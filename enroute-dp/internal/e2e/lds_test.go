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

	gatewayhostv1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1"

	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_extensions_filters_network_tcp_proxy_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	envoy_service_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"

	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/saarasio/enroute/enroute-dp/apis/generated/clientset/versioned/fake"
	"github.com/saarasio/enroute/enroute-dp/internal/assert"
	"github.com/saarasio/enroute/enroute-dp/internal/contour"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/envoy"
	"github.com/saarasio/enroute/enroute-dp/internal/k8s"
	"github.com/saarasio/enroute/enroute-dp/internal/protobuf"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestNonTLSListener(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	// assert that without any ingress objects registered
	// there are no active listeners
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "0",
		Resources: resources(t,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "0",
	}, streamLDS(t, cc))

	// i1 is a simple ingress, no hostname, no tls.
	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
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

	rh.OnAdd(&corev1.Service{
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
	})

	// add it and assert that we now have a ingress_http listener
	rh.OnAdd(i1)
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			&envoy_config_listener_v3.Listener{
				Name:         "ingress_http",
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager("ingress_http", "/dev/stdout", nil)),
			},
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "2",
	}, streamLDS(t, cc))

	// i2 is the same as i1 but has the kubernetes.io/ingress.allow-http: "false" annotation
	i2 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.allow-http": "false",
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

	// update i1 to i2 and verify that ingress_http has gone.
	rh.OnUpdate(i1, i2)
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "3",
	}, streamLDS(t, cc))

	// i3 is similar to i2, but uses the ingress.kubernetes.io/force-ssl-redirect: "true" annotation
	// to force 80 -> 443 upgrade
	i3 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
			Annotations: map[string]string{
				"ingress.kubernetes.io/force-ssl-redirect": "true",
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

	// update i2 to i3 and check that ingress_http has returned
	rh.OnUpdate(i2, i3)
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "4",
		Resources: resources(t,
			&envoy_config_listener_v3.Listener{
				Name:         "ingress_http",
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager("ingress_http", "/dev/stdout", nil)),
			},
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "4",
	}, streamLDS(t, cc))
}

func TestTLSListener(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	// s1 is a tls secret
	s1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	}

	// i1 is a tls ingress
	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
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
			TLS: []netv1.IngressTLS{{
				Hosts:      []string{"kuard.example.com"},
				SecretName: "secret",
			}},
		},
	}

	rh.OnAdd(&corev1.Service{
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
	})

	// add secret
	rh.OnAdd(s1)

	// assert that there is only a static listener
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "2",
	}, streamLDS(t, cc))

	// add ingress and assert the existence of ingress_http and ingres_https
	rh.OnAdd(i1)
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			&envoy_config_listener_v3.Listener{
				Name:         "ingress_http",
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager("ingress_http", "/dev/stdout", nil)),
			},
			&envoy_config_listener_v3.Listener{
				Name:    "ingress_https",
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
				FilterChains: filterchaintls("kuard.example.com", s1, envoy.HTTPConnectionManager("ingress_https", "/dev/stdout", nil), "h2", "http/1.1"),
			},
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "3",
	}, streamLDS(t, cc))

	// i2 is the same as i1 but has the kubernetes.io/ingress.allow-http: "false" annotation
	i2 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.allow-http": "false",
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
			TLS: []netv1.IngressTLS{{
				Hosts:      []string{"kuard.example.com"},
				SecretName: "secret",
			}},
		},
	}

	// update i1 to i2 and verify that ingress_http has gone.
	rh.OnUpdate(i1, i2)
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "4",
		Resources: resources(t,
			&envoy_config_listener_v3.Listener{
				Name:    "ingress_https",
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
				FilterChains: filterchaintls("kuard.example.com", s1, envoy.HTTPConnectionManager("ingress_https", "/dev/stdout", nil), "h2", "http/1.1"),
			},
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "4",
	}, streamLDS(t, cc))

	// delete secret and assert that ingress_https is removed
	rh.OnDelete(s1)
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "5",
		Resources: resources(t,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "5",
	}, streamLDS(t, cc))
}

