// SPDX-License-Identifier: Apache-2.0
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

package e2e

import (
	"testing"

	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/envoy"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestSDSVisibility(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	c := &Contour{
		T:          t,
		ClientConn: cc,
	}

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
	// add secret
	rh.OnAdd(s1, false)

	// assert that the secret is _not_ visible as it is
	// not referenced by any ingress/gatewayhost
	c.Request(secretType).Equals(&envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "1",
		Resources:   resources(t),
		TypeUrl:     secretType,
		Nonce:       "1",
	})

	// i1 is a tls ingress
	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: netv1.IngressSpec{
			DefaultBackend: backend("backend", intstr.FromInt(80)),
			TLS: []netv1.IngressTLS{{
				Hosts:      []string{"kuard.example.com"},
				SecretName: "secret",
			}},
		},
	}
	rh.OnAdd(i1, false)

	// TODO(dfc) #1165: secret should not be present if the ingress does not
	// have any valid routes.
	// i1 has a default route to backend:80, but there is no matching service.
	c.Request(secretType).Equals(&envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			secret(s1),
		),
		TypeUrl: secretType,
		Nonce:   "2",
	})
}

func TestSDSShouldNotIncrementVersionNumberForUnrelatedSecret(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	c := &Contour{
		T:          t,
		ClientConn: cc,
	}

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
	// add secret
	rh.OnAdd(s1, false)

	// i1 is a tls ingress
	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: netv1.IngressSpec{
			DefaultBackend: backend("backend", intstr.FromInt(80)),
			TLS: []netv1.IngressTLS{{
				Hosts:      []string{"kuard.example.com"},
				SecretName: "secret",
			}},
		},
	}
	rh.OnAdd(i1, false)

	c.Request(secretType).Equals(&envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			secret(s1),
		),
		TypeUrl: secretType,
		Nonce:   "2",
	})

	// verify that requesting the same resource without change
	// does not bump the current version_info.

	c.Request(secretType).Equals(&envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			secret(s1),
		),
		TypeUrl: secretType,
		Nonce:   "2",
	})

	// s2 is not referenced by any active ingress object.
	s2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unrelated",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("key"),
		},
	}
	rh.OnAdd(s2, false)

	t.Skipf("See issue 1166")

	// TODO(dfc) 1166: currently Contour will rebuild all the xDS tables
	// when an unrelated secret changes.
	c.Request(secretType).Equals(&envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources: resources(t,
			secret(s1),
		),
		TypeUrl: secretType,
		Nonce:   "2",
	})
}

// issue 1169, an invalid certificate should not be
// presented by SDS even if referenced by an ingress object.
func TestSDSshouldNotPublishInvalidSecret(t *testing.T) {
	rh, cc, done := setup(t)
	defer done()

	c := &Contour{
		T:          t,
		ClientConn: cc,
	}

	// s1 is NOT a tls secret
	s1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "invalid",
			Namespace: "default",
		},
		Type: "kubernetes.io/dockerconfigjson",
		Data: map[string][]byte{
			".dockerconfigjson": []byte("ewogICAgImF1dGhzIjogewogICAgICAgICJodHRwczovL2luZGV4LmRvY2tlci5pby92MS8iOiB7CiAgICAgICAgICAgICJhdXRoIjogImMzUi4uLnpFMiIKICAgICAgICB9CiAgICB9Cn0K"),
		},
	}
	// add secret
	rh.OnAdd(s1, false)

	// i1 is a tls ingress
	i1 := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "default",
		},
		Spec: netv1.IngressSpec{
			DefaultBackend: backend("backend", intstr.FromInt(80)),
			TLS: []netv1.IngressTLS{{
				Hosts:      []string{"kuard.example.com"},
				SecretName: "invalid",
			}},
		},
	}
	rh.OnAdd(i1, false)

	// SDS should be empty
	c.Request(secretType).Equals(&envoy_service_discovery_v3.DiscoveryResponse{
		VersionInfo: "2",
		Resources:   resources(t),
		TypeUrl:     secretType,
		Nonce:       "2",
	})
}

func secret(sec *corev1.Secret) *envoy_extensions_transport_sockets_tls_v3.Secret {
	return envoy.Secret(&dag.Secret{
		Object: sec,
	})
}
