// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.
package openapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
	//"github.com/go-openapi/swag"
	v1beta1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1beta1"
	"github.com/saarasio/enroute/enroute-dp/saaras"
	"github.com/saarasio/enroute/enroute-dp/saarasconfig"
	"github.com/saarasio/enroute/enroutectl/config"
	"github.com/saarasio/enroute/enroutectl/sync"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Analyzed creates a new analyzed spec document
//func Analyzed(data json.RawMessage, version string) (*Document, error) {
//        if version == "" {
//                version = "2.0"
//        }
//        if version != "2.0" {
//                return nil, fmt.Errorf("spec version %q is not supported", version)
//        }
//
//        raw := data
//        trimmed := bytes.TrimSpace(data)
//        if len(trimmed) > 0 {
//                if trimmed[0] != '{' && trimmed[0] != '[' {
//                        yml, err := swag.BytesToYAMLDoc(trimmed)
//                        if err != nil {
//                                return nil, fmt.Errorf("analyzed: %v", err)
//                        }
//                        d, err := swag.YAMLToJSON(yml)
//                        if err != nil {
//                                return nil, fmt.Errorf("analyzed: %v", err)
//                        }
//                        raw = d
//                }
//        }
//
//=>      swspec := new(spec.Swagger)
//        if err := json.Unmarshal(raw, swspec); err != nil {
//                return nil, err
//        }
//
//        origsqspec, err := cloneSpec(swspec)
//        if err != nil {
//                return nil, err
//        }
//
//        d := &Document{
//                Analyzer: analysis.New(swspec),
//                schema:   spec.MustLoadSwagger20Schema(),
//                spec:     swspec,
//                raw:      raw,
//                origSpec: origsqspec,
//        }
//        return d, nil
//}

// JSONSpec loads a spec from a json document
// Similar to JSONSpec here - go-openapi/loads@v0.19.5/spec.go
func JSONSpec(path_or_spec string, is_spec bool) (*loads.Document, error) {

	var data json.RawMessage
	var err error

	if is_spec {
		data = json.RawMessage(path_or_spec)
	} else {
		data, err = ioutil.ReadFile(path_or_spec)
	}

	if err != nil {
		fmt.Printf("Error while reading file from [%s]\n\n", path_or_spec)
		return nil, err
	}

	// Analyzed() (go-openapi/loads@v0.19.5/spec.go) checks for '{' and '['
	// But we try to guess using extension
	// This call is just a wrapper on json.Unmarshal (see comment at start of file)
	d, e := loads.Analyzed(data, "")

	if e != nil {
		data, err = yaml.YAMLToJSON(data)
		d, e = loads.Analyzed(data, "")
	}

	//WalkSpec(s)

	return d, e
}

func Spec(path string) (*spec.Swagger, error) {

	d, err := JSONSpec(path, false)

	if err != nil {
		fmt.Printf("Error while reading spec from [%s]\n\n", path)
		return nil, err
	}

	if d != nil {
		return d.Spec(), nil
	}

	fmt.Printf("Error while decoding spec from [%s]\n\n", path)

	return nil, nil
}

func JSONMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

func YAMLMarshal(t interface{}) (string, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)

	if err != nil {
		return "", err
	}

	//fmt.Printf("%s\n", string(buffer.Bytes()))

	yamldoc, err2 := yaml.JSONToYAML(buffer.Bytes())
	return string(yamldoc), err2
}

// Convert path and operation into match condition for route
func routeMatchCondition(path, opName string) string {
	// populate RouteMatchConditions and serialize it to Route_config
	var routeMatchConfig saarasconfig.RouteMatchConditions

	routeMatchConfig.Prefix = path
	if routeMatchConfig.MatchConditions == nil {
		routeMatchConfig.MatchConditions = make([]saarasconfig.RouteMatchCondition, 0)
	}

	routeMatchConfig.MatchConditions = append(routeMatchConfig.MatchConditions,
		saarasconfig.RouteMatchCondition{
			HeaderName:  ":method",
			HeaderValue: opName,
		})

	match_json, err := JSONMarshal(routeMatchConfig)

	if err != nil {
		fmt.Printf("Couldn't marshall RouteMatchConditions to JSON\n")
		return ""
	}

	return string(match_json)
}