func TestGatewayHostTLSListener(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	// secret1 is a tls secret
	secret1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	}

	// i1 is a tls gatewayhost
	i1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "kuard.example.com",
				TLS: &gatewayhostv1.TLS{
					SecretName:             "secret",
					MinimumProtocolVersion: "1.1",
				},
			},
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

	// i2 is a tls gatewayhost
	i2 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "kuard.example.com",
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
					Name: "backend",
					Port: 80,
				}},
			}},
		},
	}

	svc1 := &corev1.Service{
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
	}

	// add secret
	rh.OnAdd(secret1)

	// assert that there is only a static listener
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "1",
		Resources: resources(t,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "1",
	}, streamLDS(t, cc))

	l1 := &envoy_config_listener_v3.Listener{
		Name:    "ingress_https",
		Address: envoy.SocketAddress("0.0.0.0", 8443),
		ListenerFilters: envoy.ListenerFilters(
			envoy.TLSInspector(),
		),
		FilterChains: filterchaintls("kuard.example.com", secret1, envoy.HTTPConnectionManager("ingress_https", "/dev/stdout", nil), "h2", "http/1.1"),
	}

	// add service
	rh.OnAdd(svc1)

	// add ingress and assert the existence of ingress_http and ingres_https
	rh.OnAdd(i1)

	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			&envoy_config_listener_v3.Listener{
				Name:         "ingress_http",
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager("ingress_http", "/dev/stdout", nil)),
			},
			l1,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "3",
	}, streamLDS(t, cc))

	// delete secret and assert that ingress_https is removed
	rh.OnDelete(secret1)
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "4",
		Resources: resources(t,
			&envoy_config_listener_v3.Listener{
				Name:         "ingress_http",
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager("ingress_http", "/dev/stdout", nil)),
			},
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "4",
	}, streamLDS(t, cc))

	rh.OnDelete(i1)
	// add secret
	rh.OnAdd(secret1)
	l2 := &envoy_config_listener_v3.Listener{
		Name:    "ingress_https",
		Address: envoy.SocketAddress("0.0.0.0", 8443),
		ListenerFilters: envoy.ListenerFilters(
			envoy.TLSInspector(),
		),
		FilterChains: []*envoy_config_listener_v3.FilterChain{
			envoy.FilterChainTLS(
				"kuard.example.com",
				&dag.Secret{Object: secret1},
				envoy.Filters(
					envoy.HTTPConnectionManager("ingress_https", "/dev/stdout", nil),
				),
				envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_3,
				"h2", "http/1.1",
			),
		},
	}

	// add ingress and assert the existence of ingress_http and ingres_https
	rh.OnAdd(i2)
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "7",
		Resources: resources(t,
			&envoy_config_listener_v3.Listener{
				Name:         "ingress_http",
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager("ingress_http", "/dev/stdout", nil)),
			},
			l2,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "7",
	}, streamLDS(t, cc))
}

func TestLDSFilter(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	// s1 is a tls secret
	s1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	}

	// i1 is a tls ingress
	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
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
			TLS: []netv1.IngressTLS{{
				Hosts:      []string{"kuard.example.com"},
				SecretName: "secret",
			}},
		},
	}

	rh.OnAdd(&corev1.Service{
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
	})

	// add secret
	rh.OnAdd(s1)

	// add ingress and fetch ingress_https
	rh.OnAdd(i1)
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			&envoy_config_listener_v3.Listener{
				Name:    "ingress_https",
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
				FilterChains: filterchaintls("kuard.example.com", s1, envoy.HTTPConnectionManager("ingress_https", "/dev/stdout", nil), "h2", "http/1.1"),
			},
		),
		TypeUrl: listenerType,
		Nonce:   "3",
	}, streamLDS(t, cc, "ingress_https"))

	// fetch ingress_http
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			&envoy_config_listener_v3.Listener{
				Name:         "ingress_http",
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager("ingress_http", "/dev/stdout", nil)),
			},
		),
		TypeUrl: listenerType,
		Nonce:   "3",
	}, streamLDS(t, cc, "ingress_http"))

	// fetch something non existent.
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		TypeUrl:     listenerType,
		Nonce:       "3",
	}, streamLDS(t, cc, "HTTP"))
}

func TestLDSStreamEmpty(t *testing.T) {
	_, cc, done := setup(t)
	defer done()

	// assert that streaming LDS with no ingresses does not stall.
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "0",
		TypeUrl:     listenerType,
		Nonce:       "0",
	}, streamLDS(t, cc, "HTTP"))
}

