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
	"testing"

	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_extensions_filters_network_http_connection_manager_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/google/go-cmp/cmp"
	"github.com/saarasio/enroute/enroute-dp/internal/protobuf"
	cfg "github.com/saarasio/enroute/enroute-dp/saarasconfig"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestStatsListener(t *testing.T) {
	tests := map[string]struct {
		address string
		port    int
		want    *envoy_config_listener_v3.Listener
	}{
		"stats-health": {
			address: "127.0.0.127",
			port:    8123,
			want: &envoy_config_listener_v3.Listener{
				Name:    "stats-health",
				Address: SocketAddress("127.0.0.127", 8123),
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
									ConfigType: &envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter_TypedConfig{
										TypedConfig: &any.Any{
											TypeUrl: cfg.HTTPFilterRouter,
										},
									},
								}},
								NormalizePath: protobuf.Bool(true),
							}),
						},
					},
				),
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := StatsListener(tc.address, tc.port)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
