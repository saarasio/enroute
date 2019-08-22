package saaras

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/saarasio/enroute/apis/contour/v1beta1"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net"
	"os"
	"sort"
	"strconv"
)

const QIngressRoute2 string = `
  query get_services_by_proxy($proxy_name: String!) {
    saaras_db_proxy_service(where: {proxy: {proxy_name: {_eq: $proxy_name}}}) {
      service {
        service_id
        service_name
        fqdn
        create_ts
        update_ts
        routes {
          route_name
          route_prefix
          create_ts
          update_ts
          route_upstreams {
            upstream {
              upstream_name
              upstream_ip
              upstream_port
              create_ts
              update_ts
            }
          }
        }
      }
    }
  }
`

const QIngressRouteOutput2 string = `
{
  "data": {
    "saaras_db_proxy_service": [
      {
        "service": {
          "service_id": 1,
          "service_name": "test",
          "fqdn": "testfqdn.com",
          "create_ts": "2019-08-20T00:40:02.873368+00:00",
          "update_ts": "2019-08-19T14:57:51.841163+00:00",
          "routes": [
            {
              "route_name": "testroute",
              "route_prefix": "/",
              "create_ts": "2019-08-19T15:06:50.680275+00:00",
              "update_ts": "2019-08-20T00:52:10.882748+00:00",
              "route_upstreams": [
                {
                  "upstream": {
                    "upstream_name": "testupstream",
                    "upstream_ip": "1.1.1.1",
                    "upstream_port": 10000,
                    "create_ts": "2019-08-20T01:21:02.351317+00:00",
                    "update_ts": "2019-08-20T13:20:18.519485+00:00"
                  }
                }
              ]
            }
          ]
        }
      },
      {
        "service": {
          "service_id": 104,
          "service_name": "test2",
          "fqdn": "testfqdn.com",
          "create_ts": "2019-08-20T00:45:17.724345+00:00",
          "update_ts": "2019-08-20T00:45:17.724345+00:00",
          "routes": []
        }
      }
    ]
  }
}
`

// Note: Either the cluster members should start with uppercase or
// json tags should be defined
// Else decoding of the message would fail.
// Also the best way to setup the structs is by starting with the
// output and then defining/replacing members with structs

type SaarasCluster struct {
	Cluster_name string
	External_ip  string
	Create_ts    string
	Update_ts    string
}

type SaarasMicroserviceDetail struct {
	Microservice_name  string
	Port               int32
	External_ip        string
	Create_ts          string
	Update_ts          string
	ClusterByclusterId SaarasCluster
}

type SaarasMicroService struct {
	MicroservicesBymicroserviceId SaarasMicroserviceDetail
	Load_percentage               int32
	Microservice_id               string
	Namespace                     string
}

type SaarasSecretName struct {
	Secret_name string
}

type SaarasAppSecretName struct {
	SecretsBySecretId SaarasSecretName
}

type SaarasRoute struct {
	Route_prefix      string
	Create_ts         string
	Update_ts         string
	RouteMssByrouteId []SaarasMicroService
}

type SaarasOrg struct {
	Org_name string
	Org_id   string
}

// SaarasIngressRoute is similar to IngressRoute
// - A GraphQL query fetches state from the saaras cloud.
//		The query roughly represents an object similar to IngressRoute
//    The query constructs this object by querying multiple tables and performing joins
//    Once the query response is received, we check the query results with current state
//    This will help us determine if we already know about this object and if it is an
//			Add/Update/Delete operation. Once we know what it is, we can generate the
//      the corresponding event (Add/Update/Delete) and call the functions similar to
//      ResourceEventHandler interface
type SaarasIngressRoute struct {
	App_id                      string
	App_name                    string
	Fqdn                        string
	Create_ts                   string
	Update_ts                   string
	OrgByorgId                  SaarasOrg
	RoutessByappId              []SaarasRoute
	Application_secretsByApp_id []SaarasAppSecretName
}

type SaarasApp struct {
	Saaras_db_application []SaarasIngressRoute
}

type SaarasEndpoint struct {
	Name      string
	Namespace string
	Ip        string
	Port      int32
}

type DataPayloadSaarasApp struct {
	Data   SaarasApp
	Errors []GraphErr
}

type SaarasUpstream struct {
	Upstream_name string
	Upstream_ip   string
	Upstream_port int32
	Create_ts     string
	Update_ts     string
}

type SaarasMicroService2 struct {
	Upstream SaarasUpstream
}

type SaarasRoute2 struct {
	Route_name      string
	Route_prefix    string
	Create_ts       string
	Update_ts       string
	Route_upstreams []SaarasMicroService2
}

