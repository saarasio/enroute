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
	"sort"
	"time"

	"github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/protobuf"
)

// HTTPDefaultIdleTimeout sets the idle timeout for HTTP connections
// to 60 seconds. This is chosen as a rough default to stop idle connections
// wasting resources, without stopping slow connections from being terminated
// too quickly.
// Exported so the same value can be used here and in e2e tests.
const HTTPDefaultIdleTimeout = 60 * time.Second

// TCPDefaultIdleTimeout sets the idle timeout in seconds for
// connections through a TCP Proxy type filter.
// It's defaulted to two and a half hours for reasons documented at
// https://github.com/saarasio/enroute/enroute-dp/issues/1074
// Set to 9001 because now it's OVER NINE THOUSAND.
// Exported so the same value can be used here and in e2e tests.
const TCPDefaultIdleTimeout = 9001 * time.Second

// TLSInspector returns a new TLS inspector listener filter.
func TLSInspector() *envoy_config_listener_v3.ListenerFilter {
	return &envoy_config_listener_v3.ListenerFilter{
		Name: wellknown.TlsInspector,
	}
}

// ProxyProtocol returns a new Proxy Protocol listener filter.
func ProxyProtocol() *envoy_config_listener_v3.ListenerFilter {
	return &envoy_config_listener_v3.ListenerFilter{
		Name: wellknown.ProxyProtocol,
	}
}

// Listener returns a new envoy_config_listener_v3.Listener for the supplied address, port, and filters.
func Listener(name, address string, port int, lf []*envoy_config_listener_v3.ListenerFilter, filters ...*envoy_config_listener_v3.Filter) *envoy_config_listener_v3.Listener {
	l := &envoy_config_listener_v3.Listener{
		Name:            name,
		Address:         SocketAddress(address, port),
		ListenerFilters: lf,
	}
	if len(filters) > 0 {
		l.FilterChains = append(
			l.FilterChains,
			&envoy_config_listener_v3.FilterChain{
				Filters: filters,
			},
		)
	}
	return l
}

// HTTPConnectionManager creates a new HTTP Connection Manager filter
// for the supplied route and access log.
func HTTPConnectionManager(routename, accessLogPath string, vh *dag.Vertex) *envoy_config_listener_v3.Filter {
	return &envoy_config_listener_v3.Filter{
		Name: wellknown.HTTPConnectionManager,
		ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
			TypedConfig: toAny(&envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager{
				StatPrefix: routename,
				RouteSpecifier: &envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager_Rds{
					Rds: &envoy_extensions_filters_network_http_connection_manager_v3.Rds{
						RouteConfigName: routename,
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
				HttpFilters: httpFilters(vh),
				HttpProtocolOptions: &envoy_config_core_v3.Http1ProtocolOptions{
					// Enable support for HTTP/1.0 requests that carry
					// a Host: header. See #537.
					AcceptHttp_10: true,
				},
				AccessLog:        FileAccessLog(accessLogPath),
				UseRemoteAddress: protobuf.Bool(true),
				NormalizePath:    protobuf.Bool(true),
				CommonHttpProtocolOptions: &envoy_config_core_v3.HttpProtocolOptions{
					// Sets the idle timeout for HTTP connections to 60 seconds.
					// This is chosen as a rough default to stop idle connections wasting resources,
					// without stopping slow connections from being terminated too quickly.
					IdleTimeout: protobuf.Duration(60 * time.Second),
				},
				//RequestTimeout:   protobuf.Duration(requestTimeout),

				// issue #1487 pass through X-Request-Id if provided.
				PreserveExternalRequestId: true,
			}),
		},
	}
}

// TCPProxy creates a new TCPProxy filter.
func TCPProxy(statPrefix string, proxy *dag.TCPProxy, accessLogPath string) *envoy_config_listener_v3.Filter {
	idleTimeout := protobuf.Duration(9001 * time.Second)
	switch len(proxy.Clusters) {
	case 1:
		return &envoy_config_listener_v3.Filter{
			Name: wellknown.TCPProxy,
			ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
				TypedConfig: toAny(&envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy{
					StatPrefix: statPrefix,
					ClusterSpecifier: &envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_Cluster{
						Cluster: Clustername(proxy.Clusters[0]),
					},
					AccessLog:   FileAccessLog(accessLogPath),
					IdleTimeout: idleTimeout,
				}),
			},
		}
	default:
		var clusters []*envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_WeightedCluster_ClusterWeight
		for _, c := range proxy.Clusters {
			weight := c.Weight
			if weight == 0 {
				weight = 1
			}
			clusters = append(clusters, &envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_WeightedCluster_ClusterWeight{
				Name:   Clustername(c),
				Weight: weight,
			})
		}
		sort.Stable(clustersByNameAndWeight(clusters))
		return &envoy_config_listener_v3.Filter{
			Name: wellknown.TCPProxy,
			ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
				TypedConfig: toAny(&envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy{
					StatPrefix: statPrefix,
					ClusterSpecifier: &envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_WeightedClusters{
						WeightedClusters: &envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_WeightedCluster{
							Clusters: clusters,
						},
					},
					AccessLog:   FileAccessLog(accessLogPath),
					IdleTimeout: idleTimeout,
				}),
			},
		}
	}
}

