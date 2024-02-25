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

package e2e

import (
	"context"
	"testing"
	"time"

	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_service_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	gatewayhostv1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1"
	"github.com/saarasio/enroute/enroute-dp/internal/envoy"
	"github.com/saarasio/enroute/enroute-dp/internal/protobuf"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// saarasio/enroute#186
// Cluster.ServiceName and ClusterLoadAssignment.ClusterName should not be truncated.
func TestClusterLongServiceName(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "kuard",
		},
		Spec: netv1.IngressSpec{
			DefaultBackend: &netv1.IngressBackend{
				Service: &netv1.IngressServiceBackend{
					Name: "kbujbkuhdod66gjdmwmijz8xzgsx1nkfbrloezdjiulquzk4x3p0nnvpzi8r",
					Port: netv1.ServiceBackendPort{
						Number: 8080,
					},
				},
			},
		},
	}
	rh.OnAdd(i1, false)

	rh.OnAdd(service(
		"default",
		"kbujbkuhdod66gjdmwmijz8xzgsx1nkfbrloezdjiulquzk4x3p0nnvpzi8r",
		corev1.ServicePort{
			Protocol:   "TCP",
			Port:       8080,
			TargetPort: intstr.FromInt(8080),
		},
	), false)

	// check that it's been translated correctly.
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			cluster("default/kbujbkuh-c83ceb/8080/da39a3ee5e", "default/kbujbkuhdod66gjdmwmijz8xzgsx1nkfbrloezdjiulquzk4x3p0nnvpzi8r", "default_kbujbkuhdod66gjdmwmijz8xzgsx1nkfbrloezdjiulquzk4x3p0nnvpzi8r_8080"),
		),
		TypeUrl: clusterType,
		Nonce:   "2",
	}, streamCDS(t, cc))
}

// Test adding, updating, and removing a service
// doesn't leave turds in the CDS cache.
func TestClusterAddUpdateDelete(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: netv1.IngressSpec{
			DefaultBackend: &netv1.IngressBackend{
				Service: &netv1.IngressServiceBackend{
					Name: "kuard",
					Port: netv1.ServiceBackendPort{
						Number: 80,
					},
				},
			},
		},
	}
	rh.OnAdd(i1, false)

	i2 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuarder",
			Namespace: "default",
		},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{{
				Host: "www.example.com",
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Path: "/kuarder",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "kuard",
									Port: netv1.ServiceBackendPort{
										Name: "https",
									},
								},
							},
						}},
					},
				},
			}},
		},
	}
	rh.OnAdd(i2, false)

	// s1 is a simple tcp 80 -> 8080 service.
	s1 := service("default", "kuard", corev1.ServicePort{
		Protocol:   "TCP",
		Port:       80,
		TargetPort: intstr.FromInt(8080),
	})
	rh.OnAdd(s1, false)

	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			cluster("default/kuard/80/da39a3ee5e", "default/kuard", "default_kuard_80"),
		),
		TypeUrl: clusterType,
		Nonce:   "3",
	}, streamCDS(t, cc))

	// s2 is the same as s2, but the service port has a name
	s2 := service("default", "kuard", corev1.ServicePort{
		Name:       "http",
		Protocol:   "TCP",
		Port:       80,
		TargetPort: intstr.FromInt(8080),
	})

	// replace s1 with s2
	rh.OnUpdate(s1, s2)

	// check that we get two CDS records because the port is now named.
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "4",
		Resources: resources(t,
			cluster("default/kuard/80/da39a3ee5e", "default/kuard/http", "default_kuard_80"),
		),
		TypeUrl: clusterType,
		Nonce:   "4",
	}, streamCDS(t, cc))

	// s3 is like s2, but has a second named port. The k8s spec
	// requires all ports to be named if there is more than one of them.
	s3 := service("default", "kuard",
		corev1.ServicePort{
			Name:       "http",
			Protocol:   "TCP",
			Port:       80,
			TargetPort: intstr.FromInt(8080),
		},
		corev1.ServicePort{
			Name:       "https",
			Protocol:   "TCP",
			Port:       443,
			TargetPort: intstr.FromInt(8443),
		},
	)

	// replace s2 with s3
	rh.OnUpdate(s2, s3)

	// check that we get four CDS records. Order is important
	// because the CDS cache is sorted.
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "5",
		Resources: resources(t,
			cluster("default/kuard/443/da39a3ee5e", "default/kuard/https", "default_kuard_443"),
			cluster("default/kuard/80/da39a3ee5e", "default/kuard/http", "default_kuard_80"),
		),
		TypeUrl: clusterType,
		Nonce:   "5",
	}, streamCDS(t, cc))

	// s4 is s3 with the http port removed.
	s4 := service("default", "kuard",
		corev1.ServicePort{
			Name:       "https",
			Protocol:   "TCP",
			Port:       443,
			TargetPort: intstr.FromInt(8443),
		},
	)

	// replace s3 with s4
	rh.OnUpdate(s3, s4)

	// check that we get two CDS records only, and that the 80 and http
	// records have been removed even though the service object remains.
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "6",
		Resources: resources(t,
			cluster("default/kuard/443/da39a3ee5e", "default/kuard/https", "default_kuard_443"),
		),
		TypeUrl: clusterType,
		Nonce:   "6",
	}, streamCDS(t, cc))
}