type SaarasIngressRoute2 struct {
	Service_id   int64
	Service_name string
	Fqdn         string
	Create_ts    string
	Update_ts    string
	Routes       []SaarasRoute2
}

type SaarasIngressRouteService struct {
	Service SaarasIngressRoute2
}

type SaarasApp2 struct {
	Saaras_db_proxy_service []SaarasIngressRouteService
}

type DataPayloadSaarasApp2 struct {
	Data   SaarasApp2
	Errors []GraphErr
}

////////////// IngressRoute //////////////////////////////////////////////

func saaras_route_to_v1b1_service_slice2(sir *SaarasIngressRouteService, r SaarasRoute2) []v1beta1.Service {
	services := make([]v1beta1.Service, 0)
	for _, oneService := range r.Route_upstreams {
		s := v1beta1.Service{
			Name:   serviceName2(oneService.Upstream.Upstream_name),
			Port:   int(oneService.Upstream.Upstream_port),
			Weight: 100,
		}
		services = append(services, s)
	}
	return services
}

func saaras_ir__to__v1b1_ir2(sir *SaarasIngressRouteService) *v1beta1.IngressRoute {
	routes := make([]v1beta1.Route, 0)
	for _, oneRoute := range sir.Service.Routes {
		route := v1beta1.Route{
			Match:    oneRoute.Route_prefix,
			Services: saaras_route_to_v1b1_service_slice2(sir, oneRoute),
		}
		routes = append(routes, route)

	}
	return &v1beta1.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sir.Service.Service_name,
			Namespace: ENROUTE_NAME,
		},
		Spec: v1beta1.IngressRouteSpec{
			VirtualHost: &v1beta1.VirtualHost{
				Fqdn: sir.Service.Fqdn,
				// TODO
				//TLS: &v1beta1.TLS{
				//	SecretName: getIrSecretName(sdb),
				//},
			},
			Routes: routes,
		},
	}
}

func saaras_ir_slice__to__v1b1_ir_map(s *[]SaarasIngressRouteService, log logrus.FieldLogger) *map[string]*v1beta1.IngressRoute {
	var m map[string]*v1beta1.IngressRoute
	m = make(map[string]*v1beta1.IngressRoute)

	for _, oneSaarasIRService := range *s {
		onev1b1ir := saaras_ir__to__v1b1_ir2(&oneSaarasIRService)
		spew.Dump(onev1b1ir)
		m[strconv.FormatInt(oneSaarasIRService.Service.Service_id, 10)] = onev1b1ir
	}

	return &m
}

func getIrSecretName(sdb *SaarasIngressRoute) string {
	if len(sdb.Application_secretsByApp_id) > 0 {
		return sdb.Application_secretsByApp_id[0].SecretsBySecretId.Secret_name
	}

	return ""
}

func saaras_ir__to__v1b1_ir(sdb *SaarasIngressRoute) *v1beta1.IngressRoute {
	routes := make([]v1beta1.Route, 0)

	for _, oneRoute := range sdb.RoutessByappId {
		route := v1beta1.Route{
			Match:    oneRoute.Route_prefix,
			Services: saaras_route_to_v1b1_service_slice(sdb, oneRoute),
		}
		routes = append(routes, route)
	}

	return &v1beta1.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sdb.App_name,
			Namespace: sdb.OrgByorgId.Org_name,
		},
		Spec: v1beta1.IngressRouteSpec{
			VirtualHost: &v1beta1.VirtualHost{
				Fqdn: sdb.Fqdn,
				TLS: &v1beta1.TLS{
					SecretName: getIrSecretName(sdb),
				},
			},
			Routes: routes,
		},
	}
}

func saaras_ms_equal(log logrus.FieldLogger, ms1, ms2 SaarasMicroService) bool {
	return ((ms1.MicroservicesBymicroserviceId.Microservice_name ==
		ms2.MicroservicesBymicroserviceId.Microservice_name) &&
		(ms1.MicroservicesBymicroserviceId.Port ==
			ms2.MicroservicesBymicroserviceId.Port) &&
		(ms1.MicroservicesBymicroserviceId.External_ip ==
			ms2.MicroservicesBymicroserviceId.External_ip) &&
		(ms1.MicroservicesBymicroserviceId.ClusterByclusterId.Cluster_name ==
			ms2.MicroservicesBymicroserviceId.ClusterByclusterId.Cluster_name) &&
		(ms1.MicroservicesBymicroserviceId.ClusterByclusterId.External_ip ==
			ms2.MicroservicesBymicroserviceId.ClusterByclusterId.External_ip) &&
		(ms1.Load_percentage == ms2.Load_percentage))
}

