// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package envoy

import (
	envoy_extensions_filters_http_lua_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/lua/v3"
	envoy_extensions_filters_network_http_connection_manager_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/logger"
	cfg "github.com/saarasio/enroute/enroute-dp/saarasconfig"
)

func addLuaFilterConfigIfPresent(http_filters *[]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter, v *dag.VirtualHost) {
	lua_filter := dag.GetVHHttpFilterConfigIfPresent(cfg.FILTER_TYPE_HTTP_LUA, v)

	// We pass a nil value for VirtualHost below for httpLuaTypedConfig()
	// This is OK since during initialization, we need to set things up
	// Later when we process the filters, this value will be updated correctly.
	if lua_filter != nil {
		if logger.EL.ELogger != nil {
			logger.EL.ELogger.Debugf("internal:envoy:listener_filter_lua:addLuaFilterConfigIfPresent() Adding Lua Filter [%s]\n", lua_filter.Filter.Filter_name)
		}

		*http_filters = append(*http_filters,
			&envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter{
				Name:       "envoy.lua",
				ConfigType: httpLuaTypedConfig(lua_filter, nil),
			})
	}
}

func httpLuaTypedConfig(f *dag.HttpFilter, vh *dag.VirtualHost) *envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter_TypedConfig {

	// Aggregate config for Route and Http Lua filter config

	if vh != nil {
		// VH Lua
		lua_http_filter := dag.GetVHHttpFilterConfigIfPresent(cfg.FILTER_TYPE_HTTP_LUA, vh)

		var l_cfg envoy_extensions_filters_http_lua_v3.Lua

		if lua_http_filter != nil {
			if logger.EL.ELogger != nil {
				logger.EL.ELogger.Debugf("envoy:lua:httpLuaTypedConfig() Found VH Lua [%+s]\n",
					lua_http_filter.Filter.Filter_name)
			}
			l_cfg.InlineCode = lua_http_filter.Filter.Filter_config
		}

		lua_typed_config := &envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter_TypedConfig{
			TypedConfig: toAny(&l_cfg),
		}

		return lua_typed_config
	}

	return nil
}