func TestLDSTLSMinimumProtocolVersion(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	// s1 is a tls secret
	s1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	}
	rh.OnAdd(s1)

	// i1 is a tls ingress
	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
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
			TLS: []netv1.IngressTLS{{
				Hosts:      []string{"kuard.example.com"},
				SecretName: "secret",
			}},
		},
	}

	rh.OnAdd(i1)

	// add ingress and fetch ingress_https
	rh.OnAdd(i1)
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			&envoy_config_listener_v3.Listener{
				Name:    "ingress_https",
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
				FilterChains: filterchaintls("kuard.example.com", s1, envoy.HTTPConnectionManager("ingress_https", "/dev/stdout", nil), "h2", "http/1.1"),
			},
		),
		TypeUrl: listenerType,
		Nonce:   "3",
	}, streamLDS(t, cc, "ingress_https"))

	i2 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
			Annotations: map[string]string{
				"enroute.saaras.io/tls-minimum-protocol-version": "1.3",
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
			TLS: []netv1.IngressTLS{{
				Hosts:      []string{"kuard.example.com"},
				SecretName: "secret",
			}},
		},
	}

	// update tls version and fetch ingress_https
	rh.OnUpdate(i1, i2)

	l1 := &envoy_config_listener_v3.Listener{
		Name:    "ingress_https",
		Address: envoy.SocketAddress("0.0.0.0", 8443),
		ListenerFilters: envoy.ListenerFilters(
			envoy.TLSInspector(),
		),
		FilterChains: []*envoy_config_listener_v3.FilterChain{
			envoy.FilterChainTLS(
				"kuard.example.com",
				&dag.Secret{Object: s1},
				envoy.Filters(
					envoy.HTTPConnectionManager("ingress_https", "/dev/stdout", nil),
				),
				envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_3,
				"h2", "http/1.1",
			),
		},
	}

	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "4",
		Resources: resources(t,
			l1,
		),
		TypeUrl: listenerType,
		Nonce:   "4",
	}, streamLDS(t, cc, "ingress_https"))
}

func TestLDSIngressHTTPUseProxyProtocol(t *testing.T) {
	rh, cc, done := setup(t, func(reh *contour.ResourceEventHandler) {
		reh.Notifier.(*contour.CacheHandler).UseProxyProto = true
	})
	defer done()

	// assert that without any ingress objects registered
	// there is only a static listener
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "0",
		Resources: resources(t,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "0",
	}, streamLDS(t, cc))

	// i1 is a simple ingress, no hostname, no tls.
	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
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
	rh.OnAdd(&corev1.Service{
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
	})

	// add it and assert that we now have a ingress_http listener using
	// the proxy protocol (the true param to filterchain)
	rh.OnAdd(i1)
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			&envoy_config_listener_v3.Listener{
				Name:    "ingress_http",
				Address: envoy.SocketAddress("0.0.0.0", 8080),
				ListenerFilters: envoy.ListenerFilters(
					envoy.ProxyProtocol(),
				),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager("ingress_http", "/dev/stdout", nil)),
			},
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "2",
	}, streamLDS(t, cc))
}

func TestLDSIngressHTTPSUseProxyProtocol(t *testing.T) {
	rh, cc, done := setup(t, func(reh *contour.ResourceEventHandler) {
		reh.Notifier.(*contour.CacheHandler).UseProxyProto = true
	})
	defer done()

	// s1 is a tls secret
	s1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	}

	// i1 is a tls ingress
	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
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
			TLS: []netv1.IngressTLS{{
				Hosts:      []string{"kuard.example.com"},
				SecretName: "secret",
			}},
		},
	}

	// add secret
	rh.OnAdd(s1)

	// assert that there is only a static listener
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "1",
		Resources: resources(t,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "1",
	}, streamLDS(t, cc))

	rh.OnAdd(&corev1.Service{
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
	})

	// add ingress and assert the existence of ingress_http and ingres_https and both
	// are using proxy protocol
	rh.OnAdd(i1)

	ingress_https := &envoy_config_listener_v3.Listener{
		Name:    "ingress_https",
		Address: envoy.SocketAddress("0.0.0.0", 8443),
		ListenerFilters: envoy.ListenerFilters(
			envoy.ProxyProtocol(),
			envoy.TLSInspector(),
		),
		FilterChains: filterchaintls("kuard.example.com", s1, envoy.HTTPConnectionManager("ingress_https", "/dev/stdout", nil), "h2", "http/1.1"),
	}
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			&envoy_config_listener_v3.Listener{
				Name:    "ingress_http",
				Address: envoy.SocketAddress("0.0.0.0", 8080),
				ListenerFilters: envoy.ListenerFilters(
					envoy.ProxyProtocol(),
				),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager("ingress_http", "/dev/stdout", nil)),
			},
			ingress_https,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "3",
	}, streamLDS(t, cc))
}