// pathological hard case, one service is removed, the other is moved to a different port, and its name removed.
func TestClusterRenameUpdateDelete(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{{
				Host: "www.example.com",
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "kuard",
									Port: netv1.ServiceBackendPort{
										Name: "http",
									},
								},
							},
						}, {
							Path: "/kuarder",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "kuard",
									Port: netv1.ServiceBackendPort{
										Number: 443,
									},
								},
							},
						}},
					},
				},
			}},
		},
	}
	rh.OnAdd(i1, false)

	s1 := service("default", "kuard",
		corev1.ServicePort{
			Name:       "http",
			Protocol:   "TCP",
			Port:       80,
			TargetPort: intstr.FromInt(8080),
		},
		corev1.ServicePort{
			Name:       "https",
			Protocol:   "TCP",
			Port:       443,
			TargetPort: intstr.FromInt(8443),
		},
	)

	rh.OnAdd(s1, false)
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			cluster("default/kuard/443/da39a3ee5e", "default/kuard/https", "default_kuard_443"),
			cluster("default/kuard/80/da39a3ee5e", "default/kuard/http", "default_kuard_80"),
		),
		TypeUrl: clusterType,
		Nonce:   "2",
	}, streamCDS(t, cc))

	// s2 removes the name on port 80, moves it to port 443 and deletes the https port
	s2 := service("default", "kuard",
		corev1.ServicePort{
			Protocol:   "TCP",
			Port:       443,
			TargetPort: intstr.FromInt(8000),
		},
	)

	rh.OnUpdate(s1, s2)
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			cluster("default/kuard/443/da39a3ee5e", "default/kuard", "default_kuard_443"),
		),
		TypeUrl: clusterType,
		Nonce:   "3",
	}, streamCDS(t, cc))

	// now replace s2 with s1 to check it works in the other direction.
	rh.OnUpdate(s2, s1)
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "4",
		Resources: resources(t,
			cluster("default/kuard/443/da39a3ee5e", "default/kuard/https", "default_kuard_443"),
			cluster("default/kuard/80/da39a3ee5e", "default/kuard/http", "default_kuard_80"),
		),
		TypeUrl: clusterType,
		Nonce:   "4",
	}, streamCDS(t, cc))

	// cleanup and check
	rh.OnDelete(s1)
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "5",
		Resources:   resources(t),
		TypeUrl:     clusterType,
		Nonce:       "5",
	}, streamCDS(t, cc))
}

