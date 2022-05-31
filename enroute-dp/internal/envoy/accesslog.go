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

package envoy

import (
	envoy_config_accesslog_v3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	envoy_extensions_access_loggers_file_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
)

// FileAccessLog returns a new file based access log filter.
func FileAccessLog(path string) []*envoy_config_accesslog_v3.AccessLog {
	return []*envoy_config_accesslog_v3.AccessLog{{
		Name: wellknown.FileAccessLog,
		ConfigType: &envoy_config_accesslog_v3.AccessLog_TypedConfig{
			TypedConfig: toAny(&envoy_extensions_access_loggers_file_v3.FileAccessLog{
				Path: path,
				// TODO(dfc) FileAccessLog_Format elided.
			}),
		},
	}}
}
