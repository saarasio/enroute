// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

// Copyright © 2017 Heptio
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

package dag

import (
	"fmt"
	"testing"
	"time"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"github.com/google/go-cmp/cmp"
	gatewayhostv1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestDAGInsert(t *testing.T) {
	// The DAG is sensitive to ordering, adding an ingress, then a service,
	// should have the same result as adding a service, then an ingress.

	sec1 := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "default",
		},
		Type: v1.SecretTypeTLS,
		Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
	}

	// Invalid cert in the secret
	sec2 := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "default",
		},
		Type: v1.SecretTypeTLS,
		Data: secretdata("wrong", "wronger"),
	}

	// weird secret with a blank ca.crt that
	// cert manager creates. #1644
	sec3 := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "default",
		},
		Type: v1.SecretTypeTLS,
		Data: map[string][]byte{
			"ca.crt":            []byte(""),
			v1.TLSCertKey:       []byte(CERTIFICATE),
			v1.TLSPrivateKeyKey: []byte(RSA_PRIVATE_KEY),
		},
	}

	cert1 := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ca",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"ca.crt": []byte(CERTIFICATE),
		},
	}

	i1 := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: v1beta1.IngressSpec{
			Backend: backend("kuard", intstr.FromInt(8080))},
	}
	i1a := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.allow-http": "false",
			},
		},
		Spec: v1beta1.IngressSpec{
			Backend: backend("kuard", intstr.FromInt(8080))},
	}

	// i2 is functionally identical to i1
	i2 := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{
				IngressRuleValue: ingressrulevalue(backend("kuard", intstr.FromInt(8080))),
			}},
		},
	}

	// i2a is missing a http key from the spec.rule.
	// see issue 606
	i2a := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{
				Host: "test1.test.com",
			}},
		},
	}

	// i3 is similar to i2 but includes a hostname on the ingress rule
	i3 := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{{
				Hosts:      []string{"kuard.example.com"},
				SecretName: sec1.Name,
			}},
			Rules: []v1beta1.IngressRule{{
				Host:             "kuard.example.com",
				IngressRuleValue: ingressrulevalue(backend("kuard", intstr.FromInt(8080))),
			}},
		},
	}
	// i4 is like i1 except it uses a named service port
	i4 := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: v1beta1.IngressSpec{
			Backend: backend("kuard", intstr.FromString("http"))},
	}
	// i5 is functionally identical to i2
	i5 := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{
				IngressRuleValue: ingressrulevalue(backend("kuard", intstr.FromString("http"))),
			}},
		},
	}
	// i6 contains two named vhosts which point to the same service
	// one of those has TLS
	i6 := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "two-vhosts",
			Namespace: "default",
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{{
				Hosts:      []string{"b.example.com"},
				SecretName: sec1.Name,
			}},
			Rules: []v1beta1.IngressRule{{
				Host:             "a.example.com",
				IngressRuleValue: ingressrulevalue(backend("kuard", intstr.FromInt(8080))),
			}, {
				Host:             "b.example.com",
				IngressRuleValue: ingressrulevalue(backend("kuard", intstr.FromString("http"))),
			}},
		},
	}
	i6a := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "two-vhosts",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.allow-http": "false",
			},
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{{
				Hosts:      []string{"b.example.com"},
				SecretName: sec1.Name,
			}},
			Rules: []v1beta1.IngressRule{{
				Host:             "a.example.com",
				IngressRuleValue: ingressrulevalue(backend("kuard", intstr.FromInt(8080))),
			}, {
				Host:             "b.example.com",
				IngressRuleValue: ingressrulevalue(backend("kuard", intstr.FromString("http"))),
			}},
		},
	}
	i6b := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "two-vhosts",
			Namespace: "default",
			Annotations: map[string]string{
				"ingress.kubernetes.io/force-ssl-redirect": "true",
			},
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{{
				Hosts:      []string{"b.example.com"},
				SecretName: sec1.Name,
			}},
			Rules: []v1beta1.IngressRule{{
				Host:             "b.example.com",
				IngressRuleValue: ingressrulevalue(backend("kuard", intstr.FromString("http"))),
			}},
		},
	}
	i6c := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "two-vhosts",
			Namespace: "default",
			Annotations: map[string]string{
				"ingress.kubernetes.io/force-ssl-redirect": "true",
				"kubernetes.io/ingress.allow-http":         "false",
			},
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{{
				Hosts:      []string{"b.example.com"},
				SecretName: sec1.Name,
			}},
			Rules: []v1beta1.IngressRule{{
				Host:             "b.example.com",
				IngressRuleValue: ingressrulevalue(backend("kuard", intstr.FromString("http"))),
			}},
		},
	}

	// i7 contains a single vhost with two paths
	i7 := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "two-paths",
			Namespace: "default",
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{{
				Hosts:      []string{"b.example.com"},
				SecretName: sec1.Name,
			}},
			Rules: []v1beta1.IngressRule{{
				Host: "b.example.com",
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{{
							Backend: v1beta1.IngressBackend{
								ServiceName: "kuard",
								ServicePort: intstr.FromString("http"),
							},
						}, {
							Path: "/kuarder",
							Backend: v1beta1.IngressBackend{
								ServiceName: "kuarder",
								ServicePort: intstr.FromInt(8080),
							},
						}},
					},
				},
			}},
		},
	}

	// i8 is identical to i7 but uses multiple IngressRules
	i8 := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "two-rules",
			Namespace: "default",
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{{
				Hosts:      []string{"b.example.com"},
				SecretName: sec1.Name,
			}},
			Rules: []v1beta1.IngressRule{{
				Host: "b.example.com",
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{{
							Backend: v1beta1.IngressBackend{
								ServiceName: "kuard",
								ServicePort: intstr.FromString("http"),
							},
						}},
					},
				},
			}, {
				Host: "b.example.com",
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{{
							Path: "/kuarder",
							Backend: v1beta1.IngressBackend{
								ServiceName: "kuarder",
								ServicePort: intstr.FromInt(8080),
							},
						}},
					},
				},
			}},
		},
	}
	// i9 is identical to i8 but disables non TLS connections
	i9 := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "two-rules",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.allow-http": "false",
			},
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{{
				Hosts:      []string{"b.example.com"},
				SecretName: sec1.Name,
			}},
			Rules: []v1beta1.IngressRule{{
				Host: "b.example.com",
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{{
							Backend: v1beta1.IngressBackend{
								ServiceName: "kuard",
								ServicePort: intstr.FromString("http"),
							},
						}},
					},
				},
			}, {
				Host: "b.example.com",
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{{
							Path: "/kuarder",
							Backend: v1beta1.IngressBackend{
								ServiceName: "kuarder",
								ServicePort: intstr.FromInt(8080),
							},
						}},
					},
				},
			}},
		},
	}

	// i10 specifies a minimum tls version
	i10 := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "two-rules",
			Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/tls-minimum-protocol-version": "1.3",
			},
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{{
				Hosts:      []string{"b.example.com"},
				SecretName: sec1.Name,
			}},
			Rules: []v1beta1.IngressRule{{
				Host: "b.example.com",
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{{
							Backend: v1beta1.IngressBackend{
								ServiceName: "kuard",
								ServicePort: intstr.FromString("http"),
							},
						}},
					},
				},
			}},
		},
	}

	// i11 has a websocket route
	i11 := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "websocket",
			Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/websocket-routes": "/ws1 , /ws2",
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{{
							Backend: v1beta1.IngressBackend{
								ServiceName: "kuard",
								ServicePort: intstr.FromString("http"),
							},
						}, {
							Path: "/ws1",
							Backend: v1beta1.IngressBackend{
								ServiceName: "kuard",
								ServicePort: intstr.FromString("http"),
							},
						}},
					},
				},
			}},
		},
	}

	// i12a has an invalid timeout
	i12a := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "timeout",
			Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/request-timeout": "peanut",
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{{
							Path: "/",
							Backend: v1beta1.IngressBackend{
								ServiceName: "kuard",
								ServicePort: intstr.FromString("http"),
							},
						}},
					},
				},
			}},
		},
	}

	// i12b has a reasonable timeout
	i12b := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "timeout",
			Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/request-timeout": "1m30s", // 90 seconds y'all
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{{
							Path: "/",
							Backend: v1beta1.IngressBackend{
								ServiceName: "kuard",
								ServicePort: intstr.FromString("http"),
							},
						}},
					},
				},
			}},
		},
	}

	// i12c has an unreasonable timeout
	i12c := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "timeout",
			Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/request-timeout": "infinite",
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{
				IngressRuleValue: v1beta1.IngressRuleValue{HTTP: &v1beta1.HTTPIngressRuleValue{
					Paths: []v1beta1.HTTPIngressPath{{Path: "/",
						Backend: v1beta1.IngressBackend{ServiceName: "kuard",
							ServicePort: intstr.FromString("http")},
					}}},
				}}}},
	}

	// i13 a and b are a pair of ingresses for the same vhost
	// they represent a tricky way over 'overlaying' routes from one
	// ingress onto another
	i13a := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: "default",
			Annotations: map[string]string{
				"ingress.kubernetes.io/force-ssl-redirect": "true",
			},
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{{
				Hosts:      []string{"example.com"},
				SecretName: "example-tls",
			}},
			Rules: []v1beta1.IngressRule{{
				Host: "example.com",
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{{
							Path: "/",
							Backend: v1beta1.IngressBackend{
								ServiceName: "app-service",
								ServicePort: intstr.FromInt(8080),
							},
						}},
					},
				},
			}},
		},
	}
	i13b := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "challenge", Namespace: "nginx-ingress"},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{
				Host: "example.com",
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{{
							Path: "/.well-known/acme-challenge/gVJl5NWL2owUqZekjHkt_bo3OHYC2XNDURRRgLI5JTk",
							Backend: v1beta1.IngressBackend{
								ServiceName: "challenge-service",
								ServicePort: intstr.FromInt(8009),
							},
						}},
					},
				},
			}},
		},
	}

	i3a := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{
				IngressRuleValue: ingressrulevalue(backend("kuard", intstr.FromInt(80))),
			}},
		},
	}

	i14 := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "timeout",
			Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/retry-on":        "gateway-error",
				"enroute.saaras.io/num-retries":     "6",
				"enroute.saaras.io/per-try-timeout": "10s",
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{{
							Path: "/",
							Backend: v1beta1.IngressBackend{
								ServiceName: "kuard",
								ServicePort: intstr.FromString("http"),
							},
						}},
					},
				},
			}},
		},
	}

	// s3a and b have http/2 protocol annotations
	s3a := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/upstream-protocol.h2c": "80,http",
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8888),
			}},
		},
	}

	s3b := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s3a.Name,
			Namespace: s3a.Namespace,
			Annotations: map[string]string{
				"enroute.saaras.io/upstream-protocol.h2": "80,http",
			},
		},
		Spec: s3a.Spec,
	}

	s3c := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s3b.Name,
			Namespace: s3b.Namespace,
			Annotations: map[string]string{
				"enroute.saaras.io/upstream-protocol.tls": "80,http",
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8888),
			}},
		},
	}

	sec13 := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-tls",
			Namespace: "default",
		},
		Type: v1.SecretTypeTLS,
		Data: map[string][]byte{
			v1.TLSCertKey:       []byte("certificate"),
			v1.TLSPrivateKeyKey: []byte("key"),
		},
	}

	s13a := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-service",
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

	s13b := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "challenge-service",
			Namespace: "nginx-ingress",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       8009,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}

	ir1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
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

	// ir1a tcp forwards traffic to default/kuard:8080 by TLS terminating it
	// first.
	ir1a := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard-tcp",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "kuard.example.com",
				TLS: &gatewayhostv1.TLS{
					SecretName: sec1.Name,
				},
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			}},
			TCPProxy: &gatewayhostv1.TCPProxy{
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			},
		},
	}

	// ir1b tcp forwards traffic to default/kuard:8080 by TLS pass-throughing
	// it.
	ir1b := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard-tcp",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "kuard.example.com",
				TLS: &gatewayhostv1.TLS{
					Passthrough: true,
				},
			},
			TCPProxy: &gatewayhostv1.TCPProxy{
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			},
		},
	}

	// ir1c tcp delegates to another ingress route, concretely to
	// marketing/kuard-tcp. it.
	ir1c := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard-tcp",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "kuard.example.com",
				TLS: &gatewayhostv1.TLS{
					Passthrough: true,
				},
			},
			TCPProxy: &gatewayhostv1.TCPProxy{
				Delegate: &gatewayhostv1.Delegate{
					Name:      "kuard-tcp",
					Namespace: "marketing",
				},
			},
		},
	}

	// ir1d tcp forwards traffic to default/kuard:8080 by TLS pass-throughing
	// it.
	ir1d := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard-tcp",
			Namespace: "marketing",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			TCPProxy: &gatewayhostv1.TCPProxy{
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			},
		},
	}

	ir1e := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
					HealthCheck: &gatewayhostv1.HealthCheck{
						Path: "/healthz",
					},
				}},
			}},
		},
	}

	// ir2 is like ir1 but refers to two backend services
	ir2 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}, {
					Name: "kuarder",
					Port: 8080,
				}},
			}},
		},
	}

	// ir3 delegates a route to ir4
	ir3 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/blog",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name:      "blog",
					Namespace: "marketing",
				},
			}},
		},
	}

	// ir4 is a delegate gatewayhost, and itself delegates to another one.
	ir4 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "blog",
			Namespace: "marketing",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/blog",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "blog",
					Port: 8080,
				}},
			}, {
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/blog/admin",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name:      "marketing-admin",
					Namespace: "operations",
				},
			}},
		},
	}

	// ir5 is a delegate gatewayhost
	ir5 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "marketing-admin",
			Namespace: "operations",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/blog/admin",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "blog-admin",
					Port: 8080,
				}},
			}},
		},
	}

	// ir6 has TLS and does not specify min tls version
	ir6 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "foo.com",
				TLS: &gatewayhostv1.TLS{
					SecretName: "secret",
				},
			},
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

	// ir7 has TLS and specifies min tls version of 1.2
	ir7 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "foo.com",
				TLS: &gatewayhostv1.TLS{
					SecretName:             "secret",
					MinimumProtocolVersion: "1.2",
				},
			},
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

	// ir8 has TLS and specifies min tls version of 1.3
	ir8 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "foo.com",
				TLS: &gatewayhostv1.TLS{
					SecretName:             "secret",
					MinimumProtocolVersion: "1.3",
				},
			},
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

	// ir9 has TLS and specifies an invalid min tls version of 0.9999
	ir9 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "foo.com",
				TLS: &gatewayhostv1.TLS{
					SecretName:             "secret",
					MinimumProtocolVersion: "0.9999",
				},
			},
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

	// ir10 has a websocket route
	ir10 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			}, {
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/websocket",
				}},
				EnableWebsockets: true,
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			}},
		},
	}

	// ir11 has a prefix-rewrite route
	ir11 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			}, {
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/websocket",
				}},
				PrefixRewrite: "/",
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			}},
		},
	}

	// ir13 has two routes to the same service with different
	// weights
	ir13 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/a",
				}},
				Services: []gatewayhostv1.Service{{
					Name:   "kuard",
					Port:   8080,
					Weight: 90,
				}},
			}, {
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/b",
				}},
				Services: []gatewayhostv1.Service{{Name: "kuard",
					Port:   8080,
					Weight: 60,
				}},
			}},
		},
	}
	// ir13a has one route to the same service with two different weights
	ir13a := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/a",
				}},
				Services: []gatewayhostv1.Service{{
					Name:   "kuard",
					Port:   8080,
					Weight: 90,
				}, {
					Name:   "kuard",
					Port:   8080,
					Weight: 60,
				}},
			}},
		},
	}

	// ir14 has TLS and allows insecure
	ir14 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "foo.com",
				TLS: &gatewayhostv1.TLS{
					SecretName: "secret",
				},
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				PermitInsecure: true,
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			}},
		},
	}

	ir15 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "bar.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				RetryPolicy: &gatewayhostv1.RetryPolicy{
					NumRetries:    6,
					PerTryTimeout: "10s",
				},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			}},
		},
	}

	ir15a := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "bar.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				RetryPolicy: &gatewayhostv1.RetryPolicy{
					NumRetries:    6,
					PerTryTimeout: "please",
				},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			}},
		},
	}

	ir15b := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "bar.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				RetryPolicy: &gatewayhostv1.RetryPolicy{
					NumRetries:    0,
					PerTryTimeout: "10s",
				},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			}},
		},
	}

	ir16a := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "bar.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				TimeoutPolicy: &gatewayhostv1.TimeoutPolicy{
					Request: "peanut",
				},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			}},
		},
	}

	ir16b := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "bar.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				TimeoutPolicy: &gatewayhostv1.TimeoutPolicy{
					Request: "1m30s", // 90 seconds y'all
				},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			}},
		},
	}

	ir16c := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "bar.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				TimeoutPolicy: &gatewayhostv1.TimeoutPolicy{
					Request: "infinite",
				},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			}},
		},
	}

	ir17 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
					UpstreamValidation: &gatewayhostv1.UpstreamValidation{
						CACertificate: "ca",
						SubjectName:   "example.com",
					},
				}},
			}},
		},
	}

	s5 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "blog-admin",
			Namespace: "operations",
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

	// s1a carries the tls annotation
	s1a := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/upstream-protocol.tls": "8080",
			},
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

	// s1b carries all four ingress annotations{
	s1b := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/max-connections":      "9000",
				"enroute.saaras.io/max-pending-requests": "4096",
				"enroute.saaras.io/max-requests":         "404",
				"enroute.saaras.io/max-retries":          "7",
			},
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

	// s2 is like s1 but with a different name
	s2 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuarder",
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
	// s3 is like s1 but has a different port
	s3 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       9999,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}

	s4 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "blog",
			Namespace: "marketing",
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

	s6 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "marketing",
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

	s7 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "home",
			Namespace: "finance",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:     "http",
				Protocol: "TCP",
				Port:     8080,
			}},
		},
	}

	//s8 := &v1.Service{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name:      "green",
	//		Namespace: "marketing",
	//	},
	//	Spec: v1.ServiceSpec{
	//		Ports: []v1.ServicePort{{
	//			Name:     "http",
	//			Protocol: "TCP",
	//			Port:     80,
	//		}},
	//	},
	//}

	//s9 := &v1.Service{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name:      "nginx",
	//		Namespace: "default",
	//	},
	//	Spec: v1.ServiceSpec{
	//		Ports: []v1.ServicePort{{
	//			Protocol: "TCP",
	//			Port:     80,
	//		}},
	//	},
	//}

	//s10 := &v1.Service{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name:      "tls-passthrough",
	//		Namespace: "default",
	//	},
	//	Spec: v1.ServiceSpec{
	//		Ports: []v1.ServicePort{{
	//			Name:       "https",
	//			Protocol:   "TCP",
	//			Port:       443,
	//			TargetPort: intstr.FromInt(443),
	//		}, {
	//			Name:       "http",
	//			Protocol:   "TCP",
	//			Port:       80,
	//			TargetPort: intstr.FromInt(80),
	//		}},
	//	},
	//}

	//s11 := &v1.Service{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name:      "blog",
	//		Namespace: "it",
	//	},
	//	Spec: v1.ServiceSpec{
	//		Ports: []v1.ServicePort{{
	//			Name:     "blog",
	//			Protocol: "TCP",
	//			Port:     8080,
	//		}},
	//	},
	//}

	//s12 := &v1.Service{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name:      "kuard",
	//		Namespace: "teama",
	//	},
	//	Spec: v1.ServiceSpec{
	//		Ports: []v1.ServicePort{{
	//			Name:       "http",
	//			Protocol:   "TCP",
	//			Port:       8080,
	//			TargetPort: intstr.FromInt(8080),
	//		}},
	//	},
	//}

	//s13 := &v1.Service{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name:      "kuard",
	//		Namespace: "teamb",
	//	},
	//	Spec: v1.ServiceSpec{
	//		Ports: []v1.ServicePort{{
	//			Name:       "http",
	//			Protocol:   "TCP",
	//			Port:       8080,
	//			TargetPort: intstr.FromInt(8080),
	//		}},
	//	},
	//}

	// ir18 tcp forwards traffic to by TLS pass-throughing
	// it. It also exposes non HTTP traffic to the the non secure port of the
	// application so it can give an informational message
	//ir18 := &gatewayhostv1.GatewayHost{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name:      "kuard-tcp",
	//		Namespace: s10.Namespace,
	//	},
	//	Spec: gatewayhostv1.GatewayHostSpec{
	//		VirtualHost: &gatewayhostv1.VirtualHost{
	//			Fqdn: "kuard.example.com",
	//			TLS: &gatewayhostv1.TLS{
	//				Passthrough: true,
	//			},
	//		},
	//		Routes: []gatewayhostv1.Route{{
	//          Conditions: []gatewayhostv1.Condition{{
	//              Prefix: "/",
	//          }},
	//			Services: []gatewayhostv1.Service{{
	//				Name: s10.Name,
	//				Port: 80, // proxy non secure traffic to port 80
	//			}},
	//		}},
	//		TCPProxy: &gatewayhostv1.TCPProxy{
	//			Services: []gatewayhostv1.Service{{
	//				Name: s10.Name,
	//				Port: 443, // ssl passthrough to secure port
	//			}},
	//		},
	//	},
	//}

	//ir19 := &gatewayhostv1.GatewayHost{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name:      "app-with-tls-delegation",
	//		Namespace: s10.Namespace,
	//	},
	//	Spec: gatewayhostv1.GatewayHostSpec{
	//		VirtualHost: &gatewayhostv1.VirtualHost{
	//			Fqdn: "app-with-tls-delegation.127.0.0.1.nip.io",
	//			TLS: &gatewayhostv1.TLS{
	//				SecretName: "heptio-contour/ssl-cert", // not delegated
	//			},
	//		},
	//		Routes: []gatewayhostv1.Route{{
	//          Conditions: []gatewayhostv1.Condition{{
	//              Prefix: "/",
	//          }},
	//			Services: []gatewayhostv1.Service{{
	//				Name: s10.Name,
	//				Port: 80,
	//			}},
	//		}},
	//	},
	//}

	tests := map[string]struct {
		objs []interface{}
		want []Vertex
	}{
		"insert ingress w/ default backend": {
			objs: []interface{}{
				i1,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", route(i1, "/", httpService(s1))),
					),
				},
			),
		},
		"insert ingress w/ single unnamed backend": {
			objs: []interface{}{
				i2,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", route(i2, "/", httpService(s1))),
					),
				},
			),
		},
		"insert ingress with missing spec.rule.http key": {
			objs: []interface{}{
				i2a,
			},
			want: listeners(),
		},
		"insert ingress w/ host name and single backend": {
			objs: []interface{}{
				i3,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("kuard.example.com", route(i3, "/", httpService(s1))),
					),
				},
			),
		},
		"insert ingress w/ default backend then matching service": {
			objs: []interface{}{
				i1,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", route(i1, "/", httpService(s1))),
					),
				},
			),
		},
		"insert service then ingress w/ default backend": {
			objs: []interface{}{
				s1,
				i1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", route(i1, "/", httpService(s1))),
					),
				},
			),
		},
		"insert non matching service then ingress w/ default backend": {
			objs: []interface{}{
				s2,
				i1,
			},
			want: listeners(),
		},
		"insert ingress w/ default backend then matching service with wrong port": {
			objs: []interface{}{
				i1,
				s3,
			},
			want: listeners(),
		},
		"insert unnamed ingress w/ single backend then matching service with wrong port": {
			objs: []interface{}{
				i2,
				s3,
			},
			want: listeners(),
		},
		"insert ingress w/ default backend then matching service w/ named port": {
			objs: []interface{}{
				i4,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", route(i4, "/", httpService(s1))),
					),
				},
			),
		},
		"insert service w/ named port then ingress w/ default backend": {
			objs: []interface{}{
				s1,
				i4,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", route(i4, "/", httpService(s1))),
					),
				},
			),
		},
		"insert ingress w/ single unnamed backend w/ named service port then service": {
			objs: []interface{}{
				i5,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", route(i5, "/", httpService(s1))),
					),
				},
			),
		},
		"insert service then ingress w/ single unnamed backend w/ named service port": {
			objs: []interface{}{
				s1,
				i5,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", route(i5, "/", httpService(s1))),
					),
				},
			),
		},
		"insert secret": {
			objs: []interface{}{
				sec1,
			},
			want: []Vertex{},
		},
		"insert secret then ingress w/o tls": {
			objs: []interface{}{
				sec1,
				i1,
			},
			want: listeners(),
		},
		"insert service, secret then ingress w/o tls": {
			objs: []interface{}{
				s1,
				sec1,
				i1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", prefixroute("/", httpService(s1))),
					),
				},
			),
		},
		// 6-3-2020 - Since no service is specified, Listener shouldn't be created
		// TODO: Debug
		//"insert secret then ingress w/ tls": {
		//	objs: []interface{}{
		//		sec1,
		//		i3,
		//	},
		//	want: listeners(),
		//},
		"insert service, secret then ingress w/ tls": {
			objs: []interface{}{
				s1,
				sec1,
				i3,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("kuard.example.com", prefixroute("/", httpService(s1))),
					),
				},
				&Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						securevirtualhost("kuard.example.com", sec1, prefixroute("/", httpService(s1))),
					),
				},
			),
		},
		"insert service w/ secret with w/ blank ca.crt": {
			objs: []interface{}{
				s1,
				sec3, // issue 1644
				i3,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("kuard.example.com", prefixroute("/", httpService(s1))),
					),
				},
				&Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						securevirtualhost("kuard.example.com", sec3, prefixroute("/", httpService(s1))),
					),
				},
			),
		},

		"insert invalid secret then ingress w/o tls": {
			objs: []interface{}{
				sec2,
				i1,
			},
			want: listeners(),
		},
		// TODO: 6-3-2020 - Incorporate invalid secret handling. This test should pass
		// This test results in creating a https listener (expected behavior is that it shouldn't be created)
		// cdc2913f2d47ea0771ad383eb4eefab35000348b, cba7fe03d0c2e4a3be38d2d95aa8b8622554b0ad, 46d3ae3251a5f2c12a48a4baaaf1b869200300e6, 240366e4fbabe79a76a00e5121b1fc23980a3ead
		//"insert service, invalid secret then ingress w/o tls": {
		//	objs: []interface{}{
		//		s1,
		//		sec2,
		//		i1,
		//	},
		//	want: listeners(
		//		&Listener{
		//			Port: 80,
		//			VirtualHosts: virtualhosts(
		//				virtualhost("*", prefixroute("/", httpService(s1))),
		//			),
		//		},
		//	),
		//},
		//"insert invalid secret then ingress w/ tls": {
		//	objs: []interface{}{
		//		sec2,
		//		i3,
		//	},
		//	want: listeners(),
		//},
		//"insert service, invalid secret then ingress w/ tls": {
		//	objs: []interface{}{
		//		s1,
		//		sec2,
		//		i3,
		//	},
		//	want: listeners(
		//		&Listener{
		//			Port: 80,
		//			VirtualHosts: virtualhosts(
		//				virtualhost("kuard.example.com", prefixroute("/", httpService(s1))),
		//			),
		//		},
		//	),
		//},
		"insert ingress w/ two vhosts": {
			objs: []interface{}{
				i6,
			},
			want: nil,
		},
		"insert ingress w/ two vhosts then matching service": {
			objs: []interface{}{
				i6,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("a.example.com", prefixroute("/", httpService(s1))),
						virtualhost("b.example.com", prefixroute("/", httpService(s1))),
					),
				},
			),
		},
		"insert service then ingress w/ two vhosts": {
			objs: []interface{}{
				s1,
				i6,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("a.example.com", prefixroute("/", httpService(s1))),
						virtualhost("b.example.com", prefixroute("/", httpService(s1))),
					),
				},
			),
		},
		"insert ingress w/ two vhosts then service then secret": {
			objs: []interface{}{
				i6,
				s1,
				sec1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("a.example.com", prefixroute("/", httpService(s1))),
						virtualhost("b.example.com", prefixroute("/", httpService(s1))),
					),
				}, &Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						securevirtualhost("b.example.com", sec1, prefixroute("/", httpService(s1))),
					),
				},
			),
		},
		"insert service then secret then ingress w/ two vhosts": {
			objs: []interface{}{
				s1,
				sec1,
				i6,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("a.example.com", prefixroute("/", httpService(s1))),
						virtualhost("b.example.com", prefixroute("/", httpService(s1))),
					),
				}, &Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						securevirtualhost("b.example.com", sec1, prefixroute("/", httpService(s1))),
					),
				},
			),
		},
		"insert ingress w/ two paths then one service": {
			objs: []interface{}{
				i7,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("b.example.com",
							prefixroute("/", httpService(s1)),
						),
					),
				},
			),
		},
		"insert ingress w/ two paths then services": {
			objs: []interface{}{
				i7,
				s2,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("b.example.com",
							prefixroute("/", httpService(s1)),
							prefixroute("/kuarder", httpService(s2)),
						),
					),
				},
			),
		},
		"insert two services then ingress w/ two ingress rules": {
			objs: []interface{}{
				s1, s2, i8,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("b.example.com",
							prefixroute("/", httpService(s1)),
							prefixroute("/kuarder", httpService(s2)),
						),
					),
				},
			),
		},
		"insert ingress w/ two paths httpAllowed: false": {
			objs: []interface{}{
				i9,
			},
			want: []Vertex{},
		},
		"insert ingress w/ two paths httpAllowed: false then tls and service": {
			objs: []interface{}{
				i9,
				sec1,
				s1, s2,
			},
			want: listeners(
				&Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						securevirtualhost("b.example.com", sec1,
							prefixroute("/", httpService(s1)),
							prefixroute("/kuarder", httpService(s2)),
						),
					),
				},
			),
		},
		"insert default ingress httpAllowed: false": {
			objs: []interface{}{
				i1a,
			},
			want: []Vertex{},
		},
		"insert default ingress httpAllowed: false then tls and service": {
			objs: []interface{}{
				i1a, sec1, s1,
			},
			want: []Vertex{}, // default ingress cannot be tls
		},
		"insert ingress w/ two vhosts httpAllowed: false": {
			objs: []interface{}{
				i6a,
			},
			want: []Vertex{},
		},
		"insert ingress w/ two vhosts httpAllowed: false then tls and service": {
			objs: []interface{}{
				i6a, sec1, s1,
			},
			want: listeners(
				&Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						securevirtualhost("b.example.com", sec1, prefixroute("/", httpService(s1))),
					),
				},
			),
		},
		"insert ingress w/ force-ssl-redirect: true": {
			objs: []interface{}{
				i6b, sec1, s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("b.example.com", routeUpgrade("/", httpService(s1))),
					),
				}, &Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						securevirtualhost("b.example.com", sec1, routeUpgrade("/", httpService(s1))),
					),
				},
			),
		},

		"insert ingress w/ force-ssl-redirect: true and allow-http: false": {
			objs: []interface{}{
				i6c, sec1, s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("b.example.com", routeUpgrade("/", httpService(s1))),
					),
				}, &Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						securevirtualhost("b.example.com", sec1, routeUpgrade("/", httpService(s1))),
					),
				},
			),
		},
		"insert gatewayhost": {
			objs: []interface{}{
				ir1, s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("example.com", prefixroute("/", httpService(s1))),
					),
				},
			),
		},
		"insert gatewayhost w/ healthcheck": {
			objs: []interface{}{
				ir1e, s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("example.com",
							routeCluster("/", &Cluster{
								Upstream: httpService(s1),
								HealthCheck: &gatewayhostv1.HealthCheck{
									Path: "/healthz",
								},
							}),
						),
					),
				},
			),
		},

		"insert gatewayhost with websocket route": {
			objs: []interface{}{
				ir11, s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("example.com",
							prefixroute("/", httpService(s1)),
							routeRewrite("/websocket", "/", httpService(s1)),
						),
					),
				},
			),
		},
		"insert gatewayhost with tcp forward with TLS termination": {
			objs: []interface{}{
				ir1a, s1, sec1,
			},
			want: listeners(
				&Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "kuard.example.com",
								TCPProxy: &TCPProxy{
									Clusters: clusters(
										tcpService(s1),
									),
								},
							},
							Secret:          secret(sec1),
							MinProtoVersion: envoy_api_v2_auth.TlsParameters_TLSv1_1,
						},
					),
				},
			),
		},
		"insert gatewayhost with tcp forward without TLS termination w/ passthrough": {
			objs: []interface{}{
				ir1b, s1,
			},
			want: listeners(
				&Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "kuard.example.com",
								TCPProxy: &TCPProxy{
									Clusters: clusters(
										tcpService(s1),
									),
								},
							},
						},
					),
				},
			),
		},

		"insert root ingress route and delegate ingress route for a tcp proxy": {
			objs: []interface{}{
				ir1d, s6, ir1c,
			},
			want: listeners(
				&Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "kuard.example.com",
								TCPProxy: &TCPProxy{
									Clusters: clusters(
										tcpService(s6),
									),
								},
							},
						},
					),
				},
			),
		},
		"insert gatewayhost with prefix rewrite route": {
			objs: []interface{}{
				ir10, s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("example.com",
							prefixroute("/", httpService(s1)),
							routeWebsocket("/websocket", httpService(s1)),
						),
					),
				},
			),
		},
		"insert gatewayhost and service": {
			objs: []interface{}{
				ir1, s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("example.com", prefixroute("/", httpService(s1))),
					),
				},
			),
		},
		"insert gatewayhost without tls version": {
			objs: []interface{}{
				ir6, s1, sec1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("foo.com", routeUpgrade("/", httpService(s1))),
					),
				}, &Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						securevirtualhost("foo.com", sec1, routeUpgrade("/", httpService(s1))),
					),
				},
			),
		},
		"insert gatewayhost with TLS one insecure": {
			objs: []interface{}{
				ir14, s1, sec1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("foo.com", prefixroute("/", httpService(s1))),
					),
				}, &Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						securevirtualhost("foo.com", sec1, prefixroute("/", httpService(s1))),
					),
				},
			),
		},
		"insert gatewayhost with tls version 1.2": {
			objs: []interface{}{
				ir7, s1, sec1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("foo.com", routeUpgrade("/", httpService(s1))),
					),
				}, &Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "foo.com",
								routes: routemap(
									routeUpgrade("/", httpService(s1)),
								),
							},
							MinProtoVersion: envoy_api_v2_auth.TlsParameters_TLSv1_2,
							Secret:          secret(sec1),
						},
					),
				},
			),
		},
		"insert gatewayhost with tls version 1.3": {
			objs: []interface{}{
				ir8, s1, sec1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("foo.com", routeUpgrade("/", httpService(s1))),
					),
				}, &Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "foo.com",
								routes: routemap(
									routeUpgrade("/", httpService(s1)),
								),
							},
							MinProtoVersion: envoy_api_v2_auth.TlsParameters_TLSv1_3,
							Secret:          secret(sec1),
						},
					),
				},
			),
		},
		"insert gatewayhost with invalid tls version": {
			objs: []interface{}{
				ir9, s1, sec1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("foo.com", routeUpgrade("/", httpService(s1))),
					),
				}, &Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						securevirtualhost("foo.com", sec1, routeUpgrade("/", httpService(s1))),
					),
				},
			),
		},
		"insert gatewayhost referencing two backends, one missing": {
			objs: []interface{}{
				ir2, s2,
			},
			want: listeners(),
		},
		"insert gatewayhost referencing two backends": {
			objs: []interface{}{
				ir2, s1, s2,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("example.com", prefixroute("/", httpService(s1), httpService(s2))),
					),
				},
			),
		},
		"insert ingress w/ tls min proto annotation": {
			objs: []interface{}{
				i10,
				sec1,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("b.example.com", prefixroute("/", httpService(s1))),
					),
				}, &Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						&SecureVirtualHost{
							VirtualHost: VirtualHost{
								Name: "b.example.com",
								routes: routemap(
									prefixroute("/", httpService(s1)),
								),
							},
							MinProtoVersion: envoy_api_v2_auth.TlsParameters_TLSv1_3,
							Secret:          secret(sec1),
						},
					),
				},
			),
		},
		"insert ingress w/ websocket route annotation": {
			objs: []interface{}{
				i11,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*",
							prefixroute("/", httpService(s1)),
							routeWebsocket("/ws1", httpService(s1)),
						),
					),
				},
			),
		},
		"insert ingress w/ invalid timeout annotation": {
			objs: []interface{}{
				i12a,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", &Route{
							PathCondition: prefix("/"),
							Clusters:      clustermap(s1),
							TimeoutPolicy: &TimeoutPolicy{
								Timeout: -1, // invalid timeout equals infinity ¯\_(ツ)_/¯.
							},
						}),
					),
				},
			),
		},
		"insert gatewayhost w/ invalid timeoutpolicy": {
			objs: []interface{}{
				ir16a,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("bar.com", &Route{
							PathCondition: prefix("/"),
							Clusters:      clustermap(s1),
							TimeoutPolicy: &TimeoutPolicy{
								Timeout: -1, // invalid timeout equals infinity ¯\_(ツ)_/¯.
							},
						}),
					),
				},
			),
		},
		"insert ingress w/ valid timeout annotation": {
			objs: []interface{}{
				i12b,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", &Route{
							PathCondition: prefix("/"),
							Clusters:      clustermap(s1),
							TimeoutPolicy: &TimeoutPolicy{
								Timeout: 90 * time.Second,
							},
						}),
					),
				},
			),
		},
		"insert gatewayhost w/ valid timeoutpolicy": {
			objs: []interface{}{
				ir16b,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("bar.com", &Route{
							PathCondition: prefix("/"),
							Clusters:      clustermap(s1),
							TimeoutPolicy: &TimeoutPolicy{
								Timeout: 90 * time.Second,
							},
						}),
					),
				},
			),
		},
		"insert ingress w/ infinite timeout annotation": {
			objs: []interface{}{
				i12c,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", &Route{
							PathCondition: prefix("/"),
							Clusters:      clustermap(s1),
							TimeoutPolicy: &TimeoutPolicy{
								Timeout: -1,
							},
						}),
					),
				},
			),
		},
		"insert gatewayhost w/ infinite timeoutpolicy": {
			objs: []interface{}{
				ir16c,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("bar.com", &Route{
							PathCondition: prefix("/"),
							Clusters:      clustermap(s1),
							TimeoutPolicy: &TimeoutPolicy{
								Timeout: -1,
							},
						}),
					),
				},
			),
		},
		"insert gatewayhost w/ missing tls annotation": {
			objs: []interface{}{
				cert1, ir17, s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("example.com",
							prefixroute("/", httpService(s1)),
						),
					),
				},
			),
		},
		"insert gatewayhost w/ missing certificate": {
			objs: []interface{}{
				ir17, s1a,
			},
			want: listeners(), // no listeners, missing certificates for upstream validation
		},
		"insert gatewayhost expecting verification": {
			objs: []interface{}{
				cert1, ir17, s1a,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("example.com",
							routeCluster("/",
								&Cluster{
									Upstream: &HTTPService{
										TCPService: TCPService{
											Name:        s1a.Name,
											Namespace:   s1a.Namespace,
											ServicePort: &s1a.Spec.Ports[0],
										},
										Protocol: "tls",
									},
									UpstreamValidation: &UpstreamValidation{
										CACertificate: secret(cert1),
										SubjectName:   "example.com",
									},
								},
							),
						),
					),
				},
			),
		},
		// 6-3-2020 TODO: Revisit gatewayhost delegtion, tls delegation
		//"insert gatewayhost with missing tls delegation should not present port 80": {
		//	objs: []interface{}{
		//		s10, ir19,
		//	},
		//	want: listeners(), // no listeners, ir19 is invalid
		//},
		"insert root ingress route and delegate ingress route": {
			objs: []interface{}{
				ir5, s4, ir4, s5, ir3,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("example.com",
							prefixroute("/blog", httpService(s4)),
							prefixroute("/blog/admin", httpService(s5)),
						),
					),
				},
			),
		},
		"insert ingress with retry annotations": {
			objs: []interface{}{
				ir15,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("bar.com", &Route{
							PathCondition: prefix("/"),
							Clusters:      clustermap(s1),
							RetryPolicy: &RetryPolicy{
								RetryOn:       "5xx",
								NumRetries:    6,
								PerTryTimeout: 10 * time.Second,
							},
						}),
					),
				},
			),
		},
		"insert ingress with invalid perTryTimeout": {
			objs: []interface{}{
				ir15a,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("bar.com", &Route{
							PathCondition: prefix("/"),
							Clusters:      clustermap(s1),
							RetryPolicy: &RetryPolicy{
								RetryOn:       "5xx",
								NumRetries:    6,
								PerTryTimeout: 0,
							},
						}),
					),
				},
			),
		},

		"insert ingress with zero retry count": {
			objs: []interface{}{
				ir15b,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("bar.com", &Route{
							PathCondition: prefix("/"),
							Clusters:      clustermap(s1),
							RetryPolicy: &RetryPolicy{
								RetryOn:       "5xx",
								NumRetries:    1,
								PerTryTimeout: 10 * time.Second,
							},
						}),
					),
				},
			),
		},
		"insert gatewayhost with retrypolicy": {
			objs: []interface{}{
				i14,
				s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*", &Route{
							PathCondition: prefix("/"),
							Clusters:      clustermap(s1),
							RetryPolicy: &RetryPolicy{
								RetryOn:       "gateway-error",
								NumRetries:    6,
								PerTryTimeout: 10 * time.Second,
							},
						}),
					),
				},
			),
		},
		"insert ingress overlay": {
			objs: []interface{}{
				i13a, i13b, sec13, s13a, s13b,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("example.com",
							routeUpgrade("/", httpService(s13a)),
							prefixroute("/.well-known/acme-challenge/gVJl5NWL2owUqZekjHkt_bo3OHYC2XNDURRRgLI5JTk", httpService(s13b)),
						),
					),
				}, &Listener{
					Port: 443,
					VirtualHosts: virtualhosts(
						securevirtualhost("example.com", sec13,
							routeUpgrade("/", httpService(s13a)),
							prefixroute("/.well-known/acme-challenge/gVJl5NWL2owUqZekjHkt_bo3OHYC2XNDURRRgLI5JTk", httpService(s13b)),
						),
					),
				},
			),
		},
		"h2c service annotation": {
			objs: []interface{}{
				i3a, s3a,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*",
							prefixroute("/", &HTTPService{
								TCPService: TCPService{
									Name:        s3a.Name,
									Namespace:   s3a.Namespace,
									ServicePort: &s3a.Spec.Ports[0],
								},
								Protocol: "h2c",
							}),
						),
					),
				},
			),
		},
		"h2 service annotation": {
			objs: []interface{}{
				i3a, s3b,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*",
							prefixroute("/", &HTTPService{
								TCPService: TCPService{
									Name:        s3b.Name,
									Namespace:   s3b.Namespace,
									ServicePort: &s3b.Spec.Ports[0],
								},
								Protocol: "h2",
							}),
						),
					),
				},
			),
		},
		"tls service annotation": {
			objs: []interface{}{
				i3a, s3c,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*",
							prefixroute("/", &HTTPService{
								TCPService: TCPService{
									Name:        s3c.Name,
									Namespace:   s3c.Namespace,
									ServicePort: &s3c.Spec.Ports[0],
								},
								Protocol: "tls",
							}),
						),
					),
				},
			),
		},
		"insert ingress then service w/ upstream annotations": {
			objs: []interface{}{
				i1,
				s1b,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("*",
							prefixroute("/", &HTTPService{
								TCPService: TCPService{
									Name:               s1b.Name,
									Namespace:          s1b.Namespace,
									ServicePort:        &s1b.Spec.Ports[0],
									MaxConnections:     9000,
									MaxPendingRequests: 4096,
									MaxRequests:        404,
									MaxRetries:         7,
								},
							}),
						),
					),
				},
			),
		},
		"insert gatewayhost with two routes to the same service": {
			objs: []interface{}{
				ir13, s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("example.com",
							routeCluster("/a", &Cluster{
								Upstream: &HTTPService{
									TCPService: TCPService{
										Name:        s1.Name,
										Namespace:   s1.Namespace,
										ServicePort: &s1.Spec.Ports[0],
									},
								},
								Weight: 90,
							}),
							routeCluster("/b", &Cluster{
								Upstream: &HTTPService{
									TCPService: TCPService{
										Name:        s1.Name,
										Namespace:   s1.Namespace,
										ServicePort: &s1.Spec.Ports[0],
									},
								},
								Weight: 60,
							}),
						),
					),
				},
			),
		},
		"insert gatewayhost with one routes to the same service with two different weights": {
			objs: []interface{}{
				ir13a, s1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("example.com",
							routeCluster("/a",
								&Cluster{
									Upstream: &HTTPService{
										TCPService: TCPService{
											Name:        s1.Name,
											Namespace:   s1.Namespace,
											ServicePort: &s1.Spec.Ports[0],
										},
									},
									Weight: 90,
								}, &Cluster{
									Upstream: &HTTPService{
										TCPService: TCPService{
											Name:        s1.Name,
											Namespace:   s1.Namespace,
											ServicePort: &s1.Spec.Ports[0],
										},
									},
									Weight: 60,
								},
							),
						),
					),
				},
			),
		},
		"gatewayhost delegated to non existent object": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "example-com",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "example.com",
						},
						Routes: []gatewayhostv1.Route{{
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/finance",
							}},
							Delegate: &gatewayhostv1.Delegate{
								Name:      "non-existent",
								Namespace: "non-existent",
							},
						}},
					},
				},
			},
			want: nil, // no listener created
		},
		"gatewayhost delegates to itself": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "example-com",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "example.com",
						},
						Routes: []gatewayhostv1.Route{{
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/finance",
							}},
							Delegate: &gatewayhostv1.Delegate{
								Name:      "example-com",
								Namespace: "default",
							},
						}},
					},
				},
			},
			want: nil, // no listener created
		},
		"gatewayhost delegates to incorrect prefix": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "example-com",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "example.com",
						},
						Routes: []gatewayhostv1.Route{{
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/finance",
							}},
							Delegate: &gatewayhostv1.Delegate{
								Name:      "finance-root",
								Namespace: "finance",
							},
						}},
					},
				},
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "finance",
						Name:      "finance-root",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						Routes: []gatewayhostv1.Route{{
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/prefixDoesntMatch",
							}},
							Services: []gatewayhostv1.Service{{
								Name: "home",
							}},
						}},
					},
				},
			},
			want: nil, // no listener created
		},
		"gatewayhost delegate to prefix, but no matching path in delegate": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "example-com",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "example.com",
						},
						Routes: []gatewayhostv1.Route{{
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/foo",
							}},
							Delegate: &gatewayhostv1.Delegate{
								Name:      "finance-root",
								Namespace: "finance",
							},
						}},
					},
				},
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "finance",
						Name:      "finance-root",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						Routes: []gatewayhostv1.Route{{
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/foobar",
							}},
							Services: []gatewayhostv1.Service{{
								Name: "home",
							}},
						}, {
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/foo/bar",
							}},
							Services: []gatewayhostv1.Service{{
								Name: "home",
							}},
						}},
					},
				},
			},
			want: nil, // no listener created
		},
		"gatewayhost cycle": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "example-com",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "example.com",
						},
						Routes: []gatewayhostv1.Route{{
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/finance",
							}},
							Delegate: &gatewayhostv1.Delegate{
								Name:      "finance-root",
								Namespace: "finance",
							},
						}},
					},
				},
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "finance",
						Name:      "finance-root",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						Routes: []gatewayhostv1.Route{{
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/finance",
							}},
							Services: []gatewayhostv1.Service{{
								Name: "home",
								Port: 8080,
							}},
						}, {
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/finance/stocks",
							}},
							Delegate: &gatewayhostv1.Delegate{
								Name:      "example-com",
								Namespace: "default",
							},
						}},
					},
				},
				s7,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						virtualhost("example.com", prefixroute("/finance", httpService(s7))),
					),
				},
			),
		},
		// 6-3-2020 TODO: GatewayHost delegation
		//"gatewayhost root delegates to another ingressroute root": {
		//	objs: []interface{}{
		//		&gatewayhostv1.GatewayHost{
		//			ObjectMeta: metav1.ObjectMeta{
		//				Name:      "root-blog",
		//				Namespace: "roots",
		//			},
		//			Spec: gatewayhostv1.GatewayHostSpec{
		//				VirtualHost: &gatewayhostv1.VirtualHost{
		//					Fqdn: "blog.containersteve.com",
		//				},
		//				Routes: []gatewayhostv1.Route{{
		//                  Conditions: []gatewayhostv1.Condition{{
		//                      Prefix: "/",
		//                  }},
		//					Delegate: &gatewayhostv1.Delegate{
		//						Name:      "blog",
		//						Namespace: "marketing",
		//					},
		//				}},
		//			},
		//		},
		//		&gatewayhostv1.GatewayHost{
		//			ObjectMeta: metav1.ObjectMeta{
		//				Name:      "blog",
		//				Namespace: "marketing",
		//			},
		//			Spec: gatewayhostv1.GatewayHostSpec{
		//				VirtualHost: &gatewayhostv1.VirtualHost{
		//					Fqdn: "www.containersteve.com",
		//				},
		//				Routes: []gatewayhostv1.Route{{
		//                  Conditions: []gatewayhostv1.Condition{{
		//                      Prefix: "/",
		//                  }},
		//					Services: []gatewayhostv1.Service{{
		//						Name: "green",
		//						Port: 80,
		//					}},
		//				}},
		//			},
		//		},
		//		s8,
		//	},
		//	want: listeners(
		//		&Listener{
		//			Port: 80,
		//			VirtualHosts: virtualhosts(
		//				virtualhost("www.containersteve.com", prefixroute("/", httpService(s8))),
		//			),
		//		},
		//	),
		//},
		// issue 1399
		//		"service shared across ingress and gatewayhost tcpproxy": {
		//			objs: []interface{}{
		//				sec1,
		//				s9,
		//				&v1beta1.Ingress{
		//					ObjectMeta: metav1.ObjectMeta{
		//						Name:      "nginx",
		//						Namespace: "default",
		//					},
		//					Spec: v1beta1.IngressSpec{
		//						TLS: []v1beta1.IngressTLS{{
		//							Hosts:      []string{"example.com"},
		//							SecretName: s1.Name,
		//						}},
		//						Rules: []v1beta1.IngressRule{{
		//							Host:             "example.com",
		//							IngressRuleValue: ingressrulevalue(backend(s9.Name, intstr.FromInt(80))),
		//						}},
		//					},
		//				},
		//				&gatewayhostv1.GatewayHost{
		//					ObjectMeta: metav1.ObjectMeta{
		//						Name:      "nginx",
		//						Namespace: "default",
		//					},
		//					Spec: gatewayhostv1.GatewayHostSpec{
		//						VirtualHost: &projcontour.VirtualHost{
		//							Fqdn: "example.com",
		//							TLS: &projcontour.TLS{
		//								SecretName: sec1.Name,
		//							},
		//						},
		//						TCPProxy: &gatewayhostv1.TCPProxy{
		//							Services: []gatewayhostv1.Service{{
		//								Name: s9.Name,
		//								Port: 80,
		//							}},
		//						},
		//					},
		//				},
		//			},
		//			want: listeners(
		//				&Listener{
		//					Port: 80,
		//					VirtualHosts: virtualhosts(
		//						virtualhost("example.com", prefixroute("/", service(s9))),
		//					),
		//				},
		//				&Listener{
		//					Port: 443,
		//					VirtualHosts: virtualhosts(
		//						&SecureVirtualHost{
		//							VirtualHost: VirtualHost{
		//								Name: "example.com",
		//							},
		//							MinProtoVersion: envoy_api_v2_auth.TlsParameters_TLSv1_1,
		//							Secret:          secret(sec1),
		//							TCPProxy: &TCPProxy{
		//								Clusters: clusters(service(s9)),
		//							},
		//						},
		//					),
		//				},
		//			),
		//		},

	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var kc KubernetesCache
			for _, o := range tc.objs {
				kc.Insert(o)
			}
			dag := BuildDAG(&kc)

			got := make(map[int]*Listener)
			dag.Visit(listenerMap(got).Visit)

			want := make(map[int]*Listener)
			for _, v := range tc.want {
				if l, ok := v.(*Listener); ok {
					want[l.Port] = l
				}
			}

			opts := []cmp.Option{
				cmp.AllowUnexported(Listener{}, VirtualHost{}),
			}
			if diff := cmp.Diff(want, got, opts...); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