func getUpstreamForRoute(s *spec.Swagger) []config.RouteUpstreams {
	var rus []config.RouteUpstreams

	rus = make([]config.RouteUpstreams, 0)

	ru := config.RouteUpstreams{
		Upstream: config.Upstream{
			UpstreamName: "openapi-upstream-" + s.SwaggerProps.Host,
			UpstreamIP:   s.SwaggerProps.Host,
			// TODO: Get port or use default
			UpstreamPort: 80,
			// TODO: Provide default value
			// UpstreamHcPath: "/",
		},
	}

	rus = append(rus, ru)

	return rus
}

func truncate(l int, s, suffix string) string {
	if l >= len(s) {
		// under the limit, nothing to do
		return s
	}
	if l <= len(suffix) {
		// easy case, just return the start of the suffix
		return suffix[:min(l, len(suffix))]
	}
	return s[:l-len(suffix)-1] + "-" + suffix
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

func hashname(l int, s ...string) string {
	const shorthash = 6 // the length of the shorthash

	r := strings.Join(s, "/")
	if l > len(r) {
		// we're under the limit, nothing to do
		return r
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(r)))
	for n := len(s) - 1; n >= 0; n-- {
		s[n] = truncate(l/len(s), s[n], hash[:shorthash])
		r = strings.Join(s, "/")
		if l > len(r) {
			return r
		}
	}
	// truncated everything, but we're still too long
	// just return the hash truncated to l.
	return hash[:min(len(hash), l)]
}

func swaggerPathSpecToSaarasRoute(s *spec.Swagger, rName, opName, pathPrefix string) config.Routes {

	var oneSaarasRoute config.Routes

	// Account for basepath and sanitize
	basePath := s.SwaggerProps.BasePath
	basePath = strings.TrimSpace(basePath)
	basePath = strings.TrimRight(basePath, "/")
	pathPrefix = strings.TrimSpace(pathPrefix)
	pathPrefix = strings.TrimLeft(pathPrefix, "/")
	path := basePath + "/" + pathPrefix
	oneSaarasRoute.RouteConfig = routeMatchCondition(path, opName)

	if rName == "" {
		oneSaarasRoute.RouteName = strings.Replace(basePath+pathPrefix, "/", "", 0)
	} else {
		oneSaarasRoute.RouteName = rName
	}

	oneSaarasRoute.RouteName = oneSaarasRoute.RouteName + "-" + hashname(10, path) + "-" + hashname(10, oneSaarasRoute.RouteConfig)
	oneSaarasRoute.RouteName = "openapi-" + strings.Replace(oneSaarasRoute.RouteName, "/", "", -1)
	oneSaarasRoute.RouteName = strings.Replace(oneSaarasRoute.RouteName, " ", "-", -1)

	// TODO: Add route filters if any
	if oneSaarasRoute.RouteUpstreams == nil {
		oneSaarasRoute.RouteUpstreams = make([]config.RouteUpstreams, 0)
	}
	rus := getUpstreamForRoute(s)
	for _, oneru := range rus {
		oneSaarasRoute.RouteUpstreams = append(oneSaarasRoute.RouteUpstreams, oneru)
	}

	//fmt.Printf("swaggerPathSpecToSaarasRoute() %+v\n", oneSaarasRoute)

	return oneSaarasRoute
}

func pathWithParamsToRegexPath(pathWithParams string, hasQueryParams bool) string {
	re := regexp.MustCompile(`\{([^)]+?)\}`)
	var str = `(?P<$1>.*)`
	//var str2 = `(.*)`
	s := re.ReplaceAllString(pathWithParams, str)

	if !hasQueryParams {
		s = s + "$"
	}

	return s
}

func routeForOp(s *spec.Swagger, opName, pathPrefix string, operationProps spec.OperationProps, routes *[]config.Routes) {
	opParameters := operationProps.Parameters

	// Convert Path and Header into a match condition
	// TODO: Handle Query Parameters and other interesting stuff

	hasQueryParams := false

	// Walk through params for additional information
	for _, param := range opParameters {
		if param.ParamProps.In == "query" {
			hasQueryParams = true
		}
	}

	// Parameters found on path, convert params to regex match
	if len(opParameters) > 0 {
		pathPrefix = pathWithParamsToRegexPath(pathPrefix, hasQueryParams)
	}

	oneSaarasRoute := swaggerPathSpecToSaarasRoute(s, operationProps.ID, opName, pathPrefix)
	*routes = append(*routes, oneSaarasRoute)

}

