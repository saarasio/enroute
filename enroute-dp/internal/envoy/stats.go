// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

// Copyright © 2019 Heptio
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
	"github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/saarasio/enroute/enroute-dp/internal/protobuf"
)

// metrics on /stats.
func StatsListener(address string, port int) *envoy_config_listener_v3.Listener {
	return &envoy_config_listener_v3.Listener{
		Name:    "stats-health",
		Address: SocketAddress(address, port),
		FilterChains: FilterChains(
			&envoy_config_listener_v3.Filter{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
					TypedConfig: toAny(&envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager{
						StatPrefix: "stats",
						RouteSpecifier: &envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager_RouteConfig{
							RouteConfig: &envoy_config_route_v3.RouteConfiguration{
								VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
									Name:    "backend",
									Domains: []string{"*"},
									Routes: []*envoy_config_route_v3.Route{{
										Match: &envoy_config_route_v3.RouteMatch{
											PathSpecifier: &envoy_config_route_v3.RouteMatch_Prefix{
												Prefix: "/ready",
											},
										},
										Action: &envoy_config_route_v3.Route_Route{
											Route: &envoy_config_route_v3.RouteAction{
												ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{
													Cluster: "service-stats",
												},
											},
										},
									}, {
										Match: &envoy_config_route_v3.RouteMatch{
											PathSpecifier: &envoy_config_route_v3.RouteMatch_Prefix{
												Prefix: "/stats",
											},
										},
										Action: &envoy_config_route_v3.Route_Route{
											Route: &envoy_config_route_v3.RouteAction{
												ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{
													Cluster: "service-stats",
												},
											},
										},
									},
									},
								}},
							},
						},
						HttpFilters: []*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter{{
							Name: wellknown.Router,
						}},
						NormalizePath: protobuf.Bool(true),
					}),
				},
			},
		),
	}
}
