package envoy

import (
	"testing"

	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_extensions_filters_http_lua_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/lua/v3"
	envoy_extensions_filters_network_http_connection_manager_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"

	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/saarasio/enroute/enroute-dp/internal/assert"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	cfg "github.com/saarasio/enroute/enroute-dp/saarasconfig"
)

func TestAddLuaHTTPVHFilter(t *testing.T) {

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
						Name:       "compressor",
						ConfigType: nil,
					},

					{
						Name:       "grpcweb",
						ConfigType: nil,
					},

					{
						Name:       "router",
						ConfigType: nil,
					},
				},
			}),
		},
	}
	dag_filters_in := dag.Filter{
		Filter_type:   cfg.FILTER_TYPE_HTTP_LUA,
		Filter_config: luaCode,
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
								InlineCode: luaCode,
							}),
						},
					},
					{
						Name:       "compressor",
						ConfigType: nil,
					},

					{
						Name:       "grpcweb",
						ConfigType: nil,
					},

					{
						Name:       "router",
						ConfigType: nil,
					},
				},
			}),
		},
	}

	tests := map[string]struct {
		l           envoy_config_listener_v3.Listener
		dag_filters dag.HttpFilter
		name        string
		want        envoy_config_listener_v3.Listener
	}{
		"No listener No filter": {
			l:           envoy_config_listener_v3.Listener{},
			dag_filters: dag.HttpFilter{},
			want:        envoy_config_listener_v3.Listener{},
		},

		"Lua filter in DAG Filters": {
			l: envoy_config_listener_v3.Listener{
				FilterChains: FilterChains(filter_in),
			},
			dag_filters: dag.HttpFilter{
				Filter: dag_filters_in,
			},
			name: "",
			want: envoy_config_listener_v3.Listener{
				FilterChains: FilterChains(filter_out),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			vh := dag.VirtualHost{}
			vh.HttpFilters = []*dag.HttpFilter{
				&tc.dag_filters,
			}
			AddHttpFilterToListener(&tc.l, &vh, tc.name)
			got := tc.l
			assert.Equal(t, tc.want, got)
		})
	}
}
