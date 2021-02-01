// Copyright © 2019 VMware
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
	"github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/saarasio/enroute/enroute-dp/internal/protobuf"
)

// maxRegexProgramSize is the default value for the Envoy regex max
// program size tunable. There's no way to really know what a good
// value for this is, except that the RE2 maintainer thinks that 100
// is low. As a rule of thumb, each '.*' expression costs about 15
// units of program size. AFAIK, there's no obvious correlation
// between regex size and execution time.
//
// https://github.com/envoyproxy/envoy/pull/9171#discussion_r351974033
const maxRegexProgramSize = 1000

// SafeRegexMatch retruns a envoy_type_matcher_v3.RegexMatcher for the supplied regex.
// SafeRegexMatch does not escape regex meta characters.
func SafeRegexMatch(regex string) *envoy_type_matcher_v3.RegexMatcher {
	return &envoy_type_matcher_v3.RegexMatcher{
		EngineType: &envoy_type_matcher_v3.RegexMatcher_GoogleRe2{
			GoogleRe2: &envoy_type_matcher_v3.RegexMatcher_GoogleRE2{
				MaxProgramSize: protobuf.UInt32(maxRegexProgramSize),
			},
		},
		Regex: regex,
	}
}
