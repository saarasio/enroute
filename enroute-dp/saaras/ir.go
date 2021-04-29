// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package saaras

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/saarasio/enroute/enroute-dp/apis/enroute/v1beta1"
	"github.com/saarasio/enroute/enroute-dp/internal/logger"
	cfg "github.com/saarasio/enroute/enroute-dp/saarasconfig"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
	//"strconv"
	"net"
	"strings"
)

const QGatewayHost string = `

query get_services_by_proxy($proxy_name: String!) {
  saaras_db_proxy_service(where: {proxy: {proxy_name: {_eq: $proxy_name}}}) {
    proxy {
      proxy_globalconfigs {
        globalconfig {
          globalconfig_type
          globalconfig_name
          config
        }
      }
    }
    service {
      service_id
      service_name
      fqdn
      create_ts
      update_ts
      routes {
        route_name
        route_prefix
        route_config
        create_ts
        update_ts
        route_upstreams {
          upstream {
            upstream_name
            upstream_ip
            upstream_port
            upstream_weight
            upstream_hc_path
            upstream_hc_host
            upstream_hc_intervalseconds
            upstream_hc_timeoutseconds
            upstream_hc_unhealthythresholdcount
            upstream_hc_healthythresholdcount
            upstream_strategy
            upstream_validation_cacertificate
            upstream_validation_subjectname
            upstream_protocol
            create_ts
            update_ts
          }
        }
        route_filters {
          filter {
            filter_id
            filter_name
            filter_type
            filter_config
            filter_upstreams {
              upstream {
                    upstream_name
                    upstream_ip
                    upstream_port
                    upstream_weight
                    upstream_hc_path
                    upstream_hc_host
                    upstream_hc_intervalseconds
                    upstream_hc_timeoutseconds
                    upstream_hc_unhealthythresholdcount
                    upstream_hc_healthythresholdcount
                    upstream_strategy
                    upstream_validation_cacertificate
                    upstream_validation_subjectname
                    upstream_protocol
                    create_ts
                    update_ts
              }
            }
            create_ts
            update_ts
          }
        }
      }
      service_filters {
        filter {
          filter_id
          filter_name
          filter_type
          filter_config
          filter_upstreams {
              upstream {
                upstream_name
                    upstream_ip
                    upstream_port
                    upstream_weight
                    upstream_hc_path
                    upstream_hc_host
                    upstream_hc_intervalseconds
                    upstream_hc_timeoutseconds
                    upstream_hc_unhealthythresholdcount
                    upstream_hc_healthythresholdcount
                    upstream_strategy
                    upstream_validation_cacertificate
                    upstream_validation_subjectname
                    upstream_protocol
                    create_ts
                    update_ts
              }
            }
          create_ts
          update_ts
        }
      }
      service_secrets {
        secret {
          secret_id
          secret_name
          secret_key
          secret_sni
          secret_cert
          create_ts
          update_ts
          artifacts {
            artifact_id
            artifact_name
            artifact_type
            artifact_value
          }
        }
      }
    }
  }
}
`

const QUpstreams string = `
query GetAllUpstreams {
  saaras_db_upstream {
    upstream_name
    upstream_ip
    upstream_port
    upstream_protocol
    upstream_strategy
    upstream_validation_cacertificate
    upstream_validation_subjectname
    upstream_weight
    upstream_config
    upstream_hc_healthythresholdcount
    upstream_hc_host
    upstream_hc_intervalseconds
    upstream_hc_path
    upstream_hc_timeoutseconds
    upstream_hc_unhealthythresholdcount
    upstream_id
  }
}
`

type ConfiguredUpstreams struct {
	Data DataUpstreams `json:"data"`
}

type DataUpstreams struct {
	SaarasUpstreams []cfg.SaarasUpstream `json:"saaras_db_upstream"`
}

type Filter struct {
	FilterID         int                       `json:"filter_id"`
	Filter_name      string                    `json:"filter_name"`
	Filter_type      string                    `json:"filter_type"`
	Filter_config    string                    `json:"filter_config"`
	Filter_upstreams []cfg.SaarasMicroService2 `json:"filter_upstreams"`
}

