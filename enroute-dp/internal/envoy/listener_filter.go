// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package envoy

import (
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_extensions_filters_network_http_connection_manager_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	types "github.com/golang/protobuf/ptypes"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
)

type ListenerFilterInfo struct {
	FilterName     string
	FilterLocation string
}

func dagVirtualHost(vh *dag.Vertex) *dag.VirtualHost {
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

	return vhost
}

func buildHttpFilterMap(listener_filters *[]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter, dag_http_filters []*dag.HttpFilter,
	vh *dag.VirtualHost, m *map[string]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter) {

	for _, hf := range *listener_filters {
		(*m)[hf.Name] = hf
	}

	if dag_http_filters != nil {
		for _, df := range dag_http_filters {
			hf := DagFilterToHttpFilter(df, vh)
			if hf != nil {
				(*m)[hf.Name] = hf
			}
		}
	}
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

func AddHttpFilterToListener(l *envoy_config_listener_v3.Listener, vh *dag.VirtualHost, name string) {
	var dag_filters []*dag.HttpFilter

	dag_filters = vh.HttpFilters
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
									updateHttpVHFilters(l, &httpConnManagerConfig.HttpFilters, dag_filters, vh)
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
								updateHttpVHFilters(l, &httpConnManagerConfig.HttpFilters, dag_filters, vh)
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
