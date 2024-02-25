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

// End to ends tests for translator to grpc operations.
package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	envoy_service_route_v3 "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	envoy_type_matcher_v3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"

	gatewayhostv1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1"
	"github.com/saarasio/enroute/enroute-dp/apis/generated/clientset/versioned/fake"
	"github.com/saarasio/enroute/enroute-dp/internal/contour"
	"github.com/saarasio/enroute/enroute-dp/internal/envoy"
	"github.com/saarasio/enroute/enroute-dp/internal/k8s"
	"github.com/saarasio/enroute/enroute-dp/internal/protobuf"
	cfg "github.com/saarasio/enroute/enroute-dp/saarasconfig"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// saarasio/enroute#172. Updating an object from
//
// apiVersion: networking/v1
// kind: Ingress
// metadata:
//   name: kuard
// spec:
//   backend:
//     serviceName: kuard
//     servicePort: 80
//
// to
//
// apiVersion: networking/v1
// kind: Ingress
// metadata:
//   name: kuard
// spec:
//   rules:
//   - http:
//       paths:
//       - path: /testing
//         backend:
//           serviceName: kuard
//           servicePort: 80
//
// fails to update the virtualhost cache.
func TestEditIngress(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	meta := metav1.ObjectMeta{Name: "kuard", Namespace: "default"}

	s1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
	rh.OnAdd(s1, false)

	// add default/kuard to translator.
	old := &netv1.Ingress{
		ObjectMeta: meta,
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
	rh.OnAdd(old, false)

	// check that it's been translated correctly.
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_http",
				VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
					Name:    "*",
					Domains: []string{"*"},
					Routes: []*envoy_config_route_v3.Route{{
						Match:               envoy.RouteMatch("/"),
						Action:              routecluster("default/kuard/80/da39a3ee5e"),
						RequestHeadersToAdd: envoy.RouteHeaders(),
					}},
				}},
			},
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_https",
			},
		),
		TypeUrl: routeType,
		Nonce:   "2",
	}, streamRDS(t, cc))

	// update old to new
	rh.OnUpdate(old, &netv1.Ingress{
		ObjectMeta: meta,
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{{
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Path: "/testing",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "kuard",
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
	})

	// check that ingress_http has been updated.
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_http",
				VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
					Name:    "*",
					Domains: []string{"*"},
					Routes: []*envoy_config_route_v3.Route{{
						Match:               envoy.RouteMatch("/testing"),
						Action:              routecluster("default/kuard/80/da39a3ee5e"),
						RequestHeadersToAdd: envoy.RouteHeaders(),
					}},
				}},
			},
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_https",
			},
		),
		TypeUrl: routeType,
		Nonce:   "3",
	}, streamRDS(t, cc))
}

// saarasio/enroute#101
// The path /hello should point to default/hello/80 on "*"
//
// apiVersion: networking/v1
// kind: Ingress
// metadata:
//   name: hello
// spec:
//   rules:
//   - http:
// 	 paths:
//       - path: /hello
//         backend:
//           serviceName: hello
//           servicePort: 80
func TestIngressPathRouteWithoutHost(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	// add default/hello to translator.
	rh.OnAdd(&netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "hello", Namespace: "default"},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{{
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Path: "/hello",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "hello",
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

	s1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
	rh.OnAdd(s1, false)

	// check that it's been translated correctly.
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_http",
				VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
					Name:    "*",
					Domains: []string{"*"},
					Routes: []*envoy_config_route_v3.Route{{
						Match:               envoy.RouteMatch("/hello"),
						Action:              routecluster("default/hello/80/da39a3ee5e"),
						RequestHeadersToAdd: envoy.RouteHeaders(),
					}},
				}},
			},
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_https",
			},
		),
		TypeUrl: routeType,
		Nonce:   "2",
	}, streamRDS(t, cc))
}

func TestEditIngressInPlace(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "hello", Namespace: "default"},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{{
				Host: "hello.example.com",
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Path: "/",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "wowie",
									Port: netv1.ServiceBackendPort{
										Name: "http",
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

	s1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wowie",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
	rh.OnAdd(s1, false)

	s2 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kerpow",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       9000,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
	rh.OnAdd(s2, false)

	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_http",
				VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
					Name:    "hello.example.com",
					Domains: domains("hello.example.com"),
					Routes: []*envoy_config_route_v3.Route{{
						Match:               envoy.RouteMatch("/"),
						Action:              routecluster("default/wowie/80/da39a3ee5e"),
						RequestHeadersToAdd: envoy.RouteHeaders(),
					}},
				}},
			},
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_https",
			},
		),
		TypeUrl: routeType,
		Nonce:   "3",
	}, streamRDS(t, cc))

	// i2 is like i1 but adds a second route
	i2 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "hello", Namespace: "default"},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{{
				Host: "hello.example.com",
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Path: "/",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "wowie",
									Port: netv1.ServiceBackendPort{
										Number: 80,
									},
								},
							},
						}, {
							Path: "/whoop",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "kerpow",
									Port: netv1.ServiceBackendPort{
										Number: 9000,
									},
								},
							},
						}},
					},
				},
			}},
		},
	}
	rh.OnUpdate(i1, i2)
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "4",
		Resources: resources(t,
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_http",
				VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
					Name:    "hello.example.com",
					Domains: domains("hello.example.com"),
					Routes: []*envoy_config_route_v3.Route{{
						Match:               envoy.RouteMatch("/whoop"),
						Action:              routecluster("default/kerpow/9000/da39a3ee5e"),
						RequestHeadersToAdd: envoy.RouteHeaders(),
					}, {
						Match:               envoy.RouteMatch("/"),
						Action:              routecluster("default/wowie/80/da39a3ee5e"),
						RequestHeadersToAdd: envoy.RouteHeaders(),
					}},
				}},
			},
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_https",
			},
		),
		TypeUrl: routeType,
		Nonce:   "4",
	}, streamRDS(t, cc))

	// i3 is like i2, but adds the ingress.kubernetes.io/force-ssl-redirect: "true" annotation
	i3 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: "default",
			Annotations: map[string]string{
				"ingress.kubernetes.io/force-ssl-redirect": "true"},
		},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{{
				Host: "hello.example.com",
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Path: "/",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "wowie",
									Port: netv1.ServiceBackendPort{
										Number: 80,
									},
								},
							},
						}, {
							Path: "/whoop",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "kerpow",
									Port: netv1.ServiceBackendPort{
										Number: 9000,
									},
								},
							},
						}},
					},
				},
			}},
		},
	}
	rh.OnUpdate(i2, i3)
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "5",
		Resources: resources(t,
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_http",
				VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
					Name:    "hello.example.com",
					Domains: domains("hello.example.com"),
					Routes: []*envoy_config_route_v3.Route{{
						Match:  envoy.RouteMatch("/whoop"),
						Action: envoy.UpgradeHTTPS(),
					}, {
						Match:  envoy.RouteMatch("/"),
						Action: envoy.UpgradeHTTPS(),
					}},
				}},
			},
			&envoy_config_route_v3.RouteConfiguration{Name: "ingress_https"},
		),
		TypeUrl: routeType,
		Nonce:   "5",
	}, streamRDS(t, cc))

	rh.OnAdd(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello-kitty",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	}, false)

	// i4 is the same as i3, and includes a TLS spec object to enable ingress_https routes
	// i3 is like i2, but adds the ingress.kubernetes.io/force-ssl-redirect: "true" annotation
	i4 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: "default",
			Annotations: map[string]string{
				"ingress.kubernetes.io/force-ssl-redirect": "true"},
		},
		Spec: netv1.IngressSpec{
			TLS: []netv1.IngressTLS{{
				Hosts:      []string{"hello.example.com"},
				SecretName: "hello-kitty",
			}},
			Rules: []netv1.IngressRule{{
				Host: "hello.example.com",
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Path: "/",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "wowie",
									Port: netv1.ServiceBackendPort{
										Number: 80,
									},
								},
							},
						}, {
							Path: "/whoop",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "kerpow",
									Port: netv1.ServiceBackendPort{
										Number: 9000,
									},
								},
							},
						}},
					},
				},
			}},
		},
	}
	rh.OnUpdate(i3, i4)
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "7",
		Resources: resources(t,
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_http",
				VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
					Name:    "hello.example.com",
					Domains: domains("hello.example.com"),
					Routes: []*envoy_config_route_v3.Route{{
						Match:  envoy.RouteMatch("/whoop"),
						Action: envoy.UpgradeHTTPS(),
					}, {
						Match:  envoy.RouteMatch("/"),
						Action: envoy.UpgradeHTTPS(),
					}},
				}},
			},
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_https",
				VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
					Name:    "hello.example.com",
					Domains: domains("hello.example.com"),
					Routes: []*envoy_config_route_v3.Route{{
						Match:               envoy.RouteMatch("/whoop"),
						Action:              routecluster("default/kerpow/9000/da39a3ee5e"),
						RequestHeadersToAdd: envoy.RouteHeaders(),
					}, {
						Match:               envoy.RouteMatch("/"),
						Action:              routecluster("default/wowie/80/da39a3ee5e"),
						RequestHeadersToAdd: envoy.RouteHeaders(),
					}},
				}},
			},
		),
		TypeUrl: routeType,
		Nonce:   "7",
	}, streamRDS(t, cc))
}

