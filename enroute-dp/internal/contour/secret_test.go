// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package contour

import (
	"reflect"
	"testing"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	gatewayhostv1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/metrics"
	"google.golang.org/protobuf/testing/protocmp"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSecretCacheContents(t *testing.T) {
	tests := map[string]struct {
		contents map[string]*envoy_extensions_transport_sockets_tls_v3.Secret
		want     []proto.Message
	}{
		"empty": {
			contents: nil,
			want:     nil,
		},
		"simple": {
			contents: secretmap(
				secret("default/secret/cd1b506996", "cert", "key"),
			),
			want: []proto.Message{
				secret("default/secret/cd1b506996", "cert", "key"),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var sc SecretCache
			sc.Update(tc.contents)
			got := sc.Contents()
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSecretCacheQuery(t *testing.T) {
	tests := map[string]struct {
		contents map[string]*envoy_extensions_transport_sockets_tls_v3.Secret
		query    []string
		want     []proto.Message
	}{
		"exact match": {
			contents: secretmap(
				secret("default/secret/cd1b506996", "cert", "key"),
			),
			query: []string{"default/secret/cd1b506996"},
			want: []proto.Message{
				secret("default/secret/cd1b506996", "cert", "key"),
			},
		},
		"partial match": {
			contents: secretmap(
				secret("default/secret-a/ff2a9f58ca", "cert-a", "key-a"),
				secret("default/secret-b/0a068be4ba", "cert-b", "key-b"),
			),
			query: []string{"default/secret/cd1b506996", "default/secret-b/0a068be4ba"},
			want: []proto.Message{
				secret("default/secret-b/0a068be4ba", "cert-b", "key-b"),
			},
		},
		"no match": {
			contents: secretmap(
				secret("default/secret/cd1b506996", "cert", "key"),
			),
			query: []string{"default/secret-b/0a068be4ba"},
			want:  nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var sc SecretCache
			sc.Update(tc.contents)
			got := sc.Query(tc.query)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSecretVisit(t *testing.T) {
	tests := map[string]struct {
		objs []interface{}
		want map[string]*envoy_extensions_transport_sockets_tls_v3.Secret
	}{
		"nothing": {
			objs: nil,
			want: map[string]*envoy_extensions_transport_sockets_tls_v3.Secret{},
		},
		"unassociated secrets": {
			objs: []interface{}{
				tlssecret("default", "secret-a", secretdata("cert", "key")),
				tlssecret("default", "secret-b", secretdata("cert", "key")),
			},
			want: map[string]*envoy_extensions_transport_sockets_tls_v3.Secret{},
		},
		"simple ingress with secret": {
			objs: []interface{}{
				&netv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: netv1.IngressSpec{
						TLS: []netv1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						DefaultBackend: &netv1.IngressBackend{
							Service: &netv1.IngressServiceBackend{
								Name: "kuard",
								Port: netv1.ServiceBackendPort{
									Number: 8080,
								},
							},
						},
					},
				},
				tlssecret("default", "secret", secretdata("cert", "key")),
			},
			want: secretmap(
				secret("default/secret/cd1b506996", "cert", "key"),
			),
		},
		"multiple ingresses with shared secret": {
			objs: []interface{}{
				&netv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple-a",
						Namespace: "default",
					},
					Spec: netv1.IngressSpec{
						TLS: []netv1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						DefaultBackend: &netv1.IngressBackend{
							Service: &netv1.IngressServiceBackend{
								Name: "kuard",
								Port: netv1.ServiceBackendPort{
									Number: 8080,
								},
							},
						},
					},
				},
				&netv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple-b",
						Namespace: "default",
					},
					Spec: netv1.IngressSpec{
						TLS: []netv1.IngressTLS{{
							Hosts:      []string{"omg.example.com"},
							SecretName: "secret",
						}},
						DefaultBackend: &netv1.IngressBackend{
							Service: &netv1.IngressServiceBackend{
								Name: "kuard",
								Port: netv1.ServiceBackendPort{
									Number: 8080,
								},
							},
						},
					},
				},
				tlssecret("default", "secret", secretdata("cert", "key")),
			},
			want: secretmap(
				secret("default/secret/cd1b506996", "cert", "key"),
			),
		},
		"multiple ingresses with different secrets": {
			objs: []interface{}{
				&netv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple-a",
						Namespace: "default",
					},
					Spec: netv1.IngressSpec{
						TLS: []netv1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret-a",
						}},
						DefaultBackend: &netv1.IngressBackend{
							Service: &netv1.IngressServiceBackend{
								Name: "kuard",
								Port: netv1.ServiceBackendPort{
									Number: 8080,
								},
							},
						},
					},
				},
				&netv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple-b",
						Namespace: "default",
					},
					Spec: netv1.IngressSpec{
						TLS: []netv1.IngressTLS{{
							Hosts:      []string{"omg.example.com"},
							SecretName: "secret-b",
						}},
						DefaultBackend: &netv1.IngressBackend{
							Service: &netv1.IngressServiceBackend{
								Name: "kuard",
								Port: netv1.ServiceBackendPort{
									Number: 8080,
								},
							},
						},
					},
				},
				tlssecret("default", "secret-a", secretdata("cert-a", "key-a")),
				tlssecret("default", "secret-b", secretdata("cert-b", "key-b")),
			},
			want: secretmap(
				secret("default/secret-a/ff2a9f58ca", "cert-a", "key-a"),
				secret("default/secret-b/0a068be4ba", "cert-b", "key-b"),
			),
		},
		"simple gatewayhost with secret": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "www.example.com",
							TLS: &gatewayhostv1.TLS{
								SecretName: "secret",
							},
						},
						Routes: []gatewayhostv1.Route{{
							Services: []gatewayhostv1.Service{
								{
									Name: "backend",
									Port: 80,
								},
							}},
						},
					},
				},
				tlssecret("default", "secret", secretdata("cert", "key")),
			},
			want: secretmap(
				secret("default/secret/cd1b506996", "cert", "key"),
			),
		},
		"multiple gatewayhosts with shared secret": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple-a",
						Namespace: "default",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "www.example.com",
							TLS: &gatewayhostv1.TLS{
								SecretName: "secret",
							},
						},
						Routes: []gatewayhostv1.Route{
							{
								Services: []gatewayhostv1.Service{
									{
										Name: "backend",
										Port: 80,
									},
								},
							},
						},
					},
				},
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple-b",
						Namespace: "default",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "www.other.com",
							TLS: &gatewayhostv1.TLS{
								SecretName: "secret",
							},
						},
						Routes: []gatewayhostv1.Route{
							{
								Services: []gatewayhostv1.Service{
									{
										Name: "backend",
										Port: 80,
									},
								},
							},
						},
					},
				},
				tlssecret("default", "secret", secretdata("cert", "key")),
			},
			want: secretmap(
				secret("default/secret/cd1b506996", "cert", "key"),
			),
		},
		"multiple gatewayhosts with different secret": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple-a",
						Namespace: "default",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "www.example.com",
							TLS: &gatewayhostv1.TLS{
								SecretName: "secret-a",
							},
						},
						Routes: []gatewayhostv1.Route{
							{
								Services: []gatewayhostv1.Service{
									{
										Name: "backend",
										Port: 80,
									},
								},
							},
						},
					},
				},
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple-b",
						Namespace: "default",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "www.other.com",
							TLS: &gatewayhostv1.TLS{
								SecretName: "secret-b",
							},
						},
						Routes: []gatewayhostv1.Route{
							{
								Services: []gatewayhostv1.Service{
									{
										Name: "backend",
										Port: 80,
									},
								},
							},
						},
					},
				},
				tlssecret("default", "secret-a", secretdata("cert-a", "key-a")),
				tlssecret("default", "secret-b", secretdata("cert-b", "key-b")),
			},
			want: secretmap(
				secret("default/secret-a/ff2a9f58ca", "cert-a", "key-a"),
				secret("default/secret-b/0a068be4ba", "cert-b", "key-b"),
			),
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
				reh.OnAdd(o, false)
			}
			root := dag.BuildDAG(&reh.KubernetesCache)
			got := visitSecrets(root)
			if !reflect.DeepEqual(tc.want, got) {
				t.Fatalf("expected:\n%+v\ngot:\n%+v", tc.want, got)
			}
		})
	}
}

func secretmap(secrets ...*envoy_extensions_transport_sockets_tls_v3.Secret) map[string]*envoy_extensions_transport_sockets_tls_v3.Secret {
	m := make(map[string]*envoy_extensions_transport_sockets_tls_v3.Secret)
	for _, s := range secrets {
		m[s.Name] = s
	}
	return m
}

func secret(name, cert, key string) *envoy_extensions_transport_sockets_tls_v3.Secret {
	return &envoy_extensions_transport_sockets_tls_v3.Secret{
		Name: name,
		Type: &envoy_extensions_transport_sockets_tls_v3.Secret_TlsCertificate{
			TlsCertificate: &envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
				PrivateKey: &envoy_config_core_v3.DataSource{
					Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
						InlineBytes: []byte(key),
					},
				},
				CertificateChain: &envoy_config_core_v3.DataSource{
					Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
						InlineBytes: []byte(cert),
					},
				},
			},
		},
	}
}

// tlssecert creates a new corev1.Secret object of type kubernetes.io/tls.
func tlssecret(namespace, name string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: data,
	}
}