func specToSaarasRoutes(s *spec.Swagger) []config.Routes {
	var routes []config.Routes
	routes = make([]config.Routes, 0)

	for pathPrefix, pathItem := range s.SwaggerProps.Paths.Paths {
		// Do we have params for this path?

		var operationProps spec.OperationProps
		var opName string

		if pathItem.PathItemProps.Get != nil {
			operationProps = pathItem.PathItemProps.Get.OperationProps
			opName = "GET"
			routeForOp(s, opName, pathPrefix, operationProps, &routes)
		}
		if pathItem.PathItemProps.Put != nil {
			operationProps = pathItem.PathItemProps.Put.OperationProps
			opName = "PUT"
			routeForOp(s, opName, pathPrefix, operationProps, &routes)
		}
		if pathItem.PathItemProps.Post != nil {
			operationProps = pathItem.PathItemProps.Post.OperationProps
			opName = "POST"
			routeForOp(s, opName, pathPrefix, operationProps, &routes)
		}
		if pathItem.PathItemProps.Delete != nil {
			operationProps = pathItem.PathItemProps.Delete.OperationProps
			opName = "DELETE"
			routeForOp(s, opName, pathPrefix, operationProps, &routes)
		}
		if pathItem.PathItemProps.Options != nil {
			operationProps = pathItem.PathItemProps.Options.OperationProps
			opName = "OPTIONS"
			routeForOp(s, opName, pathPrefix, operationProps, &routes)
		}
		if pathItem.PathItemProps.Head != nil {
			operationProps = pathItem.PathItemProps.Head.OperationProps
			opName = "HEAD"
			routeForOp(s, opName, pathPrefix, operationProps, &routes)
		}
		if pathItem.PathItemProps.Patch != nil {
			operationProps = pathItem.PathItemProps.Patch.OperationProps
			opName = "PATCH"
			routeForOp(s, opName, pathPrefix, operationProps, &routes)
		}
	}

	sort.Stable(config.RoutesByName(routes))

	return routes
}

func getHostFromSwaggerHost(swaggerHost string) string {
	// TODO: Should the host be any different?
	return swaggerHost
}

func specToProxyServices(s *spec.Swagger) []config.ProxyServices {
	var proxyservices []config.ProxyServices

	proxyservices = make([]config.ProxyServices, 0)

	//fqdn := getHostFromSwaggerHost(s.SwaggerProps.Host)
	svc := config.Service{
		Fqdn:           "*",
		ServiceName:    "openapi-" + s.SwaggerProps.Host,
		ServiceFilters: nil,
		Routes:         specToSaarasRoutes(s),
	}

	psvc := config.ProxyServices{
		Service: svc,
	}

	proxyservices = append(proxyservices, psvc)

	return proxyservices
}

func SwaggerToEnroute(s *spec.Swagger, cfg *config.EnrouteConfig) {

	if cfg == nil {
		// We expect pre-allocated EnrouteConfig
		return
	}

	if cfg.Data.SaarasDbProxy == nil {
		cfg.Data.SaarasDbProxy = make([]config.SaarasDbProxy, 0)
	}

	proxy := config.SaarasDbProxy{
		ProxyName:          "gw",
		ProxyConfig:        "",
		ProxyGlobalconfigs: nil,
		ProxyServices:      specToProxyServices(s),
	}

	cfg.Data.SaarasDbProxy = append(cfg.Data.SaarasDbProxy, proxy)

	//WriteYAML(cfg)
	//CreateOnEnroute(cfg)
}

func WriteYAML(ecfg *config.EnrouteConfig) string {
	yamlstr, _ := YAMLMarshal(*ecfg)
	return yamlstr
}

func CreateOnEnroute(url string, ecfg *config.EnrouteConfig) {

	if ecfg == nil {
		return
	}

	url = strings.TrimRight(url, "/")

	ae := &sync.EnrouteSync{}
	ae.EnDrv.Base_url = url
	ae.EnDrv.Dbg = true
	ae.EnDrv.Id = ecfg.EnrouteCtlUUID

	ae.EnrouteGetStatus()

	emptyCfg := config.EnrouteConfig{

		EnrouteCtlUUID: ecfg.EnrouteCtlUUID,
		Data: config.Data{
			SaarasDbProxy: []config.SaarasDbProxy{
				{
					ProxyName: "gw",
				},
			},
		},
	}

	ae.SyncProxyAdd(&emptyCfg, ecfg)

	// We created a SaarasDbProxy above, so we can index
	for _, onesvc := range ecfg.Data.SaarasDbProxy[0].ProxyServices {
		ae.EnrouteAssociateProxyService(onesvc.Service.ServiceName)
	}
}