// contour#164: backend request timeout support
func TestRequestTimeout(t *testing.T) {
	const (
		durationInfinite  = time.Duration(0)
		duration10Minutes = 10 * time.Minute
	)

	rh, cc, done := setup(t)
	defer done()

	s1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "backend",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
	rh.OnAdd(s1, false)

	// i1 is a simple ingress bound to the default vhost.
	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "hello", Namespace: "default"},
		Spec: netv1.IngressSpec{
			DefaultBackend: backend("backend", intstr.FromInt(80)),
		},
	}
	rh.OnAdd(i1, false)
	assertRDS(t, cc, "2", []*envoy_config_route_v3.VirtualHost{{
		Name:    "*",
		Domains: []string{"*"},
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"), // match all
			Action:              routecluster("default/backend/80/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)

	// i2 adds an _invalid_ timeout, which we interpret as _infinite_.
	i2 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "hello", Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/request-timeout": "600", // not valid
			},
		},
		Spec: netv1.IngressSpec{
			DefaultBackend: backend("backend", intstr.FromInt(80)),
		},
	}
	rh.OnUpdate(i1, i2)
	assertRDS(t, cc, "3", []*envoy_config_route_v3.VirtualHost{{
		Name:    "*",
		Domains: []string{"*"},
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"), // match all
			Action:              clustertimeout("default/backend/80/da39a3ee5e", durationInfinite),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)

	// i3 corrects i2 to use a proper duration
	i3 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "hello", Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/request-timeout": "600s", // 10 * time.Minute
			},
		},
		Spec: netv1.IngressSpec{
			DefaultBackend: backend("backend", intstr.FromInt(80)),
		},
	}
	rh.OnUpdate(i2, i3)
	assertRDS(t, cc, "4", []*envoy_config_route_v3.VirtualHost{{
		Name:    "*",
		Domains: []string{"*"},
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"), // match all
			Action:              clustertimeout("default/backend/80/da39a3ee5e", duration10Minutes),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)

	// i4 updates i3 to explicitly request infinite timeout
	i4 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "hello", Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/request-timeout": "infinity",
			},
		},
		Spec: netv1.IngressSpec{
			DefaultBackend: backend("backend", intstr.FromInt(80)),
		},
	}
	rh.OnUpdate(i3, i4)
	assertRDS(t, cc, "5", []*envoy_config_route_v3.VirtualHost{{
		Name:    "*",
		Domains: []string{"*"},
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"), // match all
			Action:              clustertimeout("default/backend/80/da39a3ee5e", durationInfinite),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)
}

// contour#250 ingress.kubernetes.io/force-ssl-redirect: "true" should apply
// per route, not per vhost.
func TestSSLRedirectOverlay(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	// i1 is a stock ingress with force-ssl-redirect on the / route
	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: "default",
			Annotations: map[string]string{
				"ingress.kubernetes.io/force-ssl-redirect": "true",
			},
		},
		Spec: netv1.IngressSpec{
			TLS: []netv1.IngressTLS{{
				Hosts:      []string{"example.com"},
				SecretName: "example-tls",
			}},
			Rules: []netv1.IngressRule{{
				Host: "example.com",
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Path: "/",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "app-service",
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

	rh.OnAdd(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-tls",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	}, false)

	rh.OnAdd(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-service",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}, false)

	// i2 is an overlay to add the let's encrypt handler.
	i2 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "challenge", Namespace: "nginx-ingress"},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{{
				Host: "example.com",
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Path: "/.well-known/acme-challenge/gVJl5NWL2owUqZekjHkt_bo3OHYC2XNDURRRgLI5JTk",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "challenge-service",
									Port: netv1.ServiceBackendPort{
										Number: 8009,
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

	rh.OnAdd(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "challenge-service",
			Namespace: "nginx-ingress",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       8009,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}, false)

	assertRDS(t, cc, "5", []*envoy_config_route_v3.VirtualHost{{ // ingress_http
		Name:    "example.com",
		Domains: domains("example.com"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/.well-known/acme-challenge/gVJl5NWL2owUqZekjHkt_bo3OHYC2XNDURRRgLI5JTk"),
			Action:              routecluster("nginx-ingress/challenge-service/8009/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}, {
			Match:  envoy.RouteMatch("/"), // match all
			Action: envoy.UpgradeHTTPS(),
		}},
	}}, []*envoy_config_route_v3.VirtualHost{{ // ingress_https
		Name:    "example.com",
		Domains: domains("example.com"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/.well-known/acme-challenge/gVJl5NWL2owUqZekjHkt_bo3OHYC2XNDURRRgLI5JTk"),
			Action:              routecluster("nginx-ingress/challenge-service/8009/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}, {
			Match:               envoy.RouteMatch("/"), // match all
			Action:              routecluster("default/app-service/8080/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}})
}

func TestInvalidCertInIngress(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	// Create an invalid TLS secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-tls",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       nil,
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	}
	rh.OnAdd(secret, false)

	// Create a service
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

	// Create an ingress that uses the invalid secret
	rh.OnAdd(&netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "kuard-ing", Namespace: "default"},
		Spec: netv1.IngressSpec{
			TLS: []netv1.IngressTLS{{
				Hosts:      []string{"kuard.io"},
				SecretName: "example-tls",
			}},
			Rules: []netv1.IngressRule{{
				Host: "kuard.io",
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
						}},
					},
				},
			}},
		},
	}, false)

	assertRDS(t, cc, "3", []*envoy_config_route_v3.VirtualHost{{ // ingress_http
		Name:    "kuard.io",
		Domains: domains("kuard.io"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"),
			Action:              routecluster("default/kuard/80/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)

	// Correct the secret
	rh.OnUpdate(secret, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-tls",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("cert"),
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	})

	assertRDS(t, cc, "4", []*envoy_config_route_v3.VirtualHost{{ // ingress_http
		Name:    "kuard.io",
		Domains: domains("kuard.io"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"),
			Action:              routecluster("default/kuard/80/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, []*envoy_config_route_v3.VirtualHost{{ // ingress_https
		Name:    "kuard.io",
		Domains: domains("kuard.io"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"),
			Action:              routecluster("default/kuard/80/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}})
}

// issue #257: editing default ingress did not remove original default route
func TestIssue257(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	// apiVersion: networking/v1
	// kind: Ingress
	// metadata:
	//   name: kuard-ing
	//   labels:
	//     app: kuard
	//   annotations:
	//     kubernetes.io/ingress.class: enroute
	// spec:
	//   backend:
	//     serviceName: kuard
	//     servicePort: 80
	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard-ing",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "enroute",
			},
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

	s1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
	rh.OnAdd(s1, false)

	assertRDS(t, cc, "2", []*envoy_config_route_v3.VirtualHost{{
		Name:    "*",
		Domains: []string{"*"},
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"), // match all
			Action:              routecluster("default/kuard/80/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)

	// apiVersion: networking/v1
	// kind: Ingress
	// metadata:
	//   name: kuard-ing
	//   labhls:
	//     app: kuard
	//   annotations:
	//     kubernetes.io/ingress.class: enroute
	// spec:
	//  rules:
	//  - host: kuard.db.gd-ms.com
	//    http:
	//      paths:
	//      - backend:
	//         serviceName: kuard
	//         servicePort: 80
	//        path: /
	i2 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard-ing",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "enroute",
			},
		},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{{
				Host: "kuard.db.gd-ms.com",
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Path: "/",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "kuard",
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
	}
	rh.OnUpdate(i1, i2)

	assertRDS(t, cc, "3", []*envoy_config_route_v3.VirtualHost{{
		Name:    "kuard.db.gd-ms.com",
		Domains: domains("kuard.db.gd-ms.com"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"), // match all
			Action:              routecluster("default/kuard/80/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)
}

func TestRDSFilter(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	// i1 is a stock ingress with force-ssl-redirect on the / route
	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: "default",
			Annotations: map[string]string{
				"ingress.kubernetes.io/force-ssl-redirect": "true",
			},
		},
		Spec: netv1.IngressSpec{
			TLS: []netv1.IngressTLS{{
				Hosts:      []string{"example.com"},
				SecretName: "example-tls",
			}},
			Rules: []netv1.IngressRule{{
				Host: "example.com",
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Path: "/",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "app-service",
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

	rh.OnAdd(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-tls",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	}, false)

	s1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-service",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
	rh.OnAdd(s1, false)

	// i2 is an overlay to add the let's encrypt handler.
	i2 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "challenge", Namespace: "nginx-ingress"},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{{
				Host: "example.com",
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Path: "/.well-known/acme-challenge/gVJl5NWL2owUqZekjHkt_bo3OHYC2XNDURRRgLI5JTk",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "challenge-service",
									Port: netv1.ServiceBackendPort{
										Number: 8009,
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

	s2 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "challenge-service",
			Namespace: "nginx-ingress",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       8009,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
	rh.OnAdd(s2, false)

	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "5",
		Resources: resources(t,
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_http",
				VirtualHosts: []*envoy_config_route_v3.VirtualHost{{ // ingress_http
					Name:    "example.com",
					Domains: domains("example.com"),
					Routes: []*envoy_config_route_v3.Route{{
						Match:               envoy.RouteMatch("/.well-known/acme-challenge/gVJl5NWL2owUqZekjHkt_bo3OHYC2XNDURRRgLI5JTk"),
						Action:              routecluster("nginx-ingress/challenge-service/8009/da39a3ee5e"),
						RequestHeadersToAdd: envoy.RouteHeaders(),
					}, {
						Match:  envoy.RouteMatch("/"), // match all
						Action: envoy.UpgradeHTTPS(),
					}},
				}},
			},
		),
		TypeUrl: routeType,
		Nonce:   "5",
	}, streamRDS(t, cc, "ingress_http"))

	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "5",
		Resources: resources(t,
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_https",
				VirtualHosts: []*envoy_config_route_v3.VirtualHost{{ // ingress_https
					Name:    "example.com",
					Domains: domains("example.com"),
					Routes: []*envoy_config_route_v3.Route{{
						Match:               envoy.RouteMatch("/.well-known/acme-challenge/gVJl5NWL2owUqZekjHkt_bo3OHYC2XNDURRRgLI5JTk"),
						Action:              routecluster("nginx-ingress/challenge-service/8009/da39a3ee5e"),
						RequestHeadersToAdd: envoy.RouteHeaders(),
					}, {
						Match:               envoy.RouteMatch("/"), // match all
						Action:              routecluster("default/app-service/8080/da39a3ee5e"),
						RequestHeadersToAdd: envoy.RouteHeaders(),
					}},
				}},
			},
		),
		TypeUrl: routeType,
		Nonce:   "5",
	}, streamRDS(t, cc, "ingress_https"))
}

func TestWebsocketIngress(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	rh.OnAdd(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws",
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

	rh.OnAdd(&netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws",
			Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/websocket-routes": "/",
			},
		},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{{
				Host: "websocket.hello.world",
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Path: "/",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "ws",
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

	assertRDS(t, cc, "2", []*envoy_config_route_v3.VirtualHost{{
		Name:    "websocket.hello.world",
		Domains: domains("websocket.hello.world"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"), // match all
			Action:              websocketroute("default/ws/80/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)
}

func TestWebsocketGatewayHost(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	rh.OnAdd(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws",
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
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "websocket.hello.world"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "ws",
					Port: 80,
				}},
			}, {
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/ws-1",
				}},
				EnableWebsockets: true,
				Services: []gatewayhostv1.Service{{
					Name: "ws",
					Port: 80,
				}},
			}, {
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/ws-2",
				}},
				EnableWebsockets: true,
				Services: []gatewayhostv1.Service{{
					Name: "ws",
					Port: 80,
				}},
			}},
		},
	}, false)

	assertRDS(t, cc, "2", []*envoy_config_route_v3.VirtualHost{{
		Name:    "websocket.hello.world",
		Domains: domains("websocket.hello.world"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/ws-2"),
			Action:              websocketroute("default/ws/80/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}, {
			Match:               envoy.RouteMatch("/ws-1"),
			Action:              websocketroute("default/ws/80/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}, {
			Match:               envoy.RouteMatch("/"), // match all
			Action:              routecluster("default/ws/80/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)
}
func TestPrefixRewriteGatewayHost(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	rh.OnAdd(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws",
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
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "prefixrewrite.hello.world"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "ws",
					Port: 80,
				}},
			}, {
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/ws-1",
				}},
				PrefixRewrite: "/",
				Services: []gatewayhostv1.Service{{
					Name: "ws",
					Port: 80,
				}},
			}, {
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/ws-2",
				}},
				PrefixRewrite: "/",
				Services: []gatewayhostv1.Service{{
					Name: "ws",
					Port: 80,
				}},
			}},
		},
	}, false)

	assertRDS(t, cc, "2", []*envoy_config_route_v3.VirtualHost{{
		Name:    "prefixrewrite.hello.world",
		Domains: domains("prefixrewrite.hello.world"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/ws-2"),
			Action:              prefixrewriteroute("default/ws/80/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}, {
			Match:               envoy.RouteMatch("/ws-1"),
			Action:              prefixrewriteroute("default/ws/80/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}, {
			Match:               envoy.RouteMatch("/"), // match all
			Action:              routecluster("default/ws/80/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)
}

// issue 404
func TestDefaultBackendDoesNotOverwriteNamedHost(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	rh.OnAdd(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}, {
				Name:       "alt",
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}, false)

	rh.OnAdd(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gui",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}, false)

	rh.OnAdd(&netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello",
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
			Rules: []netv1.IngressRule{{
				Host: "test-gui",
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Path: "/",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "test-gui",
									Port: netv1.ServiceBackendPort{
										Number: 80,
									},
								},
							},
						}},
					},
				},
			}, {
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Path: "/kuard",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "kuard",
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
	}, false)

	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_http",
				VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
					Name:    "*",
					Domains: []string{"*"},
					Routes: []*envoy_config_route_v3.Route{{
						Match:               envoy.RouteMatch("/kuard"),
						Action:              routecluster("default/kuard/8080/da39a3ee5e"),
						RequestHeadersToAdd: envoy.RouteHeaders(),
					}, {
						Match:               envoy.RouteMatch("/"),
						Action:              routecluster("default/kuard/80/da39a3ee5e"),
						RequestHeadersToAdd: envoy.RouteHeaders(),
					}},
				}, {
					Name:    "test-gui",
					Domains: domains("test-gui"),
					Routes: []*envoy_config_route_v3.Route{{
						Match:               envoy.RouteMatch("/"),
						Action:              routecluster("default/test-gui/80/da39a3ee5e"),
						RequestHeadersToAdd: envoy.RouteHeaders(),
					}},
				}},
			},
		),
		TypeUrl: routeType,
		Nonce:   "3",
	}, streamRDS(t, cc, "ingress_http"))
}

func TestRDSGatewayHostInsideRootNamespaces(t *testing.T) {
	rh, cc, done := setup(t, func(reh *contour.ResourceEventHandler) {
		reh.GatewayHostRootNamespaces = []string{"roots"}
		reh.Notifier.(*contour.CacheHandler).GatewayHostStatus = &k8s.GatewayHostStatus{
			Client: fake.NewSimpleClientset(),
		}
	})
	defer done()

	rh.OnAdd(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "roots",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}, false)

	// ir1 is an gatewayhost that is in the root namespaces
	ir1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "roots",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "example.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			}},
		},
	}

	// add gatewayhost
	rh.OnAdd(ir1, false)

	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_http",
				VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
					Name:    "example.com",
					Domains: domains("example.com"),
					Routes: []*envoy_config_route_v3.Route{{
						Match:               envoy.RouteMatch("/"),
						Action:              routecluster("roots/kuard/8080/da39a3ee5e"),
						RequestHeadersToAdd: envoy.RouteHeaders(),
					}},
				}},
			},
		),
		TypeUrl: routeType,
		Nonce:   "2",
	}, streamRDS(t, cc, "ingress_http"))
}

func TestRDSGatewayHostOutsideRootNamespaces(t *testing.T) {
	rh, cc, done := setup(t, func(reh *contour.ResourceEventHandler) {
		reh.GatewayHostRootNamespaces = []string{"roots"}
		reh.Notifier.(*contour.CacheHandler).GatewayHostStatus = &k8s.GatewayHostStatus{
			Client: fake.NewSimpleClientset(),
		}
	})
	defer done()

	rh.OnAdd(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}, false)

	// ir1 is an gatewayhost that is not in the root namespaces
	ir1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "example.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			}},
		},
	}

	// add gatewayhost
	rh.OnAdd(ir1, false)

	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_http",
			}),
		TypeUrl: routeType,
		Nonce:   "2",
	}, streamRDS(t, cc, "ingress_http"))
}