type sliceOfRouteMicroservices []SaarasMicroService

func (o sliceOfRouteMicroservices) Len() int {
	return len(o)
}

func (o sliceOfRouteMicroservices) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

func (o sliceOfRouteMicroservices) Less(i, j int) bool {
	return o[i].MicroservicesBymicroserviceId.Microservice_name >
		o[j].MicroservicesBymicroserviceId.Microservice_name
}

func saaras_ms_slice_equal(log logrus.FieldLogger, ms1, ms2 []SaarasMicroService) bool {
	sort.Sort(sliceOfRouteMicroservices(ms1))
	sort.Sort(sliceOfRouteMicroservices(ms2))
	for idx, oneMs := range ms1 {
		if saaras_ms_equal(log, oneMs, ms2[idx]) {
			log.Debugf("ms %v == %v\n", oneMs, ms2[idx])
		} else {
			log.Debugf("ms %v != %v\n", oneMs, ms2[idx])
			return false
		}
	}

	return true
}

func saaras_routes_equal(log logrus.FieldLogger, r1, r2 SaarasRoute) bool {
	if r1.Route_prefix == r1.Route_prefix {
		return saaras_ms_slice_equal(log, r1.RouteMssByrouteId, r2.RouteMssByrouteId)
	}
	return false
}

type sliceOfRoutes []SaarasRoute

func (o sliceOfRoutes) Len() int {
	return len(o)
}

func (o sliceOfRoutes) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

func (o sliceOfRoutes) Less(i, j int) bool {
	return o[i].Route_prefix > o[j].Route_prefix
}

func saaras_route_slices_equal(log logrus.FieldLogger, r1 []SaarasRoute, r2 []SaarasRoute) bool {
	sort.Sort(sliceOfRoutes(r1))
	sort.Sort(sliceOfRoutes(r2))
	for idx, oneRoute := range r1 {
		if saaras_routes_equal(log, oneRoute, r2[idx]) {
			log.Debugf("Routes %v == %v\n", oneRoute, r2[idx])
		} else {
			log.Debugf("Routes %v != %v\n", oneRoute, r2[idx])
			return false
		}
	}
	return true
}

func saaras_ir_equal(log logrus.FieldLogger, sdb1, sdb2 *SaarasIngressRoute) bool {
	return ((sdb1.App_id == sdb2.App_id) &&
		(sdb1.App_name == sdb2.App_name) &&
		(sdb1.Fqdn == sdb2.Fqdn) &&
		saaras_route_slices_equal(log, sdb1.RoutessByappId, sdb2.RoutessByappId))
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
	return ir_r1.Match == ir_r2.Match &&
		ir_r1.PrefixRewrite == ir_r2.PrefixRewrite &&
		ir_r1.EnableWebsockets == ir_r2.EnableWebsockets &&
		ir_r1.PermitInsecure == ir_r2.PermitInsecure &&
		v1b1_service_slice_equal(log, ir_r1.Services, ir_r2.Services)
}

type sliceOfIRRoutes []v1beta1.Route

func (o sliceOfIRRoutes) Len() int {
	return len(o)
}

func (o sliceOfIRRoutes) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

func (o sliceOfIRRoutes) Less(i, j int) bool {
	return o[i].Match > o[j].Match
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

func v1b1_ir_equal(log logrus.FieldLogger, ir1, ir2 *v1beta1.IngressRoute) bool {
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

func saaras_route_to_v1b1_service_slice(sdb *SaarasIngressRoute, r SaarasRoute) []v1beta1.Service {
	services := make([]v1beta1.Service, 0)
	for _, oneService := range r.RouteMssByrouteId {
		s := v1beta1.Service{
			Name: serviceName(sdb.OrgByorgId.Org_name,
				oneService.MicroservicesBymicroserviceId.ClusterByclusterId.Cluster_name,
				oneService.MicroservicesBymicroserviceId.Microservice_name),
			Port:   int(oneService.MicroservicesBymicroserviceId.Port),
			Weight: int(oneService.Load_percentage),
		}
		services = append(services, s)
	}
	return services
}

func saaras_ms_to_v1_serviceport(s *SaarasMicroService) []v1.ServicePort {
	sp := make([]v1.ServicePort, 0)
	// When a service lookup happens in builder, we first compare with port number, then with name
	// A more common case for us is every service having one service port
	// Service eventually gets transformed into a cluster
	one_service_port := v1.ServicePort{
		//Name: serviceName(s.Namespace,
		//	s.MicroservicesBymicroserviceId.ClusterByclusterId.Cluster_name,
		//	s.MicroservicesBymicroserviceId.Microservice_name),
		Port: s.MicroservicesBymicroserviceId.Port,
	}
	sp = append(sp, one_service_port)
	return sp
}

// Every Service -> Multiple ServicePort
func saaras_ms_to_v1_service(s *SaarasMicroService) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.MicroservicesBymicroserviceId.Microservice_name,
			Namespace: s.Namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: saaras_ms_to_v1_serviceport(s),
		},
	}
}