type listenerMap map[int]*Listener

func (lm listenerMap) Visit(v Vertex) {
	if l, ok := v.(*Listener); ok {
		lm[l.Port] = l
	}
}

func backend(name string, port intstr.IntOrString) *v1beta1.IngressBackend {
	return &v1beta1.IngressBackend{
		ServiceName: name,
		ServicePort: port,
	}
}

func ingressrulevalue(backend *v1beta1.IngressBackend) v1beta1.IngressRuleValue {
	return v1beta1.IngressRuleValue{
		HTTP: &v1beta1.HTTPIngressRuleValue{
			Paths: []v1beta1.HTTPIngressPath{{
				Backend: *backend,
			}},
		},
	}
}

func TestBuilderLookupHTTPService(t *testing.T) {
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
	services := map[Meta]*v1.Service{
		{name: "service1", namespace: "default"}: s1,
	}

	tests := map[string]struct {
		Meta
		port intstr.IntOrString
		want *HTTPService
	}{
		"lookup service by port number": {
			Meta: Meta{name: "service1", namespace: "default"},
			port: intstr.FromInt(8080),
			want: httpService(s1),
		},
		"lookup service by port name": {
			Meta: Meta{name: "service1", namespace: "default"},
			port: intstr.FromString("http"),
			want: httpService(s1),
		},
		"lookup service by port number (as string)": {
			Meta: Meta{name: "service1", namespace: "default"},
			port: intstr.Parse("8080"),
			want: httpService(s1),
		},
		"lookup service by port number (from string)": {
			Meta: Meta{name: "service1", namespace: "default"},
			port: intstr.FromString("8080"),
			want: httpService(s1),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			b := builder{
				source: &KubernetesCache{
					services: services,
				},
			}
			got := b.lookupHTTPService(tc.Meta, tc.port)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestDAGRootNamespaces(t *testing.T) {
	ir1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "allowed1",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
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

	// ir2 is like ir1, but in a different namespace
	ir2 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "allowed2",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example2.com",
			},
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

	s2 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "allowed1",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:     "http",
				Protocol: "TCP",
				Port:     8080,
			}},
		},
	}

	s3 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "allowed2",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:     "http",
				Protocol: "TCP",
				Port:     8080,
			}},
		},
	}

	tests := map[string]struct {
		rootNamespaces []string
		objs           []interface{}
		want           int
	}{
		"nil root namespaces": {
			objs: []interface{}{ir1, s2},
			want: 1,
		},
		"empty root namespaces": {
			objs: []interface{}{ir1, s2},
			want: 1,
		},
		"single root namespace with root gatewayhost": {
			rootNamespaces: []string{"allowed1"},
			objs:           []interface{}{ir1, s2},
			want:           1,
		},
		"multiple root namespaces, one with a root gatewayhost": {
			rootNamespaces: []string{"foo", "allowed1", "bar"},
			objs:           []interface{}{ir1, s2},
			want:           1,
		},
		"multiple root namespaces, each with a root gatewayhost": {
			rootNamespaces: []string{"foo", "allowed1", "allowed2"},
			objs:           []interface{}{ir1, ir2, s2, s3},
			want:           2,
		},
		"root gatewayhost defined outside single root namespaces": {
			rootNamespaces: []string{"foo"},
			objs:           []interface{}{ir1},
			want:           0,
		},
		"root gatewayhost defined outside multiple root namespaces": {
			rootNamespaces: []string{"foo", "bar"},
			objs:           []interface{}{ir1},
			want:           0,
		},
		"two root gatewayhosts, one inside root namespace, one outside": {
			rootNamespaces: []string{"foo", "allowed2"},
			objs:           []interface{}{ir1, ir2, s3},
			want:           1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			kc := &KubernetesCache{
				GatewayHostRootNamespaces: tc.rootNamespaces,
			}
			for _, o := range tc.objs {
				kc.Insert(o)
			}

			dag := BuildDAG(kc)

			var count int
			dag.Visit(func(v Vertex) {
				v.Visit(func(v Vertex) {
					if _, ok := v.(*VirtualHost); ok {
						count++
					}
				})
			})

			if tc.want != count {
				t.Errorf("wanted %d vertices, but got %d", tc.want, count)
			}
		})
	}
}

