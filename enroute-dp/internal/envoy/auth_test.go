// SPDX-License-Identifier: Apache-1.0
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
	"github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"testing"
)

func TestUpstreamTLSContext(t *testing.T) {
	tests := map[string]struct {
		ca            []byte
		subjectName   string
		alpnProtocols []string
		want          *envoy_extensions_transport_sockets_tls_v3.UpstreamTlsContext
	}{
		"no alpn, no validation": {
			want: &envoy_extensions_transport_sockets_tls_v3.UpstreamTlsContext{
				CommonTlsContext: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext{},
			},
		},
		"h2, no validation": {
			alpnProtocols: []string{"h2c"},
			want: &envoy_extensions_transport_sockets_tls_v3.UpstreamTlsContext{
				CommonTlsContext: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext{
					AlpnProtocols: []string{"h2c"},
				},
			},
		},
		"no alpn, missing altname": {
			ca: []byte("ca"),
			want: &envoy_extensions_transport_sockets_tls_v3.UpstreamTlsContext{
				CommonTlsContext: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext{},
			},
		},
		"no alpn, missing ca": {
			subjectName: "www.example.com",
			want: &envoy_extensions_transport_sockets_tls_v3.UpstreamTlsContext{
				CommonTlsContext: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext{},
			},
		},
		"no alpn, ca and altname": {
			ca:          []byte("ca"),
			subjectName: "www.example.com",
			want: &envoy_extensions_transport_sockets_tls_v3.UpstreamTlsContext{
				CommonTlsContext: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext{
					ValidationContextType: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext_ValidationContext{
						ValidationContext: &envoy_extensions_transport_sockets_tls_v3.CertificateValidationContext{
							TrustedCa: &envoy_config_core_v3.DataSource{
								Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
									InlineBytes: []byte("ca"),
								},
							},
							MatchSubjectAltNames: StringToExactMatch([]string{"www.example.com"}),
						},
					},
				},
			},
		},
	}

	for name, tc := range tests {
		sni := ""
		t.Run(name, func(t *testing.T) {
			got := UpstreamTLSContext(sni, tc.ca, tc.subjectName, tc.alpnProtocols...)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