// issue#243. A single unnamed service with a different numeric target port
func TestIssue243(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	t.Run("single unnamed service with a different numeric target port", func(t *testing.T) {

		i1 := &netv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kuard",
				Namespace: "default",
			},
			Spec: netv1.IngressSpec{
				DefaultBackend: &netv1.IngressBackend{
					Service: &netv1.IngressServiceBackend{
						Name: "kuard",
						Port: netv1.ServiceBackendPort{
							Number: 80,
						},
					},
				},
			},
		}
		rh.OnAdd(i1, false)
		s1 := service("default", "kuard",
			corev1.ServicePort{
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			},
		)
		rh.OnAdd(s1, false)
		assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
			VersionInfo: "2",
			Resources: resources(t,
				cluster("default/kuard/80/da39a3ee5e", "default/kuard", "default_kuard_80"),
			),
			TypeUrl: clusterType,
			Nonce:   "2",
		}, streamCDS(t, cc))
	})
}

// issue 247, a single unnamed service with a named target port
func TestIssue247(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: netv1.IngressSpec{
			DefaultBackend: &netv1.IngressBackend{
				Service: &netv1.IngressServiceBackend{
					Name: "kuard",
					Port: netv1.ServiceBackendPort{
						Number: 80,
					},
				},
			},
		},
	}
	rh.OnAdd(i1, false)

	// spec:
	//   ports:
	//   - port: 80
	//     protocol: TCP
	//     targetPort: kuard
	s1 := service("default", "kuard",
		corev1.ServicePort{
			Protocol:   "TCP",
			Port:       80,
			TargetPort: intstr.FromString("kuard"),
		},
	)
	rh.OnAdd(s1, false)
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			cluster("default/kuard/80/da39a3ee5e", "default/kuard", "default_kuard_80"),
		),
		TypeUrl: clusterType,
		Nonce:   "2",
	}, streamCDS(t, cc))
}
func TestCDSResourceFiltering(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{{
				Host: "www.example.com",
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "kuard",
									Port: netv1.ServiceBackendPort{
										Number: 80,
									},
								},
							},
						}, {
							Path: "/httpbin",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "httpbin",
									Port: netv1.ServiceBackendPort{
										Number: 8080,
									},
								},
							},
						}},
					},
				},
			}},
		},
	}
	rh.OnAdd(i1, false)

	// add two services, check that they are there
	s1 := service("default", "kuard",
		corev1.ServicePort{
			Protocol:   "TCP",
			Port:       80,
			TargetPort: intstr.FromString("kuard"),
		},
	)
	rh.OnAdd(s1, false)
	s2 := service("default", "httpbin",
		corev1.ServicePort{
			Protocol:   "TCP",
			Port:       8080,
			TargetPort: intstr.FromString("httpbin"),
		},
	)
	rh.OnAdd(s2, false)
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			// note, resources are sorted by Cluster.Name
			cluster("default/httpbin/8080/da39a3ee5e", "default/httpbin", "default_httpbin_8080"),
			cluster("default/kuard/80/da39a3ee5e", "default/kuard", "default_kuard_80"),
		),
		TypeUrl: clusterType,
		Nonce:   "3",
	}, streamCDS(t, cc))

	// assert we can filter on one resource
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			cluster("default/kuard/80/da39a3ee5e", "default/kuard", "default_kuard_80"),
		),
		TypeUrl: clusterType,
		Nonce:   "3",
	}, streamCDS(t, cc, "default/kuard/80/da39a3ee5e"))

	// assert a non matching filter returns a response with no entries.
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		TypeUrl:     clusterType,
		Nonce:       "3",
	}, streamCDS(t, cc, "default/httpbin/9000"))
}