func TestLDSCustomAddressAndPort(t *testing.T) {
	rh, cc, done := setup(t, func(reh *contour.ResourceEventHandler) {
		reh.Notifier.(*contour.CacheHandler).HTTPAddress = "127.0.0.100"
		reh.Notifier.(*contour.CacheHandler).HTTPPort = 9100
		reh.Notifier.(*contour.CacheHandler).HTTPSAddress = "127.0.0.200"
		reh.Notifier.(*contour.CacheHandler).HTTPSPort = 9200
	})
	defer done()

	// s1 is a tls secret
	s1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	}

	// i1 is a tls ingress
	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
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
			TLS: []netv1.IngressTLS{{
				Hosts:      []string{"kuard.example.com"},
				SecretName: "secret",
			}},
		},
	}

	// add secret
	rh.OnAdd(s1)

	// assert that there is only a static listener
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "1",
		Resources: resources(t,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "1",
	}, streamLDS(t, cc))

	rh.OnAdd(&corev1.Service{
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
	})

	// add ingress and assert the existence of ingress_http and ingres_https and both
	// are using proxy protocol
	rh.OnAdd(i1)

	ingress_http := &envoy_config_listener_v3.Listener{
		Name:         "ingress_http",
		Address:      envoy.SocketAddress("127.0.0.100", 9100),
		FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager("ingress_http", "/dev/stdout", nil)),
	}
	ingress_https := &envoy_config_listener_v3.Listener{
		Name:    "ingress_https",
		Address: envoy.SocketAddress("127.0.0.200", 9200),
		ListenerFilters: envoy.ListenerFilters(
			envoy.TLSInspector(),
		),
		FilterChains: filterchaintls("kuard.example.com", s1, envoy.HTTPConnectionManager("ingress_https", "/dev/stdout", nil), "h2", "http/1.1"),
	}
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			ingress_http,
			ingress_https,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "3",
	}, streamLDS(t, cc))
}

func TestLDSCustomAccessLogPaths(t *testing.T) {
	rh, cc, done := setup(t, func(reh *contour.ResourceEventHandler) {
		reh.Notifier.(*contour.CacheHandler).HTTPAccessLog = "/tmp/http_access.log"
		reh.Notifier.(*contour.CacheHandler).HTTPSAccessLog = "/tmp/https_access.log"
	})
	defer done()

	// s1 is a tls secret
	s1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	}

	// i1 is a tls ingress
	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
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
			TLS: []netv1.IngressTLS{{
				Hosts:      []string{"kuard.example.com"},
				SecretName: "secret",
			}},
		},
	}

	rh.OnAdd(&corev1.Service{
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
	})

	// add secret
	rh.OnAdd(s1)

	// assert that there is only a static listener
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "2",
	}, streamLDS(t, cc))

	rh.OnAdd(i1)

	ingress_http := &envoy_config_listener_v3.Listener{
		Name:         "ingress_http",
		Address:      envoy.SocketAddress("0.0.0.0", 8080),
		FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager("ingress_http", "/tmp/http_access.log", nil)),
	}
	ingress_https := &envoy_config_listener_v3.Listener{
		Name:    "ingress_https",
		Address: envoy.SocketAddress("0.0.0.0", 8443),
		ListenerFilters: envoy.ListenerFilters(
			envoy.TLSInspector(),
		),
		FilterChains: filterchaintls("kuard.example.com", s1, envoy.HTTPConnectionManager("ingress_https", "/tmp/https_access.log", nil), "h2", "http/1.1"),
	}
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			ingress_http,
			ingress_https,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "3",
	}, streamLDS(t, cc))
}