// One App -> Multiple Routes
// Every Route -> Multiple Services
func saaras_ir_to_v1_service_saaras_ms(sdb *SaarasIngressRoute) (*map[string]*v1.Service, *map[string]*SaarasMicroService) {
	v1_services := make(map[string]*v1.Service)
	cloud_services := make(map[string]*SaarasMicroService)
	routes := sdb.RoutessByappId
	for _, v := range routes {
		mss := v.RouteMssByrouteId
		for _, ms := range mss {
			// When we build a SaarasMicroService from cloud state, it doesn't have
			// SaarasMicroService.Namespace set. Set it before passing SaarasMicroService around.
			// SaarasMicroService.Namespace was added later to SaarasMicroService so that SaarasIngressRoute
			// is not needed when looking up SaarasIngressRoute.OrgByorgId.Org_name
			// We use SaarasIngressRoute.OrgByorgId.Org_name as Namespace for a micro service for all computations
			ms.Namespace = sdb.OrgByorgId.Org_name
			v1_services[ms.Microservice_id] = saaras_ms_to_v1_service(&ms)
			// Store the Org_name under Namespace so that we can use it later when looking up service from saaras service cache
			ms.Namespace = sdb.OrgByorgId.Org_name
			cloud_services[ms.Microservice_id] = &ms
		}
	}
	return &v1_services, &cloud_services
}

func saaras_ir_to_v1_service(saaras_ir_map *map[string]*SaarasIngressRoute) (*map[string]*v1.Service, *map[string]*SaarasMicroService) {
	v1_service_map := make(map[string]*v1.Service)
	saaras_service_map := make(map[string]*SaarasMicroService)

	for _, v := range *saaras_ir_map {
		// Get services for one app
		v1_svc, saaras_svc := saaras_ir_to_v1_service_saaras_ms(v)

		for k, v := range *v1_svc {
			v1_service_map[k] = v
		}
		for k, v := range *saaras_svc {
			saaras_service_map[k] = v
		}
	}

	return &v1_service_map, &saaras_service_map
}

// TODO
func saaras_svc_equal(s1, s2 *SaarasMicroService) bool {
	return true
}

///// Endpoints ////////////////////////////////////////////////
///// Functions for Saaras Endpoints <-> v1.Endpoints /////////

func saaras_ms_to_v1_ep(mss *SaarasMicroService) *v1.Endpoints {
	ep_subsets := make([]v1.EndpointSubset, 0)
	ep_subsets_addresses := make([]v1.EndpointAddress, 0)
	ep_subsets_ports := make([]v1.EndpointPort, 0)

	ep_subsets_port := v1.EndpointPort{
		Port: mss.MicroservicesBymicroserviceId.Port,
	}
	ep_subsets_ports = append(ep_subsets_ports, ep_subsets_port)

	var e_ip string

	// Pick the EIP from the microservice if present
	// Else pick the EIP from cluster
	if mss.MicroservicesBymicroserviceId.External_ip != "" {
		e_ip = mss.MicroservicesBymicroserviceId.External_ip
	} else {
		e_ip = mss.MicroservicesBymicroserviceId.ClusterByclusterId.External_ip
	}

	ep_subsets_address := v1.EndpointAddress{
		IP: e_ip,
	}
	ep_subsets_addresses = append(ep_subsets_addresses, ep_subsets_address)

	ep_subset := v1.EndpointSubset{
		Addresses: ep_subsets_addresses,
		Ports:     ep_subsets_ports,
	}
	ep_subsets = append(ep_subsets, ep_subset)

	return &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mss.MicroservicesBymicroserviceId.Microservice_name,
			Namespace: mss.Namespace,
		},
		Subsets: ep_subsets,
	}
}