// Test DAGAdapter.IngressClass setting works, this could be done
// in LDS or RDS, or even CDS, but this test mirrors the place it's
// tested in internal/contour/route_test.go
func TestRDSGatewayHostClassAnnotation(t *testing.T) {
	rh, cc, done := setup(t, func(reh *contour.ResourceEventHandler) {
		reh.IngressClass = "linkerd"
	})
	defer done()

	rh.OnAdd(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}, false)

	ir1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard ",
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
				Services: []gatewayhostv1.Service{
					{
						Name: "kuard",
						Port: 8080,
					},
				},
			}},
		},
	}

	rh.OnAdd(ir1, false)
	assertRDS(t, cc, "2", []*envoy_config_route_v3.VirtualHost{{
		Name:    "www.example.com",
		Domains: domains("www.example.com"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"),
			Action:              routecluster("default/kuard/8080/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)

	ir2 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard ",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "enroute",
			},
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "www.example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{
					{
						Name: "kuard",
						Port: 8080,
					},
				},
			}},
		},
	}
	rh.OnUpdate(ir1, ir2)
	assertRDS(t, cc, "3", nil, nil)

	ir3 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard ",
			Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/ingress.class": "enroute",
			},
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "www.example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{
					{
						Name: "kuard",
						Port: 8080,
					},
				},
			}},
		},
	}
	rh.OnUpdate(ir2, ir3)
	assertRDS(t, cc, "3", nil, nil)

	ir4 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard ",
			Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/ingress.class": "linkerd",
			},
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "www.example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{
					{
						Name: "kuard",
						Port: 8080,
					},
				},
			}},
		},
	}
	rh.OnUpdate(ir3, ir4)
	assertRDS(t, cc, "4", []*envoy_config_route_v3.VirtualHost{{
		Name:    "www.example.com",
		Domains: domains("www.example.com"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"),
			Action:              routecluster("default/kuard/8080/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)

	ir5 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard ",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "linkerd",
			},
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "www.example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{
					{
						Name: "kuard",
						Port: 8080,
					},
				},
			}},
		},
	}
	rh.OnUpdate(ir4, ir5)

	assertRDS(t, cc, "5", []*envoy_config_route_v3.VirtualHost{{
		Name:    "www.example.com",
		Domains: domains("www.example.com"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"),
			Action:              routecluster("default/kuard/8080/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)

	rh.OnUpdate(ir5, ir3)
	assertRDS(t, cc, "6", nil, nil)
}

// Test DAGAdapter.IngressClass setting works, this could be done
// in LDS or RDS, or even CDS, but this test mirrors the place it's
// tested in internal/contour/route_test.go
func TestRDSIngressClassAnnotation(t *testing.T) {
	rh, cc, done := setup(t, func(reh *contour.ResourceEventHandler) {
		reh.IngressClass = "linkerd"
	})
	defer done()

	rh.OnAdd(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}, false)

	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard-ing",
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
	}
	rh.OnAdd(i1, false)
	assertRDS(t, cc, "2", []*envoy_config_route_v3.VirtualHost{{
		Name:    "*",
		Domains: []string{"*"},
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"),
			Action:              routecluster("default/kuard/8080/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)

	i2 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard-ing",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "enroute",
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
	}
	rh.OnUpdate(i1, i2)
	assertRDS(t, cc, "3", nil, nil)

	i3 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard-ing",
			Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/ingress.class": "enroute",
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
	}
	rh.OnUpdate(i2, i3)
	assertRDS(t, cc, "3", nil, nil)

	i4 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard-ing",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "linkerd",
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
	}
	rh.OnUpdate(i3, i4)
	assertRDS(t, cc, "4", []*envoy_config_route_v3.VirtualHost{{
		Name:    "*",
		Domains: []string{"*"},
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"),
			Action:              routecluster("default/kuard/8080/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)

	i5 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard-ing",
			Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/ingress.class": "linkerd",
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
	}
	rh.OnUpdate(i4, i5)
	assertRDS(t, cc, "5", []*envoy_config_route_v3.VirtualHost{{
		Name:    "*",
		Domains: []string{"*"},
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"),
			Action:              routecluster("default/kuard/8080/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)

	rh.OnUpdate(i5, i3)
	assertRDS(t, cc, "6", nil, nil)
}

// issue 523, check for data races caused by accidentally
// sorting the contents of an RDS entry's virtualhost list.
func TestRDSAssertNoDataRaceDuringInsertAndStream(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	stop := make(chan struct{})

	s1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
	rh.OnAdd(s1, false)

	go func() {
		for i := 0; i < 100; i++ {
			rh.OnAdd(&gatewayhostv1.GatewayHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("simple-%d", i),
					Namespace: "default",
				},
				Spec: gatewayhostv1.GatewayHostSpec{
					VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: fmt.Sprintf("example-%d.com", i)},
					Routes: []gatewayhostv1.Route{{
						Conditions: []gatewayhostv1.Condition{{
							Prefix: "/",
						}},
						Services: []gatewayhostv1.Service{{
							Name: "kuard",
							Port: 80,
						}},
					}},
				},
			}, false)
		}
		close(stop)
	}()

	for {
		select {
		case <-stop:
			return
		default:
			streamRDS(t, cc)
		}
	}
}

// issue 606: spec.rules.host without a http key causes panic.
// apiVersion: networking/v1
// kind: Ingress
// metadata:
//   name: test-ingress3
// spec:
//   rules:
//   - host: test1.test.com
//   - host: test2.test.com
//     http:
//       paths:
//       - backend:
//           serviceName: network-test
//           servicePort: 9001
//         path: /
//
// note: this test caused a panic in dag.Builder, but testing the
// context of RDS is a good place to start.
func TestRDSIngressSpecMissingHTTPKey(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress3",
			Namespace: "default",
		},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{{
				Host: "test1.test.com",
			}, {
				Host: "test2.test.com",
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{{
							Path: "/",
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: "network-test",
									Port: netv1.ServiceBackendPort{
										Number: 9001,
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

	s1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "network-test",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       9001,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
	rh.OnAdd(s1, false)

	assertRDS(t, cc, "2", []*envoy_config_route_v3.VirtualHost{{
		Name:    "test2.test.com",
		Domains: domains("test2.test.com"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"), // match all
			Action:              routecluster("default/network-test/9001/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)
}

func TestRouteWithAServiceWeight(t *testing.T) {
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

	ir1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "test2.test.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/a",
				}},
				Services: []gatewayhostv1.Service{{
					Name:   "kuard",
					Port:   80,
					Weight: 90, // ignored
				}},
			}},
		},
	}

	rh.OnAdd(ir1, false)
	assertRDS(t, cc, "2", []*envoy_config_route_v3.VirtualHost{{
		Name:    "test2.test.com",
		Domains: domains("test2.test.com"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/a"), // match all
			Action:              routecluster("default/kuard/80/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)

	ir2 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "test2.test.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/a",
				}},
				Services: []gatewayhostv1.Service{{
					Name:   "kuard",
					Port:   80,
					Weight: 90,
				}, {
					Name:   "kuard",
					Port:   80,
					Weight: 60,
				}},
			}},
		},
	}

	rh.OnUpdate(ir1, ir2)
	assertRDS(t, cc, "3", []*envoy_config_route_v3.VirtualHost{{
		Name:    "test2.test.com",
		Domains: domains("test2.test.com"),
		Routes: []*envoy_config_route_v3.Route{{
			Match: envoy.RouteMatch("/a"), // match all
			Action: routeweightedcluster(
				weightedcluster{"default/kuard/80/da39a3ee5e", 60},
				weightedcluster{"default/kuard/80/da39a3ee5e", 90},
			),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)
}

func TestRouteWithTLS(t *testing.T) {
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

	rh.OnAdd(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-tls",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	}, false)

	ir1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "test2.test.com",
				TLS: &gatewayhostv1.TLS{
					SecretName: "example-tls",
				},
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/a",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 80,
				}},
			}},
		},
	}

	rh.OnAdd(ir1, false)

	// check that ingress_http has been updated.
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_http",
				VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
					Name:    "test2.test.com",
					Domains: domains("test2.test.com"),
					Routes: []*envoy_config_route_v3.Route{{
						Match:  envoy.RouteMatch("/a"),
						Action: envoy.UpgradeHTTPS(),
					}},
				}}},
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_https",
				VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
					Name:    "test2.test.com",
					Domains: domains("test2.test.com"),
					Routes: []*envoy_config_route_v3.Route{{
						Match:               envoy.RouteMatch("/a"),
						Action:              routecluster("default/kuard/80/da39a3ee5e"),
						RequestHeadersToAdd: envoy.RouteHeaders(),
					}},
				}}},
		),
		TypeUrl: routeType,
		Nonce:   "3",
	}, streamRDS(t, cc))
}
func TestRouteWithTLS_InsecurePaths(t *testing.T) {
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

	rh.OnAdd(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc2",
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

	rh.OnAdd(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-tls",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	}, false)

	ir1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "test2.test.com",
				TLS: &gatewayhostv1.TLS{
					SecretName: "example-tls",
				},
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/insecure",
				}},
				PermitInsecure: true,
				Services: []gatewayhostv1.Service{{Name: "kuard",
					Port: 80,
				}},
			}, {
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/secure",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "svc2",
					Port: 80,
				}},
			}},
		},
	}

	rh.OnAdd(ir1, false)

	// check that ingress_http has been updated.
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "4",
		Resources: resources(t,
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_http",
				VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
					Name:    "test2.test.com",
					Domains: domains("test2.test.com"),
					Routes: []*envoy_config_route_v3.Route{
						{
							Match:  envoy.RouteMatch("/secure"),
							Action: envoy.UpgradeHTTPS(),
						}, {
							Match:               envoy.RouteMatch("/insecure"),
							Action:              routecluster("default/kuard/80/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						},
					},
				}}},
			&envoy_config_route_v3.RouteConfiguration{
				Name: "ingress_https",
				VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
					Name:    "test2.test.com",
					Domains: domains("test2.test.com"),
					Routes: []*envoy_config_route_v3.Route{
						{
							Match:               envoy.RouteMatch("/secure"),
							Action:              routecluster("default/svc2/80/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}, {
							Match:               envoy.RouteMatch("/insecure"),
							Action:              routecluster("default/kuard/80/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						},
					},
				}}},
		),
		TypeUrl: routeType,
		Nonce:   "4",
	}, streamRDS(t, cc))
}