func TestClusterCircuitbreakerAnnotations(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "kuard",
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
	}
	rh.OnAdd(i1, false)

	s1 := serviceWithAnnotations(
		"default",
		"kuard",
		map[string]string{
			"enroute.saaras.io/max-connections":      "9000",
			"enroute.saaras.io/max-pending-requests": "4096",
			"enroute.saaras.io/max-requests":         "404",
			"enroute.saaras.io/max-retries":          "7",
		},
		corev1.ServicePort{
			Protocol:   "TCP",
			Port:       8080,
			TargetPort: intstr.FromInt(8080),
		},
	)
	rh.OnAdd(s1, false)

	// check that it's been translated correctly.
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			&envoy_config_cluster_v3.Cluster{
				Name:                 "default/kuard/8080/da39a3ee5e",
				AltStatName:          "default_kuard_8080",
				ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
				EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
					EdsConfig:   envoy.ConfigSource("enroute"),
					ServiceName: "default/kuard",
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
				CommonLbConfig:  envoy.ClusterCommonLBConfig(),
				DnsLookupFamily: envoy_config_cluster_v3.Cluster_V4_ONLY,
			},
		),
		TypeUrl: clusterType,
		Nonce:   "2",
	}, streamCDS(t, cc))

	// update s1 with slightly weird values
	s2 := serviceWithAnnotations(
		"default",
		"kuard",
		map[string]string{
			"enroute.saaras.io/max-pending-requests": "9999",
			"enroute.saaras.io/max-requests":         "1e6",
			"enroute.saaras.io/max-retries":          "0",
		},
		corev1.ServicePort{
			Protocol:   "TCP",
			Port:       8080,
			TargetPort: intstr.FromInt(8080),
		},
	)
	rh.OnUpdate(s1, s2)

	// check that it's been translated correctly.
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			&envoy_config_cluster_v3.Cluster{
				Name:                 "default/kuard/8080/da39a3ee5e",
				AltStatName:          "default_kuard_8080",
				ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
				EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
					EdsConfig:   envoy.ConfigSource("enroute"),
					ServiceName: "default/kuard",
				},
				ConnectTimeout: protobuf.Duration(250 * time.Millisecond),
				LbPolicy:       envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
				CircuitBreakers: &envoy_config_cluster_v3.CircuitBreakers{
					Thresholds: []*envoy_config_cluster_v3.CircuitBreakers_Thresholds{{
						MaxPendingRequests: protobuf.UInt32(9999),
					}},
				},
				CommonLbConfig:  envoy.ClusterCommonLBConfig(),
				DnsLookupFamily: envoy_config_cluster_v3.Cluster_V4_ONLY,
			},
		),
		TypeUrl: clusterType,
		Nonce:   "3",
	}, streamCDS(t, cc))
}

// issue 581, different service parameters should generate
// a single CDS entry if they differ only in weight.
func TestClusterPerServiceParameters(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	rh.OnAdd(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}, false)

	rh.OnAdd(&gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "www.example.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/a",
				}},
				Services: []gatewayhostv1.Service{{
					Name:   "kuard",
					Port:   80,
					Weight: 90,
				}},
			}, {
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/b",
				}},
				Services: []gatewayhostv1.Service{{
					Name:   "kuard",
					Port:   80,
					Weight: 60,
				}},
			}},
		},
	}, false)

	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			cluster("default/kuard/80/da39a3ee5e", "default/kuard", "default_kuard_80"),
		),
		TypeUrl: clusterType,
		Nonce:   "2",
	}, streamCDS(t, cc))
}

// issue 581, different load balancer parameters should
// generate multiple cds entries.
func TestClusterLoadBalancerStrategyPerRoute(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	rh.OnAdd(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}, false)

	rh.OnAdd(&gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "www.example.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/a",
				}},
				Services: []gatewayhostv1.Service{{
					Name:     "kuard",
					Port:     80,
					Strategy: "Random",
				}},
			}, {
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/b",
				}},
				Services: []gatewayhostv1.Service{{
					Name:     "kuard",
					Port:     80,
					Strategy: "WeightedLeastRequest",
				}},
			}},
		},
	}, false)

	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			&envoy_config_cluster_v3.Cluster{
				Name:                 "default/kuard/80/58d888c08a",
				AltStatName:          "default_kuard_80",
				ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
				EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
					EdsConfig:   envoy.ConfigSource("enroute"),
					ServiceName: "default/kuard",
				},
				ConnectTimeout:  protobuf.Duration(250 * time.Millisecond),
				LbPolicy:        envoy_config_cluster_v3.Cluster_RANDOM,
				CommonLbConfig:  envoy.ClusterCommonLBConfig(),
				DnsLookupFamily: envoy_config_cluster_v3.Cluster_V4_ONLY,
			},
			&envoy_config_cluster_v3.Cluster{
				Name:                 "default/kuard/80/8bf87fefba",
				AltStatName:          "default_kuard_80",
				ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
				EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
					EdsConfig:   envoy.ConfigSource("enroute"),
					ServiceName: "default/kuard",
				},
				ConnectTimeout:  protobuf.Duration(250 * time.Millisecond),
				LbPolicy:        envoy_config_cluster_v3.Cluster_LEAST_REQUEST,
				CommonLbConfig:  envoy.ClusterCommonLBConfig(),
				DnsLookupFamily: envoy_config_cluster_v3.Cluster_V4_ONLY,
			},
		),
		TypeUrl: clusterType,
		Nonce:   "2",
	}, streamCDS(t, cc))

}

