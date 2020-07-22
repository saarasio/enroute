// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.
package config

type EnrouteConfig struct {
	EnrouteCtlUUID string `json:"enroutectluuid,omitempty"`
	Data           Data   `json:"data"`
}
type Globalconfig struct {
	Config           string `json:"config,omitempty"`
	GlobalconfigName string `json:"globalconfig_name,omitempty"`
	GlobalconfigType string `json:"globalconfig_type,omitempty"`
}
type ProxyGlobalconfigs struct {
	Globalconfig Globalconfig `json:"globalconfig,omitempty"`
}
type Filter struct {
	FilterName   string `json:"filter_name,omitempty"`
	FilterType   string `json:"filter_type,omitempty"`
	FilterConfig string `json:"filter_config,omitempty"`
}
type ServiceFilters struct {
	Filter Filter `json:"filter"`
}
type RouteFilters struct {
	Filter Filter `json:"filter"`
}
type Upstream struct {
	UpstreamConfig                    string `json:"upstream_config,omitempty"`
	UpstreamHcHealthythresholdcount   string `json:"upstream_hc_healthythresholdcount,omitempty"`
	UpstreamHcHost                    string `json:"upstream_hc_host,omitempty"`
	UpstreamHcIntervalseconds         string `json:"upstream_hc_intervalseconds,omitempty"`
	UpstreamHcPath                    string `json:"upstream_hc_path,omitempty"`
	UpstreamHcTimeoutseconds          string `json:"upstream_hc_timeoutseconds,omitempty"`
	UpstreamHcUnhealthythresholdcount string `json:"upstream_hc_unhealthythresholdcount,omitempty"`
	UpstreamID                        int    `json:"upstream_id,omitempty"`
	UpstreamIP                        string `json:"upstream_ip,omitempty"`
	UpstreamName                      string `json:"upstream_name,omitempty"`
	UpstreamPort                      int    `json:"upstream_port,omitempty"`
	UpstreamProtocol                  string `json:"upstream_protocol,omitempty"`
	UpstreamStrategy                  string `json:"upstream_strategy,omitempty"`
	UpstreamValidationCacertificate   string `json:"upstream_validation_cacertificate,omitempty"`
	UpstreamValidationSubjectname     string `json:"upstream_validation_subjectname,omitempty"`
	UpstreamWeight                    int    `json:"upstream_weight,omitempty"`
}
type RouteUpstreams struct {
	Upstream Upstream `json:"upstream"`
}
type Routes struct {
	RouteConfig    string           `json:"route_config,omitempty"`
	RouteName      string           `json:"route_name,omitempty"`
	RoutePrefix    string           `json:"route_prefix,omitempty"`
	RouteFilters   []RouteFilters   `json:"route_filters,omitempty"`
	RouteUpstreams []RouteUpstreams `json:"route_upstreams,omitempty"`
}

// TODO
type ServiceSecret struct {
}

type Service struct {
	Fqdn           string           `json:"fqdn,omitempty"`
	ServiceConfig  string           `json:"service_config,omitempty"`
	ServiceName    string           `json:"service_name,omitempty"`
	ServiceFilters []ServiceFilters `json:"service_filters,omitempty"`
	Secret         []ServiceSecret  `json:"secret,omitempty"`
	Routes         []Routes         `json:"routes,omitempty"`
}
type ProxyServices struct {
	Service Service `json:"service"`
}
type SaarasDbProxy struct {
	ProxyConfig        string               `json:"proxy_config,omitempty"`
	ProxyName          string               `json:"proxy_name,omitempty"`
	ProxyGlobalconfigs []ProxyGlobalconfigs `json:"proxy_globalconfigs,omitempty"`
	ProxyServices      []ProxyServices      `json:"proxy_services,omitempty"`
}
type Data struct {
	SaarasDbProxy []SaarasDbProxy `json:"saaras_db_proxy,omitempty"`
}

type RoutesByName []Routes

func (l RoutesByName) Len() int      { return len(l) }
func (l RoutesByName) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l RoutesByName) Less(i, j int) bool {
	return l[i].RouteName < l[j].RouteName
}

type ServicesByName []ProxyServices

func (l ServicesByName) Len() int      { return len(l) }
func (l ServicesByName) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l ServicesByName) Less(i, j int) bool {
	return l[i].Service.ServiceName < l[j].Service.ServiceName
}