func saaras_ep_to_v1_ep(ep *SaarasEndpoint) *v1.Endpoints {
	ep_subsets := make([]v1.EndpointSubset, 0)
	ep_subsets_addresses := make([]v1.EndpointAddress, 0)
	ep_subsets_ports := make([]v1.EndpointPort, 0)

	var ep_subsets_address v1.EndpointAddress

	ep_subsets_port := v1.EndpointPort{
		Port: ep.Port,
	}
	ep_subsets_ports = append(ep_subsets_ports, ep_subsets_port)

	// Check if we received a hostname.
	// If hostname, perform lookup and extract all IPs.

	if net.ParseIP(ep.Ip) == nil {
		// The received value from the cloud is a hostname.
		// Resolve it to IP(s)
		// TODO: This uses the DNS settings of the current host.
		ips, err := net.LookupIP(ep.Ip)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not get IPs: %v\n", err)
			os.Exit(1)
		}
		for _, ip := range ips {
			fmt.Printf("Resolved [%s] -> [%s]\n", ep.Ip, ip.String())
			ep_subsets_address = v1.EndpointAddress{
				IP: ip.String(),
			}
			ep_subsets_addresses = append(ep_subsets_addresses, ep_subsets_address)
		}
	} else {
		ep_subsets_address = v1.EndpointAddress{
			IP: ep.Ip,
		}
		ep_subsets_addresses = append(ep_subsets_addresses, ep_subsets_address)
	}

	ep_subset := v1.EndpointSubset{
		Addresses: ep_subsets_addresses,
		Ports:     ep_subsets_ports,
	}
	ep_subsets = append(ep_subsets, ep_subset)

	return &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ep.Name,
			Namespace: ep.Namespace,
		},
		Subsets: ep_subsets,
	}
}

func saaras_ms_to_saaras_ep(mss *SaarasMicroService) *SaarasEndpoint {
	var e_ip string

	// Pick the EIP from the microservice if present
	// Else pick the EIP from cluster
	if mss.MicroservicesBymicroserviceId.External_ip != "" {
		e_ip = mss.MicroservicesBymicroserviceId.External_ip
	} else {
		e_ip = mss.MicroservicesBymicroserviceId.ClusterByclusterId.External_ip
	}

	return &SaarasEndpoint{
		Name:      mss.MicroservicesBymicroserviceId.Microservice_name,
		Namespace: mss.Namespace,
		Ip:        e_ip,
		Port:      mss.MicroservicesBymicroserviceId.Port,
	}
}

func saaras_ir_to_v1_ep_saaras_ep(sdb *SaarasIngressRoute) (*map[string]*v1.Endpoints, *map[string]*SaarasEndpoint) {
	v1_ep := make(map[string]*v1.Endpoints)
	saaras_ep := make(map[string]*SaarasEndpoint)
	routes := sdb.RoutessByappId
	for _, v := range routes {
		mss := v.RouteMssByrouteId
		for _, ms := range mss {
			ms.Namespace = sdb.OrgByorgId.Org_name
			v1_ep[ms.Microservice_id] = saaras_ms_to_v1_ep(&ms)
			saaras_ep[ms.Microservice_id] = saaras_ms_to_saaras_ep(&ms)
		}
	}
	return &v1_ep, &saaras_ep
}

func saaras_irs_to_v1_ep_saaras_ep(saaras_ir_map *map[string]*SaarasIngressRoute) (*map[string]*v1.Endpoints, *map[string]*SaarasEndpoint) {
	v1_ep_map := make(map[string]*v1.Endpoints)
	saaras_ep_map := make(map[string]*SaarasEndpoint)

	for _, v := range *saaras_ir_map {
		// Get services for one app
		v1_ep, saaras_ep := saaras_ir_to_v1_ep_saaras_ep(v)

		for k, v := range *v1_ep {
			v1_ep_map[k] = v
		}
		for k, v := range *saaras_ep {
			saaras_ep_map[k] = v
		}
	}
	return &v1_ep_map, &saaras_ep_map
}

func saaras_ep_equal(ep1, ep2 *SaarasEndpoint) bool {
	return (ep1.Name == ep2.Name &&
		ep1.Namespace == ep2.Namespace &&
		ep1.Ip == ep2.Ip &&
		ep1.Port == ep2.Port)
}

var qApplicatons string = `
query
  get_applications($oname: String!) {

    saaras_db_application (
      where: {

        _and:
        [
            { orgByorgId: {org_name: {_eq :$oname}} },
            { orgByorgId: {org_name: {_eq :$oname}} }
        ]

      }
    )
      {
        app_id
        app_name
        fqdn
        create_ts
        update_ts
        orgByorgId {
          org_name
        }
        applicationMicroservicessByappId {
          create_ts
          update_ts
          load_percentage
          microservicesBymicroserviceId {
            microservice_name
            clusterByclusterId {
              cluster_name
            }
          }
        }
      }
  }
`
