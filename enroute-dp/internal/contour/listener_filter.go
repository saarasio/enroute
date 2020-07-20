// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package contour

import (
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/envoy"

	// "fmt"
	// cfg "github.com/saarasio/enroute/enroute-dp/saarasconfig"
)

// TODO: Move to test

//var luaInlineCode = `
//function envoy_on_request(request_handle)
//   request_handle:logInfo("Hello World request");
//end
//
//function envoy_on_response(response_handle)
//   response_handle:logInfo("Hello World response");
//end
//`
//
//func populateTestLuaFilter(v *dag.Vertex) {
//    if v == nil {
//        fmt.Printf("populateTestLuaFilter() Vertex nil [%+v]\n", v)
//        return
//    }
//
//	switch l := (*v).(type) {
//    case *dag.Listener:
//        if l.HttpFilters == nil {
//            l.HttpFilters = &dag.HttpFilter{}
//        }
//
//        if l.HttpFilters.Filters == nil {
//            l.HttpFilters.Filters = make([]*cfg.SaarasRouteFilter, 0)
//            l.HttpFilters.Filters = append(l.HttpFilters.Filters,
//            &cfg.SaarasRouteFilter{
//	            Filter_name:   "lua_test_filter",
//	            Filter_type:   cfg.FILTER_TYPE_HTTP_LUA,
//	            Filter_config: luaInlineCode,
//            })
//        }
//	default:
//		// not interesting
//	}
//}
//func populateTestLuaFilter2(vh *dag.VirtualHost) {
//	if vh == nil {
//		return
//	}
//
//	if vh.HttpFilters == nil {
//		vh.HttpFilters = &dag.HttpFilter{}
//	}
//	if vh.HttpFilters.Filters == nil {
//		vh.HttpFilters.Filters = make([]*cfg.SaarasRouteFilter, 0)
//		vh.HttpFilters.Filters = append(vh.HttpFilters.Filters,
//			&cfg.SaarasRouteFilter{
//				Filter_name:   "lua_test_filter",
//				Filter_type:   cfg.FILTER_TYPE_HTTP_LUA,
//				Filter_config: luaInlineCode,
//			})
//	}
//}

func (v *listenerVisitor) updateListener(name string, vh *dag.VirtualHost) {
	// populateTestLuaFilter2(vh)
	if vh.HttpFilters != nil {
		if len(vh.HttpFilters.Filters) > 0 {
			listener := v.listeners[name]
			envoy.AddHttpFilterToListener(listener, vh.HttpFilters, vh.Name)
		}
	}
}

