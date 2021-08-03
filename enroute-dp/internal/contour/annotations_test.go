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

package contour

import (
	"reflect"
	"testing"

	"github.com/gogo/protobuf/types"
	"k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWebsocketRoutes(t *testing.T) {
	tests := map[string]struct {
		a    *v1.Ingress
		want map[string]*types.BoolValue
	}{
		"empty": {
			a: &v1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{annotationWebsocketRoutes: ""},
				},
			},
			want: map[string]*types.BoolValue{},
		},
		"empty with spaces": {
			a: &v1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{annotationWebsocketRoutes: ", ,"},
				},
			},
			want: map[string]*types.BoolValue{},
		},
		"single value": {
			a: &v1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{annotationWebsocketRoutes: "/ws1"},
				},
			},
			want: map[string]*types.BoolValue{
				"/ws1": {Value: true},
			},
		},
		"multiple values": {
			a: &v1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{annotationWebsocketRoutes: "/ws1,/ws2"},
				},
			},
			want: map[string]*types.BoolValue{
				"/ws1": {Value: true},
				"/ws2": {Value: true},
			},
		},
		"multiple values with spaces and invalid entries": {
			a: &v1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{annotationWebsocketRoutes: " /ws1, , /ws2 "},
				},
			},
			want: map[string]*types.BoolValue{
				"/ws1": {Value: true},
				"/ws2": {Value: true},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := websocketRoutes(tc.a)
			if !reflect.DeepEqual(tc.want, got) {
				t.Fatalf("websocketRoutes(%q): want: %v, got: %v", tc.a, tc.want, got)
			}
		})
	}
}

func TestHttpAllowed(t *testing.T) {
	tests := map[string]struct {
		i     *v1.Ingress
		valid bool
	}{
		"basic ingress": {
			i: &v1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "simple",
					Namespace: "default",
				},
				Spec: v1.IngressSpec{
					TLS: []v1.IngressTLS{{
						Hosts:      []string{"whatever.example.com"},
						SecretName: "secret",
					}},
					DefaultBackend: backend("backend", 80),
				},
			},
			valid: true,
		},
		"kubernetes.io/ingress.allow-http: \"false\"": {
			i: &v1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "simple",
					Namespace: "default",
					Annotations: map[string]string{
						"kubernetes.io/ingress.allow-http": "false",
					},
				},
				Spec: v1.IngressSpec{
					TLS: []v1.IngressTLS{{
						Hosts:      []string{"whatever.example.com"},
						SecretName: "secret",
					}},
					DefaultBackend: backend("backend", 80),
				},
			},
			valid: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := httpAllowed(tc.i)
			want := tc.valid
			if got != want {
				t.Fatalf("got: %v, want: %v", got, want)
			}
		})
	}
}

func backend(name string, port int32) *v1.IngressBackend {
	return &v1.IngressBackend{
		Service: &v1.IngressServiceBackend{
			Name: name,
			Port: v1.ServiceBackendPort{
				Number: port,
			},
		},
	}
}
