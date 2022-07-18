// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.
//go:build !e && !c
// +build !e,!c

package envoy

import (
	envoy_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_compressor_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/compressor/v3"
	envoy_extensions_filters_http_cors_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	envoy_extensions_filters_network_http_connection_manager_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	http "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	cfg "github.com/saarasio/enroute/enroute-dp/saarasconfig"
	"github.com/saarasio/enroute/enroute-dp/saarasfilters"
)

func DagFilterToHttpFilter(df *dag.HttpFilter, vh *dag.VirtualHost) *envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter {
	if df != nil {
		switch df.Filter_type {
		case cfg.FILTER_TYPE_HTTP_LUA:
			lua_http_filter := &envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter{
				Name:       "envoy.lua",
				ConfigType: httpLuaTypedConfig(df, vh),
			}
			return lua_http_filter
		case cfg.FILTER_TYPE_VH_CORS:
			cors_http_filter := &envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter{
				Name:       "envoy.cors",
				ConfigType: httpCorsTypedConfig(df, vh),
			}
			return cors_http_filter

		default:
		}
	}

	return nil
}

func httpCorsTypedConfig(f *dag.HttpFilter, vh *dag.VirtualHost) *envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter_TypedConfig {

	if vh != nil {
		// VH Csrf
		// cors_http_filter := dag.GetVHHttpFilterConfigIfPresent(cfg.FILTER_TYPE_HTTP_CORS, vh)

		var c_cfg envoy_extensions_filters_http_cors_v3.Cors

		typed_config := &envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter_TypedConfig{
			TypedConfig: toAny(&c_cfg),
		}

		return typed_config
	}

	return nil
}

func updateHttpVHFilters(l *envoy_config_listener_v3.Listener, listener_filters *[]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter,
	dag_http_filters []*dag.HttpFilter, vh *dag.VirtualHost) []*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter {

	var m map[string]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter

	m = make(map[string]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter)

	// Aggregate HttpFilter from listener and dag, store them in the map
	buildHttpFilterMap(listener_filters, dag_http_filters, vh, &m)

	// Correctly order the HttpFilters from the map constructed in previous step

	http_filters := saarasfilters.SequenceFilters(&m)

	return http_filters
}

func httpFilters(vh *dag.Vertex) []*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter {

	var vhost *dag.VirtualHost

	if vh != nil {
		switch v := (*vh).(type) {
		case *dag.VirtualHost:
			vhost = v
		case *dag.SecureVirtualHost:
			vhost = &v.VirtualHost
		default:
			// not interesting
		}
	}

	http_filters := make([]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter, 0)

	if vh != nil {
		addLuaFilterConfigIfPresent(&http_filters, vhost)
	}

	http_filters = append(http_filters,
		&envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter{
			Name: "compressor",
			ConfigType: &http.HttpFilter_TypedConfig{
				TypedConfig: toAny(&envoy_compressor_v3.Compressor{
					CompressorLibrary: &envoy_core_v3.TypedExtensionConfig{
						Name: "gzip",
						TypedConfig: &any.Any{
							TypeUrl: cfg.HTTPFilterCompressorGzip,
						},
					},
				}),
			},
		})

	http_filters = append(http_filters,
		&envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter{

			Name: "grpcweb",
			ConfigType: &http.HttpFilter_TypedConfig{
				TypedConfig: &any.Any{
					TypeUrl: cfg.HTTPFilterGrpcWeb,
				},
			},
		})

	http_filters = append(http_filters,
		&envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter{
			Name: "router",
			ConfigType: &http.HttpFilter_TypedConfig{
				TypedConfig: &any.Any{
					TypeUrl: cfg.HTTPFilterRouter,
				},
			},
		})

	return http_filters
}

/////////////////////// VH Filters //////////////////////////////

func TypedFilterConfig(vh *dag.VirtualHost) map[string]*any.Any {
       var tfc map[string]*any.Any

       return tfc
}
