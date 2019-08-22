package saaras

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/saarasio/enroute/apis/contour/v1beta1"
	"github.com/saarasio/enroute/internal/config"
	"github.com/saarasio/enroute/internal/contour"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/diff"
	"sync"
)

var ENROUTE_NAME string

type SaarasCloudCache struct {
	mu sync.RWMutex

	sdbapps    map[string]*SaarasIngressRoute
	sdbsvcs    map[string]*SaarasMicroService
	sdbeps     map[string]*SaarasEndpoint
	sdbpg      map[string]*config.SaarasProxyGroupConfig
	sdbsecrets map[string]*v1.Secret

	ir  map[string]*v1beta1.IngressRoute
	svc map[string]*v1.Service
}

// CloudEventHandler fetches state from the cloud and generates
// events that can be consumed by ResourceEventHandler
type CloudEventHandler interface {
	OnFetch(obj interface{})
}

func (sac *SaarasCloudCache) update__v1b1_ir__cache(
	saaras_ir_cloud_map *map[string]*v1beta1.IngressRoute,
	reh *contour.ResourceEventHandler, log logrus.FieldLogger) {

	for cloud_ir_id, cloud_ir := range *saaras_ir_cloud_map {
		if len(cloud_ir_id) > 0 {
			if cached_ir, ok := sac.ir[cloud_ir_id]; ok {
				// ir in cache, compare cache and one fetched from cp
				if apiequality.Semantic.DeepEqual(cached_ir, cloud_ir) {
					// Same, ignore
					log.Infof("update__v1b1_ir__cache() - service [%s, %s] - cloud version same as cache\n",
						cloud_ir_id, cloud_ir.Spec.VirtualHost.Fqdn)
				} else {
					log.Infof("update__v1b1_ir__cache() - service [%s, %s] - cloud version NOT same as cache - OnUpdate()\n",
						cloud_ir_id, cloud_ir.Spec.VirtualHost.Fqdn)
					log.Infof("Diff [%s]\n", diff.ObjectGoPrintDiff(cached_ir, cloud_ir))
					sac.ir[cloud_ir_id] = cloud_ir
					reh.OnUpdate(cached_ir, cloud_ir)
				}
			} else {
				// ir not in cache
				fmt.Printf("update__v1b1_ir__cache() -> service [%s, %s] - not in cache - OnAdd()\n",
					cloud_ir_id, cloud_ir.Spec.VirtualHost.Fqdn)
				if sac.ir == nil {
					sac.ir = make(map[string]*v1beta1.IngressRoute)
				}
				sac.ir[cloud_ir_id] = cloud_ir
				reh.OnAdd(cloud_ir)
			}
		}
	}

	for cache_ir_id, _ := range sac.ir {
		if len(cache_ir_id) > 0 {
			if cached_ir2, ok := (*saaras_ir_cloud_map)[cache_ir_id]; !ok {
				log.Infof("update__v1b1_ir__cache() - service [%s, %s] - removed from cloud - OnDelete()\n",
					cache_ir_id, sac.ir[cache_ir_id].Spec.VirtualHost.Fqdn)
				// Not found on cloud, remove
				delete(sac.ir, cache_ir_id)
				reh.OnDelete(cached_ir2)
			}
		}
	}
}

// diff against local cache and generte OnAdd/OnUpdate/OnDelete
func (sac *SaarasCloudCache) update_saaras_service_cache(
	saaras_app_ids *[]string,
	saaras_ir_map *map[string]*SaarasIngressRoute,
	reh *contour.ResourceEventHandler, log logrus.FieldLogger) {

	v1_s_map, saaras_s_map := saaras_ir_to_v1_service(saaras_ir_map)
	log.Debugf(" id ->     v1 Service Map: [%v]\n", v1_s_map)
	log.Debugf(" id -> Saaras Service Map: [%v]\n", saaras_s_map)

	for k, saaras_cloud_svc := range *saaras_s_map {
		if saaras_cache_svc, ok := sac.sdbsvcs[k]; ok {
			if saaras_svc_equal(saaras_cache_svc, saaras_cloud_svc) {
				// No change
				log.Infof("No change in Service id [%s] on saaras cloud\n", k)
			} else {
				// OnUpdate
				log.Infof("Update Service id [%s] from saaras cloud\n", k)
				sac.sdbsvcs[k] = saaras_cloud_svc
				log.Debugf("Cache map [%v]\n", sac.sdbsvcs)
				v1_cache := saaras_ms_to_v1_service(saaras_cache_svc)
				reh.OnUpdate(v1_cache, (*v1_s_map)[k])
			}
		} else {
			// OnAdd
			log.Infof("Add Service id [%s] on saaras cloud\n", k)
			if sac.sdbsvcs == nil {
				sac.sdbsvcs = make(map[string]*SaarasMicroService)
			}
			sac.sdbsvcs[k] = saaras_cloud_svc
			log.Debugf("Cache map [%v]\n", sac.sdbsvcs)
			reh.OnAdd((*v1_s_map)[k])
		}
	}

	// Walk through services in cache and check if they are still programmed on the cloud.
	for k, saaras_cache_svc := range sac.sdbsvcs {
		if _, ok := (*saaras_s_map)[k]; ok {
		} else {
			// OnRemove
			log.Infof("Remove Service id [%s] on saaras cloud\n", k)
			v1_cache_svc := saaras_ms_to_v1_service(saaras_cache_svc)
			delete(sac.sdbsvcs, k)
			reh.OnDelete(v1_cache_svc)
		}
	}
}