func TestLDSGatewayHostInsideRootNamespaces(t *testing.T) {
	rh, cc, done := setup(t, func(reh *contour.ResourceEventHandler) {
		reh.GatewayHostRootNamespaces = []string{"roots"}
		reh.Notifier.(*contour.CacheHandler).GatewayHostStatus = &k8s.GatewayHostStatus{
			Client: fake.NewSimpleClientset(),
		}
	})
	defer done()

	// assert that there is only a static listener
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "0",
		Resources: resources(t,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "0",
	}, streamLDS(t, cc))

	// ir1 is an gatewayhost that is in the root namespace
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

	svc1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "roots",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:     "http",
				Protocol: "TCP",
				Port:     8080,
			}},
		},
	}

	// add gatewayhost & service
	rh.OnAdd(svc1)
	rh.OnAdd(ir1)

	// assert there is an active listener
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			&envoy_config_listener_v3.Listener{
				Name:         "ingress_http",
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager("ingress_http", "/dev/stdout", nil)),
			},
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "2",
	}, streamLDS(t, cc))
}

func TestLDSGatewayHostOutsideRootNamespaces(t *testing.T) {
	rh, cc, done := setup(t, func(reh *contour.ResourceEventHandler) {
		reh.GatewayHostRootNamespaces = []string{"roots"}
		reh.Notifier.(*contour.CacheHandler).GatewayHostStatus = &k8s.GatewayHostStatus{
			Client: fake.NewSimpleClientset(),
		}
	})
	defer done()

	// assert that there is only a static listener
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "0",
		Resources: resources(t,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "0",
	}, streamLDS(t, cc))

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
	rh.OnAdd(ir1)

	// assert that there is only a static listener
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "1",
		Resources: resources(t,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "1",
	}, streamLDS(t, cc))
}

func TestGatewayHostHTTPS(t *testing.T) {
	rh, cc, done := setup(t, func(reh *contour.ResourceEventHandler) {
		reh.GatewayHostRootNamespaces = []string{}
		reh.Notifier.(*contour.CacheHandler).GatewayHostStatus = &k8s.GatewayHostStatus{
			Client: fake.NewSimpleClientset(),
		}
	})
	defer done()

	// assert that there is only a static listener
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "0",
		Resources: resources(t,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "0",
	}, streamLDS(t, cc))

	// s1 is a tls secret
	s1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	}

	// ir1 is an gatewayhost that has TLS
	ir1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
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

	svc1 := &corev1.Service{
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
	}

	// add secret
	rh.OnAdd(s1)

	// add service
	rh.OnAdd(svc1)

	// add gatewayhost
	rh.OnAdd(ir1)

	ingressHTTP := &envoy_config_listener_v3.Listener{
		Name:         "ingress_http",
		Address:      envoy.SocketAddress("0.0.0.0", 8080),
		FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager("ingress_http", "/dev/stdout", nil)),
	}

	ingressHTTPS := &envoy_config_listener_v3.Listener{
		Name:    "ingress_https",
		Address: envoy.SocketAddress("0.0.0.0", 8443),
		ListenerFilters: envoy.ListenerFilters(
			envoy.TLSInspector(),
		),
		FilterChains: filterchaintls("example.com", s1, envoy.HTTPConnectionManager("ingress_https", "/dev/stdout", nil), "h2", "http/1.1"),
	}
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			ingressHTTP,
			ingressHTTPS,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "3",
	}, streamLDS(t, cc))
}

