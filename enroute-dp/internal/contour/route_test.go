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
	"testing"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	gatewayhostv1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1beta1"
	"github.com/saarasio/enroute/enroute-dp/internal/assert"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/envoy"
	"github.com/saarasio/enroute/enroute-dp/internal/metrics"
	"github.com/saarasio/enroute/enroute-dp/internal/protobuf"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestRouteCacheContents(t *testing.T) {
	tests := map[string]struct {
		contents map[string]*v2.RouteConfiguration
		want     []proto.Message
	}{
		"empty": {
			contents: nil,
			want:     nil,
		},
		"simple": {
			contents: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
			want: []proto.Message{
				&v2.RouteConfiguration{
					Name: "ingress_http",
				},
				&v2.RouteConfiguration{
					Name: "ingress_https",
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var rc RouteCache
			rc.Update(tc.contents)
			got := rc.Contents()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRouteCacheQuery(t *testing.T) {
	tests := map[string]struct {
		contents map[string]*v2.RouteConfiguration
		query    []string
		want     []proto.Message
	}{
		"exact match": {
			contents: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
				},
			},
			query: []string{"ingress_http"},
			want: []proto.Message{
				&v2.RouteConfiguration{
					Name: "ingress_http",
				},
			},
		},
		"partial match": {
			contents: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
				},
			},
			query: []string{"stats-handler", "ingress_http"},
			want: []proto.Message{
				&v2.RouteConfiguration{
					Name: "ingress_http",
				},
				&v2.RouteConfiguration{
					Name: "stats-handler",
				},
			},
		},
		"no match": {
			contents: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
				},
			},
			query: []string{"stats-handler"},
			want: []proto.Message{
				&v2.RouteConfiguration{
					Name: "stats-handler",
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var rc RouteCache
			rc.Update(tc.contents)
			got := rc.Query(tc.query)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRouteVisit(t *testing.T) {
	tests := map[string]struct {
		objs []interface{}
		want map[string]*v2.RouteConfiguration
	}{
		"nothing": {
			objs: nil,
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"one http only ingress with service": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromInt(8080),
						},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "*",
						Domains: []string{"*"},
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routecluster("default/kuard/8080/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"one http only gatewayhost": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "www.example.com",
						},
						Routes: []gatewayhostv1.Route{{
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/",
							}},
							Services: []gatewayhostv1.Service{
								{
									Name: "backend",
									Port: 80,
								},
							},
						}},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "backend",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       80,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "www.example.com",
						Domains: domains("www.example.com"),
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routecluster("default/backend/80/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"default backend ingress with secret": {
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
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata("certificate", "key"),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "*", // default backend
						Domains: []string{"*"},
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routecluster("default/kuard/8080/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https", // no https for default backend
				},
			},
		},
		"vhost ingress with secret": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"www.example.com"},
							SecretName: "secret",
						}},
						Rules: []v1beta1.IngressRule{{
							Host: "www.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Backend: v1beta1.IngressBackend{
											ServiceName: "kuard",
											ServicePort: intstr.FromString("www"),
										},
									}},
								},
							},
						}},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata("certificate", "key"),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:       "www",
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "www.example.com",
						Domains: domains("www.example.com"),
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routecluster("default/kuard/8080/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "www.example.com",
						Domains: domains("www.example.com"),
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routecluster("default/kuard/8080/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
			},
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
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/",
							}},
							Services: []gatewayhostv1.Service{{
								Name: "backend",
								Port: 8080,
							},
							}},
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata("certificate", "key"),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "backend",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:       "www",
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "www.example.com",
						Domains: domains("www.example.com"),
						Routes: []*envoy_api_v2_route.Route{{
							Match: envoy.RouteMatch("/"),
							Action: &envoy_api_v2_route.Route_Redirect{
								Redirect: &envoy_api_v2_route.RedirectAction{
									SchemeRewriteSpecifier: &envoy_api_v2_route.RedirectAction_HttpsRedirect{
										HttpsRedirect: true,
									},
								},
							},
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "www.example.com",
						Domains: domains("www.example.com"),
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routecluster("default/backend/8080/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
			},
		},
		"simple tls ingress with allow-http:false": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
						Annotations: map[string]string{
							"kubernetes.io/ingress.allow-http": "false",
						},
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"www.example.com"},
							SecretName: "secret",
						}},
						Rules: []v1beta1.IngressRule{{
							Host: "www.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Backend: v1beta1.IngressBackend{
											ServiceName: "kuard",
											ServicePort: intstr.FromString("www"),
										},
									}},
								},
							},
						}},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata("certificate", "key"),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:       "www",
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
				},
				"ingress_https": {
					Name: "ingress_https",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "www.example.com",
						Domains: domains("www.example.com"),
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routecluster("default/kuard/8080/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
			},
		},
		"simple tls ingress with force-ssl-redirect": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
						Annotations: map[string]string{
							"ingress.kubernetes.io/force-ssl-redirect": "true",
						},
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"www.example.com"},
							SecretName: "secret",
						}},
						Rules: []v1beta1.IngressRule{{
							Host: "www.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Backend: v1beta1.IngressBackend{
											ServiceName: "kuard",
											ServicePort: intstr.FromString("www"),
										},
									}},
								},
							},
						}},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata("certificate", "key"),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:       "www",
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "www.example.com",
						Domains: domains("www.example.com"),
						Routes: []*envoy_api_v2_route.Route{{
							Match: envoy.RouteMatch("/"),
							Action: &envoy_api_v2_route.Route_Redirect{
								Redirect: &envoy_api_v2_route.RedirectAction{
									SchemeRewriteSpecifier: &envoy_api_v2_route.RedirectAction_HttpsRedirect{
										HttpsRedirect: true,
									},
								},
							},
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "www.example.com",
						Domains: domains("www.example.com"),
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routecluster("default/kuard/8080/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
			},
		},
		"ingress with websocket annotation": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
						Annotations: map[string]string{
							"enroute.saaras.io/websocket-routes": "/ws1 , /ws2",
						},
					},
					Spec: v1beta1.IngressSpec{
						Rules: []v1beta1.IngressRule{{
							Host: "www.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Path: "/",
										Backend: v1beta1.IngressBackend{
											ServiceName: "kuard",
											ServicePort: intstr.FromString("www"),
										},
									}, {
										Path: "/ws1",
										Backend: v1beta1.IngressBackend{
											ServiceName: "kuard",
											ServicePort: intstr.FromString("www"),
										},
									}},
								},
							},
						}},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:       "www",
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "www.example.com",
						Domains: domains("www.example.com"),
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/ws1"),
							Action:              websocketroute("default/kuard/8080/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}, {
							Match:               envoy.RouteMatch("/"),
							Action:              routecluster("default/kuard/8080/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"ingress invalid timeout": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
						Annotations: map[string]string{
							"enroute.saaras.io/request-timeout": "heptio",
						},
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromInt(8080),
						},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "*",
						Domains: []string{"*"},
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routetimeout("default/kuard/8080/da39a3ee5e", 0),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"ingress infinite timeout": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
						Annotations: map[string]string{
							"enroute.saaras.io/request-timeout": "infinity",
						},
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromInt(8080),
						},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "*",
						Domains: []string{"*"},
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routetimeout("default/kuard/8080/da39a3ee5e", 0),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"ingress 90 second timeout": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
						Annotations: map[string]string{
							"enroute.saaras.io/request-timeout": "1m30s",
						},
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromInt(8080),
						},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "*",
						Domains: []string{"*"},
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routetimeout("default/kuard/8080/da39a3ee5e", 90*time.Second),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"vhost name exceeds 60 chars": { // saarasio/enroute#25
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-service-name",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						Rules: []v1beta1.IngressRule{{
							Host: "my-very-very-long-service-host-name.subdomain.boring-dept.my.company",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Path: "/",
										Backend: v1beta1.IngressBackend{
											ServiceName: "kuard",
											ServicePort: intstr.FromString("www"),
										},
									}},
								},
							},
						}},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:       "www",
							Protocol:   "TCP",
							Port:       80,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "d31bb322ca62bb395acad00b3cbf45a3aa1010ca28dca7cddb4f7db786fa",
						Domains: domains("my-very-very-long-service-host-name.subdomain.boring-dept.my.company"),
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routecluster("default/kuard/80/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"Ingress: empty ingress class": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "incorrect",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromInt(8080),
						},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "*",
						Domains: []string{"*"},
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routecluster("default/kuard/8080/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"Ingress: explicit kubernetes.io/ingress.class": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "incorrect",
						Namespace: "default",
						Annotations: map[string]string{
							"kubernetes.io/ingress.class": new(ResourceEventHandler).ingressClass(),
						},
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromInt(8080),
						},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "*",
						Domains: []string{"*"},
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routecluster("default/kuard/8080/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"Ingress: explicit enroute.saaras.io/ingress.class": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "incorrect",
						Namespace: "default",
						Annotations: map[string]string{
							"enroute.saaras.io/ingress.class": new(ResourceEventHandler).ingressClass(),
						},
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromInt(8080),
						},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "*",
						Domains: []string{"*"},
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routecluster("default/kuard/8080/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"GatewayHost: empty ingress annotation": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "www.example.com",
						},
						Routes: []gatewayhostv1.Route{{
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/",
							}},
							Services: []gatewayhostv1.Service{
								{
									Name: "kuard",
									Port: 8080,
								},
							},
						}},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "www.example.com",
						Domains: domains("www.example.com"),
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routecluster("default/kuard/8080/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"GatewayHost: explicit enroute.saaras.io/ingress.class": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
						Annotations: map[string]string{
							"enroute.saaras.io/ingress.class": new(ResourceEventHandler).ingressClass(),
						},
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "www.example.com",
						},
						Routes: []gatewayhostv1.Route{{
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/",
							}},
							Services: []gatewayhostv1.Service{
								{
									Name: "kuard",
									Port: 8080,
								},
							},
						}},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "www.example.com",
						Domains: domains("www.example.com"),
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routecluster("default/kuard/8080/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"GatewayHost: explicit kubernetes.io/ingress.class": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
						Annotations: map[string]string{
							"kubernetes.io/ingress.class": new(ResourceEventHandler).ingressClass(),
						},
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "www.example.com",
						},
						Routes: []gatewayhostv1.Route{{
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/",
							}},
							Services: []gatewayhostv1.Service{
								{
									Name: "kuard",
									Port: 8080,
								},
							},
						}},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "www.example.com",
						Domains: domains("www.example.com"),
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routecluster("default/kuard/8080/da39a3ee5e"),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"ingress retry-on": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
						Annotations: map[string]string{
							"enroute.saaras.io/retry-on": "5xx,gateway-error",
						},
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromInt(8080),
						},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "*",
						Domains: []string{"*"},
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routeretry("default/kuard/8080/da39a3ee5e", "5xx,gateway-error", 0, 0),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"ingress retry-on, num-retries": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
						Annotations: map[string]string{
							"enroute.saaras.io/retry-on":    "5xx,gateway-error",
							"enroute.saaras.io/num-retries": "7", // not five or six or eight, but seven.
						},
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromInt(8080),
						},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "*",
						Domains: []string{"*"},
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routeretry("default/kuard/8080/da39a3ee5e", "5xx,gateway-error", 7, 0),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"ingress retry-on, per-try-timeout": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
						Annotations: map[string]string{
							"enroute.saaras.io/retry-on":        "5xx,gateway-error",
							"enroute.saaras.io/per-try-timeout": "150ms",
						},
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "kuard",
							ServicePort: intstr.FromInt(8080),
						},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "*",
						Domains: []string{"*"},
						Routes: []*envoy_api_v2_route.Route{{
							Match:               envoy.RouteMatch("/"),
							Action:              routeretry("default/kuard/8080/da39a3ee5e", "5xx,gateway-error", 0, 150*time.Millisecond),
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"gatewayhost no weights defined": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "www.example.com",
						},
						Routes: []gatewayhostv1.Route{{
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/",
							}},
							Services: []gatewayhostv1.Service{
								{
									Name: "backend",
									Port: 80,
								},
								{
									Name: "backendtwo",
									Port: 80,
								},
							},
						}},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "backend",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       80,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "backendtwo",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       80,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "www.example.com",
						Domains: domains("www.example.com"),
						Routes: []*envoy_api_v2_route.Route{{
							Match: envoy.RouteMatch("/"),
							Action: &envoy_api_v2_route.Route_Route{
								Route: &envoy_api_v2_route.RouteAction{
									ClusterSpecifier: &envoy_api_v2_route.RouteAction_WeightedClusters{
										WeightedClusters: &envoy_api_v2_route.WeightedCluster{
											Clusters: weightedClusters(
												weightedCluster("default/backend/80/da39a3ee5e", 1),
												weightedCluster("default/backendtwo/80/da39a3ee5e", 1),
											),
											TotalWeight: protobuf.UInt32(2),
										},
									},
								},
							},
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"gatewayhost one weight defined": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "www.example.com",
						},
						Routes: []gatewayhostv1.Route{{
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/",
							}},
							Services: []gatewayhostv1.Service{
								{
									Name: "backend",
									Port: 80,
								},
								{
									Name:   "backendtwo",
									Port:   80,
									Weight: 50,
								},
							},
						}},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "backend",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       80,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "backendtwo",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       80,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "www.example.com",
						Domains: domains("www.example.com"),
						Routes: []*envoy_api_v2_route.Route{{
							Match: envoy.RouteMatch("/"),
							Action: &envoy_api_v2_route.Route_Route{
								Route: &envoy_api_v2_route.RouteAction{
									ClusterSpecifier: &envoy_api_v2_route.RouteAction_WeightedClusters{
										WeightedClusters: &envoy_api_v2_route.WeightedCluster{
											Clusters: weightedClusters(
												weightedCluster("default/backend/80/da39a3ee5e", 0),
												weightedCluster("default/backendtwo/80/da39a3ee5e", 50),
											),
											TotalWeight: protobuf.UInt32(50),
										},
									},
								},
							},
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"gatewayhost all weights defined": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{
							Fqdn: "www.example.com",
						},
						Routes: []gatewayhostv1.Route{{
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/",
							}},
							Services: []gatewayhostv1.Service{
								{
									Name:   "backend",
									Port:   80,
									Weight: 22,
								},
								{
									Name:   "backendtwo",
									Port:   80,
									Weight: 50,
								},
							},
						}},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "backend",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       80,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "backendtwo",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       80,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http",
					VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
						Name:    "www.example.com",
						Domains: domains("www.example.com"),
						Routes: []*envoy_api_v2_route.Route{{
							Match: envoy.RouteMatch("/"),
							Action: &envoy_api_v2_route.Route_Route{
								Route: &envoy_api_v2_route.RouteAction{
									ClusterSpecifier: &envoy_api_v2_route.RouteAction_WeightedClusters{
										WeightedClusters: &envoy_api_v2_route.WeightedCluster{
											Clusters: weightedClusters(
												weightedCluster("default/backend/80/da39a3ee5e", 22),
												weightedCluster("default/backendtwo/80/da39a3ee5e", 50),
											),
											TotalWeight: protobuf.UInt32(72),
										},
									},
								},
							},
							RequestHeadersToAdd: envoy.RouteHeaders(),
						}},
					}},
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
		},
		"gatewayhost w/ missing fqdn": {
			objs: []interface{}{
				&gatewayhostv1.GatewayHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: gatewayhostv1.GatewayHostSpec{
						VirtualHost: &gatewayhostv1.VirtualHost{},
						Routes: []gatewayhostv1.Route{{
							Conditions: []gatewayhostv1.Condition{{
								Prefix: "/",
							}},
							Services: []gatewayhostv1.Service{
								{
									Name: "backend",
									Port: 80,
								},
							},
						}},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "backend",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Protocol:   "TCP",
							Port:       80,
							TargetPort: intstr.FromInt(8080),
						}},
					},
				},
			},
			want: map[string]*v2.RouteConfiguration{
				"ingress_http": {
					Name: "ingress_http", // should be blank, no fqdn defined.
				},
				"ingress_https": {
					Name: "ingress_https",
				},
			},
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
			got := visitRoutes(root)
			assert.Equal(t, tc.want, got)
		})
	}
}

