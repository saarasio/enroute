// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package envoy

import (
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	httplua "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/lua/v2"
	httprl "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/rate_limit/v2"
	http "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	envoy_config_ratelimit_v2 "github.com/envoyproxy/go-control-plane/envoy/config/ratelimit/v2"
    "github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	cfg "github.com/saarasio/enroute/enroute-dp/saarasconfig"
	types "github.com/golang/protobuf/ptypes"
)

func httpRateLimitTypedConfig(vh *dag.Vertex) *http.HttpFilter_TypedConfig {
	return &http.HttpFilter_TypedConfig{
		TypedConfig: toAny(&httprl.RateLimit{
			Domain: "enroute",
			RateLimitService: &envoy_config_ratelimit_v2.RateLimitServiceConfig{
				GrpcService: &envoy_api_v2_core.GrpcService{
					TargetSpecifier: &envoy_api_v2_core.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &envoy_api_v2_core.GrpcService_EnvoyGrpc{
							ClusterName: "enroute_ratelimit",
						},
					},
				},
			},
		}),
	}
}

//- name: envoy.lua
//            typed_config:
//              "@type": type.googleapis.com/envoy.config.filter.http.lua.v2.Lua
//              inline_code: |
//                local mylibrary = require("lib.mylibrary")
//
//                function envoy_on_request(request_handle)
//                  request_handle:headers():add("foo", mylibrary.foobar())
//                end
//                function envoy_on_response(response_handle)
//                  body_size = response_handle:body():length()
//                  response_handle:headers():add("response-body-size", tostring(body_size))
//                end

var unusedLuaExampleCode = `
 function envoy_on_response(response_handle)
     body_size = response_handle:body():length()
     body_size = 100
     response_handle:headers():add("response-body-size", tostring(body_size))
   end
`

func getHttpFilterConfigIfPresent(filter_type string, v *dag.Vertex) *cfg.SaarasRouteFilter {
	// TODO: Get the HTTP filters on the listener

	var http_filters *dag.HttpFilter

	if v == nil {
		return nil
	}

	switch vh := (*v).(type) {
	case *dag.VirtualHost:
		http_filters = vh.HttpFilters
	case *dag.SecureVirtualHost:
		http_filters = vh.VirtualHost.HttpFilters
	default:
		// not interesting
	}

	if http_filters != nil {
		if len(http_filters.Filters) > 0 {
			for _, one_http_filter := range http_filters.Filters {
				if one_http_filter.Filter_type == filter_type {
					return one_http_filter
				}
			}
		}
	}
	return nil
}

func addLuaFilterConfigIfPresent(http_filters *[]*http.HttpFilter, v *dag.Vertex) {
	lua_filter := getHttpFilterConfigIfPresent(cfg.FILTER_TYPE_HTTP_LUA, v)

	if lua_filter != nil {
		*http_filters = append(*http_filters,
			&http.HttpFilter{
				Name:       "envoy.lua",
				ConfigType: httpLuaTypedConfig(lua_filter),
			})
	}
}

func addRateLimitFilterConfigIfPresent(http_filters *[]*http.HttpFilter, vh *dag.Vertex) {
	// TODO: Check all routes to determine if rate-limit is configured with any of them
	//rl_filter := getVirtualHostFilterConfigIfPresent(cfg.FILTER_TYPE_VH_RATELIMIT, vh)

	//if rl_filter != nil {
	*http_filters = append(*http_filters,
		&http.HttpFilter{
			//Name:       "envoy.rate_limit",
			Name:       wellknown.HTTPRateLimit,
			ConfigType: httpRateLimitTypedConfig(vh),
		})
	//}
}

func httpFilters(vh *dag.Vertex) []*http.HttpFilter {

	http_filters := make([]*http.HttpFilter, 0)

	addLuaFilterConfigIfPresent(&http_filters, vh)

	http_filters = append(http_filters,
		&http.HttpFilter{
			Name:       wellknown.Gzip,
			ConfigType: nil,
		})

	http_filters = append(http_filters,
		&http.HttpFilter{
			Name:       wellknown.GRPCWeb,
			ConfigType: nil,
		})

	// TODO:
	// If any of the routes have a rate-limit policy, add the envoy.rate_limit
	// filter here.
	addRateLimitFilterConfigIfPresent(&http_filters, vh)

	http_filters = append(http_filters,
		&http.HttpFilter{
			Name:       wellknown.Router,
			ConfigType: nil,
		})

	return http_filters

	//return []*http.HttpFilter{{
	//    Name:       wellknown.Gzip,
	//    ConfigType: nil,
	//}, {
	//    Name:       wellknown.GRPCWeb,
	//    ConfigType: nil,
	//}, {
	//    //  TODO
	//    //  go-control-plane defines this constant as "envoy.ratelimit"
	//    //  While we run this with envoy 1.12, "envoy.ratelimit" is not recognized
	//    //  However "envoy.rate_limit" is recognized
	//    //  Name: wellknown.RateLimit,
	//    Name:       "envoy.rate_limit",
	//    ConfigType: httpRateLimitTypedConfig(vh),
	//}, {
	//    Name:       wellknown.Router,
	//    ConfigType: nil,
	//}}
}

type ListenerFilterInfo struct {
	FilterName     string
	FilterLocation string
}

