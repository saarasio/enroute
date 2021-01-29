// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package contour

import (
	"reflect"
	"testing"

	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	gatewayhostv1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1beta1"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/metrics"
	"google.golang.org/protobuf/testing/protocmp"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestSecretCacheContents(t *testing.T) {
	tests := map[string]struct {
		contents map[string]*envoy_api_v2_auth.Secret
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
		contents map[string]*envoy_api_v2_auth.Secret
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
		want map[string]*envoy_api_v2_auth.Secret
	}{
		"nothing": {
			objs: nil,
			want: map[string]*envoy_api_v2_auth.Secret{},
		},
		"unassociated secrets": {
			objs: []interface{}{
				tlssecret("default", "secret-a", secretdata("cert", "key")),
				tlssecret("default", "secret-b", secretdata("cert", "key")),
			},
			want: map[string]*envoy_api_v2_auth.Secret{},
		},
		"simple ingress with secret": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromInt(8080),
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
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple-a",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromInt(8080),
						},
					},
				},
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple-b",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"omg.example.com"},
							SecretName: "secret",
						}},
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromInt(8080),
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
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple-a",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret-a",
						}},
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromInt(8080),
						},
					},
				},
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple-b",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"omg.example.com"},
							SecretName: "secret-b",
						}},
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromInt(8080),
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
				reh.OnAdd(o)
			}
			root := dag.BuildDAG(&reh.KubernetesCache)
			got := visitSecrets(root)
			if !reflect.DeepEqual(tc.want, got) {
				t.Fatalf("expected:\n%+v\ngot:\n%+v", tc.want, got)
			}
		})
	}
}

func secretmap(secrets ...*envoy_api_v2_auth.Secret) map[string]*envoy_api_v2_auth.Secret {
	m := make(map[string]*envoy_api_v2_auth.Secret)
	for _, s := range secrets {
		m[s.Name] = s
	}
	return m
}

func secret(name, cert, key string) *envoy_api_v2_auth.Secret {
	return &envoy_api_v2_auth.Secret{
		Name: name,
		Type: &envoy_api_v2_auth.Secret_TlsCertificate{
			TlsCertificate: &envoy_api_v2_auth.TlsCertificate{
				PrivateKey: &envoy_api_v2_core.DataSource{
					Specifier: &envoy_api_v2_core.DataSource_InlineBytes{
						InlineBytes: []byte(key),
					},
				},
				CertificateChain: &envoy_api_v2_core.DataSource{
					Specifier: &envoy_api_v2_core.DataSource_InlineBytes{
						InlineBytes: []byte(cert),
					},
				},
			},
		},
	}
}

// tlssecert creates a new v1.Secret object of type kubernetes.io/tls.
func tlssecret(namespace, name string, data map[string][]byte) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: v1.SecretTypeTLS,
		Data: data,
	}
}