// Assert that when a spec.vhost.tls spec is present with tls.passthrough
// set to true we configure envoy to forward the TLS session to the cluster
// after using SNI to determine the target.
func TestLDSGatewayHostTCPProxyTLSPassthrough(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	i1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "kuard-tcp.example.com",
				TLS: &gatewayhostv1.TLS{
					Passthrough: true,
				},
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "wrong-backend",
					Port: 80,
				}},
			}},
			TCPProxy: &gatewayhostv1.TCPProxy{
				Services: []gatewayhostv1.Service{{
					Name: "correct-backend",
					Port: 80,
				}},
			},
		},
	}
	svc := service("default", "correct-backend", corev1.ServicePort{
		Protocol:   "TCP",
		Port:       80,
		TargetPort: intstr.FromInt(8080),
	})
	rh.OnAdd(svc)
	rh.OnAdd(i1)

	ingressHTTPS := &envoy_config_listener_v3.Listener{
		Name:    "ingress_https",
		Address: envoy.SocketAddress("0.0.0.0", 8443),
		FilterChains: []*envoy_config_listener_v3.FilterChain{{
			Filters: envoy.Filters(
				tcpproxy(t, "ingress_https", "default/correct-backend/80/da39a3ee5e"),
			),
			FilterChainMatch: &envoy_config_listener_v3.FilterChainMatch{
				ServerNames: []string{"kuard-tcp.example.com"},
			},
		}},
		ListenerFilters: envoy.ListenerFilters(
			envoy.TLSInspector(),
		),
	}

	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			ingressHTTPS,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "2",
	}, streamLDS(t, cc))
}

func TestLDSGatewayHostTCPForward(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	// s1 is a tls secret
	s1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	}

	i1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "kuard-tcp.example.com",
				TLS: &gatewayhostv1.TLS{
					SecretName: "secret",
				},
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "wrong-backend",
					Port: 80,
				}},
			}},
			TCPProxy: &gatewayhostv1.TCPProxy{
				Services: []gatewayhostv1.Service{{
					Name: "correct-backend",
					Port: 80,
				}},
			},
		},
	}
	rh.OnAdd(s1)
	svc := service("default", "correct-backend", corev1.ServicePort{
		Protocol:   "TCP",
		Port:       80,
		TargetPort: intstr.FromInt(8080),
	})
	rh.OnAdd(svc)
	rh.OnAdd(i1)

	ingressHTTPS := &envoy_config_listener_v3.Listener{
		Name:         "ingress_https",
		Address:      envoy.SocketAddress("0.0.0.0", 8443),
		FilterChains: filterchaintls("kuard-tcp.example.com", s1, tcpproxy(t, "ingress_https", "default/correct-backend/80/da39a3ee5e")),
		ListenerFilters: envoy.ListenerFilters(
			envoy.TLSInspector(),
		),
	}

	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			ingressHTTPS,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "3",
	}, streamLDS(t, cc))
}

// Test that TLS Cerfiticate delegation works correctly.
func TestGatewayHostTLSCertificateDelegation(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	// assert that there is only a static listener
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "0",
		Resources: resources(t,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "0",
	}, streamLDS(t, cc))

	s1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wildcard",
			Namespace: "secret",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	}

	// add a secret object secret/wildcard.
	rh.OnAdd(s1)

	rh.OnAdd(&corev1.Service{
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
	})

	// add an gatewayhost in a different namespace mentioning secret/wildcard.
	rh.OnAdd(&gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
				TLS: &gatewayhostv1.TLS{
					SecretName: "secret/wildcard",
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
	})

	ingress_http := &envoy_config_listener_v3.Listener{
		Name:         "ingress_http",
		Address:      envoy.SocketAddress("0.0.0.0", 8080),
		FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager("ingress_http", "/dev/stdout", nil)),
	}

	// assert there is no ingress_https because there is no matching secret.
	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "3",
		Resources: resources(t,
			ingress_http,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "3",
	}, streamLDS(t, cc))

	// t1 is a TLSCertificateDelegation that permits default to access secret/wildcard
	t1 := &gatewayhostv1.TLSCertificateDelegation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "delegation",
			Namespace: "secret",
		},
		Spec: gatewayhostv1.TLSCertificateDelegationSpec{
			Delegations: []gatewayhostv1.CertificateDelegation{{
				SecretName: "wildcard",
				TargetNamespaces: []string{
					"default",
				},
			}},
		},
	}
	rh.OnAdd(t1)

	ingress_https := &envoy_config_listener_v3.Listener{
		Name:    "ingress_https",
		Address: envoy.SocketAddress("0.0.0.0", 8443),
		ListenerFilters: envoy.ListenerFilters(
			envoy.TLSInspector(),
		),
		FilterChains: filterchaintls("example.com", s1, envoy.HTTPConnectionManager("ingress_https", "/dev/stdout", nil), "h2", "http/1.1"),
	}

	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "4",
		Resources: resources(t,
			ingress_http,
			ingress_https,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "4",
	}, streamLDS(t, cc))

	// t2 is a TLSCertificateDelegation that permits access to secret/wildcard from all namespaces.
	t2 := &gatewayhostv1.TLSCertificateDelegation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "delegation",
			Namespace: "secret",
		},
		Spec: gatewayhostv1.TLSCertificateDelegationSpec{
			Delegations: []gatewayhostv1.CertificateDelegation{{
				SecretName: "wildcard",
				TargetNamespaces: []string{
					"*",
				},
			}},
		},
	}
	rh.OnUpdate(t1, t2)

	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "5",
		Resources: resources(t,
			ingress_http,
			ingress_https,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "5",
	}, streamLDS(t, cc))

	// t3 is a TLSCertificateDelegation that permits access to secret/different all namespaces.
	t3 := &gatewayhostv1.TLSCertificateDelegation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "delegation",
			Namespace: "secret",
		},
		Spec: gatewayhostv1.TLSCertificateDelegationSpec{
			Delegations: []gatewayhostv1.CertificateDelegation{{
				SecretName: "different",
				TargetNamespaces: []string{
					"*",
				},
			}},
		},
	}
	rh.OnUpdate(t2, t3)

	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "6",
		Resources: resources(t,
			ingress_http,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "6",
	}, streamLDS(t, cc))

	// t4 is a TLSCertificateDelegation that permits access to secret/wildcard from the kube-secret namespace.
	t4 := &gatewayhostv1.TLSCertificateDelegation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "delegation",
			Namespace: "secret",
		},
		Spec: gatewayhostv1.TLSCertificateDelegationSpec{
			Delegations: []gatewayhostv1.CertificateDelegation{{
				SecretName: "wildcard",
				TargetNamespaces: []string{
					"kube-secret",
				},
			}},
		},
	}
	rh.OnUpdate(t3, t4)

	assert.Equal(t, &envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "7",
		Resources: resources(t,
			ingress_http,
			staticListener(),
		),
		TypeUrl: listenerType,
		Nonce:   "7",
	}, streamLDS(t, cc))

}