func (sac *SaarasCloudCache) update__v1b1_service_cache(
	v1b1_service_slice *[]*v1.Service,
	reh *contour.ResourceEventHandler,
	log logrus.FieldLogger) {

	for _, cloud_svc := range *v1b1_service_slice {
		if cached_svc, ok := sac.svc[cloud_svc.ObjectMeta.Namespace+cloud_svc.ObjectMeta.Name]; ok {
			if apiequality.Semantic.DeepEqual(cached_svc, cloud_svc) {
				log.Infof("update__v1b1_service_cache() - Service [%s] on saaras cloud same as cache\n", cloud_svc.ObjectMeta.Name)
			} else {
				log.Infof("update__v1b1_service_cache() - Service [%s] on saaras cloud changed - OnUpdate()\n", cloud_svc.ObjectMeta.Name)
				sac.svc[cloud_svc.ObjectMeta.Namespace+cloud_svc.ObjectMeta.Name] = cloud_svc
				reh.OnUpdate(cached_svc, cloud_svc)
			}
		} else {
			if sac.svc == nil {
				sac.svc = make(map[string]*v1.Service)
			}
			sac.svc[cloud_svc.ObjectMeta.Namespace+cloud_svc.ObjectMeta.Name] = cloud_svc
			log.Infof("update__v1b1_service_cache() - Service [%s] on saaras cloud added - OnAdd()\n", cloud_svc.ObjectMeta.Name)
			reh.OnAdd(cloud_svc)
		}
	}

	// TODO: Generate OnDelete
}

func (sac *SaarasCloudCache) saaras_ir_slice__to__v1b1_service_slice(
	s *[]SaarasIngressRouteService, log logrus.FieldLogger) *[]*v1.Service {
	svc := make([]*v1.Service, 0)
	for _, oneSaarasIRService := range *s {
		for _, oneRoute := range oneSaarasIRService.Service.Routes {
			for _, oneService := range oneRoute.Route_upstreams {
				sp := make([]v1.ServicePort, 0)
				one_service_port := v1.ServicePort{
					Port: oneService.Upstream.Upstream_port,
				}
				sp = append(sp, one_service_port)
				one_service := &v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      oneService.Upstream.Upstream_name,
						Namespace: ENROUTE_NAME,
					},
					Spec: v1.ServiceSpec{
						Ports: sp,
					},
				}
				svc = append(svc, one_service)
			}
		}
	}
	return &svc
}

func (sac *SaarasCloudCache) update_saaras_endpoint_cache(
	saaras_app_ids *[]string,
	saaras_ir_map *map[string]*SaarasIngressRoute,
	et *contour.EndpointsTranslator, log logrus.FieldLogger) {
	v1_ep_map, saaras_ep_map := saaras_irs_to_v1_ep_saaras_ep(saaras_ir_map)

	for k, saaras_cloud_ep := range *saaras_ep_map {
		if saaras_cache_ep, ok := sac.sdbeps[k]; ok {
			// Found in cache
			if saaras_ep_equal(saaras_cloud_ep, saaras_cache_ep) {
				log.Infof("No change in Endpoint id [%s] on saaras cloud\n", k)
			} else {
				log.Infof("Update Endpoint id [%s] from saaras cloud\n", k)
				old_ep := saaras_ep_to_v1_ep(saaras_cache_ep)
				// OnUpdate expects v1.Endpoints,
				// lookup the v1.Endpoints corresponding to SaarasEndpoint
				new_ep := (*v1_ep_map)[k]
				// Generate the OnUpdate event on EndpointTranslator
				et.OnUpdate(old_ep, new_ep)
				sac.sdbeps[k] = saaras_cloud_ep
			}
		} else {
			// Not found in cache, run OnAdd, update cache
			log.Infof("Add Endpoint id [%s] from Saaras cloud\n", k)
			if sac.sdbeps == nil {
				sac.sdbeps = make(map[string]*SaarasEndpoint)
			}
			sac.sdbeps[k] = saaras_cloud_ep
			v1_ep := saaras_ep_to_v1_ep(saaras_cloud_ep)
			et.OnAdd(v1_ep)
		}
	}

	// TODO: Track and generate OnDelete
}

