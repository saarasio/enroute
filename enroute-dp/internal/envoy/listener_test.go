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

package envoy

import (
	"testing"
	"time"

	"github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"

	envoy_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_compressor_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/compressor/v3"
	http "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/google/go-cmp/cmp"
	"github.com/saarasio/enroute/enroute-dp/internal/assert"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/protobuf"
	cfg "github.com/saarasio/enroute/enroute-dp/saarasconfig"
	"google.golang.org/protobuf/testing/protocmp"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestListener(t *testing.T) {
	tests := map[string]struct {
		name, address string
		port          int
		lf            []*envoy_config_listener_v3.ListenerFilter
		f             []*envoy_config_listener_v3.Filter
		want          *envoy_config_listener_v3.Listener
	}{
		"insecure listener": {
			name:    "http",
			address: "0.0.0.0",
			port:    9000,
			f: []*envoy_config_listener_v3.Filter{
				HTTPConnectionManager("http", "/dev/null", nil),
			},
			want: &envoy_config_listener_v3.Listener{
				Name:    "http",
				Address: SocketAddress("0.0.0.0", 9000),
				FilterChains: FilterChains(
					HTTPConnectionManager("http", "/dev/null", nil),
				),
			},
		},
		"insecure listener w/ proxy": {
			name:    "http-proxy",
			address: "0.0.0.0",
			port:    9000,
			lf: []*envoy_config_listener_v3.ListenerFilter{
				ProxyProtocol(),
			},
			f: []*envoy_config_listener_v3.Filter{
				HTTPConnectionManager("http-proxy", "/dev/null", nil),
			},
			want: &envoy_config_listener_v3.Listener{
				Name:    "http-proxy",
				Address: SocketAddress("0.0.0.0", 9000),
				ListenerFilters: ListenerFilters(
					ProxyProtocol(),
				),
				FilterChains: FilterChains(
					HTTPConnectionManager("http-proxy", "/dev/null", nil),
				),
			},
		},
		"secure listener": {
			name:    "https",
			address: "0.0.0.0",
			port:    9000,
			lf: []*envoy_config_listener_v3.ListenerFilter{
				TLSInspector(),
			},
			want: &envoy_config_listener_v3.Listener{
				Name:    "https",
				Address: SocketAddress("0.0.0.0", 9000),
				ListenerFilters: ListenerFilters(
					TLSInspector(),
				),
			},
		},
		"secure listener w/ proxy": {
			name:    "https-proxy",
			address: "0.0.0.0",
			port:    9000,
			lf: []*envoy_config_listener_v3.ListenerFilter{
				ProxyProtocol(),
				TLSInspector(),
			},
			want: &envoy_config_listener_v3.Listener{
				Name:    "https-proxy",
				Address: SocketAddress("0.0.0.0", 9000),
				ListenerFilters: ListenerFilters(
					ProxyProtocol(),
					TLSInspector(),
				),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := Listener(tc.name, tc.address, tc.port, tc.lf, tc.f...)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSocketAddress(t *testing.T) {
	const (
		addr = "foo.example.com"
		port = 8123
	)

	got := SocketAddress(addr, port)
	want := &envoy_config_core_v3.Address{
		Address: &envoy_config_core_v3.Address_SocketAddress{
			SocketAddress: &envoy_config_core_v3.SocketAddress{
				Protocol: envoy_config_core_v3.SocketAddress_TCP,
				Address:  addr,
				PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
					PortValue: port,
				},
			},
		},
	}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Fatal(diff)
	}

	got = SocketAddress("::", port)
	want = &envoy_config_core_v3.Address{
		Address: &envoy_config_core_v3.Address_SocketAddress{
			SocketAddress: &envoy_config_core_v3.SocketAddress{
				Protocol:   envoy_config_core_v3.SocketAddress_TCP,
				Address:    "::",
				Ipv4Compat: true, // Set only for ipv6-any "::"
				PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
					PortValue: port,
				},
			},
		},
	}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Fatal(diff)
	}
}

func TestDownstreamTLSContext(t *testing.T) {
	const secretName = "default/tls-cert"

	got := DownstreamTLSContext(secretName, envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_1, "h2", "http/1.1")
	want := &envoy_extensions_transport_sockets_tls_v3.DownstreamTlsContext{
		CommonTlsContext: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext{
			TlsParams: &envoy_extensions_transport_sockets_tls_v3.TlsParameters{
				TlsMinimumProtocolVersion: envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_1,
				TlsMaximumProtocolVersion: envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_3,
				CipherSuites: []string{
					"[ECDHE-ECDSA-AES128-GCM-SHA256|ECDHE-ECDSA-CHACHA20-POLY1305]",
					"[ECDHE-RSA-AES128-GCM-SHA256|ECDHE-RSA-CHACHA20-POLY1305]",
					"ECDHE-ECDSA-AES128-SHA",
					"ECDHE-RSA-AES128-SHA",
					"ECDHE-ECDSA-AES256-GCM-SHA384",
					"ECDHE-RSA-AES256-GCM-SHA384",
					"ECDHE-ECDSA-AES256-SHA",
					"ECDHE-RSA-AES256-SHA",
				},
			},
			TlsCertificateSdsSecretConfigs: []*envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig{{
				Name: secretName,
				SdsConfig: &envoy_config_core_v3.ConfigSource{
					ConfigSourceSpecifier: &envoy_config_core_v3.ConfigSource_ApiConfigSource{
						ApiConfigSource: &envoy_config_core_v3.ApiConfigSource{
							ApiType: envoy_config_core_v3.ApiConfigSource_GRPC,
							GrpcServices: []*envoy_config_core_v3.GrpcService{{
								TargetSpecifier: &envoy_config_core_v3.GrpcService_EnvoyGrpc_{
									EnvoyGrpc: &envoy_config_core_v3.GrpcService_EnvoyGrpc{
										ClusterName: "enroute",
									},
								},
							}},
							TransportApiVersion: envoy_config_core_v3.ApiVersion_V3,
						},
					},
					ResourceApiVersion: envoy_config_core_v3.ApiVersion_V3,
				},
			}},
			AlpnProtocols: []string{"h2", "http/1.1"},
		},
	}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Fatal(diff)
	}
}

