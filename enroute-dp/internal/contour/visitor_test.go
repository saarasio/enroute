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

package contour

import (
	"testing"
	"time"

	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"

	"github.com/google/go-cmp/cmp"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/envoy"
	"github.com/saarasio/enroute/enroute-dp/internal/protobuf"
	"google.golang.org/protobuf/testing/protocmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestVisitClusters(t *testing.T) {
	tests := map[string]struct {
		root dag.Visitable
		want map[string]*envoy_config_cluster_v3.Cluster
	}{
		"TCPService forward": {
			root: &dag.Listener{
				Port: 443,
				VirtualHosts: virtualhosts(
					&dag.SecureVirtualHost{
						VirtualHost: dag.VirtualHost{
							Name: "www.example.com",
							TCPProxy: &dag.TCPProxy{
								Clusters: []*dag.Cluster{{
									Upstream: &dag.TCPService{
										Name:      "example",
										Namespace: "default",
										ServicePort: &v1.ServicePort{
											Protocol:   "TCP",
											Port:       443,
											TargetPort: intstr.FromInt(8443),
										},
									},
								}},
							},
						},
						Secret: new(dag.Secret),
					},
				),
			},
			want: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/example/443/da39a3ee5e",
					AltStatName:          "default_example_443",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/example",
					},
					ConnectTimeout:  protobuf.Duration(250 * time.Millisecond),
					LbPolicy:        envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					CommonLbConfig:  envoy.ClusterCommonLBConfig(),
					DnsLookupFamily: envoy_config_cluster_v3.Cluster_V4_ONLY,
				},
			),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := visitClusters(tc.root)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestVisitListeners(t *testing.T) {
	p1 := &dag.TCPProxy{
		Clusters: []*dag.Cluster{{
			Upstream: &dag.TCPService{
				Name:      "example",
				Namespace: "default",
				ServicePort: &v1.ServicePort{
					Protocol:   "TCP",
					Port:       443,
					TargetPort: intstr.FromInt(8443),
				},
			},
		}},
	}

	tests := map[string]struct {
		root dag.Visitable
		want map[string]*envoy_config_listener_v3.Listener
	}{
		"TCPService forward": {
			root: &dag.Listener{
				Port: 443,
				VirtualHosts: virtualhosts(
					&dag.SecureVirtualHost{
						VirtualHost: dag.VirtualHost{
							Name:     "tcpproxy.example.com",
							TCPProxy: p1,
						},
						Secret: &dag.Secret{
							Object: &v1.Secret{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "secret",
									Namespace: "default",
								},
								Data: secretdata("certificate", "key"),
							},
						},
						MinProtoVersion: envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_1,
					},
				),
			},
			want: listenermap(
				&envoy_config_listener_v3.Listener{
					Name:    ENVOY_HTTPS_LISTENER,
					Address: envoy.SocketAddress("0.0.0.0", 8443),
					FilterChains: []*envoy_config_listener_v3.FilterChain{{
						FilterChainMatch: &envoy_config_listener_v3.FilterChainMatch{
							ServerNames: []string{"tcpproxy.example.com"},
						},
						TransportSocket: transportSocket(envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_1),
						Filters:         envoy.Filters(envoy.TCPProxy(ENVOY_HTTPS_LISTENER, p1, DEFAULT_HTTPS_ACCESS_LOG)),
					}},
					ListenerFilters: envoy.ListenerFilters(
						envoy.TLSInspector(),
					),
				},
			),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := visitListeners(tc.root, new(ListenerVisitorConfig))
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestVisitSecrets(t *testing.T) {
	tests := map[string]struct {
		root dag.Visitable
		want map[string]*envoy_extensions_transport_sockets_tls_v3.Secret
	}{
		"TCPService forward": {
			root: &dag.Listener{
				Port: 443,
				VirtualHosts: virtualhosts(
					&dag.SecureVirtualHost{
						VirtualHost: dag.VirtualHost{
							Name: "www.example.com",
							TCPProxy: &dag.TCPProxy{
								Clusters: []*dag.Cluster{{
									Upstream: &dag.TCPService{
										Name:      "example",
										Namespace: "default",
										ServicePort: &v1.ServicePort{
											Protocol:   "TCP",
											Port:       443,
											TargetPort: intstr.FromInt(8443),
										},
									},
								}},
							},
						},
						Secret: &dag.Secret{
							Object: &v1.Secret{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "secret",
									Namespace: "default",
								},
								Data: secretdata("certificate", "key"),
							},
						},
					},
				),
			},
			want: secretmap(&envoy_extensions_transport_sockets_tls_v3.Secret{
				Name: "default/secret/735ad571c1",
				Type: &envoy_extensions_transport_sockets_tls_v3.Secret_TlsCertificate{
					TlsCertificate: &envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
						PrivateKey: &envoy_config_core_v3.DataSource{
							Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
								InlineBytes: []byte("key"),
							},
						},
						CertificateChain: &envoy_config_core_v3.DataSource{
							Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
								InlineBytes: []byte("certificate"),
							},
						},
					},
				},
			}),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := visitSecrets(tc.root)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func virtualhosts(vx ...dag.Vertex) map[string]dag.Vertex {
	m := make(map[string]dag.Vertex)
	for _, v := range vx {
		switch v := v.(type) {
		case *dag.VirtualHost:
			m[v.Name] = v
		case *dag.SecureVirtualHost:
			m[v.VirtualHost.Name] = v
		}
	}
	return m
}
