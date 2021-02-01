// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package envoy

import (
	"github.com/envoyproxy/go-control-plane/envoy/config/core/v3"

	"github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/envoy/config/ratelimit/v3"
	"github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/lua/v3"
	"github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ratelimit/v3"
	"github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	types "github.com/golang/protobuf/ptypes"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	cfg "github.com/saarasio/enroute/enroute-dp/saarasconfig"
)

func httpRateLimitTypedConfig(vh dag.Vertex) *envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter_TypedConfig {
	return &envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter_TypedConfig{
		TypedConfig: toAny(&envoy_extensions_filters_http_ratelimit_v3.RateLimit{
			Domain: "enroute",
			RateLimitService: &envoy_config_ratelimit_v3.RateLimitServiceConfig{
				GrpcService: &envoy_config_core_v3.GrpcService{
					TargetSpecifier: &envoy_config_core_v3.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &envoy_config_core_v3.GrpcService_EnvoyGrpc{
							ClusterName: "enroute_ratelimit",
						},
					},
				},
			},
		}),
	}
}

func getVHHttpFilterConfigIfPresent(filter_type string, v *dag.Vertex) *cfg.SaarasRouteFilter {
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

func addLuaFilterConfigIfPresent(http_filters *[]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter, v *dag.Vertex) {
	lua_filter := getVHHttpFilterConfigIfPresent(cfg.FILTER_TYPE_HTTP_LUA, v)

	if lua_filter != nil {
		*http_filters = append(*http_filters,
			&envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter{
				Name:       "envoy.lua",
				ConfigType: httpLuaTypedConfig(lua_filter),
			})
	}
}

func routeHasRateLimitFilter(routes map[string]*dag.Route) bool {
	for _, r := range routes {
		if r.RouteFilters != nil {
			for _, rf := range r.RouteFilters.Filters {
				if rf.Filter_type == cfg.FILTER_TYPE_RT_RATELIMIT {
					return true
				}
			}
		}
	}
	return false
}

func addRateLimitFilterConfigIfPresent(http_filters *[]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter, v dag.Vertex) {
	if v == nil {
		return
	}

	has := false

	switch vh := v.(type) {
	case *dag.VirtualHost:
		routes := vh.GetVirtualHostRoutes()
		has = routeHasRateLimitFilter(routes)
	case *dag.SecureVirtualHost:
		routes := vh.VirtualHost.GetVirtualHostRoutes()
		has = routeHasRateLimitFilter(routes)
	default:
		// not interesting
	}

	if has {
		*http_filters = append(*http_filters,
			&envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter{
				Name:       wellknown.HTTPRateLimit,
				ConfigType: httpRateLimitTypedConfig(v),
			})
	}
}

func httpFilters(vh *dag.Vertex) []*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter {

	http_filters := make([]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter, 0)

	if vh != nil {
		addLuaFilterConfigIfPresent(&http_filters, vh)
	}

	http_filters = append(http_filters,
		&envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter{
			Name:       wellknown.Gzip,
			ConfigType: nil,
		})

	http_filters = append(http_filters,
		&envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter{
			Name:       wellknown.GRPCWeb,
			ConfigType: nil,
		})

	if vh != nil {
		addRateLimitFilterConfigIfPresent(&http_filters, *vh)
	}

	http_filters = append(http_filters,
		&envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter{
			Name:       wellknown.Router,
			ConfigType: nil,
		})

	return http_filters
}

type ListenerFilterInfo struct {
	FilterName     string
	FilterLocation string
}

func httpLuaTypedConfig(filter_config *cfg.SaarasRouteFilter) *envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter_TypedConfig {
	return &envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter_TypedConfig{
		TypedConfig: toAny(&envoy_extensions_filters_http_lua_v3.Lua{
			InlineCode: filter_config.Filter_config,
			// TODO: Remove
			// InlineCode: luaInlineCode,
		}),
	}
}

func dagFilterToHttpFilter(df *cfg.SaarasRouteFilter) *envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter {
	if df != nil {
		switch df.Filter_type {
		case cfg.FILTER_TYPE_HTTP_LUA:
			lua_http_filter := &envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter{
				Name:       "envoy.lua",
				ConfigType: httpLuaTypedConfig(df),
			}
			return lua_http_filter
		default:
		}
	}

	return nil
}

func buildHttpFilterMap(listener_filters *[]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter, dag_http_filters *dag.HttpFilter,
	m *map[string]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter) {
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

func updateHttpVHFilters(listener_filters *[]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter,
	dag_http_filters *dag.HttpFilter) []*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter {

	http_filters := make([]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter, 0)

	var m map[string]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter

	m = make(map[string]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter)

	// Aggregate HttpFilter from listener and dag, store them in the map
	buildHttpFilterMap(listener_filters, dag_http_filters, &m)

	// Correctly order the HttpFilters from the map constructed in previous step

	// Lua
	if hf, ok := m[wellknown.Lua]; ok {
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
	if hf, ok := m[wellknown.HTTPRateLimit]; ok {
		http_filters = append(http_filters, hf)
	}

	// Router
	if hf, ok := m[wellknown.Router]; ok {
		http_filters = append(http_filters, hf)
	}

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

func remove(slice []*envoy_config_listener_v3.Filter, s int) []*envoy_config_listener_v3.Filter {
	return append(slice[:s], slice[s+1:]...)
}

func AddHttpFilterToListener(l *envoy_config_listener_v3.Listener, dag_filters *dag.HttpFilter, name string) {
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
							httpConnManagerConfig := &envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager{}
							if config := one_filter.GetTypedConfig(); config != nil {
								types.UnmarshalAny(config, httpConnManagerConfig)
								httpConnManagerConfig.HttpFilters =
									updateHttpVHFilters(&httpConnManagerConfig.HttpFilters, dag_filters)
								one_filter.ConfigType = &envoy_config_listener_v3.Filter_TypedConfig{
									TypedConfig: toAny(httpConnManagerConfig),
								}

								// Update modified filter
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
						httpConnManagerConfig := &envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager{}
						if config := one_filter.GetTypedConfig(); config != nil {
							types.UnmarshalAny(config, httpConnManagerConfig)
							httpConnManagerConfig.HttpFilters =
								updateHttpVHFilters(&httpConnManagerConfig.HttpFilters, dag_filters)
							one_filter.ConfigType = &envoy_config_listener_v3.Filter_TypedConfig{
								TypedConfig: toAny(httpConnManagerConfig),
							}

							// update modified filter
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