func TestHTTPConnectionManager(t *testing.T) {
	tests := map[string]struct {
		routename string
		accesslog string
		want      *envoy_config_listener_v3.Filter
	}{
		"default": {
			routename: "default/kuard",
			accesslog: "/dev/stdout",
			want: &envoy_config_listener_v3.Filter{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
					TypedConfig: toAny(&envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager{
						StatPrefix: "default/kuard",
						RouteSpecifier: &envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager_Rds{
							Rds: &envoy_extensions_filters_network_http_connection_manager_v3.Rds{
								RouteConfigName: "default/kuard",
								ConfigSource: &envoy_config_core_v3.ConfigSource{
									ConfigSourceSpecifier: &envoy_config_core_v3.ConfigSource_ApiConfigSource{
										ApiConfigSource: &envoy_config_core_v3.ApiConfigSource{
											ApiType: envoy_config_core_v3.ApiConfigSource_GRPC,
											GrpcServices: []*envoy_config_core_v3.GrpcService{{
												TargetSpecifier: &envoy_config_core_v3.GrpcService_EnvoyGrpc_{
													EnvoyGrpc: &envoy_config_core_v3.GrpcService_EnvoyGrpc{
														ClusterName: "enroute",
													},
												},
											}},
											TransportApiVersion: envoy_config_core_v3.ApiVersion_V3,
										},
									},
									ResourceApiVersion: envoy_config_core_v3.ApiVersion_V3,
								},
							},
						},
						HttpFilters: []*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter{{
							Name: "compressor",
							ConfigType: &http.HttpFilter_TypedConfig{
								TypedConfig: toAny(&envoy_compressor_v3.Compressor{
									CompressorLibrary: &envoy_core_v3.TypedExtensionConfig{
										Name: "gzip",
										TypedConfig: &any.Any{
											TypeUrl: cfg.HTTPFilterCompressorGzip,
										},
									},
								}),
							},
						}, {
							Name: "grpcweb",
							ConfigType: &http.HttpFilter_TypedConfig{
								TypedConfig: &any.Any{
									TypeUrl: cfg.HTTPFilterGrpcWeb,
								},
							},
						},

							//    {
							//		Name: wellknown.HTTPRateLimit,
							//		ConfigType: &envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter_TypedConfig{
							//			TypedConfig: toAny(&httprl.RateLimit{
							//				Domain: "enroute",
							//				RateLimitService: &envoy_config_ratelimit_v2.RateLimitServiceConfig{
							//					GrpcService: &envoy_config_core_v3.GrpcService{
							//						TargetSpecifier: &envoy_config_core_v3.GrpcService_EnvoyGrpc_{
							//							EnvoyGrpc: &envoy_config_core_v3.GrpcService_EnvoyGrpc{
							//								ClusterName: "enroute_ratelimit",
							//							},
							//						},
							//					},
							//				},
							//			}),
							//		},
							//	},

							{
								Name: "router",
								ConfigType: &http.HttpFilter_TypedConfig{
									TypedConfig: &any.Any{
										TypeUrl: cfg.HTTPFilterRouter,
									},
								},
							}},
						HttpProtocolOptions: &envoy_config_core_v3.Http1ProtocolOptions{
							// Enable support for HTTP/1.0 requests that carry
							// a Host: header. See #537.
							AcceptHttp_10: true,
						},
						AccessLog:        FileAccessLog("/dev/stdout"),
						UseRemoteAddress: protobuf.Bool(true),
						NormalizePath:    protobuf.Bool(true),
						CommonHttpProtocolOptions: &envoy_config_core_v3.HttpProtocolOptions{
							IdleTimeout: protobuf.Duration(60 * time.Second),
						},
						PreserveExternalRequestId: true,
					}),
				},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := HTTPConnectionManager(tc.routename, tc.accesslog, nil)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTCPProxy(t *testing.T) {
	const (
		statPrefix    = "ingress_https"
		accessLogPath = "/dev/stdout"
	)

	c1 := &dag.Cluster{
		Upstream: &dag.TCPService{
			Name:      "example",
			Namespace: "default",
			ServicePort: &v1.ServicePort{
				Protocol:   "TCP",
				Port:       443,
				TargetPort: intstr.FromInt(8443),
			},
		},
	}
	c2 := &dag.Cluster{
		Upstream: &dag.TCPService{
			Name:      "example2",
			Namespace: "default",
			ServicePort: &v1.ServicePort{
				Protocol:   "TCP",
				Port:       443,
				TargetPort: intstr.FromInt(8443),
			},
		},
		Weight: 20,
	}

	tests := map[string]struct {
		proxy *dag.TCPProxy
		want  *envoy_config_listener_v3.Filter
	}{
		"single cluster": {
			proxy: &dag.TCPProxy{
				Clusters: []*dag.Cluster{c1},
			},
			want: &envoy_config_listener_v3.Filter{
				Name: wellknown.TCPProxy,
				ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
					TypedConfig: toAny(&envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy{
						StatPrefix: statPrefix,
						ClusterSpecifier: &envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_Cluster{
							Cluster: Clustername(c1),
						},
						AccessLog:   FileAccessLog(accessLogPath),
						IdleTimeout: protobuf.Duration(9001 * time.Second),
					}),
				},
			},
		},
		"multiple cluster": {
			proxy: &dag.TCPProxy{
				Clusters: []*dag.Cluster{c2, c1},
			},
			want: &envoy_config_listener_v3.Filter{
				Name: wellknown.TCPProxy,
				ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
					TypedConfig: toAny(&envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy{
						StatPrefix: statPrefix,
						ClusterSpecifier: &envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_WeightedClusters{
							WeightedClusters: &envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_WeightedCluster{
								Clusters: []*envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_WeightedCluster_ClusterWeight{{
									Name:   Clustername(c1),
									Weight: 1,
								}, {
									Name:   Clustername(c2),
									Weight: 20,
								}},
							},
						},
						AccessLog:   FileAccessLog(accessLogPath),
						IdleTimeout: protobuf.Duration(9001 * time.Second),
					}),
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := TCPProxy(statPrefix, tc.proxy, accessLogPath)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