func httpLuaTypedConfig(filter_config *cfg.SaarasRouteFilter) *http.HttpFilter_TypedConfig {
	return &http.HttpFilter_TypedConfig{
		TypedConfig: toAny(&httplua.Lua{
			InlineCode: filter_config.Filter_config,
			// TODO: Remove
			// InlineCode: luaInlineCode,
		}),
	}
}

func dagFilterToHttpFilter(df *cfg.SaarasRouteFilter) *http.HttpFilter {
	if df != nil {
		switch df.Filter_type {
		case cfg.FILTER_TYPE_HTTP_LUA:
			lua_http_filter := &http.HttpFilter{
				Name:       "envoy.lua",
				ConfigType: httpLuaTypedConfig(df),
			}
			return lua_http_filter
			// case cfg.FILTER_TYPE_HTTP_RATELIMIT:
		default:
		}
	}

	return nil
}

func buildHttpFilterMap(listener_filters *[]*http.HttpFilter, dag_http_filters *dag.HttpFilter,
	m *map[string]*http.HttpFilter) {
	for _, hf := range *listener_filters {
		(*m)[hf.Name] = hf
	}

	if dag_http_filters != nil {
		for _, df := range dag_http_filters.Filters {
			hf := dagFilterToHttpFilter(df)
			if hf != nil {
				(*m)[hf.Name] = hf
			}
		}
	}
}

func updateHttpFilters(listener_filters *[]*http.HttpFilter,
	dag_http_filters *dag.HttpFilter) []*http.HttpFilter {

	http_filters := make([]*http.HttpFilter, 0)

	var m map[string]*http.HttpFilter

	m = make(map[string]*http.HttpFilter)

	// Aggregate HttpFilter from listener and dag, store them in the map
	buildHttpFilterMap(listener_filters, dag_http_filters, &m)

	// Correctly order the HttpFilters from the map constructed in previous step

	// fmt.Printf("updateHttpFilters(): Filters to install \n[%+v]\n", m)

	// Lua
	if hf, ok := m["envoy.lua"]; ok {
		// fmt.Printf("updateHttpFilters() Adding LUA filter\n")
		http_filters = append(http_filters, hf)
	}

	// Gzip
	if hf, ok := m[wellknown.Gzip]; ok {
		http_filters = append(http_filters, hf)
	}

	// GRPCWeb
	if hf, ok := m[wellknown.GRPCWeb]; ok {
		http_filters = append(http_filters, hf)
	}

	// Rate Limit
	if hf, ok := m["envoy.rate_limit"]; ok {
		http_filters = append(http_filters, hf)
	}

	// Router
	if hf, ok := m[wellknown.Router]; ok {
		http_filters = append(http_filters, hf)
	}

	// fmt.Printf("updateHttpFilters() Http Filters to install [%+v]\n", http_filters)

	return http_filters
}

func Find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

func remove(slice []*envoy_api_v2_listener.Filter, s int) []*envoy_api_v2_listener.Filter {
	return append(slice[:s], slice[s+1:]...)
}

func AddHttpFilterToListener(l *v2.Listener, dag_filters *dag.HttpFilter, name string) {
	if l != nil && l.FilterChains != nil {
		var done bool = false
		for _, filterchain := range l.FilterChains {
			var found bool = false

			// For SNI listener, find the correct Filter Chain
			if filterchain.FilterChainMatch != nil && filterchain.FilterChainMatch.ServerNames != nil {
				_, found = Find(filterchain.FilterChainMatch.ServerNames, name)
				if found {
					for idx, one_filter := range filterchain.Filters {
						// Find HTTPConnectionManager filter and update HttpFilters on it
						if one_filter.Name == wellknown.HTTPConnectionManager {
							httpConnManagerConfig := &http.HttpConnectionManager{}
							if config := one_filter.GetTypedConfig(); config != nil {
								types.UnmarshalAny(config, httpConnManagerConfig)
								httpConnManagerConfig.HttpFilters =
									updateHttpFilters(&httpConnManagerConfig.HttpFilters, dag_filters)
								one_filter.ConfigType = &envoy_api_v2_listener.Filter_TypedConfig{
									TypedConfig: toAny(httpConnManagerConfig),
								}

								// Delete one_filter from slice and add it back again
								filterchain.Filters = remove(filterchain.Filters, idx)
								filterchain.Filters = append(filterchain.Filters, one_filter)

								done = true
							}
						}
					}
				}
			}
		}

		// No SNI matching listener found
		// This is a non-SNI listener,
		if !done {
			for _, filterchain := range l.FilterChains {
				for idx, one_filter := range filterchain.Filters {
					// Find HTTPConnectionManager filter and update HttpFilters on it
					if one_filter.Name == wellknown.HTTPConnectionManager {
						httpConnManagerConfig := &http.HttpConnectionManager{}
						if config := one_filter.GetTypedConfig(); config != nil {
							types.UnmarshalAny(config, httpConnManagerConfig)
							httpConnManagerConfig.HttpFilters =
								updateHttpFilters(&httpConnManagerConfig.HttpFilters, dag_filters)
							one_filter.ConfigType = &envoy_api_v2_listener.Filter_TypedConfig{
								TypedConfig: toAny(httpConnManagerConfig),
							}

							// Delete one_filter from slice and add it back again
							filterchain.Filters = remove(filterchain.Filters, idx)
							filterchain.Filters = append(filterchain.Filters, one_filter)

							done = true
						}
					}
				}
			}
		}
	}
}
