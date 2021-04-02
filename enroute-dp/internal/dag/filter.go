// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package dag

import (
)

func GetVHHttpFilterConfigIfPresent(filter_type string, v *VirtualHost) *HttpFilter {
	var http_filters []*HttpFilter

	if v == nil {
		return nil
	}

	http_filters = v.HttpFilters

	if http_filters != nil {
		if len(http_filters) > 0 {
			for _, one_http_filter := range http_filters {
				if one_http_filter.Filter_type == filter_type {
					return one_http_filter
				}
			}
		}
	}
	return nil
}
