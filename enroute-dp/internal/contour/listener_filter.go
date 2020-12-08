// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package contour

import (
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/envoy"
)

func (v *listenerVisitor) updateListener(name string, vh *dag.VirtualHost) {
	if vh.HttpFilters != nil {
		if len(vh.HttpFilters.Filters) > 0 {
			listener := v.listeners[name]
			envoy.AddHttpFilterToListener(listener, vh.HttpFilters, vh.Name)
		}
	}
}

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