func TestMatchesPathPrefix(t *testing.T) {
	tests := map[string]struct {
		path    string
		prefix  string
		matches bool
	}{
		"no path cannot match the prefix": {
			prefix:  "/foo",
			path:    "",
			matches: false,
		},
		"any path has the empty string as the prefix": {
			prefix:  "",
			path:    "/foo",
			matches: true,
		},
		"strict match": {
			prefix:  "/foo",
			path:    "/foo",
			matches: true,
		},
		"strict match with / at the end": {
			prefix:  "/foo/",
			path:    "/foo/",
			matches: true,
		},
		"no match": {
			prefix:  "/foo",
			path:    "/bar",
			matches: false,
		},
		"string prefix match should not match": {
			prefix:  "/foo",
			path:    "/foobar",
			matches: false,
		},
		"prefix match": {
			prefix:  "/foo",
			path:    "/foo/bar",
			matches: true,
		},
		"prefix match with trailing slash in prefix": {
			prefix:  "/foo/",
			path:    "/foo/bar",
			matches: true,
		},
		"prefix match with trailing slash in path": {
			prefix:  "/foo",
			path:    "/foo/bar/",
			matches: true,
		},
		"prefix match with trailing slashes": {
			prefix:  "/foo/",
			path:    "/foo/bar/",
			matches: true,
		},
		"prefix match two levels": {
			prefix:  "/foo/bar",
			path:    "/foo/bar",
			matches: true,
		},
		"prefix match two levels trailing slash in prefix": {
			prefix:  "/foo/bar/",
			path:    "/foo/bar",
			matches: true,
		},
		"prefix match two levels trailing slash in path": {
			prefix:  "/foo/bar",
			path:    "/foo/bar/",
			matches: true,
		},
		"no match two levels": {
			prefix:  "/foo/bar",
			path:    "/foo/baz",
			matches: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := matchesPathPrefix(tc.path, tc.prefix)
			if got != tc.matches {
				t.Errorf("expected %v but got %v", tc.matches, got)
			}
		})
	}
}

