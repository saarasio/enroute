package saarasconfig

import (
	"encoding/json"
	"github.com/pkg/errors"
	"strings"
)

type SaarasMicroService2 struct {
	Upstream SaarasUpstream `json:"upstream"`
}

type SaarasUpstream struct {
	Upstream_name                       string `json:"upstream_name"`
	Upstream_ip                         string `json:"upstream_ip"`
	Upstream_port                       int32  `json:"upstream_port"`
	Upstream_weight                     int32  `json:"upstream_weight"`
	Upstream_hc_path                    string `json:"upstream_hc_path"`
	Upstream_hc_host                    string `json:"upstream_hc_host"`
	Upstream_hc_intervalseconds         int64  `json:"upstream_hc_intervalseconds"`
	Upstream_hc_timeoutseconds          int64  `json:"upstream_hc_timeoutseconds"`
	Upstream_hc_unhealthythresholdcount uint32 `json:"upstream_hc_unhealthythresholdcount"`
	Upstream_hc_healthythresholdcount   uint32 `json:"upstream_hc_healthythresholdcount"`
	Upstream_strategy                   string `json:"upstream_strategy"`
	Upstream_validation_cacertificate   string `json:"upstream_validation_cacertificate"`
	Upstream_validation_subjectname     string `json:"upstream_validation_subjectname"`
	Upstream_protocol                   string `json:"upstream_protocol"`
	Create_ts                           string `json:"create_ts"`
	Update_ts                           string `json:"update_ts"`
}

type SaarasRouteFilter struct {
	Filter_name   string `json:"filter_name"`
	Filter_type   string `json:"filter_type"`
	Filter_config string `json:"filter_config"`
}

const FILTER_TYPE_HTTP_LUA string = "http_filter_lua"
const FILTER_TYPE_HTTP_CORS string = "http_filter_cors"
const FILTER_TYPE_RT_LUA string = "route_filter_lua"
const FILTER_TYPE_HTTP_RATELIMIT string = "http_filter_ratelimit"
const FILTER_TYPE_RT_RATELIMIT string = "route_filter_ratelimit"
const FILTER_TYPE_HTTP_JWT string = "http_filter_jwt"
const FILTER_TYPE_HTTP_ACCESSLOG string = "http_filter_accesslog"
const FILTER_TYPE_RT_CIRCUITBREAKERS string = "route_filter_circuitbreakers"

const PROXY_CONFIG_RATELIMIT string = "globalconfig_ratelimit"
const PROXY_CONFIG_ACCESSLOG string = "globalconfig_accesslog"

const JAEGER_TRACING_CLUSTER string = "jaeger-trace"
const EDS_CONFIG_CLUSTER string = "contour"

// https://github.com/envoyproxy/envoy/blob/2eb63590357ffe9689256184fde74eda5d21a648/api/envoy/config/route/v3/route_components.proto#L510
type CorsMatchCondition struct {
	// +optional
	Exact string `json:"exact,omitempty"`

	// +optional
	Prefix string `json:"prefix,omitempty"`

	// +optional
	Suffix string `json:"suffix,omitempty"`

	// +optional
	Contains string `json:"contains,omitempty"`

	// +optional
	Regex string `json:"regex,omitempty"`
}

type CorsFilterConfig struct {

	// +optional
	MatchCondition CorsMatchCondition `json:"match_condition,omitempty"`

	// +optional
	AccessControlAllowMethods string `json:"access_control_allow_methods,omitempty"`

	// +optional
	AccessControlAllowHeaders string `json:"access_control_allow_headers,omitempty"`

	// +optional
	AccessControlExposeHeaders string `json:"access_control_expose_headers,omitempty"`

	// +optional
	AccessControlMaxAge string `json:"access_control_max_age,omitempty"`
}

func UnmarshalCorsFilterConfig(filter_config string) (*CorsFilterConfig, error) {
	var gr CorsFilterConfig
	var err error

	// filter_config = test_cors_config

	buf := strings.NewReader(filter_config)
	if err = json.NewDecoder(buf).Decode(&gr); err != nil {
		errors.Wrap(err, "decoding response")
	}

	return &gr, err
}

type GenericKeyType struct {
	DescriptorValue string `json:"descriptor_value,omitempty"`
}

type RequestHeadersType struct {
	HeaderName    string `json:"header_name,omitempty"`
	DescriptorKey string `json:"descriptor_key,omitempty"`
}

type Descriptors struct {
	GenericKey         *GenericKeyType     `json:"generic_key,omitempty"`
	RequestHeaders     *RequestHeadersType `json:"request_headers,omitempty"`
	SourceCluster      string              `json:"source_cluster,omitempty"`
	DestinationCluster string              `json:"destination_cluster,omitempty"`
	RemoteAddress      string              `json:"remote_address,omitempty"`
}

type RouteActionDescriptors struct {
	Descriptors []Descriptors `json:"descriptors,omitempty"`
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

type RouteMatchConditions struct {
	Prefix          string                `json:"prefix"`
	MatchConditions []RouteMatchCondition `json:"header"`
}

type RouteMatchConditionsByHeaderNameVal []RouteMatchCondition

func (l RouteMatchConditionsByHeaderNameVal) Len() int      { return len(l) }
func (l RouteMatchConditionsByHeaderNameVal) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l RouteMatchConditionsByHeaderNameVal) Less(i, j int) bool {
	return l[i].Name+l[i].Exact < l[j].Name+l[j].Exact
}

type RouteMatchCondition struct {
	// Name is the name of the header to match on. Name is required.
	// Header names are case insensitive.
	Name string `json:"name"`

	// Present is true if the Header is present in the request.
	// +optional
	Present bool `json:"present,omitempty"`

	// Contains is true if the Header containing this string is present
	// in the request.
	// +optional
	Contains string `json:"contains,omitempty"`

	// NotContains is true if the Header containing this string is not present
	// in the request.
	// +optional
	NotContains string `json:"notcontains,omitempty"`

	// Exact is true if the Header containing this string matches exactly
	// in the request.
	// +optional
	Exact string `json:"exact,omitempty"`

	// NotExact is true if the Header containing this string doesn't match exactly
	// in the request.
	// +optional
	NotExact string `json:"notexact,omitempty"`
}

func UnmarshalRouteMatchCondition(route_config string) (RouteMatchConditions, error) {
	var mc RouteMatchConditions
	var err error

	buf := strings.NewReader(route_config)
	if err = json.NewDecoder(buf).Decode(&mc); err != nil {
		errors.Wrap(err, "decoding response")
	}

	return mc, err
}

type CircuitBreakerConfig struct {
	// Circuit breaking limits

	// Max connections is maximum number of connections
	// that Envoy will make to the upstream cluster.
	MaxConnections uint32 `json:"max_connections"`

	// MaxPendingRequests is maximum number of pending
	// requests that Envoy will allow to the upstream cluster.
	MaxPendingRequests uint32 `json:"max_pending_requests"`

	// MaxRequests is the maximum number of parallel requests that
	// Envoy will make to the upstream cluster.
	MaxRequests uint32 `json:"max_requests"`

	// MaxRetries is the maximum number of parallel retries that
	// Envoy will allow to the upstream cluster.
	MaxRetries uint32 `json:"max_retries"`
}

func UnmarshalCircuitBreakerconfig(cc_config string) (CircuitBreakerConfig, error) {
	var cbc CircuitBreakerConfig
	var err error

	buf := strings.NewReader(cc_config)
	if err = json.NewDecoder(buf).Decode(&cbc); err != nil {
		errors.Wrap(err, "decoding response")
	}

	return cbc, err
}
