// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

// Copyright © 2018 Heptio
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package e2e provides end-to-end tests.
package e2e

import (
	"context"
	"net"
	"testing"

	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	envoy_service_secret_v3 "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"

	resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	"github.com/saarasio/enroute/enroute-dp/internal/contour"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/saarasio/enroute/enroute-dp/apis/generated/clientset/versioned/fake"
	cgrpc "github.com/saarasio/enroute/enroute-dp/internal/grpc"
	"github.com/saarasio/enroute/enroute-dp/internal/k8s"
	"github.com/saarasio/enroute/enroute-dp/internal/metrics"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	endpointType = resource.EndpointType
	clusterType  = resource.ClusterType
	routeType    = resource.RouteType
	listenerType = resource.ListenerType
	secretType   = resource.SecretType
	statsAddress = "0.0.0.0"
	statsPort    = 8002
)

type testWriter struct {
	*testing.T
}

func (t *testWriter) Write(buf []byte) (int, error) {
	t.Logf("%s", buf)
	return len(buf), nil
}

type discardWriter struct {
}

func (d *discardWriter) Write(buf []byte) (int, error) {
	return len(buf), nil
}

func setup(t *testing.T, opts ...func(*contour.ResourceEventHandler)) (cache.ResourceEventHandler, *grpc.ClientConn, func()) {
	log := logrus.New()
	log.Out = &testWriter{t}

	et := &contour.EndpointsTranslator{
		FieldLogger: log,
	}

	r := prometheus.NewRegistry()
	ch := &contour.CacheHandler{
		GatewayHostStatus: &k8s.GatewayHostStatus{
			Client: fake.NewSimpleClientset(),
		},
		Metrics:       metrics.NewMetrics(r),
		ListenerCache: contour.NewListenerCache(statsAddress, statsPort),
		FieldLogger:   log,
	}

	reh := contour.ResourceEventHandler{
		Notifier:    ch,
		Metrics:     ch.Metrics,
		FieldLogger: log,
	}

	for _, opt := range opts {
		opt(&reh)
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	check(t, err)
	discard := logrus.New()
	discard.Out = new(discardWriter)
	// Resource types in xDS envoy_service_discovery_v3.
	srv := cgrpc.NewAPI(discard, map[string]cgrpc.Resource{
		ch.ClusterCache.TypeURL():  &ch.ClusterCache,
		ch.RouteCache.TypeURL():    &ch.RouteCache,
		ch.ListenerCache.TypeURL(): &ch.ListenerCache,
		ch.SecretCache.TypeURL():   &ch.SecretCache,
		et.TypeURL():               et,
	})

	done := make(chan error, 1)
	go func() {
		done <- srv.Serve(l) // srv now owns l and will close l before returning
	}()
	cc, err := grpc.Dial(l.Addr().String(), grpc.WithInsecure())
	check(t, err)

	rh := &resourceEventHandler{
		ResourceEventHandler: &reh,
		EndpointsTranslator:  et,
	}

	return rh, cc, func() {
		// close client connection
		cc.Close()

		// stop server and wait for it to stop
		srv.Stop()

		<-done
	}
}

// resourceEventHandler composes a contour.Translator and a contour.EndpointsTranslator
// into a single ResourceEventHandler type.
type resourceEventHandler struct {
	*contour.ResourceEventHandler
	*contour.EndpointsTranslator
}

func (r *resourceEventHandler) OnAdd(obj interface{}, isInInitialList bool) {
	switch obj.(type) {
	case *v1.Endpoints:
		r.EndpointsTranslator.OnAdd(obj, false)
	default:
		r.ResourceEventHandler.OnAdd(obj, false)
	}
}

func (r *resourceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	switch newObj.(type) {
	case *v1.Endpoints:
		r.EndpointsTranslator.OnUpdate(oldObj, newObj)
	default:
		r.ResourceEventHandler.OnUpdate(oldObj, newObj)
	}
}

func (r *resourceEventHandler) OnDelete(obj interface{}) {
	switch obj.(type) {
	case *v1.Endpoints:
		r.EndpointsTranslator.OnDelete(obj)
	default:
		r.ResourceEventHandler.OnDelete(obj)
	}
}

func check(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func resources(t *testing.T, protos ...proto.Message) []*any.Any {
	t.Helper()
	var anys []*any.Any
	for _, a := range protos {
		anys = append(anys, toAny(t, a))
	}
	return anys
}

func toAny(t *testing.T, pb proto.Message) *any.Any {
	t.Helper()
	a, err := ptypes.MarshalAny(pb)
	check(t, err)
	return a
}

type grpcStream interface {
	Send(*envoy_service_discovery_v3.DiscoveryRequest) error
	Recv() (*envoy_service_discovery_v3.DiscoveryResponse, error)
}

func stream(t *testing.T, st grpcStream, req *envoy_service_discovery_v3.DiscoveryRequest) *envoy_service_discovery_v3.DiscoveryResponse {
	t.Helper()
	err := st.Send(req)
	check(t, err)
	resp, err := st.Recv()
	check(t, err)
	return resp
}

type Contour struct {
	*grpc.ClientConn
	*testing.T
}

func (c *Contour) Request(typeurl string, names ...string) *Response {
	c.Helper()
	var st grpcStream
	ctx := context.Background()
	switch typeurl {
	case secretType:
		sds := envoy_service_secret_v3.NewSecretDiscoveryServiceClient(c.ClientConn)
		sts, err := sds.StreamSecrets(ctx)
		c.check(err)
		st = sts
	default:
		c.Fatal("unknown typeURL: " + typeurl)
	}
	resp := c.sendRequest(st, &envoy_service_discovery_v3.DiscoveryRequest{
		TypeUrl:       typeurl,
		ResourceNames: names,
	})
	return &Response{
		Contour:           c,
		DiscoveryResponse: resp,
	}
}

func (c *Contour) sendRequest(stream grpcStream, req *envoy_service_discovery_v3.DiscoveryRequest) *envoy_service_discovery_v3.DiscoveryResponse {
	err := stream.Send(req)
	c.check(err)
	resp, err := stream.Recv()
	c.check(err)
	return resp
}

func (c *Contour) check(err error) {
	if err != nil {
		c.Fatal(err)
	}
}

type Response struct {
	*Contour
	*envoy_service_discovery_v3.DiscoveryResponse
}

func (r *Response) Equals(want *envoy_service_discovery_v3.DiscoveryResponse) {
	r.Helper()
	assertEqual(r.T, want, r.DiscoveryResponse)
}

func assertEqual(t *testing.T, want, got *envoy_service_discovery_v3.DiscoveryResponse) {
	t.Helper()
	m := proto.TextMarshaler{Compact: true, ExpandAny: true}
	a := m.Text(want)
	b := m.Text(got)
	if a != b {
		m := proto.TextMarshaler{
			Compact:   false,
			ExpandAny: true,
		}
		t.Fatalf("\nexpected:\n%v\ngot:\n%v", m.Text(want), m.Text(got))
	}
}