type clustersByNameAndWeight []*envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_WeightedCluster_ClusterWeight

func (c clustersByNameAndWeight) Len() int      { return len(c) }
func (c clustersByNameAndWeight) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c clustersByNameAndWeight) Less(i, j int) bool {
	if c[i].Name == c[j].Name {
		return c[i].Weight < c[j].Weight
	}
	return c[i].Name < c[j].Name
}

// SocketAddress creates a new TCP envoy_config_core_v3.Address.
func SocketAddress(address string, port int) *envoy_config_core_v3.Address {
	if address == "::" {
		return &envoy_config_core_v3.Address{
			Address: &envoy_config_core_v3.Address_SocketAddress{
				SocketAddress: &envoy_config_core_v3.SocketAddress{
					Protocol:   envoy_config_core_v3.SocketAddress_TCP,
					Address:    address,
					Ipv4Compat: true,
					PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
						PortValue: uint32(port),
					},
				},
			},
		}
	}
	return &envoy_config_core_v3.Address{
		Address: &envoy_config_core_v3.Address_SocketAddress{
			SocketAddress: &envoy_config_core_v3.SocketAddress{
				Protocol: envoy_config_core_v3.SocketAddress_TCP,
				Address:  address,
				PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
					PortValue: uint32(port),
				},
			},
		},
	}
}

// Filters returns a []*envoy_config_listener_v3.Filter for the supplied filters.
func Filters(filters ...*envoy_config_listener_v3.Filter) []*envoy_config_listener_v3.Filter {
	if len(filters) == 0 {
		return nil
	}
	return filters
}

// FilterChain retruns a *envoy_config_listener_v3.FilterChain for the supplied filters.
func FilterChain(filters ...*envoy_config_listener_v3.Filter) *envoy_config_listener_v3.FilterChain {
	return &envoy_config_listener_v3.FilterChain{
		Filters: filters,
	}
}

// FilterChains returns a []*envoy_config_listener_v3.FilterChain for the supplied filters.
func FilterChains(filters ...*envoy_config_listener_v3.Filter) []*envoy_config_listener_v3.FilterChain {
	if len(filters) == 0 {
		return nil
	}
	return []*envoy_config_listener_v3.FilterChain{
		FilterChain(filters...),
	}
}

// FilterChainTLS returns a TLS enabled envoy_config_listener_v3.FilterChain,
func FilterChainTLS(domain string, secret *dag.Secret, filters []*envoy_config_listener_v3.Filter, tlsMinProtoVersion envoy_extensions_transport_sockets_tls_v3.TlsParameters_TlsProtocol, alpnProtos ...string) *envoy_config_listener_v3.FilterChain {
	fc := &envoy_config_listener_v3.FilterChain{
		Filters: filters,
		FilterChainMatch: &envoy_config_listener_v3.FilterChainMatch{
			ServerNames: []string{domain},
		},
	}
	// attach certificate data to this listener if provided.
	if secret != nil {
		fc.TransportSocket = DownstreamTLSTransportSocket(
			DownstreamTLSContext(Secretname(secret), tlsMinProtoVersion, alpnProtos...),
		)
	}
	return fc
}

// ListenerFilters returns a []*envoy_config_listener_v3.ListenerFilter for the supplied listener filters.
func ListenerFilters(filters ...*envoy_config_listener_v3.ListenerFilter) []*envoy_config_listener_v3.ListenerFilter {
	return filters
}

func toAny(pb proto.Message) *any.Any {
	a, err := ptypes.MarshalAny(pb)
	if err != nil {
		panic(err.Error())
	}
	return a
}
