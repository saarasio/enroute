// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2021 Saaras Inc.

package envoy

import (
	"testing"

	"github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"

	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/saarasio/enroute/enroute-dp/internal/assert"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	cfg "github.com/saarasio/enroute/enroute-dp/saarasconfig"
	"github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/lua/v3"

)



func TestAddLuaHTTPRouteFilter(t *testing.T) {

	var luaCode = `
    function envoy_on_request(request_handle)
       request_handle:logInfo("Hello World request");
    end
    
    function envoy_on_response(response_handle)
       response_handle:logInfo("Hello World response");
    end
    `

	filter_in := &envoy_config_listener_v3.Filter{
		Name: wellknown.HTTPConnectionManager,
		ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
			TypedConfig: toAny(&envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager{
				HttpFilters: []*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter{
					{
						Name:       wellknown.Gzip,
						ConfigType: nil,
					},

					{
						Name:       wellknown.GRPCWeb,
						ConfigType: nil,
					},

					{
						Name:       wellknown.Router,
						ConfigType: nil,
					},
				},
			}),
		},
	}

	dag_route_filters_in := []*dag.RouteFilter{
		{
			Filter: dag.Filter{
				Filter_name:   "luaroutefilter",
				Filter_type:   cfg.FILTER_TYPE_RT_LUA,
				Filter_config: luaCode,
			},
		},
	}

	sc := make(map[string]*envoy_config_core_v3.DataSource)
	sc[dag_route_filters_in[0].Filter.Filter_name] = &envoy_config_core_v3.DataSource{
		Specifier: &envoy_config_core_v3.DataSource_InlineString{
			InlineString: dag_route_filters_in[0].Filter.Filter_config,
		},
	}

	filter_out := &envoy_config_listener_v3.Filter{
		Name: wellknown.HTTPConnectionManager,
		ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
			TypedConfig: toAny(&envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager{
				HttpFilters: []*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter{
					{
						Name: "envoy.lua",
						ConfigType: &envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter_TypedConfig{
							TypedConfig: toAny(&envoy_extensions_filters_http_lua_v3.Lua{
								InlineCode:  luaCode,
								SourceCodes: sc,
							}),
						},
					},
					{
						Name:       wellknown.Gzip,
						ConfigType: nil,
					},

					{
						Name:       wellknown.GRPCWeb,
						ConfigType: nil,
					},

					{
						Name:       wellknown.Router,
						ConfigType: nil,
					},
				},
			}),
		},
	}

	tests := map[string]struct {
		l          envoy_config_listener_v3.Listener
		dag_routes map[string]*dag.Route
		name       string
		want       envoy_config_listener_v3.Listener
	}{

		"Lua filter on a VirtualHost Route": {
			l: envoy_config_listener_v3.Listener{
				FilterChains: FilterChains(filter_in),
			},
			dag_routes: map[string]*dag.Route{
				"r": {RouteFilters: dag_route_filters_in},
			},
			name: "",
			want: envoy_config_listener_v3.Listener{
				FilterChains: FilterChains(filter_out),
				// AccessLog:    FileAccessLog("/dev/stdout"),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			vh := dag.VirtualHost{}
			vh.Routes = tc.dag_routes
			AddHttpFilterToListener(&tc.l, &vh, tc.name)
			assert.Equal(t, tc.want, tc.l)
		})
	}
}