func TestClusterWithHealthChecks(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	rh.OnAdd(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}, false)

	rh.OnAdd(&gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "www.example.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/a",
				}},
				Services: []gatewayhostv1.Service{{
					Name:   "kuard",
					Port:   80,
					Weight: 90,
					HealthCheck: &gatewayhostv1.HealthCheck{
						Path: "/healthz",
					},
				}},
			}},
		},
	}, false)

	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			clusterWithHealthCheck("default/kuard/80/bc862a33ca", "default/kuard", "default_kuard_80", "/healthz", true),
		),
		TypeUrl: clusterType,
		Nonce:   "2",
	}, streamCDS(t, cc))
}

// Test that contour correctly recognizes the "enroute.saaras.io/upstream-protocol.tls"
// service annotation.
func TestClusterServiceTLSBackend(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: netv1.IngressSpec{
			DefaultBackend: &netv1.IngressBackend{
				Service: &netv1.IngressServiceBackend{
					Name: "kuard",
					Port: netv1.ServiceBackendPort{
						Number: 443,
					},
				},
			},
		},
	}
	rh.OnAdd(i1, false)

	s1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/upstream-protocol.tls": "securebackend",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "securebackend",
				Protocol:   "TCP",
				Port:       443,
				TargetPort: intstr.FromInt(8888),
			}},
		},
	}
	rh.OnAdd(s1, false)

	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			tlscluster("default/kuard/443/da39a3ee5e", "default/kuard/securebackend", "default_kuard_443", nil, ""),
		),
		TypeUrl: clusterType,
		Nonce:   "2",
	}, streamCDS(t, cc))
}

func TestClusterServiceTLSBackendCAValidation(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	rh.OnAdd(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/upstream-protocol.tls": "securebackend,443",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "securebackend",
				Protocol:   "TCP",
				Port:       443,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}, false)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		},
		Data: map[string][]byte{
			envoy.CACertificateKey: []byte("ca"),
		},
	}

	rh.OnAdd(secret, false)

	ir1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "www.example.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/a",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 443,
				}},
			}},
		},
	}

	rh.OnAdd(ir1, false)

	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			tlscluster(
				"default/kuard/443/da39a3ee5e",
				"default/kuard/securebackend",
				"default_kuard_443",
				nil,
				""),
		),
		TypeUrl: clusterType,
		Nonce:   "3",
	}, streamCDS(t, cc))

	ir2 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "www.example.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/a",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 443,
					UpstreamValidation: &gatewayhostv1.UpstreamValidation{
						CACertificate: "foo",
						SubjectName:   "subjname",
					},
				}},
			}},
		},
	}

	rh.OnUpdate(ir1, ir2)

	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "4",
		Resources: resources(t,
			tlscluster(
				"default/kuard/443/98c0f31c72",
				"default/kuard/securebackend",
				"default_kuard_443",
				[]byte("ca"),
				"subjname"),
		),
		TypeUrl: clusterType,
		Nonce:   "4",
	}, streamCDS(t, cc))
}