func domains(hostname string) []string {
	return []string{hostname, hostname + ":*"}
}

func routecluster(cluster string) *envoy_api_v2_route.Route_Route {
	return &envoy_api_v2_route.Route_Route{
		Route: &envoy_api_v2_route.RouteAction{
			ClusterSpecifier: &envoy_api_v2_route.RouteAction_Cluster{
				Cluster: cluster,
			},
		},
	}

}

func websocketroute(c string) *envoy_api_v2_route.Route_Route {
	r := routecluster(c)
	r.Route.UpgradeConfigs = append(r.Route.UpgradeConfigs,
		&envoy_api_v2_route.RouteAction_UpgradeConfig{
			UpgradeType: "websocket",
		},
	)
	return r
}

func routetimeout(cluster string, timeout time.Duration) *envoy_api_v2_route.Route_Route {
	r := routecluster(cluster)
	r.Route.Timeout = protobuf.Duration(timeout)
	return r
}

func routeretry(cluster string, retryOn string, numRetries uint32, perTryTimeout time.Duration) *envoy_api_v2_route.Route_Route {
	r := routecluster(cluster)
	r.Route.RetryPolicy = &envoy_api_v2_route.RetryPolicy{
		RetryOn: retryOn,
	}
	if numRetries > 0 {
		r.Route.RetryPolicy.NumRetries = protobuf.UInt32(numRetries)
	}
	if perTryTimeout > 0 {
		r.Route.RetryPolicy.PerTryTimeout = protobuf.Duration(perTryTimeout)
	}
	return r
}

func weightedClusters(first, second *envoy_api_v2_route.WeightedCluster_ClusterWeight, rest ...*envoy_api_v2_route.WeightedCluster_ClusterWeight) []*envoy_api_v2_route.WeightedCluster_ClusterWeight {
	return append([]*envoy_api_v2_route.WeightedCluster_ClusterWeight{first, second}, rest...)
}

func weightedCluster(name string, weight uint32) *envoy_api_v2_route.WeightedCluster_ClusterWeight {
	return &envoy_api_v2_route.WeightedCluster_ClusterWeight{
		Name:   name,
		Weight: protobuf.UInt32(weight),
	}
}
