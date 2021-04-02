// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package contour

import (
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/envoy"
	"github.com/saarasio/enroute/enroute-dp/internal/logger"
)

func (v *listenerVisitor) updateListener(name string, vh *dag.VirtualHost) {

	if logger.EL.ELogger != nil {
		logger.EL.ELogger.Debugf("contour:updateListener() [%s] [%+v]\n", name, vh)
	}

	// populateTestLuaFilter2(vh)
	if vh.HttpFilters != nil {
		if len(vh.HttpFilters) > 0 {
			listener := v.listeners[name]
			envoy.AddHttpFilterToListener(listener, vh, vh.Name)
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