// Generate OnAdd/OnUpdate/OnDelete on reh
func (sac *SaarasCloudCache) update_saaras_pg_cache(
	saaras_pg_ids *[]string,
	saaras_pg_map *map[string]*config.SaarasProxyGroupConfig,
	reh *contour.ResourceEventHandler,
	et *contour.EndpointsTranslator,
	log logrus.FieldLogger) {

	for k, saaras_pg_cloud := range *saaras_pg_map {
		saaras_cloud_v1_ep := saaras_pg_to_v1_ep(saaras_pg_cloud)
		if saaras_pg_cache, ok := sac.sdbpg[k]; ok {
			saaras_cache_v1_ep := saaras_pg_to_v1_ep(saaras_pg_cache)
			if pg_equal(saaras_pg_cache, saaras_pg_cloud) {
			} else {
				sac.sdbpg[k] = saaras_pg_cloud

				// Now generate event
				reh.OnUpdate(saaras_pg_cloud, saaras_pg_cloud)
				et.OnUpdate(saaras_cache_v1_ep, saaras_cloud_v1_ep)
			}
		} else {
			if sac.sdbpg == nil {
				sac.sdbpg = make(map[string]*config.SaarasProxyGroupConfig)
				config.GCC.Sdbpg = &(sac.sdbpg)
			}
			sac.sdbpg[k] = saaras_pg_cloud

			// Now generate event
			reh.OnAdd(saaras_pg_cloud)
			et.OnAdd(saaras_cloud_v1_ep)
		}
	}

	for k, saaras_pg_cache := range sac.sdbpg {
		saaras_cache_v1_ep := saaras_pg_to_v1_ep(saaras_pg_cache)
		if _, ok := (*saaras_pg_map)[k]; ok {
		} else {
			reh.OnDelete(saaras_pg_cache)
			et.OnDelete(saaras_cache_v1_ep)
			delete(sac.sdbpg, k)
		}
	}
}

func v1_secret_slice_to_v1_secret_map(secrets *[]v1.Secret) map[string]*v1.Secret {
	var secret_map map[string]*v1.Secret
	secret_map = make(map[string]*v1.Secret)
	for _, onesecret := range *secrets {
		// TODO - encode this in a function, check other usages in this file
		secret_map_key := onesecret.Namespace + "_" + onesecret.Name
		secret_map[secret_map_key] = &onesecret
	}
	return secret_map
}

// Generate OnAdd/OnUpdate/OnDelete on reh
func (sac *SaarasCloudCache) update_saaras_secret_cache(
	reh *contour.ResourceEventHandler,
	secrets *[]v1.Secret,
	log logrus.FieldLogger) {
	// Can use equality from here to compare v1.Secret
	// kubernetes/pkg/api/testing/serialization_test.go
	// apiequality "k8s.io/apimachinery/pkg/api/equality"
	// apiequality.Semantic.DeepEqual(v1secret, res)

	// TODO: SaarasCache can hold kubernetes api objects and the api equality methods can be used for comparison.
	// that way we can do away with our own comparison operations
	for _, secret := range *secrets {
		// TODO - encode this in a function
		secret_cache_map_key := secret.Namespace + "_" + secret.Name
		if sac.sdbsecrets == nil {
			sac.sdbsecrets = make(map[string]*v1.Secret)
			// config.GCC.Sdbpg = &(sac.sdbpg)
		}

		if cached_v1_secret, ok := sac.sdbsecrets[secret_cache_map_key]; ok {
			// Found secret in cache
			if apiequality.Semantic.DeepEqual(*cached_v1_secret, secret) {
			} else {
				fmt.Printf("Diff [%s]\n", diff.ObjectGoPrintDiff(*cached_v1_secret, secret))
				sac.sdbsecrets[secret_cache_map_key] = &secret
				reh.OnUpdate(cached_v1_secret, &secret)
			}
		} else {
			// Secret not found in cache
			fmt.Printf("update_saaras_secret_cache() Insert secret in Saaras Cache[%s]\n", secret_cache_map_key)
			sac.sdbsecrets[secret_cache_map_key] = &secret
			reh.OnAdd(&secret)
		}
	}

	cloud_v1_secrets_map := v1_secret_slice_to_v1_secret_map(secrets)

	for cached_v1_secret_key, cached_v1_secret := range sac.sdbsecrets {
		if _, ok := cloud_v1_secrets_map[cached_v1_secret_key]; ok {
		} else {
			reh.OnDelete(cached_v1_secret)
			delete(sac.sdbsecrets, cached_v1_secret_key)
		}
	}
}

// Convert saaras cloud state to k8s state
// Generate events on SaarasCloudCache
// - Generate OnFetch() for all state similar to k8s (ingress_route, service, endpoint)