func TestDAGGatewayHostStatus(t *testing.T) {
	// ir1 is a valid gatewayhost
	ir1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "example",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "home",
					Port: 8080,
				}},
			}, {
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/prefix",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name: "delegated",
				}},
			},
		},
	}

	// ir2 is invalid because it contains a service with negative port
	ir2 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "example",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "home",
					Port: -80,
				}},
			}},
		},
	}

	// ir3 is invalid because it lives outside the roots namespace
	ir3 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "finance",
			Name:      "example",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foobar",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "home",
					Port: 8080,
				}},
			}},
		},
	}

	// 6-4-2020 We have removed this validation from builder.go where
	// we used to check delegate's prefix is child of parent's prefix
	// ir4 is invalid because its match prefix does not match its parent's (ir1)
	//ir4 := &gatewayhostv1.GatewayHost{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Namespace: "roots",
	//		Name:      "delegated",
	//	},
	//	Spec: gatewayhostv1.GatewayHostSpec{
	//		Routes: []gatewayhostv1.Route{{
	//                        Conditions: []gatewayhostv1.Condition{{
	//		                   Prefix: "/doesnotmatch",
	//		                }},
	//			Services: []gatewayhostv1.Service{{
	//				Name: "home",
	//				Port: 8080,
	//			}},
	//		}},
	//	},
	//}

	// ir6 is invalid because it delegates to itself, producing a cycle
	ir6 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "self",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name: "self",
				},
			}},
		},
	}

	// ir7 delegates to ir8, which is invalid because it delegates back to ir7
	ir7 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "parent",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name: "child",
				},
			}},
		},
	}

	ir8 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "child",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name: "parent",
				},
			}},
		},
	}

	// ir9 is invalid because it has a route that both delegates and has a list of services
	ir9 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "parent",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name: "child",
				},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			}},
		},
	}

	// ir10 delegates to ir11 and ir 12.
	ir10 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "parent",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name: "validChild",
				},
			}, {
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/bar",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name: "invalidChild",
				},
			}},
		},
	}

	ir11 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "validChild",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "foo2",
					Port: 8080,
				}},
			}},
		},
	}

	// ir12 is invalid because it contains an invalid port
	ir12 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "invalidChild",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/bar",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "foo3",
					Port: 12345678,
				}},
			}},
		},
	}

	// ir13 is invalid because it does not specify and FQDN
	ir13 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "parent",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "foo",
					Port: 8080,
				}},
			}},
		},
	}

	// ir14 delegates tp ir15 but it is invalid because it is missing fqdn
	ir14 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "invalidParent",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name: "validChild",
				},
			}},
		},
	}

	// ir15 is invalid because it contains a wildcarded fqdn
	//	ir15 := &gatewayhostv1.GatewayHost{
	//		ObjectMeta: metav1.ObjectMeta{
	//			Namespace: "roots",
	//			Name:      "example",
	//		},
	//		Spec: gatewayhostv1.GatewayHostSpec{
	//			VirtualHost: &gatewayhostv1.VirtualHost{
	//				Fqdn: "example.*.com",
	//			},
	//			Routes: []gatewayhostv1.Route{{
	//				Conditions: []gatewayhostv1.Condition{{
	//					Prefix: "/foo",
	//				}},
	//				Services: []gatewayhostv1.Service{{
	//					Name: "home",
	//					Port: 8080,
	//				}},
	//			}},
	//		},
	//	}

	// ir16 is invalid because it references an invalid service
	ir16 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "invalidir",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "invalid",
					Port: 8080,
				}},
			}},
		},
	}

	s4 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "home",
			Namespace: "roots",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:     "http",
				Protocol: "TCP",
				Port:     8080,
			}},
		},
	}

	s5 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "parent",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:     "http",
				Protocol: "TCP",
				Port:     8080,
			}},
		},
	}

	s6 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "foo2",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:     "http",
				Protocol: "TCP",
				Port:     8080,
			}},
		},
	}

	s7 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo3",
			Namespace: "roots",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:     "http",
				Protocol: "TCP",
				Port:     12345678,
			}},
		},
	}

	tests := map[string]struct {
		objs []interface{}
		want []Status
	}{
		"valid gatewayhost": {
			objs: []interface{}{ir1, s4},
			want: []Status{{Object: ir1, Status: "valid", Description: "valid GatewayHost", Vhost: "example.com"}},
		},
		"invalid port in service": {
			objs: []interface{}{ir2},
			want: []Status{{Object: ir2, Status: "invalid", Description: `service "home": port must be in the range 1-65535`, Vhost: "example.com"}},
		},
		"root gatewayhost outside of roots namespace": {
			objs: []interface{}{ir3},
			want: []Status{{Object: ir3, Status: "invalid", Description: "root GatewayHost cannot be defined in this namespace"}},
		},
		//"delegated route's match prefix does not match parent's prefix": {
		//	objs: []interface{}{ir1, ir4, s4},
		//	want: []Status{
		//		{Object: ir1, Status: "valid", Description: "valid GatewayHost", Vhost: "example.com"},
		//		{Object: ir4, Status: "invalid", Description: `the path prefix "/doesnotmatch" does not match the parent's path prefix "/prefix"`, Vhost: "example.com"},
		//	},
		//},
		"root gatewayhost does not specify FQDN": {
			objs: []interface{}{ir13},
			want: []Status{{Object: ir13, Status: "invalid", Description: "Spec.VirtualHost.Fqdn must be specified"}},
		},
		"self-edge produces a cycle": {
			objs: []interface{}{ir6},
			want: []Status{{Object: ir6, Status: "invalid", Description: "route creates a delegation cycle: roots/self -> roots/self", Vhost: "example.com"}},
		},
		"child delegates to parent, producing a cycle": {
			objs: []interface{}{ir7, ir8},
			want: []Status{
				{Object: ir7, Status: "valid", Description: "valid GatewayHost", Vhost: "example.com"},
				{Object: ir8, Status: "invalid", Description: "route creates a delegation cycle: roots/parent -> roots/child -> roots/parent", Vhost: "example.com"},
			},
		},
		"route has a list of services and also delegates": {
			objs: []interface{}{ir9},
			want: []Status{{Object: ir9, Status: "invalid", Description: `cannot specify services and delegate in the same route`, Vhost: "example.com"}},
		},
		"gatewayhost is an orphaned route": {
			objs: []interface{}{ir8},
			want: []Status{{Object: ir8, Status: "orphaned", Description: "this GatewayHost is not part of a delegation chain from a root GatewayHost"}},
		},
		"gatewayhost delegates to multiple ingressroutes, one is invalid": {
			objs: []interface{}{ir10, ir11, ir12, s6, s7},
			want: []Status{
				{Object: ir11, Status: "valid", Description: "valid GatewayHost", Vhost: "example.com"},
				{Object: ir12, Status: "invalid", Description: `service "foo3": port must be in the range 1-65535`, Vhost: "example.com"},
				{Object: ir10, Status: "valid", Description: "valid GatewayHost", Vhost: "example.com"},
			},
		},
		"invalid parent orphans children": {
			objs: []interface{}{ir14, ir11},
			want: []Status{
				{Object: ir14, Status: "invalid", Description: "Spec.VirtualHost.Fqdn must be specified"},
				{Object: ir11, Status: "orphaned", Description: "this GatewayHost is not part of a delegation chain from a root GatewayHost"},
			},
		},
		"multi-parent children is not orphaned when one of the parents is invalid": {
			objs: []interface{}{ir14, ir11, ir10, s5, s6},
			want: []Status{
				{Object: ir14, Status: "invalid", Description: "Spec.VirtualHost.Fqdn must be specified"},
				{Object: ir11, Status: "valid", Description: "valid GatewayHost", Vhost: "example.com"},
				{Object: ir10, Status: "valid", Description: "valid GatewayHost", Vhost: "example.com"},
			},
		},
		//"invalid FQDN contains wildcard": {
		//	objs: []interface{}{ir15},
		//	want: []Status{{Object: ir15, Status: "invalid", Description: `Spec.VirtualHost.Fqdn "example.*.com" cannot use wildcards`, Vhost: "example.*.com"}},
		//},
		"missing service shows invalid status": {
			objs: []interface{}{ir16},
			want: []Status{{Object: ir16, Status: "invalid", Description: `Service [invalid:8080] is invalid or missing`, Vhost: ""}},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			kc := &KubernetesCache{
				GatewayHostRootNamespaces: []string{"roots"},
			}
			for _, o := range tc.objs {
				kc.Insert(o)
			}

			dag := BuildDAG(kc)
			got := dag.Statuses()
			if len(tc.want) != len(got) {
				t.Fatalf("expected:\n%v\ngot\n%v", tc.want, got)
			}

			for _, ex := range tc.want {
				var found bool
				for _, g := range got {
					if cmp.Equal(ex, g) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected to find:\n%v\nbut did not find it in:\n%v", ex, got)
				}
			}
		})
	}
}

