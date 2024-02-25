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

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	gatewayhostv1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1"
	"github.com/saarasio/enroute/enroute-dp/internal/assert"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/envoy"
	"github.com/saarasio/enroute/enroute-dp/internal/metrics"
	"google.golang.org/protobuf/testing/protocmp"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListenerCacheContents(t *testing.T) {
	tests := map[string]struct {
		contents map[string]*envoy_config_listener_v3.Listener
		want     []proto.Message
	}{
		"empty": {
			contents: nil,
			want:     nil,
		},
		"simple": {
			contents: listenermap(&envoy_config_listener_v3.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, DEFAULT_HTTP_ACCESS_LOG, nil)),
			}),
			want: []proto.Message{
				&envoy_config_listener_v3.Listener{
					Name:         ENVOY_HTTP_LISTENER,
					Address:      envoy.SocketAddress("0.0.0.0", 8080),
					FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, DEFAULT_HTTP_ACCESS_LOG, nil)),
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var lc ListenerCache
			lc.Update(tc.contents)
			got := lc.Contents()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestListenerCacheQuery(t *testing.T) {
	tests := map[string]struct {
		contents map[string]*envoy_config_listener_v3.Listener
		query    []string
		want     []proto.Message
	}{
		"exact match": {
			contents: listenermap(&envoy_config_listener_v3.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, DEFAULT_HTTP_ACCESS_LOG, nil)),
			}),
			query: []string{ENVOY_HTTP_LISTENER},
			want: []proto.Message{
				&envoy_config_listener_v3.Listener{
					Name:         ENVOY_HTTP_LISTENER,
					Address:      envoy.SocketAddress("0.0.0.0", 8080),
					FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, DEFAULT_HTTP_ACCESS_LOG, nil)),
				},
			},
		},
		"partial match": {
			contents: listenermap(&envoy_config_listener_v3.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, DEFAULT_HTTP_ACCESS_LOG, nil)),
			}),
			query: []string{ENVOY_HTTP_LISTENER, "stats-listener"},
			want: []proto.Message{
				&envoy_config_listener_v3.Listener{
					Name:         ENVOY_HTTP_LISTENER,
					Address:      envoy.SocketAddress("0.0.0.0", 8080),
					FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, DEFAULT_HTTP_ACCESS_LOG, nil)),
				},
			},
		},
		"no match": {
			contents: listenermap(&envoy_config_listener_v3.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, DEFAULT_HTTP_ACCESS_LOG, nil)),
			}),
			query: []string{"stats-listener"},
			want:  nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var lc ListenerCache
			lc.Update(tc.contents)
			got := lc.Query(tc.query)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestListenerVisit(t *testing.T) {
	tests := map[string]struct {
		ListenerVisitorConfig
		objs []interface{}
		want map[string]*envoy_config_listener_v3.Listener
	}{
		"nothing": {
			objs: nil,
			want: map[string]*envoy_config_listener_v3.Listener{},
		},
		"one http only ingress": {
			objs: []interface{}{
				&netv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: netv1.IngressSpec{
						DefaultBackend: &netv1.IngressBackend{
							Service: &netv1.IngressServiceBackend{
								Name: "kuard",
								Port: netv1.ServiceBackendPort{
									Number: 8080,
								},
							},
						},
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&envoy_config_listener_v3.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, DEFAULT_HTTP_ACCESS_LOG, nil)),
			}),
		},
		"one http only gatewayhost": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "www.example.com",
						},
						Routes: []gatewayhostv1.Route{{
							Services: []gatewayhostv1.Service{
								{
									Name: "backend",
									Port: 80,
								},
							},
						}},
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "backend",
						Namespace: "default",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     80,
						}},
					},
				},
			},
			want: listenermap(&envoy_config_listener_v3.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, DEFAULT_HTTP_ACCESS_LOG, nil)),
			}),
		},
		"simple ingress with secret": {
			objs: []interface{}{
				&netv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: netv1.IngressSpec{
						TLS: []netv1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						DefaultBackend: &netv1.IngressBackend{
							Service: &netv1.IngressServiceBackend{
								Name: "kuard",
								Port: netv1.ServiceBackendPort{
									Number: 8080,
								},
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata("certificate", "key"),
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&envoy_config_listener_v3.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, DEFAULT_HTTP_ACCESS_LOG, nil)),
			}, &envoy_config_listener_v3.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
				FilterChains: []*envoy_config_listener_v3.FilterChain{{
					FilterChainMatch: &envoy_config_listener_v3.FilterChainMatch{
						ServerNames: []string{"whatever.example.com"},
					},
					TransportSocket: transportSocket(envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_1, "h2", "http/1.1"),
					Filters:         envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, DEFAULT_HTTPS_ACCESS_LOG, nil)),
				}},
			}),
		},
		"multiple tls ingress with secrets should be sorted": {
			objs: []interface{}{
				&netv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sortedsecond",
						Namespace: "default",
					},
					Spec: netv1.IngressSpec{
						TLS: []netv1.IngressTLS{{
							Hosts:      []string{"sortedsecond.example.com"},
							SecretName: "secret",
						}},
						DefaultBackend: &netv1.IngressBackend{
							Service: &netv1.IngressServiceBackend{
								Name: "kuard",
								Port: netv1.ServiceBackendPort{
									Number: 8080,
								},
							},
						},
					},
				},
				&netv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sortedfirst",
						Namespace: "default",
					},
					Spec: netv1.IngressSpec{
						TLS: []netv1.IngressTLS{{
							Hosts:      []string{"sortedfirst.example.com"},
							SecretName: "secret",
						}},
						DefaultBackend: &netv1.IngressBackend{
							Service: &netv1.IngressServiceBackend{
								Name: "kuard",
								Port: netv1.ServiceBackendPort{
									Number: 8080,
								},
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata("certificate", "key"),
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&envoy_config_listener_v3.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, DEFAULT_HTTP_ACCESS_LOG, nil)),
			}, &envoy_config_listener_v3.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
				FilterChains: []*envoy_config_listener_v3.FilterChain{
					{
						FilterChainMatch: &envoy_config_listener_v3.FilterChainMatch{
							ServerNames: []string{"sortedfirst.example.com"},
						},
						TransportSocket: transportSocket(envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_1, "h2", "http/1.1"),
						Filters:         envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, DEFAULT_HTTPS_ACCESS_LOG, nil)),
					},
					{
						FilterChainMatch: &envoy_config_listener_v3.FilterChainMatch{
							ServerNames: []string{"sortedsecond.example.com"},
						},
						TransportSocket: transportSocket(envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_1, "h2", "http/1.1"),
						Filters:         envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, DEFAULT_HTTPS_ACCESS_LOG, nil)),
					},
				},
			}),
		},
		"simple ingress with missing secret": {
			objs: []interface{}{
				&netv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: netv1.IngressSpec{
						TLS: []netv1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "missing",
						}},
						DefaultBackend: &netv1.IngressBackend{
							Service: &netv1.IngressServiceBackend{
								Name: "kuard",
								Port: netv1.ServiceBackendPort{
									Number: 8080,
								},
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata("certificate", "key"),
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&envoy_config_listener_v3.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, DEFAULT_HTTP_ACCESS_LOG, nil)),
			}),
		},
		"simple gatewayhost with secret": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "www.example.com",
							TLS: &gatewayhostv1.TLS{
								SecretName: "secret",
							},
						},
						Routes: []gatewayhostv1.Route{
							{
								Services: []gatewayhostv1.Service{
									{
										Name: "backend",
										Port: 80,
									},
								},
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata("certificate", "key"),
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "backend",
						Namespace: "default",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     80,
						}},
					},
				},
			},
			want: listenermap(&envoy_config_listener_v3.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, DEFAULT_HTTP_ACCESS_LOG, nil)),
			}, &envoy_config_listener_v3.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				FilterChains: []*envoy_config_listener_v3.FilterChain{{
					FilterChainMatch: &envoy_config_listener_v3.FilterChainMatch{
						ServerNames: []string{"www.example.com"},
					},
					TransportSocket: transportSocket(envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_1, "h2", "http/1.1"),
					Filters:         envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, DEFAULT_HTTPS_ACCESS_LOG, nil)),
				}},
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
			}),
		},
		"ingress with allow-http: false": {
			objs: []interface{}{
				&netv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
						Annotations: map[string]string{
							"kubernetes.io/ingress.allow-http": "false",
						},
					},
					Spec: netv1.IngressSpec{
						DefaultBackend: &netv1.IngressBackend{
							Service: &netv1.IngressServiceBackend{
								Name: "kuard",
								Port: netv1.ServiceBackendPort{
									Number: 8080,
								},
							},
						},
					},
				},
			},
			want: map[string]*envoy_config_listener_v3.Listener{},
		},
		"simple tls ingress with allow-http:false": {
			objs: []interface{}{
				&netv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
						Annotations: map[string]string{
							"kubernetes.io/ingress.allow-http": "false",
						},
					},
					Spec: netv1.IngressSpec{
						TLS: []netv1.IngressTLS{{
							Hosts:      []string{"www.example.com"},
							SecretName: "secret",
						}},
						DefaultBackend: &netv1.IngressBackend{
							Service: &netv1.IngressServiceBackend{
								Name: "kuard",
								Port: netv1.ServiceBackendPort{
									Number: 8080,
								},
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata("certificate", "key"),
				},
			},
			want: listenermap(&envoy_config_listener_v3.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				FilterChains: []*envoy_config_listener_v3.FilterChain{{
					FilterChainMatch: &envoy_config_listener_v3.FilterChainMatch{
						ServerNames: []string{"www.example.com"},
					},
					TransportSocket: transportSocket(envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_1, "h2", "http/1.1"),
					Filters:         envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, DEFAULT_HTTPS_ACCESS_LOG, nil)),
				}},
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
			}),
		},
		"http listener on non default port": { // issue 72
			ListenerVisitorConfig: ListenerVisitorConfig{
				HTTPAddress:  "127.0.0.100",
				HTTPPort:     9100,
				HTTPSAddress: "127.0.0.200",
				HTTPSPort:    9200,
			},
			objs: []interface{}{
				&netv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: netv1.IngressSpec{
						TLS: []netv1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						DefaultBackend: &netv1.IngressBackend{
							Service: &netv1.IngressServiceBackend{
								Name: "kuard",
								Port: netv1.ServiceBackendPort{
									Number: 8080,
								},
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata("certificate", "key"),
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&envoy_config_listener_v3.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("127.0.0.100", 9100),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, DEFAULT_HTTP_ACCESS_LOG, nil)),
			}, &envoy_config_listener_v3.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress("127.0.0.200", 9200),
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
				FilterChains: []*envoy_config_listener_v3.FilterChain{{
					FilterChainMatch: &envoy_config_listener_v3.FilterChainMatch{
						ServerNames: []string{"whatever.example.com"},
					},
					TransportSocket: transportSocket(envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_1, "h2", "http/1.1"),
					Filters:         envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, DEFAULT_HTTPS_ACCESS_LOG, nil)),
				}},
			}),
		},
		"use proxy proto": {
			ListenerVisitorConfig: ListenerVisitorConfig{
				UseProxyProto: true,
			},
			objs: []interface{}{
				&netv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: netv1.IngressSpec{
						TLS: []netv1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						DefaultBackend: &netv1.IngressBackend{
							Service: &netv1.IngressServiceBackend{
								Name: "kuard",
								Port: netv1.ServiceBackendPort{
									Number: 8080,
								},
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata("certificate", "key"),
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&envoy_config_listener_v3.Listener{
				Name:    ENVOY_HTTP_LISTENER,
				Address: envoy.SocketAddress("0.0.0.0", 8080),
				ListenerFilters: envoy.ListenerFilters(
					envoy.ProxyProtocol(),
				),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, DEFAULT_HTTP_ACCESS_LOG, nil)),
			}, &envoy_config_listener_v3.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				ListenerFilters: envoy.ListenerFilters(
					envoy.ProxyProtocol(),
					envoy.TLSInspector(),
				),
				FilterChains: []*envoy_config_listener_v3.FilterChain{{
					FilterChainMatch: &envoy_config_listener_v3.FilterChainMatch{
						ServerNames: []string{"whatever.example.com"},
					},
					TransportSocket: transportSocket(envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_1, "h2", "http/1.1"),
					Filters:         envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, DEFAULT_HTTPS_ACCESS_LOG, nil)),
				}},
			}),
		},
		"--envoy-http-access-log": {
			ListenerVisitorConfig: ListenerVisitorConfig{
				HTTPAccessLog:  "/tmp/http_access.log",
				HTTPSAccessLog: "/tmp/https_access.log",
			},
			objs: []interface{}{
				&netv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: netv1.IngressSpec{
						TLS: []netv1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						DefaultBackend: &netv1.IngressBackend{
							Service: &netv1.IngressServiceBackend{
								Name: "kuard",
								Port: netv1.ServiceBackendPort{
									Number: 8080,
								},
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata("certificate", "key"),
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&envoy_config_listener_v3.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress(DEFAULT_HTTP_LISTENER_ADDRESS, DEFAULT_HTTP_LISTENER_PORT),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, "/tmp/http_access.log", nil)),
			}, &envoy_config_listener_v3.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress(DEFAULT_HTTPS_LISTENER_ADDRESS, DEFAULT_HTTPS_LISTENER_PORT),
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
				FilterChains: []*envoy_config_listener_v3.FilterChain{{
					FilterChainMatch: &envoy_config_listener_v3.FilterChainMatch{
						ServerNames: []string{"whatever.example.com"},
					},
					TransportSocket: transportSocket(envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_1, "h2", "http/1.1"),
					Filters:         envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, "/tmp/https_access.log", nil)),
				}},
			}),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			reh := ResourceEventHandler{
				FieldLogger: testLogger(t),
				Notifier:    new(nullNotifier),
				Metrics:     metrics.NewMetrics(prometheus.NewRegistry()),
			}
			for _, o := range tc.objs {
				reh.OnAdd(o, false)
			}
			root := dag.BuildDAG(&reh.KubernetesCache)
			got := visitListeners(root, &tc.ListenerVisitorConfig)
			if !cmp.Equal(tc.want, got, protocmp.Transform()) {
				t.Fatalf("expected:\n%+v\ngot:\n%+v", tc.want, got)
			}
		})
	}
}

func transportSocket(tlsMinProtoVersion envoy_extensions_transport_sockets_tls_v3.TlsParameters_TlsProtocol, alpnprotos ...string) *envoy_config_core_v3.TransportSocket {
	return envoy.DownstreamTLSTransportSocket(
		envoy.DownstreamTLSContext("default/secret/735ad571c1", tlsMinProtoVersion, alpnprotos...),
	)
}

func secretdata(cert, key string) map[string][]byte {
	return map[string][]byte{
		corev1.TLSCertKey:       []byte(cert),
		corev1.TLSPrivateKeyKey: []byte(key),
	}
}

func listenermap(listeners ...*envoy_config_listener_v3.Listener) map[string]*envoy_config_listener_v3.Listener {
	m := make(map[string]*envoy_config_listener_v3.Listener)
	for _, l := range listeners {
		m[l.Name] = l
	}
	return m
}
