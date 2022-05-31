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
	"testing"

	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestLBEndpoint(t *testing.T) {
	got := LBEndpoint(SocketAddress("microsoft.com", 81))
	want := &envoy_config_endpoint_v3.LbEndpoint{
		HostIdentifier: &envoy_config_endpoint_v3.LbEndpoint_Endpoint{
			Endpoint: &envoy_config_endpoint_v3.Endpoint{
				Address: SocketAddress("microsoft.com", 81),
			},
		},
	}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Fatal(diff)
	}
}

func TestEndpoints(t *testing.T) {
	got := Endpoints(
		SocketAddress("github.com", 443),
		SocketAddress("microsoft.com", 80),
	)
	want := []*envoy_config_endpoint_v3.LocalityLbEndpoints{{
		LbEndpoints: []*envoy_config_endpoint_v3.LbEndpoint{{
			HostIdentifier: &envoy_config_endpoint_v3.LbEndpoint_Endpoint{
				Endpoint: &envoy_config_endpoint_v3.Endpoint{
					Address: SocketAddress("github.com", 443),
				},
			},
		}, {
			HostIdentifier: &envoy_config_endpoint_v3.LbEndpoint_Endpoint{
				Endpoint: &envoy_config_endpoint_v3.Endpoint{
					Address: SocketAddress("microsoft.com", 80),
				},
			},
		}},
	}}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Fatal(diff)
	}
}

func TestClusterLoadAssignment(t *testing.T) {
	got := ClusterLoadAssignment("empty")
	want := &envoy_config_endpoint_v3.ClusterLoadAssignment{
		ClusterName: "empty",
	}

	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Fatal(diff)
	}

	got = ClusterLoadAssignment("one addr", SocketAddress("microsoft.com", 81))
	want = &envoy_config_endpoint_v3.ClusterLoadAssignment{
		ClusterName: "one addr",
		Endpoints:   Endpoints(SocketAddress("microsoft.com", 81)),
	}

	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Fatal(diff)
	}

	got = ClusterLoadAssignment("two addrs",
		SocketAddress("microsoft.com", 81),
		SocketAddress("github.com", 443),
	)
	want = &envoy_config_endpoint_v3.ClusterLoadAssignment{
		ClusterName: "two addrs",
		Endpoints: Endpoints(
			SocketAddress("microsoft.com", 81),
			SocketAddress("github.com", 443),
		),
	}

	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Fatal(diff)
	}
}