func streamLDS(t *testing.T, cc *grpc.ClientConn, rn ...string) *envoy_service_discovery_v3.DiscoveryResponse {
	t.Helper()
	rds := envoy_service_listener_v3.NewListenerDiscoveryServiceClient(cc)
	st, err := rds.StreamListeners(context.TODO())
	check(t, err)
	return stream(t, st, &envoy_service_discovery_v3.DiscoveryRequest{
		TypeUrl:       listenerType,
		ResourceNames: rn,
	})
}

func backend(name string, port intstr.IntOrString) *netv1.IngressBackend {
	if port.Type == intstr.Int {
		return &netv1.IngressBackend{
			Service: &netv1.IngressServiceBackend{
				Name: name,
				Port: netv1.ServiceBackendPort{
					Number: port.IntVal,
				},
			},
		}
	} else {
		return &netv1.IngressBackend{
			Service: &netv1.IngressServiceBackend{
				Name: name,
				Port: netv1.ServiceBackendPort{
					Name: port.StrVal,
				},
			},
		}
	}
}

func filterchaintls(domain string, secret *corev1.Secret, filter *envoy_config_listener_v3.Filter, alpn ...string) []*envoy_config_listener_v3.FilterChain {
	return []*envoy_config_listener_v3.FilterChain{
		envoy.FilterChainTLS(
			domain,
			&dag.Secret{Object: secret},
			[]*envoy_config_listener_v3.Filter{
				filter,
			},
			envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_1,
			alpn...,
		),
	}
}

func tcpproxy(t *testing.T, statPrefix, cluster string) *envoy_config_listener_v3.Filter {
	return &envoy_config_listener_v3.Filter{
		Name: wellknown.TCPProxy,
		ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
			TypedConfig: toAny(t, &envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy{
				StatPrefix: statPrefix,
				ClusterSpecifier: &envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_Cluster{
					Cluster: cluster,
				},
				AccessLog:   envoy.FileAccessLog("/dev/stdout"),
				IdleTimeout: protobuf.Duration(9001 * time.Second),
			}),
		},
	}
}

func staticListener() *envoy_config_listener_v3.Listener {
	return envoy.StatsListener(statsAddress, statsPort)
}