type RouteFilters struct {
	Filter Filter `json:"filter"`
}

type SaarasRoute2 struct {
	Route_name      string                    `json:"route_name"`
	Route_prefix    string                    `json:"route_prefix"`
	Route_config    string                    `json:"route_config"`
	Create_ts       string                    `json:"create_ts"`
	Update_ts       string                    `json:"update_ts"`
	Route_upstreams []cfg.SaarasMicroService2 `json:"route_upstreams"`
	Route_filters   []RouteFilters            `json:"route_filters"`
}

type SaarasArtifact struct {
	Artifact_id    int64
	Artifact_name  string
	Artifact_type  string
	Artifact_value string
}

type SaarasSecret struct {
	Secret_id   int64
	Secret_name string
	Secret_key  string
	Secret_cert string
	Secret_sni  string
	Artifacts   []SaarasArtifact
	Create_ts   string
	Update_ts   string
}

type SaarasSecrets struct {
	Secret SaarasSecret
}

type ServiceFilters struct {
	Filter Filter `json:"filter"`
}

type SaarasGatewayHost2 struct {
	Service_id      int64            `json:"service_id"`
	Service_name    string           `json:"service_name"`
	Fqdn            string           `json:"fqdn"`
	Create_ts       string           `json:"create_ts"`
	Update_ts       string           `json:"update_ts"`
	Routes          []SaarasRoute2   `json:"routes"`
	Service_secrets []SaarasSecrets  `json:"service_secrets"`
	Service_filters []ServiceFilters `json:"service_filters"`
}

type SaarasGatewayHostService struct {
	Service SaarasGatewayHost2 `json:"service"`
	Proxy   struct {
		ProxyGlobalconfigs []struct {
			Globalconfig struct {
				GlobalconfigType string `json:"globalconfig_type"`
				GlobalconfigName string `json:"globalconfig_name"`
				Config           string `json:"config"`
			} `json:"globalconfig"`
		} `json:"proxy_globalconfigs"`
	} `json:"proxy"`
}

type SaarasApp2 struct {
	Saaras_db_proxy_service []SaarasGatewayHostService
}

type DataPayloadSaarasApp2 struct {
	Data   SaarasApp2
	Errors []GraphErr
}

////////////// GatewayHost //////////////////////////////////////////////

func upstream_hc(oneService *cfg.SaarasMicroService2) *v1beta1.HealthCheck {

	if need_hc(oneService) {

		hc := v1beta1.HealthCheck{}
		if len(oneService.Upstream.Upstream_hc_path) > 0 {
			hc.Path = oneService.Upstream.Upstream_hc_path
		}

		if len(oneService.Upstream.Upstream_hc_host) > 0 {
			hc.Host = oneService.Upstream.Upstream_hc_host
		}

		if oneService.Upstream.Upstream_hc_intervalseconds > 0 {
			hc.IntervalSeconds = oneService.Upstream.Upstream_hc_intervalseconds
		}

		if oneService.Upstream.Upstream_hc_timeoutseconds > 0 {
			hc.TimeoutSeconds = oneService.Upstream.Upstream_hc_timeoutseconds
		}

		if oneService.Upstream.Upstream_hc_unhealthythresholdcount > 0 {
			hc.UnhealthyThresholdCount = oneService.Upstream.Upstream_hc_unhealthythresholdcount
		}

		if oneService.Upstream.Upstream_hc_healthythresholdcount > 0 {
			hc.HealthyThresholdCount = oneService.Upstream.Upstream_hc_healthythresholdcount
		}

		return &hc

	}
	return nil
}

func need_hc(oneService *cfg.SaarasMicroService2) bool {

	if len(oneService.Upstream.Upstream_hc_path) > 0 ||
		len(oneService.Upstream.Upstream_hc_host) > 0 ||
		oneService.Upstream.Upstream_hc_intervalseconds > 0 ||
		oneService.Upstream.Upstream_hc_timeoutseconds > 0 ||
		oneService.Upstream.Upstream_hc_unhealthythresholdcount > 0 ||
		oneService.Upstream.Upstream_hc_healthythresholdcount > 0 {

		return true
	}

	return false
}