// issue 665, support for retry-on, num-retries, and per-try-timeout annotations.
func TestRouteRetryAnnotations(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	s1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "backend",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
	rh.OnAdd(s1, false)

	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hello", Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/retry-on":        "5xx,gateway-error",
				"enroute.saaras.io/num-retries":     "7",
				"enroute.saaras.io/per-try-timeout": "120ms",
			},
		},
		Spec: netv1.IngressSpec{
			DefaultBackend: &netv1.IngressBackend{
				Service: &netv1.IngressServiceBackend{
					Name: "backend",
					Port: netv1.ServiceBackendPort{
						Number: 80,
					},
				},
			},
		},
	}
	rh.OnAdd(i1, false)
	assertRDS(t, cc, "2", []*envoy_config_route_v3.VirtualHost{{
		Name:    "*",
		Domains: []string{"*"},
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"), // match all
			Action:              routeretry("default/backend/80/da39a3ee5e", "5xx,gateway-error", 7, 120*time.Millisecond),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)
}

// issue 815, support for retry-on, num-retries, and per-try-timeout in GatewayHost
func TestRouteRetryGatewayHost(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	s1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "backend",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
	rh.OnAdd(s1, false)

	i1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "test2.test.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				RetryPolicy: &gatewayhostv1.RetryPolicy{
					NumRetries:    7,
					PerTryTimeout: "120ms",
				},
				Services: []gatewayhostv1.Service{{
					Name: "backend",
					Port: 80,
				}},
			}},
		},
	}

	rh.OnAdd(i1, false)
	assertRDS(t, cc, "2", []*envoy_config_route_v3.VirtualHost{{
		Name:    "test2.test.com",
		Domains: domains("test2.test.com"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"), // match all
			Action:              routeretry("default/backend/80/da39a3ee5e", "5xx", 7, 120*time.Millisecond),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)
}

// issue 815, support for timeoutpolicy in GatewayHost
func TestRouteTimeoutPolicyGatewayHost(t *testing.T) {
	const (
		durationInfinite  = time.Duration(0)
		duration10Minutes = 10 * time.Minute
	)

	rh, cc, done := setup(t)
	defer done()

	s1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "backend",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
	rh.OnAdd(s1, false)

	// i1 is an _invalid_ timeout, which we interpret as _infinite_.
	i1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "test2.test.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "backend",
					Port: 80,
				}},
			}},
		},
	}
	rh.OnAdd(i1, false)
	assertRDS(t, cc, "2", []*envoy_config_route_v3.VirtualHost{{
		Name:    "test2.test.com",
		Domains: domains("test2.test.com"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"), // match all
			Action:              routecluster("default/backend/80/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)

	// i2 adds an _invalid_ timeout, which we interpret as _infinite_.
	i2 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "test2.test.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				TimeoutPolicy: &gatewayhostv1.TimeoutPolicy{
					Request: "600",
				},
				Services: []gatewayhostv1.Service{{
					Name: "backend",
					Port: 80,
				}},
			}},
		},
	}
	rh.OnUpdate(i1, i2)
	assertRDS(t, cc, "3", []*envoy_config_route_v3.VirtualHost{{
		Name:    "test2.test.com",
		Domains: domains("test2.test.com"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"), // match all
			Action:              clustertimeout("default/backend/80/da39a3ee5e", durationInfinite),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)
	// i3 corrects i2 to use a proper duration
	i3 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "test2.test.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				TimeoutPolicy: &gatewayhostv1.TimeoutPolicy{
					Request: "600s", // 10 * time.Minute
				},
				Services: []gatewayhostv1.Service{{
					Name: "backend",
					Port: 80,
				}},
			}},
		},
	}
	rh.OnUpdate(i2, i3)
	assertRDS(t, cc, "4", []*envoy_config_route_v3.VirtualHost{{
		Name:    "test2.test.com",
		Domains: domains("test2.test.com"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"), // match all
			Action:              clustertimeout("default/backend/80/da39a3ee5e", duration10Minutes),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)
	// i4 updates i3 to explicitly request infinite timeout
	i4 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "test2.test.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				TimeoutPolicy: &gatewayhostv1.TimeoutPolicy{
					Request: "infinity",
				},
				Services: []gatewayhostv1.Service{{
					Name: "backend",
					Port: 80,
				}},
			}},
		},
	}
	rh.OnUpdate(i3, i4)
	assertRDS(t, cc, "5", []*envoy_config_route_v3.VirtualHost{{
		Name:    "test2.test.com",
		Domains: domains("test2.test.com"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/"), // match all
			Action:              clustertimeout("default/backend/80/da39a3ee5e", durationInfinite),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)
}

func TestRouteWithSessionAffinity(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	rh.OnAdd(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}, {
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}, false)

	// simple single service
	ir1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "www.example.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/cart",
				}},
				Services: []gatewayhostv1.Service{{
					Name:     "app",
					Port:     80,
					Strategy: "Cookie",
				}},
			}},
		},
	}

	rh.OnAdd(ir1, false)
	assertRDS(t, cc, "2", []*envoy_config_route_v3.VirtualHost{{
		Name:    "www.example.com",
		Domains: domains("www.example.com"),
		Routes: []*envoy_config_route_v3.Route{{
			Match:               envoy.RouteMatch("/cart"),
			Action:              withSessionAffinity(routecluster("default/app/80/e4f81994fe")),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)

	// two backends
	ir2 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "www.example.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/cart",
				}},
				Services: []gatewayhostv1.Service{{
					Name:     "app",
					Port:     80,
					Strategy: "Cookie",
				}, {
					Name:     "app",
					Port:     8080,
					Strategy: "Cookie",
				}},
			}},
		},
	}
	rh.OnUpdate(ir1, ir2)
	assertRDS(t, cc, "3", []*envoy_config_route_v3.VirtualHost{{
		Name:    "www.example.com",
		Domains: domains("www.example.com"),
		Routes: []*envoy_config_route_v3.Route{{
			Match: envoy.RouteMatch("/cart"),
			Action: withSessionAffinity(
				routeweightedcluster(
					weightedcluster{"default/app/80/e4f81994fe", 1},
					weightedcluster{"default/app/8080/e4f81994fe", 1},
				),
			),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)

	// two mixed backends
	ir3 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "www.example.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/cart",
				}},
				Services: []gatewayhostv1.Service{{
					Name:     "app",
					Port:     80,
					Strategy: "Cookie",
				}, {
					Name: "app",
					Port: 8080,
				}},
			}, {
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "app",
					Port: 80,
				}},
			}},
		},
	}
	rh.OnUpdate(ir2, ir3)
	assertRDS(t, cc, "4", []*envoy_config_route_v3.VirtualHost{{
		Name:    "www.example.com",
		Domains: domains("www.example.com"),
		Routes: []*envoy_config_route_v3.Route{{
			Match: envoy.RouteMatch("/cart"),
			Action: withSessionAffinity(
				routeweightedcluster(
					weightedcluster{"default/app/80/e4f81994fe", 1},
					weightedcluster{"default/app/8080/da39a3ee5e", 1},
				),
			),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}, {
			Match:               envoy.RouteMatch("/"),
			Action:              routecluster("default/app/80/da39a3ee5e"),
			RequestHeadersToAdd: envoy.RouteHeaders(),
		}},
	}}, nil)

}