func TestDAGGatewayHostUniqueFQDNs(t *testing.T) {
	ir1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
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

	// ir2 reuses the fqdn used in ir1
	ir2 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-example",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
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

	tests := map[string]struct {
		objs       []interface{}
		want       []Vertex
		wantStatus map[Meta]Status
	}{
		"insert gatewayhost": {
			objs: []interface{}{
				s1, ir1,
			},
			want: listeners(
				&Listener{
					Port: 80,
					VirtualHosts: virtualhosts(
						&VirtualHost{
							Name: "example.com",
							routes: routemap(
								prefixroute("/", httpService(s1)),
							),
						},
					),
				},
			),
			wantStatus: map[Meta]Status{
				{name: ir1.Name, namespace: ir1.Namespace}: {
					Object:      ir1,
					Status:      StatusValid,
					Description: "valid GatewayHost",
					Vhost:       "example.com",
				},
			},
		},
		"insert conflicting gatewayhosts due to fqdn reuse": {
			objs: []interface{}{
				ir1, ir2,
			},
			want: []Vertex{},
			wantStatus: map[Meta]Status{
				{name: ir1.Name, namespace: ir1.Namespace}: {
					Object:      ir1,
					Status:      StatusInvalid,
					Description: `fqdn "example.com" is used in multiple GatewayHosts: default/example-com, default/other-example`,
					Vhost:       "example.com",
				},
				{name: ir2.Name, namespace: ir2.Namespace}: {
					Object:      ir2,
					Status:      StatusInvalid,
					Description: `fqdn "example.com" is used in multiple GatewayHosts: default/example-com, default/other-example`,
					Vhost:       "example.com",
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var kc KubernetesCache
			for _, o := range tc.objs {
				kc.Insert(o)
			}

			dag := BuildDAG(&kc)
			got := make(map[int]*Listener)
			dag.Visit(listenerMap(got).Visit)

			want := make(map[int]*Listener)
			for _, v := range tc.want {
				if l, ok := v.(*Listener); ok {
					want[l.Port] = l
				}
			}

			opts := []cmp.Option{
				cmp.AllowUnexported(Listener{}, VirtualHost{}),
			}
			if diff := cmp.Diff(want, got, opts...); diff != "" {
				t.Fatal(diff)
			}

			gotStatus := dag.statuses
			if diff := cmp.Diff(tc.wantStatus, gotStatus); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestHttpPaths(t *testing.T) {
	tests := map[string]struct {
		rule v1beta1.IngressRule
		want []v1beta1.HTTPIngressPath
	}{
		"zero value": {
			rule: v1beta1.IngressRule{},
			want: nil,
		},
		"empty paths": {
			rule: v1beta1.IngressRule{
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{},
				},
			},
			want: nil,
		},
		"several paths": {
			rule: v1beta1.IngressRule{
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{{
							Backend: v1beta1.IngressBackend{
								ServiceName: "kuard",
								ServicePort: intstr.FromString("http"),
							},
						}, {
							Path: "/kuarder",
							Backend: v1beta1.IngressBackend{
								ServiceName: "kuarder",
								ServicePort: intstr.FromInt(8080),
							},
						}},
					},
				},
			},
			want: []v1beta1.HTTPIngressPath{{
				Backend: v1beta1.IngressBackend{
					ServiceName: "kuard",
					ServicePort: intstr.FromString("http"),
				},
			}, {
				Path: "/kuarder",
				Backend: v1beta1.IngressBackend{ServiceName: "kuarder",
					ServicePort: intstr.FromInt(8080),
				},
			}},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := httppaths(tc.rule)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf(diff)
			}
		})
	}
}
func TestEnforceRoute(t *testing.T) {
	tests := map[string]struct {
		tlsEnabled     bool
		permitInsecure bool
		want           bool
	}{
		"tls not enabled": {
			tlsEnabled:     false,
			permitInsecure: false,
			want:           false,
		},
		"tls enabled": {
			tlsEnabled:     true,
			permitInsecure: false,
			want:           true,
		},
		"tls enabled but insecure requested": {
			tlsEnabled:     true,
			permitInsecure: true,
			want:           false,
		},
		"tls not enabled but insecure requested": {
			tlsEnabled:     false,
			permitInsecure: true,
			want:           false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := routeEnforceTLS(tc.tlsEnabled, tc.permitInsecure)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf(diff)
			}
		})
	}
}