func upstream_service(oneService *cfg.SaarasMicroService2) v1beta1.Service {

	s := v1beta1.Service{
		Name: serviceName2(oneService.Upstream.Upstream_name),
		Port: int(oneService.Upstream.Upstream_port),
	}

	if oneService.Upstream.Upstream_weight > 0 {
		s.Weight = uint32(oneService.Upstream.Upstream_weight)
	}

	if need_hc(oneService) {
		s.HealthCheck = upstream_hc(oneService)
	}

	return s
}

func saaras_set_sni_for_externalname_service(sir *cfg.SaarasMicroService2) string {
	ip := net.ParseIP(sir.Upstream.Upstream_ip)
	if ip == nil {
		// Couldn't parse IP, assume it's a domain name
		// 09-08-2020 return service domain used to connect to service as SNI for validation
		return sir.Upstream.Upstream_ip
	}

	return ""
}

func saaras_upstream_to_v1b1_uv(sir *cfg.SaarasMicroService2) *v1beta1.UpstreamValidation {

	if sir != nil && sir.Upstream.Upstream_validation_cacertificate != "" && sir.Upstream.Upstream_validation_subjectname != "" {
		return &v1beta1.UpstreamValidation{
			CACertificate: sir.Upstream.Upstream_validation_cacertificate,
			SubjectName:   sir.Upstream.Upstream_validation_subjectname,
		}
	}
	return nil
}

func saaras_service__to__v1b1_service(sm *cfg.SaarasMicroService2) v1beta1.Service {
	s := v1beta1.Service{
		Name:               serviceName2(sm.Upstream.Upstream_name),
		Port:               int(sm.Upstream.Upstream_port),
		Weight:             uint32(sm.Upstream.Upstream_weight),
		HealthCheck:        upstream_hc(sm),
		Strategy:           sm.Upstream.Upstream_strategy,
		UpstreamValidation: saaras_upstream_to_v1b1_uv(sm),
	}

	return s
}

func saaras_route_to_v1b1_service_slice2(sir *SaarasGatewayHostService, r SaarasRoute2) []v1beta1.Service {
	services := make([]v1beta1.Service, 0)
	for _, oneService := range r.Route_upstreams {
		s := saaras_service__to__v1b1_service(&oneService)
		services = append(services, s)
	}
	return services
}

func getIrSecretName2(sir *SaarasGatewayHostService) string {
	// If there are multiple secrets, we pick the first one.

	var secret_name string

	if len(sir.Service.Service_secrets) > 0 {
		secret_name = sir.Service.Service_secrets[0].Secret.Secret_name
	}

	return secret_name
}

func getIrTLS(sir *SaarasGatewayHostService) *v1beta1.TLS {
	secret_name := getIrSecretName2(sir)
	if len(secret_name) > 0 {
		return &v1beta1.TLS{
			SecretName: secret_name,
		}
	} else {
		return nil
	}
}

func saaras_ir_host_filter__to__v1b1_host_filter(sir *SaarasGatewayHostService) []v1beta1.HostAttachedFilter {
	haf_slice := []v1beta1.HostAttachedFilter{}

	for _, oneServiceFilter := range sir.Service.Service_filters {
		v1b1_haf := v1beta1.HostAttachedFilter{
			Name: oneServiceFilter.Filter.Filter_name,
			Type: oneServiceFilter.Filter.Filter_type,
		}
		haf_slice = append(haf_slice, v1b1_haf)
	}

	return haf_slice
}

func saaras_ir_route_filter__to__v1b1_route_filter(r SaarasRoute2) []v1beta1.RouteAttachedFilter {
	raf_slice := []v1beta1.RouteAttachedFilter{}

	for _, oneRouteFilter := range r.Route_filters {
		v1b1_raf := v1beta1.RouteAttachedFilter{
			Name: oneRouteFilter.Filter.Filter_name,
			Type: oneRouteFilter.Filter.Filter_type,
		}
		raf_slice = append(raf_slice, v1b1_raf)
	}
	return raf_slice
}

