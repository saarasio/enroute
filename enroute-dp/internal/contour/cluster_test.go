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
	//"os"

	"github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/prometheus/client_golang/prometheus"
	gatewayhostv1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1beta1"
	"github.com/saarasio/enroute/enroute-dp/internal/assert"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/envoy"
	//"github.com/saarasio/enroute/enroute-dp/internal/debug"
	"github.com/saarasio/enroute/enroute-dp/internal/metrics"
	"github.com/saarasio/enroute/enroute-dp/internal/protobuf"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestClusterCacheContents(t *testing.T) {
	tests := map[string]struct {
		contents map[string]*envoy_config_cluster_v3.Cluster
		want     []proto.Message
	}{
		"empty": {
			contents: nil,
			want:     nil,
		},
		"simple": {
			contents: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/kuard/443/da39a3ee5e",
					AltStatName:          "default_kuard_443",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/kuard",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				}),
			want: []proto.Message{
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/kuard/443/da39a3ee5e",
					AltStatName:          "default_kuard_443",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/kuard",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var cc ClusterCache
			cc.Update(tc.contents)
			got := cc.Contents()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestClusterCacheQuery(t *testing.T) {
	tests := map[string]struct {
		contents map[string]*envoy_config_cluster_v3.Cluster
		query    []string
		want     []proto.Message
	}{
		"exact match": {
			contents: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/kuard/443/da39a3ee5e",
					AltStatName:          "default_kuard_443",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/kuard",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				}),
			query: []string{"default/kuard/443/da39a3ee5e"},
			want: []proto.Message{
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/kuard/443/da39a3ee5e",
					AltStatName:          "default_kuard_443",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/kuard",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				},
			},
		},
		"partial match": {
			contents: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/kuard/443/da39a3ee5e",
					AltStatName:          "default_kuard_443",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/kuard",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				}),
			query: []string{"default/kuard/443/da39a3ee5e", "foo/bar/baz"},
			want: []proto.Message{
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/kuard/443/da39a3ee5e",
					AltStatName:          "default_kuard_443",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/kuard",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				},
			},
		},
		"no match": {
			contents: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/kuard/443/da39a3ee5e",
					AltStatName:          "default_kuard_443",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/kuard",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				}),
			query: []string{"foo/bar/baz"},
			want:  nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var cc ClusterCache
			cc.Update(tc.contents)
			got := cc.Query(tc.query)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestClusterVisit(t *testing.T) {
	tests := map[string]struct {
		objs []interface{}
		want map[string]*envoy_config_cluster_v3.Cluster
	}{
		"nothing": {
			objs: nil,
			want: map[string]*envoy_config_cluster_v3.Cluster{},
		},
		"single unnamed service": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromInt(443),
						},
					},
				},
				service("default", "kuard",
					v1.ServicePort{
						Protocol:   "TCP",
						Port:       443,
						TargetPort: intstr.FromInt(8443),
					},
				),
			},
			want: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/kuard/443/da39a3ee5e",
					AltStatName:          "default_kuard_443",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/kuard",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				}),
		},
		"single named service": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromString("https"),
						},
					},
				},
				service("default", "kuard",
					v1.ServicePort{
						Name:       "https",
						Protocol:   "TCP",
						Port:       443,
						TargetPort: intstr.FromInt(8443),
					},
				),
			},
			want: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/kuard/443/da39a3ee5e",
					AltStatName:          "default_kuard_443",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/kuard/https",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				}),
		},
		"h2c upstream": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromString("http"),
						},
					},
				},
				serviceWithAnnotations(
					"default",
					"kuard",
					map[string]string{
						"enroute.saaras.io/upstream-protocol.h2c": "80,http",
					},
					v1.ServicePort{
						Protocol: "TCP",
						Name:     "http",
						Port:     80,
					},
				),
			},
			want: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/kuard/80/da39a3ee5e",
					AltStatName:          "default_kuard_80",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/kuard/http",
					},
					ConnectTimeout:       protobuf.Duration(250 * time.Millisecond),
					LbPolicy:             envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					Http2ProtocolOptions: &envoy_config_core_v3.Http2ProtocolOptions{},
					CommonLbConfig:       envoy.ClusterCommonLBConfig(),
				},
			),
		},
		"long namespace and service name": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "webserver-1-unimatrix-zero-one",
						Namespace: "beurocratic-company-test-domain-1",
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "tiny-cog-department-test-instance",
							ServicePort: intstr.FromInt(443),
						},
					},
				},
				service("beurocratic-company-test-domain-1", "tiny-cog-department-test-instance",
					v1.ServicePort{
						Name:       "svc-0",
						Protocol:   "TCP",
						Port:       443,
						TargetPort: intstr.FromInt(8443),
					},
				),
			},
			want: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "beurocra-7fe4b4/tiny-cog-7fe4b4/443/da39a3ee5e",
					AltStatName:          "beurocratic-company-test-domain-1_tiny-cog-department-test-instance_443",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "beurocratic-company-test-domain-1/tiny-cog-department-test-instance/svc-0",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				}),
		},
		"two service ports": {
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
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/",
							}},
							Services: []gatewayhostv1.Service{{
								Name: "backend",
								Port: 80,
							}, {
								Name: "backend",
								Port: 8080,
							}},
						}},
					},
				},
				service("default", "backend", v1.ServicePort{
					Name:       "http",
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromInt(6502),
				}, v1.ServicePort{
					Name:       "alt",
					Protocol:   "TCP",
					Port:       8080,
					TargetPort: intstr.FromString("9001"),
				}),
			},
			want: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/backend/80/da39a3ee5e",
					AltStatName:          "default_backend_80",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/backend/http",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				},
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/backend/8080/da39a3ee5e",
					AltStatName:          "default_backend_8080",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/backend/alt",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				},
			),
		},
		"gatewayhost with simple path healthcheck": {
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
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/",
							}},
							Services: []gatewayhostv1.Service{{
								Name: "backend",
								Port: 80,
								HealthCheck: &gatewayhostv1.HealthCheck{
									Path: "/healthy",
								},
							}},
						}},
					},
				},
				service("default", "backend", v1.ServicePort{
					Name:       "http",
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromInt(6502),
				}),
			},
			want: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/backend/80/c184349821",
					AltStatName:          "default_backend_80",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/backend/http",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					HealthChecks: []*envoy_config_core_v3.HealthCheck{{
						Timeout:            &duration.Duration{Seconds: 2},
						Interval:           &duration.Duration{Seconds: 10},
						UnhealthyThreshold: protobuf.UInt32(3),
						HealthyThreshold:   protobuf.UInt32(2),
						HealthChecker: &envoy_config_core_v3.HealthCheck_HttpHealthCheck_{
							HttpHealthCheck: &envoy_config_core_v3.HealthCheck_HttpHealthCheck{
								Path: "/healthy",
								Host: "contour-envoy-healthcheck",
							},
						},
					}},
					CommonLbConfig:                      envoy.ClusterCommonLBConfig(),
					CloseConnectionsOnHostHealthFailure: true,
				},
			),
		},
		"gatewayhost with custom healthcheck": {
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
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/",
							}},
							Services: []gatewayhostv1.Service{{
								Name: "backend",
								Port: 80,
								HealthCheck: &gatewayhostv1.HealthCheck{
									Host:                    "foo-bar-host",
									Path:                    "/healthy",
									TimeoutSeconds:          99,
									IntervalSeconds:         98,
									UnhealthyThresholdCount: 97,
									HealthyThresholdCount:   96,
								},
							}},
						}},
					},
				},
				service("default", "backend", v1.ServicePort{
					Name:       "http",
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromInt(6502),
				}),
			},
			want: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/backend/80/7f8051653a",
					AltStatName:          "default_backend_80",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/backend/http",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					HealthChecks: []*envoy_config_core_v3.HealthCheck{{
						Timeout:            &duration.Duration{Seconds: 99},
						Interval:           &duration.Duration{Seconds: 98},
						UnhealthyThreshold: protobuf.UInt32(97),
						HealthyThreshold:   protobuf.UInt32(96),
						HealthChecker: &envoy_config_core_v3.HealthCheck_HttpHealthCheck_{
							HttpHealthCheck: &envoy_config_core_v3.HealthCheck_HttpHealthCheck{
								Path: "/healthy",
								Host: "foo-bar-host",
							},
						},
					}},
					CommonLbConfig:                      envoy.ClusterCommonLBConfig(),
					CloseConnectionsOnHostHealthFailure: true,
				},
			),
		},
		"gatewayhost with RoundRobin lb algorithm": {
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
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/",
							}},
							Services: []gatewayhostv1.Service{{
								Name:     "backend",
								Port:     80,
								Strategy: "RoundRobin",
							}},
						}},
					},
				},
				service("default", "backend", v1.ServicePort{
					Name:       "http",
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromInt(6502),
				}),
			},
			want: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/backend/80/f3b72af6a9",
					AltStatName:          "default_backend_80",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/backend/http",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				},
			),
		},
		"gatewayhost with WeightedLeastRequest lb algorithm": {
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
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/",
							}},
							Services: []gatewayhostv1.Service{{
								Name:     "backend",
								Port:     80,
								Strategy: "WeightedLeastRequest",
							}},
						}},
					},
				},
				service("default", "backend", v1.ServicePort{
					Name:       "http",
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromInt(6502),
				}),
			},
			want: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/backend/80/8bf87fefba",
					AltStatName:          "default_backend_80",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/backend/http",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_LEAST_REQUEST,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				},
			),
		},
		"gatewayhost with Random lb algorithm": {
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
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/",
							}},
							Services: []gatewayhostv1.Service{{
								Name:     "backend",
								Port:     80,
								Strategy: "Random",
							}},
						}},
					},
				},
				service("default", "backend", v1.ServicePort{
					Name:       "http",
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromInt(6502),
				}),
			},
			want: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/backend/80/58d888c08a",
					AltStatName:          "default_backend_80",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/backend/http",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_RANDOM,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				},
			),
		},
		"gatewayhost with differing lb algorithms": {
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
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/a",
							}},
							Services: []gatewayhostv1.Service{{
								Name:     "backend",
								Port:     80,
								Strategy: "Random",
							}},
						}, {
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/b",
							}},
							Services: []gatewayhostv1.Service{{
								Name:     "backend",
								Port:     80,
								Strategy: "WeightedLeastRequest",
							}},
						}},
					},
				},
				service("default", "backend", v1.ServicePort{
					Name:       "http",
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromInt(6502),
				}),
			},
			want: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/backend/80/58d888c08a",
					AltStatName:          "default_backend_80",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/backend/http",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_RANDOM,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				},
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/backend/80/8bf87fefba",
					AltStatName:          "default_backend_80",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/backend/http",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_LEAST_REQUEST,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				},
			),
		},
		"gatewayhost with unknown lb algorithm": {
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
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/",
							}},
							Services: []gatewayhostv1.Service{{
								Name:     "backend",
								Port:     80,
								Strategy: "lulz",
							}},
						}},
					},
				},
				service("default", "backend", v1.ServicePort{
					Name:       "http",
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromInt(6502),
				}),
			},
			want: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/backend/80/86d7a9c129",
					AltStatName:          "default_backend_80",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/backend/http",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				},
			),
		},
		"circuitbreaker annotations": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromString("http"),
						},
					},
				},
				serviceWithAnnotations(
					"default",
					"kuard",
					map[string]string{
						"enroute.saaras.io/max-connections":      "9000",
						"enroute.saaras.io/max-pending-requests": "4096",
						"enroute.saaras.io/max-requests":         "404",
						"enroute.saaras.io/max-retries":          "7",
					},
					v1.ServicePort{
						Protocol: "TCP",
						Name:     "http",
						Port:     80,
					},
				),
			},
			want: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/kuard/80/da39a3ee5e",
					AltStatName:          "default_kuard_80",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/kuard/http",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					CircuitBreakers: &envoy_config_cluster_v3.CircuitBreakers{
						Thresholds: []*envoy_config_cluster_v3.CircuitBreakers_Thresholds{{
							MaxConnections:     protobuf.UInt32(9000),
							MaxPendingRequests: protobuf.UInt32(4096),
							MaxRequests:        protobuf.UInt32(404),
							MaxRetries:         protobuf.UInt32(7),
						}},
					},
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
				},
			),
		},
		"enroute.saaras.io/num-retries annotation": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
						Annotations: map[string]string{
							"enroute.saaras.io/num-retries": "7",
							"enroute.saaras.io/retry-on":    "gateway-error",
						},
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromString("https"),
						},
					},
				},
				service("default", "kuard",
					v1.ServicePort{
						Name:       "https",
						Protocol:   "TCP",
						Port:       443,
						TargetPort: intstr.FromInt(8443),
					},
				),
			},
			want: clustermap(
				&envoy_config_cluster_v3.Cluster{
					Name:                 "default/kuard/443/da39a3ee5e",
					AltStatName:          "default_kuard_443",
					ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
					EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
						EdsConfig:   envoy.ConfigSource("enroute"),
						ServiceName: "default/kuard/https",
					},
					ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
					LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
					CommonLbConfig: envoy.ClusterCommonLBConfig(),
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
				reh.OnAdd(o)
			}
			root := dag.BuildDAG(&reh.KubernetesCache)

			//dw := &debug.DotWriter{
			//    Kc: &reh.KubernetesCache,
			//}
			//dw.WriteDot(os.Stderr)

			got := visitClusters(root)
			assert.Equal(t, tc.want, got)
		})
	}
}

func service(ns, name string, ports ...v1.ServicePort) *v1.Service {
	return serviceWithAnnotations(ns, name, nil, ports...)
}

func serviceWithAnnotations(ns, name string, annotations map[string]string, ports ...v1.ServicePort) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
			Annotations: annotations,
		},
		Spec: v1.ServiceSpec{
			Ports: ports,
		},
	}
}

func cluster(c *envoy_config_cluster_v3.Cluster) *envoy_config_cluster_v3.Cluster {
	// NOTE: Keep this in sync with envoy.defaultCluster().
	defaults := &envoy_config_cluster_v3.Cluster{
		ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
		CommonLbConfig: envoy.ClusterCommonLBConfig(),
		LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
	}

	proto.Merge(defaults, c)
	return defaults
}

func clustermap(clusters ...*envoy_config_cluster_v3.Cluster) map[string]*envoy_config_cluster_v3.Cluster {
	m := make(map[string]*envoy_config_cluster_v3.Cluster)
	for _, c := range clusters {
		m[c.Name] = cluster(c)
	}
	return m
}