// issue 681 Increase the e2e coverage of lb strategies
func TestLoadBalancingStrategies(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	st := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "template",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}

	services := []struct {
		name       string
		lbHash     string
		lbStrategy string
		lbDesc     string
	}{
		{"s1", "f3b72af6a9", "RoundRobin", "RoundRobin lb algorithm"},
		{"s2", "8bf87fefba", "WeightedLeastRequest", "WeightedLeastRequest lb algorithm"},
		{"s5", "58d888c08a", "Random", "Random lb algorithm"},
		{"s6", "da39a3ee5e", "", "Default lb algorithm"},
	}
	ss := make([]gatewayhostv1.Service, len(services))
	wc := make([]weightedcluster, len(services))
	for i, x := range services {
		s := st
		s.ObjectMeta.Name = x.name
		rh.OnAdd(&s, false)
		ss[i] = gatewayhostv1.Service{
			Name:     x.name,
			Port:     80,
			Strategy: x.lbStrategy,
		}
		wc[i] = weightedcluster{fmt.Sprintf("default/%s/80/%s", x.name, x.lbHash), 1}
	}

	ir := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{Fqdn: "test2.test.com"},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/a",
				}},
				Services: ss,
			}},
		},
	}

	rh.OnAdd(ir, false)
	want := []*envoy_config_route_v3.VirtualHost{{
		Name:    "test2.test.com",
		Domains: domains("test2.test.com"),
		Routes: []*envoy_config_route_v3.Route{
			{
				Match:               envoy.RouteMatch("/a"),
				Action:              routeweightedcluster(wc...),
				RequestHeadersToAdd: envoy.RouteHeaders(),
			},
		},
	}}
	assertRDS(t, cc, "5", want, nil)
}