func (sac *SaarasCloudCache) OnFetch(obj interface{}, reh *contour.ResourceEventHandler, et *contour.EndpointsTranslator, log logrus.FieldLogger) {
	sac.mu.Lock()
	defer sac.mu.Unlock()
	switch obj := obj.(type) {
	case []SaarasIngressRouteService:
		fmt.Printf("-- SaarasCloudCache.OnFetch() --\n")
		v1b1_ir_map := saaras_ir_slice__to__v1b1_ir_map(&obj, log)
		sac.update__v1b1_ir__cache(v1b1_ir_map, reh, log)
		//spew.Dump(v1b1_ir_map)
		v1b1_service_slice := sac.saaras_ir_slice__to__v1b1_service_slice(&obj, log)
		//spew.Dump(v1b1_service_slice)
		sac.update__v1b1_service_cache(v1b1_service_slice, reh, log)
		//spew.Dump(v1b1_service_map)
		//v1b1_endpoint_map := saaras_ir_slice__to__v1b1_endpoint_map(&obj, log)
		//sac.update__v1b1__endpoint_cache(v1b1_endpoint_map, reh, log)
		break

	case []SaarasIngressRoute:
		saaras_app_ids, saaras_ir_map := saaras_ir_slice_to_map(&obj, log)
		sac.update_saaras_service_cache(saaras_app_ids, saaras_ir_map, reh, log)
		sac.update_saaras_endpoint_cache(saaras_app_ids, saaras_ir_map, et, log)
		break

	case []config.SaarasProxyGroupConfig:
		saaras_cluster_ids, saaras_cluster_map := saaras_cluster_slice_to_map(&obj, log)
		sac.update_saaras_pg_cache(saaras_cluster_ids, saaras_cluster_map, reh, et, log)
		break

	case []Secret:
		v1_secrets := Saaras_secret__to__v1_secret(&obj)
		sac.update_saaras_secret_cache(reh, v1_secrets, log)
		break

	default:
		// not an interesting object
	}
}

// ENTRY POINT, this function is called periodically in a loop
func FetchIngressRoute(reh *contour.ResourceEventHandler, et *contour.EndpointsTranslator, scc *SaarasCloudCache, log logrus.FieldLogger) {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	args["proxy_name"] = ENROUTE_NAME

	// Fetch Application
	if err := FetchConfig(QIngressRoute2, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	var gr DataPayloadSaarasApp2
	if err := json.NewDecoder(&buf).Decode(&gr); err != nil {
		errors.Wrap(err, "decoding response")
		log.Errorf("Error when decoding json [%v]\n", err)
	}
	sdb_spew_dump := bytes.NewBufferString(`Saaras_db_proxy_service`)
	//spew.Fdump(sdb_spew_dump, gr.Data.Saaras_db_application)
	spew.Fdump(sdb_spew_dump, gr)
	log.Debugf("-> %s", sdb_spew_dump.String())
	scc.OnFetch(gr.Data.Saaras_db_proxy_service, reh, et, log)

	//// Fetch ProxyGroup
	//buf.Reset()
	//args = make(map[string]string)
	//args["pgname"] = "pg3"
	//if err := FetchConfig(QProxyGroupCfg, &buf, args, log); err != nil {
	//	log.Errorf("FetchIngressRoute(): Error when running http request [%v]\n", err)
	//}

	//var pg DataPayloadProxyGroup
	//if err := json.NewDecoder(&buf).Decode(&pg); err != nil {
	//	errors.Wrap(err, "decoding response")
	//	log.Errorf("Error when decoding json [%v]\n", err)
	//}

	//spew_dump := bytes.NewBufferString(`pg`)
	//spew.Fdump(spew_dump, pg.Data.Saaras_db_proxygroup_config)
	//log.Debugf("%s", spew_dump.String())

	//scc.OnFetch(pg.Data.Saaras_db_proxygroup_config, reh, et, log)

	//// Fetch Secrets
	//buf.Reset()
	//args = make(map[string]string)
	//args["org_name"] = "trial_org_1"
	//if err := FetchConfig(qGetSecretsByOrgName2, &buf, args, log); err != nil {
	//	log.Errorf("FetchIngressRoute(): Error when running http request [%v]\n", err)
	//}

	//var sec DataPayloadSecrets
	//if err := json.NewDecoder(&buf).Decode(&sec); err != nil {
	//	errors.Wrap(err, "decoding response")
	//	log.Errorf("Error when decoding json [%v]\n", err)
	//}

	//spew_dump_secrets := bytes.NewBufferString(`secrets`)
	//spew.Fdump(spew_dump_secrets, sec.Data.Saaras_db_application_secret)
	//log.Debugf("%s", spew_dump_secrets.String())

	//scc.OnFetch(sec.Data.Saaras_db_application_secret, reh, et, log)

}
