// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package envoy

import (
	"testing"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/google/go-cmp/cmp"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"google.golang.org/protobuf/testing/protocmp"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSecret(t *testing.T) {
	tests := map[string]struct {
		secret *dag.Secret
		want   *envoy_extensions_transport_sockets_tls_v3.Secret
	}{
		"simple secret": {
			secret: &dag.Secret{
				Object: &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Data: map[string][]byte{
						v1.TLSCertKey:       []byte("cert"),
						v1.TLSPrivateKeyKey: []byte("key"),
					},
				},
			},
			want: &envoy_extensions_transport_sockets_tls_v3.Secret{
				Name: "default/simple/cd1b506996",
				Type: &envoy_extensions_transport_sockets_tls_v3.Secret_TlsCertificate{
					TlsCertificate: &envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
						PrivateKey: &envoy_config_core_v3.DataSource{
							Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
								InlineBytes: []byte("key"),
							},
						},
						CertificateChain: &envoy_config_core_v3.DataSource{
							Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
								InlineBytes: []byte("cert"),
							},
						},
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := Secret(tc.secret)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSecretname(t *testing.T) {
	tests := map[string]struct {
		secret *dag.Secret
		want   string
	}{
		"simple": {
			secret: &dag.Secret{
				Object: &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Data: map[string][]byte{
						v1.TLSCertKey:       []byte("cert"),
						v1.TLSPrivateKeyKey: []byte("key"),
					},
				},
			},
			want: "default/simple/cd1b506996",
		},
		"far too long": {
			secret: &dag.Secret{
				Object: &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "must-be-in-want-of-a-wife",
						Namespace: "it-is-a-truth-universally-acknowledged-that-a-single-man-in-possession-of-a-good-fortune",
					},
					Data: map[string][]byte{
						v1.TLSCertKey:       []byte("cert"),
						v1.TLSPrivateKeyKey: []byte("key"),
					},
				},
			},
			want: "it-is-a-truth-7e164b/must-be-in-wa-7e164b/cd1b506996",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := Secretname(tc.secret)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