// TODO: This needs a test
func saaras_routecondition_to_v1b1_ir_routecondition(r SaarasRoute2) []v1beta1.Condition {
	conds := make([]v1beta1.Condition, 0)

	// If Route_prefix is populated, ignore Route_config
	if len(r.Route_prefix) > 0 {
		conds = append(conds, v1beta1.Condition{Prefix: r.Route_prefix})
		return conds
	}

	// Route configuration provided in Route_config, unmarshal and convert to dag Conditions
	saarasRouteCond, err := cfg.UnmarshalRouteMatchCondition(r.Route_config)
	if err != nil {
		conds = append(conds, v1beta1.Condition{Prefix: "/"})
		return conds
	}

	cond := v1beta1.Condition{
		Prefix: saarasRouteCond.Prefix,
	}

	conds = append(conds, cond)

	for _, rmc := range saarasRouteCond.MatchConditions {
		cond2 := v1beta1.Condition{
			Header: &v1beta1.HeaderCondition{
				Name:  rmc.Name,
				Exact: strings.TrimSpace(rmc.Exact),
			},
		}

		conds = append(conds, cond2)
	}

	return conds
}

func Saaras_ir__to__v1b1_ir2(sir *SaarasGatewayHostService) *v1beta1.GatewayHost {
	routes := make([]v1beta1.Route, 0)
	for _, oneRoute := range sir.Service.Routes {
		routes = append(routes, v1beta1.Route{

			Conditions: saaras_routecondition_to_v1b1_ir_routecondition(oneRoute),
			Services:   saaras_route_to_v1b1_service_slice2(sir, oneRoute),
			Filters:    saaras_ir_route_filter__to__v1b1_route_filter(oneRoute),
		})
	}
	return &v1beta1.GatewayHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sir.Service.Service_name,
			Namespace: ENROUTE_NAME,
		},
		Spec: v1beta1.GatewayHostSpec{
			VirtualHost: &v1beta1.VirtualHost{
				Fqdn: sir.Service.Fqdn,
				// TODO
				TLS:     getIrTLS(sir),
				Filters: saaras_ir_host_filter__to__v1b1_host_filter(sir),
			},
			Routes: routes,
		},
	}
}

func saaras_ir_slice__to__v1b1_ir_map(s *[]SaarasGatewayHostService, log logrus.FieldLogger) *map[string]*v1beta1.GatewayHost {
	var m map[string]*v1beta1.GatewayHost
	m = make(map[string]*v1beta1.GatewayHost)

	for _, oneSaarasIRService := range *s {
		onev1b1ir := Saaras_ir__to__v1b1_ir2(&oneSaarasIRService)
		if logger.EL.ELogger != nil && logger.EL.ELogger.GetLevel() >= logrus.DebugLevel {
			spew.Dump(onev1b1ir)
		}
		m[onev1b1ir.Name+"_"+onev1b1ir.Namespace] = onev1b1ir
	}

	return &m
}

func v1b1_tcpproxy_equal(log logrus.FieldLogger, t1, t2 *v1beta1.TCPProxy) bool {
	if t1 == nil && t2 == nil {
		return true
	}
	if t1 == nil && t2 != nil {
		return false
	}
	if t1 != nil && t2 == nil {
		return false
	}
	return v1b1_service_slice_equal(log, t1.Services, t2.Services)
}

type sliceOfIRService []v1beta1.Service

func (o sliceOfIRService) Len() int {
	return len(o)
}

func (o sliceOfIRService) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

func (o sliceOfIRService) Less(i, j int) bool {
	return o[i].Name > o[j].Name
}

func v1b1_service_slice_equal(log logrus.FieldLogger, s1, s2 []v1beta1.Service) bool {
	sort.Sort(sliceOfIRService(s1))
	sort.Sort(sliceOfIRService(s2))

	for idx, oneSvc := range s1 {
		oneSvc2 := s2[idx]

		// TODO: Compare HealthCheck
		if oneSvc.Name == oneSvc2.Name &&
			oneSvc.Port == oneSvc2.Port &&
			oneSvc.Weight == oneSvc2.Weight &&
			oneSvc.Strategy == oneSvc2.Strategy {
		} else {
			return false
		}
	}

	return true
}

