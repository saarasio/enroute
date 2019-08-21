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
}

type SaarasCloudCache2 struct {
	mu sync.RWMutex
	ir map[string]*v1beta1.IngressRoute
}

// CloudEventHandler fetches state from the cloud and generates
// events that can be consumed by ResourceEventHandler
type CloudEventHandler interface {
	OnFetch(obj interface{})
}

// TODO: Replace update_saaras_ir_cache with update_saaras_ir_cache2
func (sac *SaarasCloudCache) update_saaras_ir_cache2(saaras_app_ids *[]string,
	saaras_ir_cloud_map *map[string]*SaarasIngressRoute,
	reh *contour.ResourceEventHandler,
	log logrus.FieldLogger) {

	for k, saaras_ir_cloud := range *saaras_ir_cloud_map {
		v1b1_ir_cloud := saaras_ir__to__v1b1_ir(saaras_ir_cloud)
		if saaras_ir_cache, ok := sac.sdbapps[k]; ok {
			v1b1_ir_cache := saaras_ir__to__v1b1_ir(saaras_ir_cache)

			if v1b1_ir_equal(log, v1b1_ir_cache, v1b1_ir_cloud) {
				// No Change
			} else {
				reh.OnUpdate(v1b1_ir_cache, v1b1_ir_cloud)
			}
		} else {
			sac.sdbapps[k] = saaras_ir_cloud
			reh.OnAdd(v1b1_ir_cloud)
		}
	}

	for k, v1b1_ir_cache := range sac.sdbapps {
		if _, ok := (*saaras_ir_cloud_map)[k]; ok {
		} else {
			// OnDelete
			reh.OnDelete(v1b1_ir_cache)
		}
	}
}

func (sac *SaarasCloudCache) update_saaras_ir_cache(
	saaras_app_ids *[]string,
	saaras_ir_cloud_map *map[string]*SaarasIngressRoute,
	reh *contour.ResourceEventHandler, log logrus.FieldLogger) {

	// Iterate through app ids received from cloud.
	// Check the cache for their presence/absence and convert it to update/add operation respectively
	for _, one_app_id := range *saaras_app_ids {
		if len(one_app_id) > 0 {

			cloud_app := (*saaras_ir_cloud_map)[one_app_id]
			ir_cloud_app := saaras_ir__to__v1b1_ir(cloud_app)

			cloud_app_spew_dump := bytes.NewBufferString(`CloudApp -
			`)
			spew.Fdump(cloud_app_spew_dump, cloud_app)
			log.Debugf("%s", cloud_app_spew_dump.String())
			ingress_route_spew_dump := bytes.NewBufferString(`IngressRoute -
			`)
			spew.Fdump(ingress_route_spew_dump, ir_cloud_app)
			log.Debugf("%s", ingress_route_spew_dump.String())

			log.Infof("Processing app id [%s] from saaras cloud\n", one_app_id)
			if cache_app, ok := sac.sdbapps[one_app_id]; ok {
				ir_cache := saaras_ir__to__v1b1_ir(cache_app)

				if v1b1_ir_equal(log, ir_cloud_app, ir_cache) {
					// App found in cache, call OnUpdate only if apps different
					//if saaras_ir_equal(log, cache_app, cloud_app) {
					log.Infof("No change to app %v in cache. Version from cloud same as the one in cache.\n", one_app_id)
				} else {
					log.Infof("Change to app %v in cache. Update cache. Generate OnUpdate\n", one_app_id)
					sac.sdbapps[one_app_id] = (*saaras_ir_cloud_map)[one_app_id]
					reh.OnUpdate(ir_cache, ir_cloud_app)
				}
			} else {
				// App not found in cache, call OnAdd
				log.Infof("App %v not in cache. Add to cache. Generate OnAdd event\n", one_app_id)
				if sac.sdbapps == nil {
					sac.sdbapps = make(map[string]*SaarasIngressRoute)
				}
				sac.sdbapps[one_app_id] = (*saaras_ir_cloud_map)[one_app_id]
				log.Debugf("Cache map [%v]\n", sac.sdbapps)
				reh.OnAdd(ir_cloud_app)
			}
		}
	}

	// Walk through the apps in the cache and check if they are still present on the cloud
	for one_app_id, _ := range sac.sdbapps {
		if len(one_app_id) > 0 {
			if _, ok := (*saaras_ir_cloud_map)[one_app_id]; !ok {
				cloud_app_from_cache := sac.sdbapps[one_app_id]
				ir := saaras_ir__to__v1b1_ir(cloud_app_from_cache)

				delete(sac.sdbapps, one_app_id)

				// App not found on cloud, call OnRemove
				log.Infof("App <%s> removed from cloud. Remove from cache. Generate OnDelete \n", one_app_id)
				reh.OnDelete(ir)
			} else {
				log.Infof("App not removed from cloud. No change to app %v in cache. App present both on the cloud and in cache.\n", one_app_id)
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
		v1b1_ir_map := saaras_ir_slice__to__v1b1_ir_map(&obj, log)
		fmt.Printf("%+v\n", v1b1_ir_map)
	case []SaarasIngressRoute:
		saaras_app_ids, saaras_ir_map := saaras_ir_slice_to_map(&obj, log)
		sac.update_saaras_ir_cache(saaras_app_ids, saaras_ir_map, reh, log)
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

	// Fetch ProxyGroup
	buf.Reset()
	args = make(map[string]string)
	args["pgname"] = "pg3"
	if err := FetchConfig(QProxyGroupCfg, &buf, args, log); err != nil {
		log.Errorf("FetchIngressRoute(): Error when running http request [%v]\n", err)
	}

	var pg DataPayloadProxyGroup
	if err := json.NewDecoder(&buf).Decode(&pg); err != nil {
		errors.Wrap(err, "decoding response")
		log.Errorf("Error when decoding json [%v]\n", err)
	}

	spew_dump := bytes.NewBufferString(`pg`)
	spew.Fdump(spew_dump, pg.Data.Saaras_db_proxygroup_config)
	log.Debugf("%s", spew_dump.String())

	scc.OnFetch(pg.Data.Saaras_db_proxygroup_config, reh, et, log)

	// Fetch Secrets
	buf.Reset()
	args = make(map[string]string)
	args["org_name"] = "trial_org_1"
	if err := FetchConfig(qGetSecretsByOrgName2, &buf, args, log); err != nil {
		log.Errorf("FetchIngressRoute(): Error when running http request [%v]\n", err)
	}

	var sec DataPayloadSecrets
	if err := json.NewDecoder(&buf).Decode(&sec); err != nil {
		errors.Wrap(err, "decoding response")
		log.Errorf("Error when decoding json [%v]\n", err)
	}

	spew_dump_secrets := bytes.NewBufferString(`secrets`)
	spew.Fdump(spew_dump_secrets, sec.Data.Saaras_db_application_secret)
	log.Debugf("%s", spew_dump_secrets.String())

	scc.OnFetch(sec.Data.Saaras_db_application_secret, reh, et, log)

}