// Test processing a service type ExternalName
func TestExternalNameService(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: netv1.IngressSpec{
			DefaultBackend: &netv1.IngressBackend{
				Service: &netv1.IngressServiceBackend{
					Name: "kuard",
					Port: netv1.ServiceBackendPort{
						Number: 80,
					},
				},
			},
		},
	}
	rh.OnAdd(i1, false)

	// s1 is a simple tcp 80 -> 8080 service.
	s1 := externalnameservice("default", "kuard", "foo.io", corev1.ServicePort{
		Protocol:   "TCP",
		Port:       80,
		TargetPort: intstr.FromInt(8080),
	})
	rh.OnAdd(s1, false)

	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			externalnamecluster("default/kuard/80/da39a3ee5e", "default/kuard/", "default_kuard_80", "foo.io", 80),
		),
		TypeUrl: clusterType,
		Nonce:   "2",
	}, streamCDS(t, cc))
}

func serviceWithAnnotations(ns, name string, annotations map[string]string, ports ...corev1.ServicePort) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	}
}

func streamCDS(t *testing.T, cc *grpc.ClientConn, rn ...string) *envoy_service_discovery_v3.DiscoveryResponse {
	t.Helper()
	rds := envoy_service_cluster_v3.NewClusterDiscoveryServiceClient(cc)
	st, err := rds.StreamClusters(context.TODO())
	check(t, err)
	return stream(t, st, &envoy_service_discovery_v3.DiscoveryRequest{
		TypeUrl:       clusterType,
		ResourceNames: rn,
	})
}

func cluster(name, servicename, statName string) *envoy_config_cluster_v3.Cluster {
	return &envoy_config_cluster_v3.Cluster{
		Name:                 name,
		ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_EDS),
		AltStatName:          statName,
		EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
			EdsConfig:   envoy.ConfigSource("enroute"),
			ServiceName: servicename,
		},
		ConnectTimeout:  protobuf.Duration(250 * time.Millisecond),
		LbPolicy:        envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
		CommonLbConfig:  envoy.ClusterCommonLBConfig(),
		DnsLookupFamily: envoy_config_cluster_v3.Cluster_V4_ONLY,
	}
}

func externalnamecluster(name, servicename, statName, externalName string, port int) *envoy_config_cluster_v3.Cluster {
	return &envoy_config_cluster_v3.Cluster{
		Name:                 name,
		ClusterDiscoveryType: envoy.ClusterDiscoveryType(envoy_config_cluster_v3.Cluster_STRICT_DNS),
		AltStatName:          statName,
		ConnectTimeout:       protobuf.Duration(250 * time.Millisecond),
		LbPolicy:             envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
		CommonLbConfig:       envoy.ClusterCommonLBConfig(),
		LoadAssignment: &envoy_config_endpoint_v3.ClusterLoadAssignment{
			ClusterName: servicename,
			Endpoints: envoy.Endpoints(
				envoy.SocketAddress(externalName, port),
			),
		},
		DnsLookupFamily: envoy_config_cluster_v3.Cluster_V4_ONLY,
	}
}

func tlscluster(name, servicename, statsName string, ca []byte, subjectName string) *envoy_config_cluster_v3.Cluster {
	c := cluster(name, servicename, statsName)
	sni := ""
	c.TransportSocket = envoy.UpstreamTLSTransportSocket(
		envoy.UpstreamTLSContext(sni, ca, subjectName),
	)
	return c
}

func clusterWithHealthCheck(name, servicename, statName, healthCheckPath string, drainConnOnHostRemoval bool) *envoy_config_cluster_v3.Cluster {
	c := cluster(name, servicename, statName)
	c.HealthChecks = []*envoy_config_core_v3.HealthCheck{{
		Timeout:            protobuf.Duration(2 * time.Second),
		Interval:           protobuf.Duration(10 * time.Second),
		UnhealthyThreshold: protobuf.UInt32(3),
		HealthyThreshold:   protobuf.UInt32(2),
		HealthChecker: &envoy_config_core_v3.HealthCheck_HttpHealthCheck_{
			HttpHealthCheck: &envoy_config_core_v3.HealthCheck_HttpHealthCheck{
				Host: "contour-envoy-healthcheck",
				Path: healthCheckPath,
			},
		},
	}}
	c.CloseConnectionsOnHostHealthFailure = drainConnOnHostRemoval
	return c
}