func ProxyConfigToHttpFilters(proxy config.SaarasDbProxy) v1beta1.HttpFilterList {
	return v1beta1.HttpFilterList{}
}

func ProxyConfigToRouteFilters(proxy config.SaarasDbProxy) v1beta1.RouteFilterList {
	return v1beta1.RouteFilterList{}
}

func ru_to_saaras_ru(ru []config.RouteUpstreams) []saaras.SaarasMicroService2 {
	sru := make([]saaras.SaarasMicroService2, 0)

	for _, one := range ru {

		var Upstream_hc_intervalseconds int64
		var Upstream_hc_timeoutseconds int64
		var Upstream_hc_healthythresholdcount int64
		var Upstream_hc_unhealthythresholdcount int64

		Upstream_hc_intervalseconds, _ = strconv.ParseInt(one.Upstream.UpstreamHcIntervalseconds, 10, 64)
		Upstream_hc_timeoutseconds, _ = strconv.ParseInt(one.Upstream.UpstreamHcTimeoutseconds, 10, 64)
		Upstream_hc_healthythresholdcount, _ = strconv.ParseInt(one.Upstream.UpstreamHcHealthythresholdcount, 10, 32)
		Upstream_hc_unhealthythresholdcount, _ = strconv.ParseInt(one.Upstream.UpstreamHcUnhealthythresholdcount, 10, 32)

		osru := saaras.SaarasMicroService2{

			Upstream: saaras.SaarasUpstream{
				Upstream_hc_host:                    one.Upstream.UpstreamHcHost,
				Upstream_hc_intervalseconds:         Upstream_hc_intervalseconds,
				Upstream_hc_timeoutseconds:          Upstream_hc_timeoutseconds,
				Upstream_hc_healthythresholdcount:   uint32(Upstream_hc_healthythresholdcount),
				Upstream_hc_unhealthythresholdcount: uint32(Upstream_hc_unhealthythresholdcount),
				Upstream_port:                       int32(one.Upstream.UpstreamPort),
				Upstream_weight:                     int32(one.Upstream.UpstreamWeight),
				Upstream_hc_path:                    one.Upstream.UpstreamHcPath,
				Upstream_ip:                         one.Upstream.UpstreamIP,
				Upstream_name:                       one.Upstream.UpstreamName,
				Upstream_protocol:                   one.Upstream.UpstreamProtocol,
				Upstream_strategy:                   one.Upstream.UpstreamStrategy,
				Upstream_validation_cacertificate:   one.Upstream.UpstreamValidationCacertificate,
				Upstream_validation_subjectname:     one.Upstream.UpstreamValidationSubjectname,
			},
		}

		sru = append(sru, osru)
	}

	return sru
}

func ss_to_saaras_ss(sf []config.ServiceSecret) []saaras.SaarasSecrets {
	// TODO: Right now there are no secrets coming in from openapi
	return nil
}

func sf_to_saaras_sf(sf []config.ServiceFilters) []saaras.SaarasServiceFilters {
	ssf := make([]saaras.SaarasServiceFilters, 0)

	for _, one := range sf {
		onesf := saaras.SaarasServiceFilters{
			Filter: saaras.SaarasServiceFilter{
				Filter_name:   one.Filter.FilterName,
				Filter_type:   one.Filter.FilterType,
				Filter_config: one.Filter.FilterConfig,
			},
		}

		ssf = append(ssf, onesf)
	}

	return ssf
}

func rf_to_saaras_rf(rf []config.RouteFilters) []saaras.SaarasRFilter {
	srf := make([]saaras.SaarasRFilter, 0)
	for _, one := range rf {
		onerf := saaras.SaarasRFilter{
			Filter: saarasconfig.SaarasRouteFilter{
				Filter_name:   one.Filter.FilterName,
				Filter_type:   one.Filter.FilterType,
				Filter_config: one.Filter.FilterConfig,
			},
		}

		srf = append(srf, onerf)
	}
	return srf
}

func sr_to_saaras_sr(sr []config.Routes) []saaras.SaarasRoute2 {
	r := make([]saaras.SaarasRoute2, 0)

	for _, one := range sr {
		onesr := saaras.SaarasRoute2{
			Route_name:      one.RouteName,
			Route_prefix:    one.RoutePrefix,
			Route_config:    one.RouteConfig,
			Route_filters:   rf_to_saaras_rf(one.RouteFilters),
			Route_upstreams: ru_to_saaras_ru(one.RouteUpstreams),
		}

		r = append(r, onesr)
	}

	return r
}

