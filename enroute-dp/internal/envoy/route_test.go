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

	v31 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/saarasio/enroute/enroute-dp/internal/assert"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/protobuf"
	cfg "github.com/saarasio/enroute/enroute-dp/saarasconfig"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestRouteRoute(t *testing.T) {
	s1 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
	c1 := &dag.Cluster{
		Upstream: &dag.TCPService{
			Name:        s1.Name,
			Namespace:   s1.Namespace,
			ServicePort: &s1.Spec.Ports[0],
		},
	}
	c2 := &dag.Cluster{
		Upstream: &dag.TCPService{
			Name:        s1.Name,
			Namespace:   s1.Namespace,
			ServicePort: &s1.Spec.Ports[0],
		},
		LoadBalancerStrategy: "Cookie",
	}

	tests := map[string]struct {
		route *dag.Route
		want  *envoy_config_route_v3.Route_Route
	}{
		"single service": {
			route: &dag.Route{
				Clusters: []*dag.Cluster{c1},
			},
			want: &envoy_config_route_v3.Route_Route{
				Route: &envoy_config_route_v3.RouteAction{
					ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{
						Cluster: "default/kuard/8080/da39a3ee5e",
					},
				},
			},
		},
		"websocket": {
			route: &dag.Route{
				Websocket: true,
				Clusters:  []*dag.Cluster{c1},
			},
			want: &envoy_config_route_v3.Route_Route{
				Route: &envoy_config_route_v3.RouteAction{
					ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{
						Cluster: "default/kuard/8080/da39a3ee5e",
					},
					UpgradeConfigs: []*envoy_config_route_v3.RouteAction_UpgradeConfig{{
						UpgradeType: "websocket",
					}},
				},
			},
		},
		"multiple": {
			route: &dag.Route{
				Clusters: []*dag.Cluster{{
					Upstream: &dag.TCPService{
						Name:        s1.Name,
						Namespace:   s1.Namespace,
						ServicePort: &s1.Spec.Ports[0],
					},
					Weight: 90,
				}, {
					Upstream: &dag.TCPService{
						Name:        s1.Name,
						Namespace:   s1.Namespace, // it's valid to mention the same service several times per route.
						ServicePort: &s1.Spec.Ports[0],
					},
				}},
			},
			want: &envoy_config_route_v3.Route_Route{
				Route: &envoy_config_route_v3.RouteAction{
					ClusterSpecifier: &envoy_config_route_v3.RouteAction_WeightedClusters{
						WeightedClusters: &envoy_config_route_v3.WeightedCluster{
							Clusters: []*envoy_config_route_v3.WeightedCluster_ClusterWeight{{
								Name:   "default/kuard/8080/da39a3ee5e",
								Weight: protobuf.UInt32(0),
							}, {
								Name:   "default/kuard/8080/da39a3ee5e",
								Weight: protobuf.UInt32(90),
							}},
							TotalWeight: protobuf.UInt32(90),
						},
					},
				},
			},
		},
		"multiple websocket": {
			route: &dag.Route{
				Websocket: true,
				Clusters: []*dag.Cluster{{
					Upstream: &dag.TCPService{
						Name:        s1.Name,
						Namespace:   s1.Namespace,
						ServicePort: &s1.Spec.Ports[0],
					},
					Weight: 90,
				}, {
					Upstream: &dag.TCPService{
						Name:        s1.Name,
						Namespace:   s1.Namespace, // it's valid to mention the same service several times per route.
						ServicePort: &s1.Spec.Ports[0],
					},
				}},
			},
			want: &envoy_config_route_v3.Route_Route{
				Route: &envoy_config_route_v3.RouteAction{
					ClusterSpecifier: &envoy_config_route_v3.RouteAction_WeightedClusters{
						WeightedClusters: &envoy_config_route_v3.WeightedCluster{
							Clusters: []*envoy_config_route_v3.WeightedCluster_ClusterWeight{{
								Name:   "default/kuard/8080/da39a3ee5e",
								Weight: protobuf.UInt32(0),
							}, {
								Name:   "default/kuard/8080/da39a3ee5e",
								Weight: protobuf.UInt32(90),
							}},
							TotalWeight: protobuf.UInt32(90),
						},
					},
					UpgradeConfigs: []*envoy_config_route_v3.RouteAction_UpgradeConfig{{
						UpgradeType: "websocket",
					}},
				},
			},
		},
		"single service without retry-on": {
			route: &dag.Route{
				RetryPolicy: &dag.RetryPolicy{
					NumRetries:    7,                // ignored
					PerTryTimeout: 10 * time.Second, // ignored
				},
				Clusters: []*dag.Cluster{c1},
			},
			want: &envoy_config_route_v3.Route_Route{
				Route: &envoy_config_route_v3.RouteAction{
					ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{
						Cluster: "default/kuard/8080/da39a3ee5e",
					},
				},
			},
		},
		"retry-on: 503": {
			route: &dag.Route{
				RetryPolicy: &dag.RetryPolicy{
					RetryOn:       "503",
					NumRetries:    6,
					PerTryTimeout: 100 * time.Millisecond,
				},
				Clusters: []*dag.Cluster{c1},
			},
			want: &envoy_config_route_v3.Route_Route{
				Route: &envoy_config_route_v3.RouteAction{
					ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{
						Cluster: "default/kuard/8080/da39a3ee5e",
					},
					RetryPolicy: &envoy_config_route_v3.RetryPolicy{
						RetryOn:       "503",
						NumRetries:    protobuf.UInt32(6),
						PerTryTimeout: protobuf.Duration(100 * time.Millisecond),
					},
				},
			},
		},
		"timeout 90s": {
			route: &dag.Route{
				TimeoutPolicy: &dag.TimeoutPolicy{
					Timeout: 90 * time.Second,
				},
				Clusters: []*dag.Cluster{c1},
			},
			want: &envoy_config_route_v3.Route_Route{
				Route: &envoy_config_route_v3.RouteAction{
					ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{
						Cluster: "default/kuard/8080/da39a3ee5e",
					},
					Timeout: protobuf.Duration(90 * time.Second),
				},
			},
		},
		"timeout infinity": {
			route: &dag.Route{
				TimeoutPolicy: &dag.TimeoutPolicy{
					Timeout: -1,
				},
				Clusters: []*dag.Cluster{c1},
			},
			want: &envoy_config_route_v3.Route_Route{
				Route: &envoy_config_route_v3.RouteAction{
					ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{
						Cluster: "default/kuard/8080/da39a3ee5e",
					},
					Timeout: protobuf.Duration(0),
				},
			},
		},
		// TODO 6-4-2020 Bring in IdleTimeout/ResponseTimeout changes
		//		"idle timeout 10m": {
		//			route: &dag.Route{
		//				TimeoutPolicy: &dag.TimeoutPolicy{
		//					IdleTimeout: 10 * time.Minute,
		//				},
		//				Clusters: []*dag.Cluster{c1},
		//			},
		//			want: &envoy_config_route_v3.Route_Route{
		//				Route: &envoy_config_route_v3.RouteAction{
		//					ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{
		//						Cluster: "default/kuard/8080/da39a3ee5e",
		//					},
		//					IdleTimeout: protobuf.Duration(600 * time.Second),
		//				},
		//			},
		//		},
		//		"idle timeout infinity": {
		//			route: &dag.Route{
		//				TimeoutPolicy: &dag.TimeoutPolicy{
		//					IdleTimeout: -1,
		//				},
		//				Clusters: []*dag.Cluster{c1},
		//			},
		//			want: &envoy_config_route_v3.Route_Route{
		//				Route: &envoy_config_route_v3.RouteAction{
		//					ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{
		//						Cluster: "default/kuard/8080/da39a3ee5e",
		//					},
		//					IdleTimeout: protobuf.Duration(0),
		//				},
		//			},
		//		},

		"single service w/ session affinity": {
			route: &dag.Route{
				Clusters: []*dag.Cluster{c2},
			},
			want: &envoy_config_route_v3.Route_Route{
				Route: &envoy_config_route_v3.RouteAction{
					ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{
						Cluster: "default/kuard/8080/e4f81994fe",
					},
					HashPolicy: []*envoy_config_route_v3.RouteAction_HashPolicy{{
						PolicySpecifier: &envoy_config_route_v3.RouteAction_HashPolicy_Cookie_{
							Cookie: &envoy_config_route_v3.RouteAction_HashPolicy_Cookie{
								Name: "X-Contour-Session-Affinity",
								Ttl:  protobuf.Duration(0),
								Path: "/",
							},
						},
					}},
				},
			},
		},
		"multiple service w/ session affinity": {
			route: &dag.Route{
				Clusters: []*dag.Cluster{c2, c2},
			},
			want: &envoy_config_route_v3.Route_Route{
				Route: &envoy_config_route_v3.RouteAction{
					ClusterSpecifier: &envoy_config_route_v3.RouteAction_WeightedClusters{
						WeightedClusters: &envoy_config_route_v3.WeightedCluster{
							Clusters: []*envoy_config_route_v3.WeightedCluster_ClusterWeight{{
								Name:   "default/kuard/8080/e4f81994fe",
								Weight: protobuf.UInt32(1),
							}, {
								Name:   "default/kuard/8080/e4f81994fe",
								Weight: protobuf.UInt32(1),
							}},
							TotalWeight: protobuf.UInt32(2),
						},
					},
					HashPolicy: []*envoy_config_route_v3.RouteAction_HashPolicy{{
						PolicySpecifier: &envoy_config_route_v3.RouteAction_HashPolicy_Cookie_{
							Cookie: &envoy_config_route_v3.RouteAction_HashPolicy_Cookie{
								Name: "X-Contour-Session-Affinity",
								Ttl:  protobuf.Duration(0),
								Path: "/",
							},
						},
					}},
				},
			},
		},
		"mixed service w/ session affinity": {
			route: &dag.Route{
				Clusters: []*dag.Cluster{c2, c1},
			},
			want: &envoy_config_route_v3.Route_Route{
				Route: &envoy_config_route_v3.RouteAction{
					ClusterSpecifier: &envoy_config_route_v3.RouteAction_WeightedClusters{
						WeightedClusters: &envoy_config_route_v3.WeightedCluster{
							Clusters: []*envoy_config_route_v3.WeightedCluster_ClusterWeight{{
								Name:   "default/kuard/8080/da39a3ee5e",
								Weight: protobuf.UInt32(1),
							}, {
								Name:   "default/kuard/8080/e4f81994fe",
								Weight: protobuf.UInt32(1),
							}},
							TotalWeight: protobuf.UInt32(2),
						},
					},
					HashPolicy: []*envoy_config_route_v3.RouteAction_HashPolicy{{
						PolicySpecifier: &envoy_config_route_v3.RouteAction_HashPolicy_Cookie_{
							Cookie: &envoy_config_route_v3.RouteAction_HashPolicy_Cookie{
								Name: "X-Contour-Session-Affinity",
								Ttl:  protobuf.Duration(0),
								Path: "/",
							},
						},
					}},
				},
			},
		},
		"host rewrite literal replace": {
			route: &dag.Route{
				RouteFilters: []*dag.RouteFilter{
					&dag.RouteFilter{
						Filter: dag.Filter{
							Filter_name: "host-rewrite",
							Filter_type: cfg.FILTER_TYPE_RT_HOST_REWRITE,
							Filter_config: `{
								"substitution" : "newhost.com"
							}`,
						},
					},
				},
				Clusters: []*dag.Cluster{c1},
			},
			want: &envoy_config_route_v3.Route_Route{
				Route: &envoy_config_route_v3.RouteAction{
					HostRewriteSpecifier: &envoy_config_route_v3.RouteAction_HostRewriteLiteral{
						HostRewriteLiteral: "newhost.com",
					},
					ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{
						Cluster: "default/kuard/8080/da39a3ee5e",
					},
				},
			},
		},
		"host rewrite regex replace": {
			route: &dag.Route{
				RouteFilters: []*dag.RouteFilter{
					&dag.RouteFilter{
						Filter: dag.Filter{
							Filter_name: "host-rewrite-regex-replace",
							Filter_type: cfg.FILTER_TYPE_RT_HOST_REWRITE,
							Filter_config: `{
								"pattern_regex" : "^/(.+)/.+$",
								"substitution" : "\\1"
							}`,
						},
					},
				},
				Clusters: []*dag.Cluster{c1},
			},
			want: &envoy_config_route_v3.Route_Route{
				Route: &envoy_config_route_v3.RouteAction{
					HostRewriteSpecifier: &envoy_config_route_v3.RouteAction_HostRewritePathRegex{
						HostRewritePathRegex: &v31.RegexMatchAndSubstitute{
							Pattern: &v31.RegexMatcher{
								EngineType: &v31.RegexMatcher_GoogleRe2{},
								Regex:      "^/(.+)/.+$",
							},
							Substitution: `\1`,
						},
					},
					ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{
						Cluster: "default/kuard/8080/da39a3ee5e",
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := RouteRoute(tc.route)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRouteRedirects(t *testing.T) {
	s1 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
	c1 := &dag.Cluster{
		Upstream: &dag.TCPService{
			Name:        s1.Name,
			Namespace:   s1.Namespace,
			ServicePort: &s1.Spec.Ports[0],
		},
	}
	tests := map[string]struct {
		route *dag.Route
		want  *envoy_config_route_v3.Route
	}{
		"redirect response": {
			route: &dag.Route{
				RouteFilters: []*dag.RouteFilter{
					&dag.RouteFilter{
						Filter: dag.Filter{
							Filter_name: "redirect-response",
							Filter_type: cfg.FILTER_TYPE_RT_REDIRECT,
							Filter_config: `{
								"response_code" : 302,
								"prefix_rewrite" : "/rewrite"
							}`,
						},
					},
				},
				Clusters: []*dag.Cluster{c1},
			},
			want: &envoy_config_route_v3.Route{
				Action: &envoy_config_route_v3.Route_Redirect{
					Redirect: &envoy_config_route_v3.RedirectAction{
						ResponseCode: envoy_config_route_v3.RedirectAction_FOUND,
						PathRewriteSpecifier: &envoy_config_route_v3.RedirectAction_PrefixRewrite{
							PrefixRewrite: "/rewrite",
						},
					},
				},
			},
		},
	}
	var rr envoy_config_route_v3.Route
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			SetupRouteRedirects(tc.route, &rr)
			assert.Equal(t, tc.want, rr)
		})
	}
}

func TestWeightedClusters(t *testing.T) {
	tests := map[string]struct {
		clusters []*dag.Cluster
		want     *envoy_config_route_v3.WeightedCluster
	}{
		"multiple services w/o weights": {
			clusters: []*dag.Cluster{{
				Upstream: &dag.TCPService{
					Name:      "kuard",
					Namespace: "default",
					ServicePort: &v1.ServicePort{
						Port: 8080,
					},
				},
			}, {
				Upstream: &dag.TCPService{
					Name:      "nginx",
					Namespace: "default",
					ServicePort: &v1.ServicePort{
						Port: 8080,
					},
				},
			}},
			want: &envoy_config_route_v3.WeightedCluster{
				Clusters: []*envoy_config_route_v3.WeightedCluster_ClusterWeight{{
					Name:   "default/kuard/8080/da39a3ee5e",
					Weight: protobuf.UInt32(1),
				}, {
					Name:   "default/nginx/8080/da39a3ee5e",
					Weight: protobuf.UInt32(1),
				}},
				TotalWeight: protobuf.UInt32(2),
			},
		},
		"multiple weighted services": {
			clusters: []*dag.Cluster{{
				Upstream: &dag.TCPService{
					Name:      "kuard",
					Namespace: "default",
					ServicePort: &v1.ServicePort{
						Port: 8080,
					},
				},
				Weight: 80,
			}, {
				Upstream: &dag.TCPService{
					Name:      "nginx",
					Namespace: "default",
					ServicePort: &v1.ServicePort{
						Port: 8080,
					},
				},
				Weight: 20,
			}},
			want: &envoy_config_route_v3.WeightedCluster{
				Clusters: []*envoy_config_route_v3.WeightedCluster_ClusterWeight{{
					Name:   "default/kuard/8080/da39a3ee5e",
					Weight: protobuf.UInt32(80),
				}, {
					Name:   "default/nginx/8080/da39a3ee5e",
					Weight: protobuf.UInt32(20),
				}},
				TotalWeight: protobuf.UInt32(100),
			},
		},
		"multiple weighted services and one with no weight specified": {
			clusters: []*dag.Cluster{{
				Upstream: &dag.TCPService{
					Name:      "kuard",
					Namespace: "default",
					ServicePort: &v1.ServicePort{
						Port: 8080,
					},
				},
				Weight: 80,
			}, {
				Upstream: &dag.TCPService{
					Name:      "nginx",
					Namespace: "default",
					ServicePort: &v1.ServicePort{
						Port: 8080,
					},
				},
				Weight: 20,
			}, {
				Upstream: &dag.TCPService{
					Name:      "notraffic",
					Namespace: "default",
					ServicePort: &v1.ServicePort{
						Port: 8080,
					},
				},
			}},
			want: &envoy_config_route_v3.WeightedCluster{
				Clusters: []*envoy_config_route_v3.WeightedCluster_ClusterWeight{{
					Name:   "default/kuard/8080/da39a3ee5e",
					Weight: protobuf.UInt32(80),
				}, {
					Name:   "default/nginx/8080/da39a3ee5e",
					Weight: protobuf.UInt32(20),
				}, {
					Name:   "default/notraffic/8080/da39a3ee5e",
					Weight: protobuf.UInt32(0),
				}},
				TotalWeight: protobuf.UInt32(100),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := weightedClusters(tc.clusters)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRouteConfiguration(t *testing.T) {
	tests := map[string]struct {
		name         string
		virtualhosts []*envoy_config_route_v3.VirtualHost
		want         *envoy_config_route_v3.RouteConfiguration
	}{

		"empty": {
			name: "ingress_http",
			want: &envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_http",
				RequestHeadersToAdd: []*envoy_config_core_v3.HeaderValueOption{{
					Header: &envoy_config_core_v3.HeaderValue{
						Key:   "x-request-start",
						Value: "t=%START_TIME(%s.%3f)%",
					},
					Append: protobuf.Bool(true),
				}},
			},
		},
		"one virtualhost": {
			name: "ingress_https",
			virtualhosts: virtualhosts(
				VirtualHost("www.example.com"),
			),
			want: &envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_https",
				VirtualHosts: virtualhosts(
					VirtualHost("www.example.com"),
				),
				RequestHeadersToAdd: []*envoy_config_core_v3.HeaderValueOption{{
					Header: &envoy_config_core_v3.HeaderValue{
						Key:   "x-request-start",
						Value: "t=%START_TIME(%s.%3f)%",
					},
					Append: protobuf.Bool(true),
				}},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := RouteConfiguration(tc.name, tc.virtualhosts...)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestVirtualHost(t *testing.T) {
	tests := map[string]struct {
		hostname string
		port     int
		want     *envoy_config_route_v3.VirtualHost
	}{
		"default hostname": {
			hostname: "*",
			port:     9999,
			want: &envoy_config_route_v3.VirtualHost{
				Name:    "*",
				Domains: []string{"*"},
			},
		},
		"www.example.com": {
			hostname: "www.example.com",
			port:     9999,
			want: &envoy_config_route_v3.VirtualHost{
				Name:    "www.example.com",
				Domains: []string{"www.example.com", "www.example.com:*"},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := VirtualHost(tc.hostname)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestUpgradeHTTPS(t *testing.T) {
	got := UpgradeHTTPS()
	want := &envoy_config_route_v3.Route_Redirect{
		Redirect: &envoy_config_route_v3.RedirectAction{
			SchemeRewriteSpecifier: &envoy_config_route_v3.RedirectAction_HttpsRedirect{
				HttpsRedirect: true,
			},
		},
	}

	assert.Equal(t, want, got)
}

func TestRouteMatchNew(t *testing.T) {
	tests := map[string]struct {
		route *dag.Route
		want  *envoy_config_route_v3.RouteMatch
	}{
		"contains match with dashes": {
			route: &dag.Route{
				HeaderConditions: []dag.HeaderCondition{{
					Name:      "x-header",
					Value:     "11-22-33-44",
					MatchType: "contains",
					Invert:    false,
				}},
			},
			want: &envoy_config_route_v3.RouteMatch{
				Headers: []*envoy_config_route_v3.HeaderMatcher{{
					Name:        "x-header",
					InvertMatch: false,
					HeaderMatchSpecifier: &envoy_config_route_v3.HeaderMatcher_SafeRegexMatch{
						SafeRegexMatch: SafeRegexMatch(".*11-22-33-44.*"),
					},
				}},
			},
		},
		"contains match with dots": {
			route: &dag.Route{
				HeaderConditions: []dag.HeaderCondition{{
					Name:      "x-header",
					Value:     "11.22.33.44",
					MatchType: "contains",
					Invert:    false,
				}},
			},
			want: &envoy_config_route_v3.RouteMatch{
				Headers: []*envoy_config_route_v3.HeaderMatcher{{
					Name:        "x-header",
					InvertMatch: false,
					HeaderMatchSpecifier: &envoy_config_route_v3.HeaderMatcher_SafeRegexMatch{
						SafeRegexMatch: SafeRegexMatch(".*11\\.22\\.33\\.44.*"),
					},
				}},
			},
		},
		"contains match with regex group": {
			route: &dag.Route{
				HeaderConditions: []dag.HeaderCondition{{
					Name:      "x-header",
					Value:     "11.[22].*33.44",
					MatchType: "contains",
					Invert:    false,
				}},
			},
			want: &envoy_config_route_v3.RouteMatch{
				Headers: []*envoy_config_route_v3.HeaderMatcher{{
					Name:        "x-header",
					InvertMatch: false,
					HeaderMatchSpecifier: &envoy_config_route_v3.HeaderMatcher_SafeRegexMatch{
						SafeRegexMatch: SafeRegexMatch(".*11\\.\\[22\\]\\.\\*33\\.44.*"),
					},
				}},
			},
		},
		"path prefix": {
			route: &dag.Route{
				PathCondition: &dag.PrefixCondition{
					Prefix: "/foo",
				},
			},
			want: &envoy_config_route_v3.RouteMatch{
				PathSpecifier: &envoy_config_route_v3.RouteMatch_Prefix{
					Prefix: "/foo",
				},
			},
		},
		"path regex": {
			route: &dag.Route{
				PathCondition: &dag.RegexCondition{
					Regex: "/v.1/*",
				},
			},
			want: &envoy_config_route_v3.RouteMatch{
				PathSpecifier: &envoy_config_route_v3.RouteMatch_SafeRegex{
					// note, unlike header conditions this is not a quoted regex because
					// the value comes directly from the Ingress.Paths.Path value which
					// is permitted to be a bare regex.
					SafeRegex: SafeRegexMatch("/v.1/*"),
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := RouteMatchNew(tc.route)
			assert.Equal(t, tc.want, got)
		})
	}
}

func virtualhosts(v ...*envoy_config_route_v3.VirtualHost) []*envoy_config_route_v3.VirtualHost {
	return v
}