func TestCorsFilter(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	st := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "template",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}

	services := []struct {
		name       string
		lbHash     string
		lbStrategy string
		lbDesc     string
	}{
		{"s1", "f3b72af6a9", "RoundRobin", "RoundRobin lb algorithm"},
		{"s2", "8bf87fefba", "WeightedLeastRequest", "WeightedLeastRequest lb algorithm"},
		{"s5", "58d888c08a", "Random", "Random lb algorithm"},
		{"s6", "da39a3ee5e", "", "Default lb algorithm"},
	}
	ss := make([]gatewayhostv1.Service, len(services))
	wc := make([]weightedcluster, len(services))
	for i, x := range services {
		s := st
		s.ObjectMeta.Name = x.name
		rh.OnAdd(&s, false)
		ss[i] = gatewayhostv1.Service{
			Name:     x.name,
			Port:     80,
			Strategy: x.lbStrategy,
		}
		wc[i] = weightedcluster{fmt.Sprintf("default/%s/80/%s", x.name, x.lbHash), 1}
	}

	host_filters := make([]gatewayhostv1.HostAttachedFilter, 0)
	host_filters = append(host_filters, gatewayhostv1.HostAttachedFilter{Name: "cors_filter", Type: cfg.FILTER_TYPE_VH_CORS})

	hf := gatewayhostv1.HttpFilter{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cors_filter",
			Namespace: "default",
		},
		Spec: gatewayhostv1.HttpFilterSpec{
			Name: "cors_filter",
			Type: cfg.FILTER_TYPE_VH_CORS,
			HttpFilterConfig: gatewayhostv1.GenericHttpFilterConfig{
				Config: `{
		"match_condition" : {
			"regex" : "https://*foo.example"
		},
		"access_control_allow_methods" : "POST, GET, OPTIONS",
		"access_control_allow_headers" : "X-PINGOTHER, Content-Type",
		"access_control_expose_headers" : "*",
		"access_control_max_age" : "60"

				}`,
			},
		},
	}

	rh.OnAdd(&hf, false)

	ir := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn:    "test2.test.com",
				Filters: host_filters,
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/a",
				}},
				Services: ss,
			}},
		},
	}

	rh.OnAdd(ir, false)
	want := []*envoy_config_route_v3.VirtualHost{{
		Name:    "test2.test.com",
		Domains: domains("test2.test.com"),
		Cors:    CorsConfig("https://*foo.example", "POST, GET, OPTIONS", "X-PINGOTHER, Content-Type", "*", "60"),
		Routes: []*envoy_config_route_v3.Route{
			{
				Match:               envoy.RouteMatch("/a"),
				Action:              routeweightedcluster(wc...),
				RequestHeadersToAdd: envoy.RouteHeaders(),
			},
		},
	}}
	assertRDS(t, cc, "6", want, nil)
}