func TestSplitSecret(t *testing.T) {
	tests := map[string]struct {
		secret, defns string
		want          Meta
	}{
		"no namespace": {
			secret: "secret",
			defns:  "default",
			want: Meta{
				name:      "secret",
				namespace: "default",
			},
		},
		"with namespace": {
			secret: "ns1/secret",
			defns:  "default",
			want: Meta{
				name:      "secret",
				namespace: "ns1",
			},
		},
		"missing namespace": {
			secret: "/secret",
			defns:  "default",
			want: Meta{
				name:      "secret",
				namespace: "default",
			},
		},
		"missing secret name": {
			secret: "secret/",
			defns:  "default",
			want: Meta{
				name:      "",
				namespace: "secret",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := splitSecret(tc.secret, tc.defns)
			opts := []cmp.Option{
				cmp.AllowUnexported(Meta{}),
			}
			if diff := cmp.Diff(tc.want, got, opts...); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func routemap(routes ...*Route) map[string]*Route {
	if len(routes) == 0 {
		return nil
	}
	m := make(map[string]*Route)
	for _, r := range routes {
		m[conditionsToString(r)] = r
	}
	return m
}

func prefixroute(prefix string, services ...*HTTPService) *Route {
	route := Route{
		PathCondition: &PrefixCondition{Prefix: prefix},
	}
	for _, s := range services {
		route.Clusters = append(route.Clusters, &Cluster{
			Upstream: s,
		})
	}
	return &route
}

func routeCluster(prefix string, clusters ...*Cluster) *Route {
	route := Route{
		PathCondition: &PrefixCondition{Prefix: prefix},
		Clusters:      clusters,
	}
	return &route
}

func routeUpgrade(prefix string, services ...*HTTPService) *Route {
	r := prefixroute(prefix, services...)
	r.HTTPSUpgrade = true
	return r
}

func routeRewrite(prefix, rewrite string, services ...*HTTPService) *Route {
	r := prefixroute(prefix, services...)
	r.PrefixRewrite = rewrite
	return r
}

func routeWebsocket(prefix string, services ...*HTTPService) *Route {
	r := prefixroute(prefix, services...)
	r.Websocket = true
	return r
}

func clusters(services ...Service) (c []*Cluster) {
	for _, s := range services {
		c = append(c, &Cluster{
			Upstream: s,
		})
	}
	return c
}

func tcpService(s *v1.Service) *TCPService {
	return &TCPService{
		Name:        s.Name,
		Namespace:   s.Namespace,
		ServicePort: &s.Spec.Ports[0],
	}
}

func httpService(s *v1.Service) *HTTPService {
	return &HTTPService{
		TCPService: TCPService{
			Name:        s.Name,
			Namespace:   s.Namespace,
			ServicePort: &s.Spec.Ports[0],
		},
	}
}

func clustermap(services ...*v1.Service) []*Cluster {
	var c []*Cluster
	for _, s := range services {
		c = append(c, &Cluster{
			Upstream: httpService(s),
		})
	}
	return c
}

func secret(s *v1.Secret) *Secret {
	return &Secret{
		Object: s,
	}
}

func virtualhosts(vx ...Vertex) map[string]Vertex {
	m := make(map[string]Vertex)
	for _, v := range vx {
		switch v := v.(type) {
		case *VirtualHost:
			m[v.Name] = v
		case *SecureVirtualHost:
			m[v.VirtualHost.Name] = v
		default:
			panic(fmt.Sprintf("unable to handle Vertex type %T", v))
		}
	}
	return m
}

func virtualhost(name string, routes ...*Route) *VirtualHost {
	return &VirtualHost{
		Name:   name,
		routes: routemap(routes...),
	}
}

func securevirtualhost(name string, sec *v1.Secret, routes ...*Route) *SecureVirtualHost {
	return &SecureVirtualHost{
		VirtualHost: VirtualHost{
			Name:   name,
			routes: routemap(routes...),
		},
		MinProtoVersion: envoy_api_v2_auth.TlsParameters_TLSv1_1,
		Secret:          secret(sec),
	}
}

func listeners(ls ...*Listener) []Vertex {
	var v []Vertex
	for _, l := range ls {
		v = append(v, l)
	}
	return v
}

func prefix(prefix string) Condition { return &PrefixCondition{Prefix: prefix} }
func regex(regex string) Condition   { return &RegexCondition{Regex: regex} }
