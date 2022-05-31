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
	"time"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/google/go-cmp/cmp"
	gatewayhostv1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/protobuf"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestHealthCheck(t *testing.T) {
	tests := map[string]struct {
		cluster *dag.Cluster
		want    *envoy_config_core_v3.HealthCheck
	}{
		// this is an odd case because contour.edshealthcheck will not call envoy.HealthCheck
		// when hc is nil, so if hc is not nil, at least one of the parameters on it must be set.
		"blank healthcheck": {
			cluster: &dag.Cluster{
				HealthCheck: new(gatewayhostv1.HealthCheck),
			},
			want: &envoy_config_core_v3.HealthCheck{
				Timeout:            protobuf.Duration(hcTimeout),
				Interval:           protobuf.Duration(hcInterval),
				UnhealthyThreshold: protobuf.UInt32(3),
				HealthyThreshold:   protobuf.UInt32(2),
				HealthChecker: &envoy_config_core_v3.HealthCheck_HttpHealthCheck_{
					HttpHealthCheck: &envoy_config_core_v3.HealthCheck_HttpHealthCheck{
						// TODO(dfc) this doesn't seem right
						Host: "contour-envoy-healthcheck",
					},
				},
			},
		},
		"healthcheck path only": {
			cluster: &dag.Cluster{
				HealthCheck: &gatewayhostv1.HealthCheck{
					Path: "/healthy",
				},
			},
			want: &envoy_config_core_v3.HealthCheck{
				Timeout:            protobuf.Duration(hcTimeout),
				Interval:           protobuf.Duration(hcInterval),
				UnhealthyThreshold: protobuf.UInt32(3),
				HealthyThreshold:   protobuf.UInt32(2),
				HealthChecker: &envoy_config_core_v3.HealthCheck_HttpHealthCheck_{
					HttpHealthCheck: &envoy_config_core_v3.HealthCheck_HttpHealthCheck{
						Path: "/healthy",
						Host: "contour-envoy-healthcheck",
					},
				},
			},
		},
		"explicit healthcheck": {
			cluster: &dag.Cluster{
				HealthCheck: &gatewayhostv1.HealthCheck{
					Host:                    "foo-bar-host",
					Path:                    "/healthy",
					TimeoutSeconds:          99,
					IntervalSeconds:         98,
					UnhealthyThresholdCount: 97,
					HealthyThresholdCount:   96,
				},
			},
			want: &envoy_config_core_v3.HealthCheck{
				Timeout:            protobuf.Duration(99 * time.Second),
				Interval:           protobuf.Duration(98 * time.Second),
				UnhealthyThreshold: protobuf.UInt32(97),
				HealthyThreshold:   protobuf.UInt32(96),
				HealthChecker: &envoy_config_core_v3.HealthCheck_HttpHealthCheck_{
					HttpHealthCheck: &envoy_config_core_v3.HealthCheck_HttpHealthCheck{
						Path: "/healthy",
						Host: "foo-bar-host",
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := healthCheck(tc.cluster)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Fatal(diff)
			}

		})
	}
}