func CorsConfig(allowOrigin, allowMethods, allowHeaders, exposeHeaders, maxAge string) *envoy_config_route_v3.CorsPolicy {
	sm := envoy_type_matcher_v3.StringMatcher{
		MatchPattern: &envoy_type_matcher_v3.StringMatcher_SafeRegex{
			SafeRegex: &envoy_type_matcher_v3.RegexMatcher{
				EngineType: &envoy_type_matcher_v3.RegexMatcher_GoogleRe2{
					GoogleRe2: &envoy_type_matcher_v3.RegexMatcher_GoogleRE2{},
				},
				Regex: allowOrigin,
			},
		},
	}

	return &envoy_config_route_v3.CorsPolicy{
		AllowOriginStringMatch: []*envoy_type_matcher_v3.StringMatcher{&sm},
		AllowMethods:           allowMethods,
		AllowHeaders:           allowHeaders,
		ExposeHeaders:          exposeHeaders,
		MaxAge:                 maxAge,
	}
}

func assertRDS(t *testing.T, cc *grpc.ClientConn, versioninfo string, ingress_http, ingress_https []*envoy_config_route_v3.VirtualHost) {
	t.Helper()
	assertEqual(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: versioninfo,
		Resources: resources(t,
			&envoy_config_route_v3.RouteConfiguration{
				Name:         "ingress_http",
				VirtualHosts: ingress_http,
			},
			&envoy_config_route_v3.RouteConfiguration{
				Name:         "ingress_https",
				VirtualHosts: ingress_https,
			},
		),
		TypeUrl: routeType,
		Nonce:   versioninfo,
	}, streamRDS(t, cc))
}

