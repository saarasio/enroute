// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package envoy

import (
	"crypto/sha1"
	"fmt"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
)

// Secretname returns the name of the SDS secret for this secret.
func Secretname(s *dag.Secret) string {
	hash := sha1.Sum(s.Cert())
	ns := s.Namespace()
	name := s.Name()
	return hashname(60, ns, name, fmt.Sprintf("%x", hash[:5]))
}

// Secret creates new envoy_extensions_transport_sockets_tls_v3.Secret from secret.
func Secret(s *dag.Secret) *envoy_extensions_transport_sockets_tls_v3.Secret {
	return &envoy_extensions_transport_sockets_tls_v3.Secret{
		Name: Secretname(s),
		Type: &envoy_extensions_transport_sockets_tls_v3.Secret_TlsCertificate{
			TlsCertificate: &envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
				PrivateKey: &envoy_config_core_v3.DataSource{
					Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
						InlineBytes: s.PrivateKey(),
					},
				},
				CertificateChain: &envoy_config_core_v3.DataSource{
					Specifier: &envoy_config_core_v3.DataSource_InlineBytes{
						InlineBytes: s.Cert(),
					},
				},
			},
		},
	}
}
