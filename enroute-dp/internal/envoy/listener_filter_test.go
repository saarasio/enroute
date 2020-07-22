package envoy

import (
	"testing"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	httplua "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/lua/v2"
	http "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
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

	filter_in := &envoy_api_v2_listener.Filter{
		Name: wellknown.HTTPConnectionManager,
		ConfigType: &envoy_api_v2_listener.Filter_TypedConfig{
			TypedConfig: toAny(&http.HttpConnectionManager{
				HttpFilters: []*http.HttpFilter{
					&http.HttpFilter{
						Name:       wellknown.Gzip,
						ConfigType: nil,
					},

					&http.HttpFilter{
						Name:       wellknown.GRPCWeb,
						ConfigType: nil,
					},

					&http.HttpFilter{
						Name:       wellknown.Router,
						ConfigType: nil,
					},
				},
			}),
		},
	}
	dag_filters_in := []*cfg.SaarasRouteFilter{
		&cfg.SaarasRouteFilter{
			Filter_type:   cfg.FILTER_TYPE_HTTP_LUA,
			Filter_config: luaCode,
		},
	}

	filter_out := &envoy_api_v2_listener.Filter{
		Name: wellknown.HTTPConnectionManager,
		ConfigType: &envoy_api_v2_listener.Filter_TypedConfig{
			TypedConfig: toAny(&http.HttpConnectionManager{
				HttpFilters: []*http.HttpFilter{
					&http.HttpFilter{
						Name: wellknown.Lua,
						ConfigType: &http.HttpFilter_TypedConfig{
							TypedConfig: toAny(&httplua.Lua{
								InlineCode: luaCode,
							}),
						},
					},
					&http.HttpFilter{
						Name:       wellknown.Gzip,
						ConfigType: nil,
					},

					&http.HttpFilter{
						Name:       wellknown.GRPCWeb,
						ConfigType: nil,
					},

					&http.HttpFilter{
						Name:       wellknown.Router,
						ConfigType: nil,
					},
				},
			}),
		},
	}

	tests := map[string]struct {
		l           v2.Listener
		dag_filters dag.HttpFilter
		name        string
		want        v2.Listener
	}{
		"No listener No filter": {
			l:           v2.Listener{},
			dag_filters: dag.HttpFilter{},
			want:        v2.Listener{},
		},

		"Lua filter in DAG Filters": {
			l: v2.Listener{
				FilterChains: FilterChains(filter_in),
			},
			dag_filters: dag.HttpFilter{
				Filters: dag_filters_in,
			},
			name: "",
			want: v2.Listener{
				FilterChains: FilterChains(filter_out),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			AddHttpFilterToListener(&tc.l, &tc.dag_filters, tc.name)
			got := tc.l
			assert.Equal(t, tc.want, got)
		})
	}
}