func domains(hostname string) []string {
	if hostname == "*" {
		return []string{"*"}
	}
	return []string{hostname, hostname + ":*"}
}

func streamRDS(t *testing.T, cc *grpc.ClientConn, rn ...string) *envoy_service_discovery_v3.DiscoveryResponse {
	t.Helper()
	rds := envoy_service_route_v3.NewRouteDiscoveryServiceClient(cc)
	st, err := rds.StreamRoutes(context.TODO())
	check(t, err)
	return stream(t, st, &envoy_service_discovery_v3.DiscoveryRequest{
		TypeUrl:       routeType,
		ResourceNames: rn,
	})
}

type weightedcluster struct {
	name   string
	weight uint32
}

func withSessionAffinity(r *envoy_config_route_v3.Route_Route) *envoy_config_route_v3.Route_Route {
	r.Route.HashPolicy = append(r.Route.HashPolicy, &envoy_config_route_v3.RouteAction_HashPolicy{
		PolicySpecifier: &envoy_config_route_v3.RouteAction_HashPolicy_Cookie_{
			Cookie: &envoy_config_route_v3.RouteAction_HashPolicy_Cookie{
				Name: "X-Contour-Session-Affinity",
				Ttl:  protobuf.Duration(0),
				Path: "/",
			},
		},
	})
	return r
}

func routecluster(cluster string) *envoy_config_route_v3.Route_Route {
	return &envoy_config_route_v3.Route_Route{
		Route: &envoy_config_route_v3.RouteAction{
			ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{
				Cluster: cluster,
			},
		},
	}
}

func routeweightedcluster(clusters ...weightedcluster) *envoy_config_route_v3.Route_Route {
	return &envoy_config_route_v3.Route_Route{
		Route: &envoy_config_route_v3.RouteAction{
			ClusterSpecifier: &envoy_config_route_v3.RouteAction_WeightedClusters{
				WeightedClusters: weightedclusters(clusters),
			},
		},
	}
}

func weightedclusters(clusters []weightedcluster) *envoy_config_route_v3.WeightedCluster {
	var wc envoy_config_route_v3.WeightedCluster
	var total uint32
	for _, c := range clusters {
		total += c.weight
		wc.Clusters = append(wc.Clusters, &envoy_config_route_v3.WeightedCluster_ClusterWeight{
			Name:   c.name,
			Weight: protobuf.UInt32(c.weight),
		})
	}
	wc.TotalWeight = protobuf.UInt32(total)
	return &wc
}

func websocketroute(c string) *envoy_config_route_v3.Route_Route {
	cl := routecluster(c)
	cl.Route.UpgradeConfigs = append(cl.Route.UpgradeConfigs,
		&envoy_config_route_v3.RouteAction_UpgradeConfig{
			UpgradeType: "websocket",
		},
	)
	return cl
}

func prefixrewriteroute(c string) *envoy_config_route_v3.Route_Route {
	cl := routecluster(c)
	cl.Route.PrefixRewrite = "/"
	return cl
}

func clustertimeout(c string, timeout time.Duration) *envoy_config_route_v3.Route_Route {
	cl := routecluster(c)
	cl.Route.Timeout = protobuf.Duration(timeout)
	return cl
}

func service(ns, name string, ports ...corev1.ServicePort) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	}
}

func externalnameservice(ns, name, externalname string, ports ...corev1.ServicePort) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Ports:        ports,
			ExternalName: externalname,
			Type:         corev1.ServiceTypeExternalName,
		},
	}
}

func routeretry(cluster string, retryOn string, numRetries uint32, perTryTimeout time.Duration) *envoy_config_route_v3.Route_Route {
	r := routecluster(cluster)
	r.Route.RetryPolicy = &envoy_config_route_v3.RetryPolicy{
		RetryOn: retryOn,
	}
	if numRetries > 0 {
		r.Route.RetryPolicy.NumRetries = protobuf.UInt32(numRetries)
	}
	if perTryTimeout > 0 {
		r.Route.RetryPolicy.PerTryTimeout = protobuf.Duration(perTryTimeout)
	}
	return r
}
