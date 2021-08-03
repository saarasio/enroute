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

	gatewayhostv1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/metrics"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGatewayHostMetrics(t *testing.T) {
	// ir1 is a valid ingressroute
	ir1 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "example",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "home",
					Port: 8080,
				}},
			}, {
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/prefix",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name: "delegated",
				}},
			},
		},
	}

	// ir2 is invalid because it contains a service with negative port
	ir2 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "example",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "home",
					Port: -80,
				}},
			}},
		},
	}

	// ir3 is invalid because it lives outside the roots namespace
	ir3 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "finance",
			Name:      "example",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foobar",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "home",
					Port: 8080,
				}},
			}},
		},
	}

	// ir4 is invalid because its match prefix does not match its parent's (ir1)
	//ir4 := &gatewayhostv1.GatewayHost{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Namespace: "roots",
	//		Name:      "delegated",
	//	},
	//	Spec: gatewayhostv1.GatewayHostSpec{
	//		Routes: []gatewayhostv1.Route{{
	//			Match: "/doesnotmatch",
	//			Services: []gatewayhostv1.Service{{
	//				Name: "home",
	//				Port: 8080,
	//			}},
	//		}},
	//	},
	//}

	// ir6 is invalid because it delegates to itself, producing a cycle
	ir6 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "self",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name: "self",
				},
			}},
		},
	}

	// ir7 delegates to ir8, which is invalid because it delegates back to ir7
	ir7 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "parent",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name: "child",
				},
			}},
		},
	}

	ir8 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "child",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name: "parent",
				},
			}},
		},
	}

	// ir9 is invalid because it has a route that both delegates and has a list of services
	ir9 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "parent",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name: "child",
				},
				Services: []gatewayhostv1.Service{{
					Name: "kuard",
					Port: 8080,
				}},
			}},
		},
	}

	// ir10 delegates to ir11 and ir 12.
	ir10 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "parent",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name: "validChild",
				},
			}, {
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/bar",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name: "invalidChild",
				},
			}},
		},
	}

	ir11 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "validChild",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "foo",
					Port: 8080,
				}},
			}},
		},
	}

	// ir12 is invalid because it contains an invalid port
	ir12 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "invalidChild",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/bar",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "foo",
					Port: 12345678,
				}},
			}},
		},
	}

	// ir13 is invalid because it does not specify and FQDN
	ir13 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "parent",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Services: []gatewayhostv1.Service{{
					Name: "foo",
					Port: 8080,
				}},
			}},
		},
	}

	// ir14 delegates tp ir15 but it is invalid because it is missing fqdn
	ir14 := &gatewayhostv1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "invalidParent",
		},
		Spec: gatewayhostv1.GatewayHostSpec{
			VirtualHost: &gatewayhostv1.VirtualHost{},
			Routes: []gatewayhostv1.Route{{
				Conditions: []gatewayhostv1.Condition{{
					Prefix: "/foo",
				}},
				Delegate: &gatewayhostv1.Delegate{
					Name: "validChild",
				},
			}},
		},
	}

	s1 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "foo",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:     "http",
				Protocol: "TCP",
				Port:     12345678,
			}},
		},
	}

	s2 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "foo",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:     "http",
				Protocol: "TCP",
				Port:     8080,
			}},
		},
	}

	s3 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "roots",
			Name:      "home",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:     "http",
				Protocol: "TCP",
				Port:     8080,
			}},
		},
	}

	tests := map[string]struct {
		objs           []interface{}
		want           metrics.GatewayHostMetric
		rootNamespaces []string
	}{
		"valid ingressroute": {
			objs: []interface{}{ir1, s3},
			want: metrics.GatewayHostMetric{
				Invalid: map[metrics.Meta]int{},
				Valid: map[metrics.Meta]int{
					{Namespace: "roots", VHost: "example.com"}: 1,
				},
				Orphaned: map[metrics.Meta]int{},
				Root: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
				Total: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
			},
		},
		"invalid port in service": {
			objs: []interface{}{ir2},
			want: metrics.GatewayHostMetric{
				Invalid: map[metrics.Meta]int{
					{Namespace: "roots", VHost: "example.com"}: 1,
				},
				Valid:    map[metrics.Meta]int{},
				Orphaned: map[metrics.Meta]int{},
				Root: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
				Total: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
			},
		},
		"root ingressroute outside of roots namespace": {
			objs: []interface{}{ir3},
			want: metrics.GatewayHostMetric{
				Invalid: map[metrics.Meta]int{
					{Namespace: "finance"}: 1,
				},
				Valid:    map[metrics.Meta]int{},
				Orphaned: map[metrics.Meta]int{},
				Root: map[metrics.Meta]int{
					{Namespace: "finance"}: 1,
				},
				Total: map[metrics.Meta]int{
					{Namespace: "finance"}: 1,
				},
			},
			rootNamespaces: []string{"foo"},
		},
		//"delegated route's match prefix does not match parent's prefix": {
		//	objs: []interface{}{ir1, ir4, s3},
		//	want: metrics.GatewayHostMetric{
		//		Invalid: map[metrics.Meta]int{
		//			{Namespace: "roots", VHost: "example.com"}: 1,
		//		},
		//		Valid: map[metrics.Meta]int{
		//			{Namespace: "roots", VHost: "example.com"}: 1,
		//		},
		//		Orphaned: map[metrics.Meta]int{},
		//		Root: map[metrics.Meta]int{
		//			{Namespace: "roots"}: 1,
		//		},
		//		Total: map[metrics.Meta]int{
		//			{Namespace: "roots"}: 2,
		//		},
		//	},
		//},
		"root ingressroute does not specify FQDN": {
			objs: []interface{}{ir13},
			want: metrics.GatewayHostMetric{
				Invalid: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
				Valid:    map[metrics.Meta]int{},
				Orphaned: map[metrics.Meta]int{},
				Root: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
				Total: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
			},
		},
		"self-edge produces a cycle": {
			objs: []interface{}{ir6},
			want: metrics.GatewayHostMetric{
				Invalid: map[metrics.Meta]int{
					{Namespace: "roots", VHost: "example.com"}: 1,
				},
				Valid:    map[metrics.Meta]int{},
				Orphaned: map[metrics.Meta]int{},
				Root: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
				Total: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
			},
		},
		"child delegates to parent, producing a cycle": {
			objs: []interface{}{ir7, ir8},
			want: metrics.GatewayHostMetric{
				Invalid: map[metrics.Meta]int{
					{Namespace: "roots", VHost: "example.com"}: 1,
				},
				Valid: map[metrics.Meta]int{
					{Namespace: "roots", VHost: "example.com"}: 1,
				},
				Orphaned: map[metrics.Meta]int{},
				Root: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
				Total: map[metrics.Meta]int{
					{Namespace: "roots"}: 2,
				},
			},
		},
		"route has a list of services and also delegates": {
			objs: []interface{}{ir9},
			want: metrics.GatewayHostMetric{
				Invalid: map[metrics.Meta]int{
					{Namespace: "roots", VHost: "example.com"}: 1,
				},
				Valid:    map[metrics.Meta]int{},
				Orphaned: map[metrics.Meta]int{},
				Root: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
				Total: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
			},
		},
		"ingressroute is an orphaned route": {
			objs: []interface{}{ir8},
			want: metrics.GatewayHostMetric{
				Invalid: map[metrics.Meta]int{},
				Valid:   map[metrics.Meta]int{},
				Orphaned: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
				Root: map[metrics.Meta]int{},
				Total: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
			},
		},
		"ingressroute delegates to multiple gatewayhosts, one is invalid": {
			objs: []interface{}{ir10, ir11, ir12, s1, s2},
			want: metrics.GatewayHostMetric{
				Invalid: map[metrics.Meta]int{
					{Namespace: "roots", VHost: "example.com"}: 1,
				},
				Valid: map[metrics.Meta]int{
					{Namespace: "roots", VHost: "example.com"}: 2,
				},
				Orphaned: map[metrics.Meta]int{},
				Root: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
				Total: map[metrics.Meta]int{
					{Namespace: "roots"}: 3,
				},
			},
		},
		"invalid parent orphans children": {
			objs: []interface{}{ir14, ir11},
			want: metrics.GatewayHostMetric{
				Invalid: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
				Valid: map[metrics.Meta]int{},
				Orphaned: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
				Root: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
				Total: map[metrics.Meta]int{
					{Namespace: "roots"}: 2,
				},
			},
		},
		"multi-parent children is not orphaned when one of the parents is invalid": {
			objs: []interface{}{ir14, ir11, ir10, s2},
			want: metrics.GatewayHostMetric{
				Invalid: map[metrics.Meta]int{
					{Namespace: "roots"}: 1,
				},
				Valid: map[metrics.Meta]int{
					{Namespace: "roots", VHost: "example.com"}: 2,
				},
				Orphaned: map[metrics.Meta]int{},
				Root: map[metrics.Meta]int{
					{Namespace: "roots"}: 2,
				},
				Total: map[metrics.Meta]int{
					{Namespace: "roots"}: 3,
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			kc := &dag.KubernetesCache{
				GatewayHostRootNamespaces: tc.rootNamespaces,
			}
			for _, o := range tc.objs {
				kc.Insert(o)
			}
			dag := dag.BuildDAG(kc)
			gotMetrics := calculateGatewayHostMetric(dag)
			if !reflect.DeepEqual(tc.want.Root, gotMetrics.Root) {
				t.Fatalf("(metrics-Root) expected to find: %v but got: %v", tc.want.Root, gotMetrics.Root)
			}
			if !reflect.DeepEqual(tc.want.Valid, gotMetrics.Valid) {
				t.Fatalf("(metrics-Valid) expected to find: %v but got: %v", tc.want.Valid, gotMetrics.Valid)
			}
			if !reflect.DeepEqual(tc.want.Invalid, gotMetrics.Invalid) {
				t.Fatalf("(metrics-Invalid) expected to find: %v but got: %v", tc.want.Invalid, gotMetrics.Invalid)
			}
			if !reflect.DeepEqual(tc.want.Orphaned, gotMetrics.Orphaned) {
				t.Fatalf("(metrics-Orphaned) expected to find: %v but got: %v", tc.want.Orphaned, gotMetrics.Orphaned)
			}
			if !reflect.DeepEqual(tc.want.Total, gotMetrics.Total) {
				t.Fatalf("(metrics-Total) expected to find: %v but got: %v", tc.want.Total, gotMetrics.Total)
			}
		})
	}
}