func EnrouteCtlServiceToSaarasGatewayHost(saarassvc *config.Service) *saaras.SaarasGatewayHostService {

	name := saarassvc.Fqdn

	if saarassvc.Fqdn == "*" {
		name = "wildcard"
	}
	sghs := saaras.SaarasGatewayHostService{
		Service: saaras.SaarasGatewayHost2{
			Service_name:    "openapi-" + name,
			Fqdn:            saarassvc.Fqdn,
			Routes:          sr_to_saaras_sr(saarassvc.Routes),
			Service_secrets: ss_to_saaras_ss(saarassvc.Secret),
			Service_filters: sf_to_saaras_sf(saarassvc.ServiceFilters),
		},
	}

	return &sghs
}

func SanitizeGatewayHost(gh *v1beta1.GatewayHost) {
	//v1b1gh.apiVersion = "enroute.saaras.io/v1beta1"
	gh.ObjectMeta.Namespace = "openapi"
	gh.TypeMeta.APIVersion = "enroute.saaras.io/v1beta1"
	gh.TypeMeta.Kind = "GatewayHost"
}

func SaarasServiceToGatewayHost(saarassvc config.Service) *v1beta1.GatewayHost {

	// Convert config.Service to saaras.SaarasGatewayHostService
	// Call Saaras_ir__to__v1b1_ir2() to convert saaras.SaarasGatewayHostService to v1beta1.GatewayHost
	sghs := EnrouteCtlServiceToSaarasGatewayHost(&saarassvc)
	v1b1gh := saaras.Saaras_ir__to__v1b1_ir2(sghs)
	SanitizeGatewayHost(v1b1gh)
	return v1b1gh
}

func ProxyConfigToGatewayHosts(proxy config.SaarasDbProxy) *v1beta1.GatewayHostList {
	items := make([]v1beta1.GatewayHost, 0)

	for _, oneSvc := range proxy.ProxyServices {
		gwHost := SaarasServiceToGatewayHost(oneSvc.Service)
		items = append(items, *gwHost)
	}
	return &v1beta1.GatewayHostList{
		Items: items,
	}
}

func ProxyConfigToGlobalConfig(proxy config.SaarasDbProxy) v1beta1.GlobalConfigList {
	var gcl v1beta1.GlobalConfigList
	for _, oneGc := range proxy.ProxyGlobalconfigs {
		gc := oneGc.Globalconfig

		if gcl.Items == nil {
			gcl.Items = make([]v1beta1.GlobalConfig, 1)
		}
		oneGlobalConfig := v1beta1.GlobalConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      gc.GlobalconfigName,
				Namespace: "gw" + "-" + "openapi", //TODO
			},
			Spec: v1beta1.GlobalConfigSpec{
				Name:   "proxy_config_name",
				Type:   gc.GlobalconfigType,
				Config: gc.Config,
			},
		}
		gcl.Items = append(gcl.Items, oneGlobalConfig)
	}
	return gcl
}

func SyncProxyToK8sApi(ecfg *config.EnrouteConfig) *v1beta1.GatewayHostList {
	if ecfg == nil {
		return nil
	}

	if len(ecfg.Data.SaarasDbProxy) == 0 {
		return nil
	}

	// If there is more than one proxy, we just pick the first one.
	proxy := ecfg.Data.SaarasDbProxy[0]

	// Deal with only gw proxy
	if proxy.ProxyName != "gw" {
		return nil
	}

	//k8sGlobalConfigList := ProxyConfigToGlobalConfig(proxy)
	k8sGatewayHostsList := ProxyConfigToGatewayHosts(proxy)
	//k8sRouteFiltersList := ProxyConfigToRouteFilters(proxy)
	//k8sHttpFiltersList := ProxyConfigToHttpFilters(proxy)
	return k8sGatewayHostsList
}

func CreateKSYaml(filename string, ecfg *config.EnrouteConfig) {

	if ecfg == nil {
		return
	}

	k8sgh := SyncProxyToK8sApi(ecfg)
	for _, one := range k8sgh.Items {
		yamlstr, _ := YAMLMarshal(one)
		fmt.Printf(" --  experimental -- \n")
		fmt.Printf("%s\n", yamlstr)
	}
}

func WalkSpec(s *spec.Swagger) (*config.EnrouteConfig, error) {
	var ecfg config.EnrouteConfig

	SwaggerToEnroute(s, &ecfg)
	//WriteYAML(&ecfg)
	//CreateOnEnroute(&ecfg)

	return &ecfg, nil
}
