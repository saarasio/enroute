// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.
// +build !e,!c

package saarasfilters

import (
	_ "github.com/davecgh/go-spew/spew"
	"github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	_ "github.com/saarasio/enroute/enroute-dp/internal/logger"
	_ "github.com/sirupsen/logrus"
)

func SequenceFilters(m *map[string]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter) []*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter {
	http_filters := make([]*envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter, 0)

	// Lua
	if hf, ok := (*m)["envoy.lua"]; ok {
		http_filters = append(http_filters, hf)
	}

	// Gzip
	if hf, ok := (*m)[wellknown.Gzip]; ok {
		http_filters = append(http_filters, hf)
	}

	// GRPCWeb
	if hf, ok := (*m)[wellknown.GRPCWeb]; ok {
		http_filters = append(http_filters, hf)
	}

	// Cors
	if hf, ok := (*m)["envoy.cors"]; ok {
		http_filters = append(http_filters, hf)
	}

	// Router
	if hf, ok := (*m)[wellknown.Router]; ok {
		http_filters = append(http_filters, hf)
	}

	return http_filters
}
