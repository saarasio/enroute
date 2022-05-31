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
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	_ "github.com/saarasio/enroute/enroute-dp/internal/logger"
)

var (
	// This is the list of default ciphers used by contour 1.9.1. A handful are
	// commented out, as they're arguably less secure. They're also unnecessary
	// - most of the clients that might need to use the commented ciphers are
	// unable to connect without TLS 1.0, which contour never enables.
	//
	// This list is ignored if the client and server negotiate TLS 1.3.
	//
	// The commented ciphers are left in place to simplify updating this list for future
	// versions of envoy.
	ciphers = []string{
		"[ECDHE-ECDSA-AES128-GCM-SHA256|ECDHE-ECDSA-CHACHA20-POLY1305]",
		"[ECDHE-RSA-AES128-GCM-SHA256|ECDHE-RSA-CHACHA20-POLY1305]",
		"ECDHE-ECDSA-AES128-SHA",
		"ECDHE-RSA-AES128-SHA",
		//"AES128-GCM-SHA256",
		//"AES128-SHA",
		"ECDHE-ECDSA-AES256-GCM-SHA384",
		"ECDHE-RSA-AES256-GCM-SHA384",
		"ECDHE-ECDSA-AES256-SHA",
		"ECDHE-RSA-AES256-SHA",
		//"AES256-GCM-SHA384",
		//"AES256-SHA",
	}
)

// UpstreamTLSContext creates an envoy_extensions_transport_sockets_tls_v3.UpstreamTlsContext. By default
// UpstreamTLSContext returns a HTTP/1.1 TLS enabled context. A list of
// additional ALPN protocols can be provided.
func UpstreamTLSContext(sni string, ca []byte, subjectName string, alpnProtocols ...string) *envoy_extensions_transport_sockets_tls_v3.UpstreamTlsContext {
	context := &envoy_extensions_transport_sockets_tls_v3.UpstreamTlsContext{
		CommonTlsContext: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext{
			AlpnProtocols: alpnProtocols,
		},
		Sni: sni,
	}

	// we have to do explicitly assign the value from validationContext
	// to context.CommonTlsContext.ValidationContextType because the latter
	// is an interface, returning nil from validationContext directly into
	// this field boxes the nil into the unexported type of this grpc OneOf field
	// which causes proto marshaling to explode later on. Not happy Jan.
	vc := validationContext(ca, subjectName)
	if vc != nil {
		context.CommonTlsContext.ValidationContextType = vc
	}

	return context
}

func UpstreamTLSContextWithClientValidation(sni string, ca []byte, cv_cert []byte, cv_key []byte, subjectName string, alpnProtocols ...string) *envoy_extensions_transport_sockets_tls_v3.UpstreamTlsContext {

	context := &envoy_extensions_transport_sockets_tls_v3.UpstreamTlsContext{
		CommonTlsContext: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext{
			AlpnProtocols: alpnProtocols,
		},
		Sni: sni,
	}

	// we have to do explicitly assign the value from validationContext
	// to context.CommonTlsContext.ValidationContextType because the latter
	// is an interface, returning nil from validationContext directly into
	// this field boxes the nil into the unexported type of this grpc OneOf field
	// which causes proto marshaling to explode later on. Not happy Jan.
	vc := validationContext(ca, subjectName)
	if vc != nil {
		context.CommonTlsContext.ValidationContextType = vc
	}

	ctls := clientTlsCertificates(cv_cert, cv_key)
	if ctls != nil {
		if context.CommonTlsContext.TlsCertificates == nil {
			context.CommonTlsContext.TlsCertificates = make([]*envoy_extensions_transport_sockets_tls_v3.TlsCertificate, 0)
		}
		context.CommonTlsContext.TlsCertificates = append(context.CommonTlsContext.TlsCertificates, ctls)
	}

	return context
}

func StringToExactMatch(in []string) []*matcher.StringMatcher {
	if len(in) == 0 {
		return nil
	}
	res := make([]*matcher.StringMatcher, 0, len(in))
	for _, s := range in {
		res = append(res, &matcher.StringMatcher{
			MatchPattern: &matcher.StringMatcher_Exact{Exact: s},
		})
	}
	return res
}

func clientTlsCertificates(cv_cert, cv_key []byte) *envoy_extensions_transport_sockets_tls_v3.TlsCertificate {
	return &envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
		CertificateChain: &envoy_config_core_v3.DataSource{
			Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
				InlineBytes: cv_cert,
			},
		},
		PrivateKey: &envoy_config_core_v3.DataSource{
			Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
				InlineBytes: cv_key,
			},
		},
	}
}

func validationContext(ca []byte, subjectName string) *envoy_extensions_transport_sockets_tls_v3.CommonTlsContext_ValidationContext {
	if len(ca) < 1 {
		// no ca provided, nothing to do
		return nil
	}

	if len(subjectName) < 1 {
		// no subject name provided, nothing to do
		return nil
	}

	return &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext_ValidationContext{
		ValidationContext: &envoy_extensions_transport_sockets_tls_v3.CertificateValidationContext{
			TrustedCa: &envoy_config_core_v3.DataSource{
				// TODO(dfc) update this for SDS
				Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
					InlineBytes: ca,
				},
			},
			MatchSubjectAltNames: StringToExactMatch([]string{subjectName}),
		},
	}
}

// DownstreamTLSContext creates a new DownstreamTlsContext.
func DownstreamTLSContext(secretName string, tlsMinProtoVersion envoy_extensions_transport_sockets_tls_v3.TlsParameters_TlsProtocol, alpnProtos ...string) *envoy_extensions_transport_sockets_tls_v3.DownstreamTlsContext {
	return &envoy_extensions_transport_sockets_tls_v3.DownstreamTlsContext{
		CommonTlsContext: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext{
			TlsParams: &envoy_extensions_transport_sockets_tls_v3.TlsParameters{
				TlsMinimumProtocolVersion: tlsMinProtoVersion,
				TlsMaximumProtocolVersion: envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_3,
				CipherSuites:              ciphers,
			},
			TlsCertificateSdsSecretConfigs: []*envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig{{
				Name:      secretName,
				SdsConfig: ConfigSource("enroute"),
			}},
			AlpnProtocols: alpnProtos,
		},
	}
}
