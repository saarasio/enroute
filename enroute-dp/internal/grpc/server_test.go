// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

// Copyright © 2017 Heptio
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

package grpc

import (
	"context"
	"io/ioutil"
	"net"
	"testing"
	"time"

	envoy_service_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	envoy_service_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	envoy_service_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	envoy_service_route_v3 "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	envoy_service_secret_v3 "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"

	resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/saarasio/enroute/enroute-dp/internal/contour"
	"github.com/saarasio/enroute/enroute-dp/internal/metrics"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestGRPC(t *testing.T) {
	// tr and et is recreated before the start of each test.
	var et *contour.EndpointsTranslator
	var reh *contour.ResourceEventHandler

	tests := map[string]func(*testing.T, *grpc.ClientConn){
		"StreamClusters": func(t *testing.T, cc *grpc.ClientConn) {
			reh.OnAdd(&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "simple",
					Namespace: "default",
				},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{
						"app": "simple",
					},
					Ports: []corev1.ServicePort{{
						Protocol:   "TCP",
						Port:       80,
						TargetPort: intstr.FromInt(6502),
					}},
				},
			}, false)

			sds := envoy_service_cluster_v3.NewClusterDiscoveryServiceClient(cc)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			stream, err := sds.StreamClusters(ctx)
			check(t, err)
			sendreq(t, stream, resource.ClusterType) // send initial notification
			checkrecv(t, stream)                     // check we receive one notification
			checktimeout(t, stream)                  // check that the second receive times out
		},
		"StreamEndpoints": func(t *testing.T, cc *grpc.ClientConn) {
			et.OnAdd(&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kube-scheduler",
					Namespace: "kube-system",
				},
				Subsets: []corev1.EndpointSubset{{
					Addresses: []corev1.EndpointAddress{{
						IP: "130.211.139.167",
					}},
					Ports: []corev1.EndpointPort{{
						Port: 80,
					}, {
						Port: 443,
					}},
				}},
			}, false)

			eds := envoy_service_endpoint_v3.NewEndpointDiscoveryServiceClient(cc)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			stream, err := eds.StreamEndpoints(ctx)
			check(t, err)
			sendreq(t, stream, resource.EndpointType) // send initial notification
			checkrecv(t, stream)                      // check we receive one notification
			checktimeout(t, stream)                   // check that the second receive times out
		},
		"StreamListeners": func(t *testing.T, cc *grpc.ClientConn) {
			// add an ingress, which will create a non tls listener
			reh.OnAdd(&netv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "httpbin-org",
					Namespace: "default",
				},
				Spec: netv1.IngressSpec{
					Rules: []netv1.IngressRule{{
						Host: "httpbin.org",
						IngressRuleValue: netv1.IngressRuleValue{
							HTTP: &netv1.HTTPIngressRuleValue{
								Paths: []netv1.HTTPIngressPath{{

									Backend: netv1.IngressBackend{
										Service: &netv1.IngressServiceBackend{
											Name: "httpbin-org",
											Port: netv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								}},
							},
						},
					}},
				},
			}, false)

			lds := envoy_service_listener_v3.NewListenerDiscoveryServiceClient(cc)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			stream, err := lds.StreamListeners(ctx)
			check(t, err)
			sendreq(t, stream, resource.ListenerType) // send initial notification
			checkrecv(t, stream)                      // check we receive one notification
			checktimeout(t, stream)                   // check that the second receive times out
		},
		"StreamRoutes": func(t *testing.T, cc *grpc.ClientConn) {
			reh.OnAdd(&netv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "httpbin-org",
					Namespace: "default",
				},
				Spec: netv1.IngressSpec{
					Rules: []netv1.IngressRule{{
						Host: "httpbin.org",
						IngressRuleValue: netv1.IngressRuleValue{
							HTTP: &netv1.HTTPIngressRuleValue{
								Paths: []netv1.HTTPIngressPath{{
									Backend: netv1.IngressBackend{
										Service: &netv1.IngressServiceBackend{
											Name: "httpbin-org",
											Port: netv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								}},
							},
						},
					}},
				},
			}, false)

			rds := envoy_service_route_v3.NewRouteDiscoveryServiceClient(cc)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			stream, err := rds.StreamRoutes(ctx)
			check(t, err)
			sendreq(t, stream, resource.RouteType) // send initial notification
			checkrecv(t, stream)                   // check we receive one notification
			checktimeout(t, stream)                // check that the second receive times out
		},
		"StreamSecrets": func(t *testing.T, cc *grpc.ClientConn) {
			reh.OnAdd(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					corev1.TLSCertKey:       []byte("certificate"),
					corev1.TLSPrivateKeyKey: []byte("key"),
				},
			}, false)

			sds := envoy_service_secret_v3.NewSecretDiscoveryServiceClient(cc)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			stream, err := sds.StreamSecrets(ctx)
			check(t, err)
			sendreq(t, stream, resource.SecretType) // send initial notification
			checkrecv(t, stream)                    // check we receive one notification
			checktimeout(t, stream)                 // check that the second receive times out
		},
	}

	log := logrus.New()
	log.SetOutput(ioutil.Discard)
	for name, fn := range tests {
		t.Run(name, func(t *testing.T) {
			et = &contour.EndpointsTranslator{
				FieldLogger: log,
			}
			ch := contour.CacheHandler{
				Metrics: metrics.NewMetrics(prometheus.NewRegistry()),
			}
			reh = &contour.ResourceEventHandler{
				Notifier:    &ch,
				Metrics:     ch.Metrics,
				FieldLogger: log,
			}
			srv := NewAPI(log, map[string]Resource{
				ch.ClusterCache.TypeURL():  &ch.ClusterCache,
				ch.RouteCache.TypeURL():    &ch.RouteCache,
				ch.ListenerCache.TypeURL(): &ch.ListenerCache,
				ch.SecretCache.TypeURL():   &ch.SecretCache,
				et.TypeURL():               et,
			})
			l, err := net.Listen("tcp", "127.0.0.1:0")
			check(t, err)
			done := make(chan error, 1)
			go func() {
				done <- srv.Serve(l)
			}()
			defer func() {
				srv.Stop()
				<-done
			}()
			cc, err := grpc.Dial(l.Addr().String(), grpc.WithInsecure())
			check(t, err)
			defer cc.Close()
			fn(t, cc)
		})
	}
}

func check(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func sendreq(t *testing.T, stream interface {
	Send(*envoy_service_discovery_v3.DiscoveryRequest) error
}, typeurl string) {
	t.Helper()
	err := stream.Send(&envoy_service_discovery_v3.DiscoveryRequest{
		TypeUrl: typeurl,
	})
	check(t, err)
}

func checkrecv(t *testing.T, stream interface {
	Recv() (*envoy_service_discovery_v3.DiscoveryResponse, error)
}) {
	t.Helper()
	_, err := stream.Recv()
	check(t, err)
}

func checktimeout(t *testing.T, stream interface {
	Recv() (*envoy_service_discovery_v3.DiscoveryResponse, error)
}) {
	t.Helper()
	_, err := stream.Recv()
	if err == nil {
		t.Fatal("expected timeout")
	}
	s, ok := status.FromError(err)
	if !ok {
		t.Fatalf("%T %v", err, err)
	}

	// Work around grpc/grpc-go#1645 which sometimes seems to
	// set the status code to Unknown, even when the message is derived from context.DeadlineExceeded.
	if s.Code() != codes.DeadlineExceeded && s.Message() != context.DeadlineExceeded.Error() {
		t.Fatalf("expected %q, got %q %T %v", codes.DeadlineExceeded, s.Code(), err, err)
	}
}