func v1b1_route_equal(log logrus.FieldLogger, ir_r1, ir_r2 v1beta1.Route) bool {
	if len(ir_r1.Conditions) > 0 && len(ir_r2.Conditions) > 0 {
		if len(ir_r1.Conditions) == len(ir_r2.Conditions) {
			// TODO: We only compare the prefix here (if present)
			if ir_r1.Conditions[0].Prefix != "" && ir_r2.Conditions[0].Prefix != "" {
				return ir_r1.Conditions[0].Prefix == ir_r2.Conditions[0].Prefix &&
					ir_r1.PrefixRewrite == ir_r2.PrefixRewrite &&
					ir_r1.EnableWebsockets == ir_r2.EnableWebsockets &&
					ir_r1.PermitInsecure == ir_r2.PermitInsecure &&
					v1b1_service_slice_equal(log, ir_r1.Services, ir_r2.Services)

			}
		}
	}

	return false
}

type sliceOfIRRoutes []v1beta1.Route

func (o sliceOfIRRoutes) Len() int {
	return len(o)
}

func (o sliceOfIRRoutes) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

func conditionsToString(r *v1beta1.Route) string {
	s := []string{}
	for _, cond := range r.Conditions {
		if cond.Header != nil {
			s = append(s, cond.Prefix+cond.Header.Name)
		} else {
			s = append(s, cond.Prefix)
		}
	}
	return strings.Join(s, ",")
}

func (o sliceOfIRRoutes) Less(i, j int) bool {
	return conditionsToString(&o[i]) > conditionsToString(&o[j])
}

func v1b1_route_slice_equal(log logrus.FieldLogger, r1, r2 []v1beta1.Route) bool {
	sort.Sort(sliceOfIRRoutes(r1))
	sort.Sort(sliceOfIRRoutes(r2))
	for idx, oneRoute := range r1 {
		if v1b1_route_equal(log, oneRoute, r2[idx]) {
			log.Debugf("Routes %v == %v\n", oneRoute, r2[idx])
			// continue
		} else {
			log.Debugf("Routes %v != %v\n", oneRoute, r2[idx])
			return false
		}
	}
	return true
}

func v1b1_tls_equal(tls1, tls2 *v1beta1.TLS) bool {
	if tls1 == nil && tls2 == nil {
		return true
	}
	if tls1 == nil && tls2 != nil {
		return false
	}
	if tls1 != nil && tls2 == nil {
		return false
	}

	return (tls1.SecretName == tls2.SecretName) &&
		(tls1.MinimumProtocolVersion == tls2.MinimumProtocolVersion)
}

func v1b1_vh_equal(log logrus.FieldLogger, vh1, vh2 *v1beta1.VirtualHost) bool {
	return vh1.Fqdn == vh2.Fqdn &&
		v1b1_tls_equal(vh1.TLS, vh2.TLS)
}

func v1b1_ir_equal(log logrus.FieldLogger, ir1, ir2 *v1beta1.GatewayHost) bool {
	return ir1.Name == ir2.Name &&
		ir1.Namespace == ir2.Namespace &&
		v1b1_vh_equal(log, ir1.Spec.VirtualHost, ir2.Spec.VirtualHost) &&
		v1b1_route_slice_equal(log, ir1.Spec.Routes, ir2.Spec.Routes) &&
		v1b1_tcpproxy_equal(log, ir1.Spec.TCPProxy, ir2.Spec.TCPProxy)
}

///// Services ////////////////////////////////////////////////

func serviceName(org_name, cluster_name, microservice_name string) string {
	//s := fmt.Sprintf("%s-%s-%s", org_name, cluster_name, microservice_name)
	//return s
	return microservice_name
}

func serviceName2(microservice_name string) string {
	//s := fmt.Sprintf("%s-%s-%s", org_name, cluster_name, microservice_name)
	//return s
	return microservice_name
}
