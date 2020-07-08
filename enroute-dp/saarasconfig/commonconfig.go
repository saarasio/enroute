package saarasconfig

import (
	"encoding/json"
	"github.com/pkg/errors"
	"strings"
)

type SaarasRouteFilter struct {
	Filter_name   string
	Filter_type   string
	Filter_config string
}

const FILTER_TYPE_HTTP_LUA string = "http_filter_lua"
const FILTER_TYPE_RT_LUA string = "route_filter_lua"
const FILTER_TYPE_HTTP_RATELIMIT string = "http_filter_ratelimit"
const FILTER_TYPE_RT_RATELIMIT string = "route_filter_ratelimit"

const PROXY_CONFIG_RATELIMIT string = "globalconfig_ratelimit"

const JAEGER_TRACING_CLUSTER string = "jaeger-trace"
const EDS_CONFIG_CLUSTER string = "contour"

type GenericKeyType struct {
	DescriptorValue string `json:"descriptor_value,omitempty"`
}

type RequestHeadersType struct {
	HeaderName    string `json:"header_name,omitempty"`
	DescriptorKey string `json:"descriptor_key,omitempty"`
}

type RouteActionDescriptors struct {
	Descriptors []struct {
		GenericKey         *GenericKeyType     `json:"generic_key,omitempty"`
		RequestHeaders     *RequestHeadersType `json:"request_headers,omitempty"`
		SourceCluster      string              `json:"source_cluster,omitempty"`
		DestinationCluster string              `json:"destination_cluster,omitempty"`
		RemoteAddress      string              `json:"remote_address,omitempty"`
	} `json:"descriptors,omitempty"`
}

func UnmarshalRateLimitRouteFilterConfig(filter_config string) (RouteActionDescriptors, error) {

	var gr RouteActionDescriptors
	var err error

	buf := strings.NewReader(filter_config)
	if err = json.NewDecoder(buf).Decode(&gr); err != nil {
		errors.Wrap(err, "decoding response")
	}

	return gr, err
}
