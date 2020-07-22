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
	"time"

	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/protobuf"
)

const (
	// Default healthcheck / lb algorithm values
	hcTimeout            = 2 * time.Second
	hcInterval           = 10 * time.Second
	hcUnhealthyThreshold = 3
	hcHealthyThreshold   = 2
	hcHost               = "contour-envoy-healthcheck"
)

// healthCheck returns a *envoy_api_v2_core.HealthCheck value.
func healthCheck(cluster *dag.Cluster) *envoy_api_v2_core.HealthCheck {
	hc := cluster.HealthCheck
	host := hcHost
	if hc.Host != "" {
		host = hc.Host
	}

	//// https://golang.org/pkg/time/#Duration
	//To convert an integer number of units to a Duration, multiply:

	//seconds := 10
	//fmt.Print(time.Duration(seconds)*time.Second) // prints 10s

	timeoutSecondsDuration := time.Duration(hc.TimeoutSeconds) * time.Second
	intervalSecondsDuration := time.Duration(hc.IntervalSeconds) * time.Second

	// TODO(dfc) why do we need to specify our own default, what is the default
	// that envoy applies if these fields are left nil?
	return &envoy_api_v2_core.HealthCheck{
		Timeout:            durationOrDefault(timeoutSecondsDuration, hcTimeout),
		Interval:           durationOrDefault(intervalSecondsDuration, hcInterval),
		UnhealthyThreshold: countOrDefault(hc.UnhealthyThresholdCount, hcUnhealthyThreshold),
		HealthyThreshold:   countOrDefault(hc.HealthyThresholdCount, hcHealthyThreshold),
		HealthChecker: &envoy_api_v2_core.HealthCheck_HttpHealthCheck_{
			HttpHealthCheck: &envoy_api_v2_core.HealthCheck_HttpHealthCheck{
				Path: hc.Path,
				Host: host,
			},
		},
	}
}

func durationOrDefault(d, def time.Duration) *duration.Duration {
	if d != 0 {
		return protobuf.Duration(d)
	}
	return protobuf.Duration(def)
}

func countOrDefault(count uint32, def uint32) *wrappers.UInt32Value {
	switch count {
	case 0:
		return protobuf.UInt32(def)
	default:
		return protobuf.UInt32(count)
	}
}