///dag.Listener
///	dag.VirtualHost/envoy.Listener
///		// envoy.Listener.Address - hard coded to ENVOY_HTTP_LISTENER
///		// envoy.Listener.Address - has IP/Port - hard coded from listener visitor config
///		// envoy.Listener.FilterChain - -- CAN HAVE MULTIPLE OF THESE ON A LISTENER --
///			// envoy.Listener.FilterChain.FilterChainMatch - empty
///			// envoy.Listener.FilterChain.TLSContext - empty
///			// envoy.Listener.FilterChain.Filters
///		    	// * :ref:`envoy.client_ssl_auth<config_network_filters_client_ssl_auth>`
///		    	// * :ref:`envoy.echo <config_network_filters_echo>`
///		    	// * :ref:`envoy.http_connection_manager <config_http_conn_man>`
///		    		// * :ref:`envoy.http_connection_manager.HttpFilters
///		    			// * :ref:`envoy.http_connection_manager.HttpFilters.Gzip - hardcoded
///		    			// * :ref:`envoy.http_connection_manager.HttpFilters.GRPCWeb - hardcoded
///		    			// * :ref:`envoy.http_connection_manager.HttpFilters.Router - hardcoded
///		    		// * :ref:`envoy.http_connection_manager.HttpProtocolOptions
///		    		// * :ref:`envoy.http_connection_manager.AccessLog
///		    		// * :ref:`envoy.http_connection_manager.UseRemoteAddress
///		    		// * :ref:`envoy.http_connection_manager.NormalizePath
///		    		// * :ref:`envoy.http_connection_manager.IdleTimeout
///		    		// * :ref:`envoy.http_connection_manager.RequestTimeout
///		    		// * :ref:`envoy.http_connection_manager.PreserveExternalRequestId
///		    	// * :ref:`envoy.mongo_proxy <config_network_filters_mongo_proxy>`
///		    	// * :ref:`envoy.ratelimit <config_network_filters_rate_limit>`
///		    	// * :ref:`envoy.redis_proxy <config_network_filters_redis_proxy>`
///		    	// * :ref:`envoy.tcp_proxy <config_network_filters_tcp_proxy>`
///
///	dag.SecureVirtualHost/envoy.Listener
///		// envoy.Listener.Address - hard coded to ENVOY_HTTPS_LISTENER
///		// envoy.Listener.Address - has IP/Port - hard coded from listener visitor config
///		// envoy.Listener.FilterChain - -- CAN HAVE MULTIPLE OF THESE ON A LISTENER --
///			// envoy.Listener.FilterChain.FilterChainMatch - SNI - populated from SecureVirtualHost
///			// envoy.Listener.FilterChain.TLSContext - Certificate populated from SecureVirtualHost
///			// envoy.Listener.FilterChain.Filters
///		    	// * :ref:`envoy.client_ssl_auth<config_network_filters_client_ssl_auth>`
///		    	// * :ref:`envoy.echo <config_network_filters_echo>`
///		    	// * :ref:`envoy.http_connection_manager <config_http_conn_man>`
///		    		// * :ref:`envoy.http_connection_manager.HttpFilters`
///		    			// * :ref:`envoy.http_connection_manager.HttpFilters.Gzip - hardcoded
///		    			// * :ref:`envoy.http_connection_manager.HttpFilters.GRPCWeb - hardcoded
///		    			// * :ref:`envoy.http_connection_manager.HttpFilters.Router - hardcoded
///		    		// * :ref:`envoy.http_connection_manager.HttpProtocolOptions
///		    		// * :ref:`envoy.http_connection_manager.AccessLog
///		    		// * :ref:`envoy.http_connection_manager.UseRemoteAddress
///		    		// * :ref:`envoy.http_connection_manager.NormalizePath
///		    		// * :ref:`envoy.http_connection_manager.IdleTimeout
///		    		// * :ref:`envoy.http_connection_manager.RequestTimeout
///		    		// * :ref:`envoy.http_connection_manager.PreserveExternalRequestId
///		    	// * :ref:`envoy.mongo_proxy <config_network_filters_mongo_proxy>`
///		    	// * :ref:`envoy.mongo_proxy <config_network_filters_mongo_proxy>`
///		    	// * :ref:`envoy.ratelimit <config_network_filters_rate_limit>`
///		    	// * :ref:`envoy.redis_proxy <config_network_filters_redis_proxy>`
///		    	// * :ref:`envoy.tcp_proxy <config_network_filters_tcp_proxy>`
///

// Read HttpFilters configured on VirtualHost/SecureVirtualHost (DAG) and update filters on listeners
// Walk through all the VirtualHost and SecureVirtualHost for the listener
// For every VirtualHost, SecureVirtualHost determine the dag.HttpFilters
// Lookup Listener.FilterChain.Filters.HttpConnectionManager.HttpFilters corresponding to
// 	(Listener, VirtualHost) or (Listener, SecureVirtualHost)
// Update Listener.FilterChain.Filters.HttpConnectionManager.HttpFilters to install filters
// configured in dag.HttpFilters
func (v *listenerVisitor) setupHttpFilters(vertex dag.Vertex) {
	switch vh := vertex.(type) {
	case *dag.VirtualHost:
		if vh != nil {
			v.updateListener(ENVOY_HTTP_LISTENER, vh)
		}
	case *dag.SecureVirtualHost:
		if vh != nil {
			v.updateListener(ENVOY_HTTPS_LISTENER, &(vh.VirtualHost))
		}

	default:
		// recurse
		vertex.Visit(v.setupHttpFilters)
	}
}
