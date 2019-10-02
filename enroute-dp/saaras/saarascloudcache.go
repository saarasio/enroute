package saaras

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/saarasio/enroute/enroute-dp/apis/contour/v1beta1"
	"github.com/saarasio/enroute/enroute-dp/internal/config"
	"github.com/saarasio/enroute/enroute-dp/internal/contour"
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

	sdbpg      map[string]*config.SaarasProxyGroupConfig
	sdbsecrets map[string]*v1.Secret

	ir  map[string]*v1beta1.IngressRoute
	svc map[string]*v1.Service
	ep  map[string]*v1.Endpoints
	sec map[string]*v1.Secret
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
					log.Infof("update__v1b1_ir__cache() - IR [%s, %s] - cloud version same as cache\n",
						cloud_ir_id, cloud_ir.Spec.VirtualHost.Fqdn)
				} else {
					log.Infof("update__v1b1_ir__cache() - IR [%s, %s] - cloud version NOT same as cache - OnUpdate()\n",
						cloud_ir_id, cloud_ir.Spec.VirtualHost.Fqdn)
					log.Infof("Diff [%s]\n", diff.ObjectGoPrintDiff(cached_ir, cloud_ir))
					sac.ir[cloud_ir_id] = cloud_ir
					reh.OnUpdate(cached_ir, cloud_ir)
				}
			} else {
				// ir not in cache
				fmt.Printf("update__v1b1_ir__cache() -> IR [%s, %s] - not in cache - OnAdd()\n",
					cloud_ir_id, cloud_ir.Spec.VirtualHost.Fqdn)
				if sac.ir == nil {
					sac.ir = make(map[string]*v1beta1.IngressRoute)
				}
				sac.ir[cloud_ir_id] = cloud_ir
				reh.OnAdd(cloud_ir)
			}
		}
	}

	for cache_ir_id, cached_ir := range sac.ir {
		if len(cache_ir_id) > 0 {
			if _, ok := (*saaras_ir_cloud_map)[cache_ir_id]; !ok {
				log.Infof("update__v1b1_ir__cache() - IR [%s, %s] - removed from cloud - OnDelete()\n",
					cache_ir_id, sac.ir[cache_ir_id].Spec.VirtualHost.Fqdn)
				// Not found on cloud, remove
				reh.OnDelete(cached_ir)
				delete(sac.ir, cache_ir_id)
			}
		}
	}
}

func (sac *SaarasCloudCache) update__v1b1_service__cache(
	v1b1_service_map *map[string]*v1.Service,
	reh *contour.ResourceEventHandler,
	log logrus.FieldLogger) {

	for _, cloud_svc := range *v1b1_service_map {
		if cached_svc, ok := sac.svc[cloud_svc.ObjectMeta.Namespace+cloud_svc.ObjectMeta.Name]; ok {
			if apiequality.Semantic.DeepEqual(cached_svc, cloud_svc) {
				log.Infof("update__v1b1_service__cache() - SVC [%s] on saaras cloud same as cache\n", cloud_svc.ObjectMeta.Name)
			} else {
				log.Infof("update__v1b1_service__cache() - SVC [%s] on saaras cloud changed - OnUpdate()\n", cloud_svc.ObjectMeta.Name)
				sac.svc[cloud_svc.ObjectMeta.Namespace+cloud_svc.ObjectMeta.Name] = cloud_svc
				reh.OnUpdate(cached_svc, cloud_svc)
			}
		} else {
			if sac.svc == nil {
				sac.svc = make(map[string]*v1.Service)
			}
			sac.svc[cloud_svc.ObjectMeta.Namespace+cloud_svc.ObjectMeta.Name] = cloud_svc
			log.Infof("update__v1b1_service__cache() - SVC [%s] on saaras cloud added - OnAdd()\n", cloud_svc.ObjectMeta.Name)
			reh.OnAdd(cloud_svc)
		}
	}

	// TODO: Generate OnDelete
	for cache_svc_id, _ := range sac.svc {
		if len(cache_svc_id) > 0 {
			if _, ok := (*v1b1_service_map)[cache_svc_id]; !ok {
				log.Infof("update__v1b1_service__cache() - SVC [%s] removed from cloud- OnDelete()\n", cache_svc_id)
				reh.OnDelete(cache_svc_id)
				delete(sac.svc, cache_svc_id)
			}
		}
	}
}

func saaras_ir_slice__to__v1b1_service_map(
	s *[]SaarasIngressRouteService, log logrus.FieldLogger) *map[string]*v1.Service {
	svc := make(map[string]*v1.Service, 0)
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
				svc[one_service.ObjectMeta.Namespace+one_service.ObjectMeta.Name] = one_service
			}
		}
	}
	return &svc
}

func saaras_upstream__to__v1_ep(mss *SaarasMicroService2) *v1.Endpoints {
	ep_subsets := make([]v1.EndpointSubset, 0)
	ep_subsets_addresses := make([]v1.EndpointAddress, 0)
	ep_subsets_ports := make([]v1.EndpointPort, 0)

	ep_subsets_port := v1.EndpointPort{
		Port: mss.Upstream.Upstream_port,
	}
	ep_subsets_ports = append(ep_subsets_ports, ep_subsets_port)

	ep_subsets_address := v1.EndpointAddress{
		IP: mss.Upstream.Upstream_ip,
	}
	ep_subsets_addresses = append(ep_subsets_addresses, ep_subsets_address)

	ep_subset := v1.EndpointSubset{
		Addresses: ep_subsets_addresses,
		Ports:     ep_subsets_ports,
	}
	ep_subsets = append(ep_subsets, ep_subset)

	return &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mss.Upstream.Upstream_name,
			Namespace: ENROUTE_NAME,
		},
		Subsets: ep_subsets,
	}
}

func saaras_ir_slice__to__v1b1_endpoint_map(
	s *[]SaarasIngressRouteService, log logrus.FieldLogger) *map[string]*v1.Endpoints {
	eps := make(map[string]*v1.Endpoints, 0)
	for _, oneSaarasIRService := range *s {
		for _, oneRoute := range oneSaarasIRService.Service.Routes {
			for _, oneService := range oneRoute.Route_upstreams {
				v1_ep := saaras_upstream__to__v1_ep(&oneService)
				eps[v1_ep.ObjectMeta.Namespace+v1_ep.ObjectMeta.Name] = v1_ep
			}
		}
	}
	return &eps
}

func (sac *SaarasCloudCache) update__v1b1__endpoint_cache(v1b1_endpoint_map *map[string]*v1.Endpoints,
	et *contour.EndpointsTranslator,
	log logrus.FieldLogger) {

	for _, cloud_ep := range *v1b1_endpoint_map {
		if cached_ep, ok := sac.ep[cloud_ep.ObjectMeta.Namespace+cloud_ep.ObjectMeta.Name]; ok {
			if apiequality.Semantic.DeepEqual(cached_ep, cloud_ep) {
				log.Infof("update__v1b1_endpoint__cache() - EP [%s] on saaras cloud same as cache\n", cloud_ep.ObjectMeta.Name)
			} else {
				log.Infof("update__v1b1_endpoint__cache() - EP [%s] on saaras cloud changed OnUpdate()\n", cloud_ep.ObjectMeta.Name)
				sac.ep[cached_ep.ObjectMeta.Namespace+cached_ep.ObjectMeta.Name] = cloud_ep
				et.OnUpdate(cached_ep, cloud_ep)
			}
		} else {
			log.Infof("update__v1b1_endpoint__cache() - EP [%s] NOT on saaras cloud - OnAdd()\n", cloud_ep.ObjectMeta.Name)

			if sac.ep == nil {
				sac.ep = make(map[string]*v1.Endpoints)
			}
			sac.ep[cloud_ep.ObjectMeta.Namespace+cloud_ep.ObjectMeta.Name] = cloud_ep
			et.OnAdd(cloud_ep)
		}
	}

	// Generate OnDelete()
	for cache_ep_id, cache_ep := range sac.ep {
		if len(cache_ep_id) > 0 {
			if _, ok := (*v1b1_endpoint_map)[cache_ep_id]; !ok {
				log.Infof("update__v1b1_endpoint__cache() - EP [%s] removed from cloud- OnDelete()\n", cache_ep_id)
				et.OnDelete(cache_ep)
				delete(sac.ep, cache_ep_id)
			}
		}
	}
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

func (sac *SaarasCloudCache) update__v1__secret_cache(v1_secret_map *map[string]*v1.Secret, reh *contour.ResourceEventHandler, log logrus.FieldLogger) {
	for _, cloud_secret := range *v1_secret_map {
		if cached_secret, ok := sac.sec[cloud_secret.ObjectMeta.Namespace+cloud_secret.ObjectMeta.Name]; ok {
			if apiequality.Semantic.DeepEqual(cached_secret, cloud_secret) {
				log.Infof("update__v1__secret_cache() - SEC [%s] on saaras cloud same as cache\n", cloud_secret.ObjectMeta.Name)
			} else {
				log.Infof("update__v1__secret_cache() - SEC [%s] on saaras different from cache - OnUpdate()\n", cloud_secret.ObjectMeta.Name)
				sac.sec[cloud_secret.ObjectMeta.Namespace+cloud_secret.ObjectMeta.Name] = cloud_secret
				reh.OnUpdate(cached_secret, cloud_secret)
			}
		} else {
			if sac.sec == nil {
				sac.sec = make(map[string]*v1.Secret, 0)
			}
			log.Infof("update__v1__secret_cache() - SEC [%s] not in cache OnAdd()\n", cloud_secret.ObjectMeta.Name)
			sac.sec[cloud_secret.ObjectMeta.Namespace+cloud_secret.ObjectMeta.Name] = cloud_secret
			reh.OnAdd(cloud_secret)
		}
	}

	for cache_sec_id, cached_secret := range sac.sec {
		if len(cache_sec_id) > 0 {
			if _, ok := (*v1_secret_map)[cache_sec_id]; !ok {
				log.Infof("update__v1__secret_cache() - SEC [%s] removed from cloud - OnDelete()\n", cached_secret.ObjectMeta.Name)
				delete(sac.sec, cache_sec_id)
				reh.OnDelete(cached_secret)
			}
		}
	}
}

func v1_secret(saaras_secret *SaarasSecret) *v1.Secret {

	var v1secret v1.Secret
	//				v1_service := &v1.Secret{
	//					ObjectMeta: metav1.ObjectMeta{
	//						Name:      saaras_secret.Secret_name,
	//						Namespace: ENROUTE_NAME,
	//					},
	//				}

	v1secret.ObjectMeta.Name = saaras_secret.Secret_name
	v1secret.ObjectMeta.Namespace = ENROUTE_NAME

	//TODO: This needs to be captured in the DB
	v1secret.Type = v1.SecretTypeTLS
	if v1secret.Data == nil {
		v1secret.Data = make(map[string][]byte, 0)
	}
	v1secret.Data[v1.TLSCertKey] = []byte(saaras_secret.Secret_cert)
	v1secret.Data[v1.TLSPrivateKeyKey] = []byte(saaras_secret.Secret_key)

	//	for _, artifact := range saaras_secret.Artifacts {
	//		if artifact.Artifact_type == v1.TLSCertKey {
	//			if v1secret.Data == nil {
	//				v1secret.Data = make(map[string][]byte, 0)
	//			}
	//			v1secret.Data[v1.TLSCertKey] = []byte(artifact.Artifact_value)
	//		}
	//
	//		if artifact.Artifact_type == v1.TLSPrivateKeyKey {
	//			v1secret.Data[v1.TLSPrivateKeyKey] = []byte(artifact.Artifact_value)
	//		}
	//	}

	return &v1secret
}

func saaras_ir_slice__to__v1_secret(s *[]SaarasIngressRouteService, log logrus.FieldLogger) *map[string]*v1.Secret {
	secrets := make(map[string]*v1.Secret, 0)
	for _, oneSaarasIRService := range *s {
		for _, oneSecret := range oneSaarasIRService.Service.Service_secrets {
			secrets[ENROUTE_NAME+oneSecret.Secret.Secret_name] = v1_secret(&oneSecret.Secret)
		}
	}
	return &secrets
}

// Convert saaras cloud state to k8s state
// Generate events on SaarasCloudCache
// - Generate OnFetch() for all state similar to k8s (ingress_route, service, endpoint)

func (sac *SaarasCloudCache) OnFetch(obj interface{}, reh *contour.ResourceEventHandler, et *contour.EndpointsTranslator, log logrus.FieldLogger) {
	sac.mu.Lock()
	defer sac.mu.Unlock()
	switch obj := obj.(type) {
	case []SaarasIngressRouteService:
		log.Infof("-- SaarasCloudCache.OnFetch() --\n")
		v1b1_ir_map := saaras_ir_slice__to__v1b1_ir_map(&obj, log)
		sac.update__v1b1_ir__cache(v1b1_ir_map, reh, log)
		//spew.Dump(v1b1_ir_map)
		v1b1_service_map := saaras_ir_slice__to__v1b1_service_map(&obj, log)
		//spew.Dump(v1b1_service_slice)
		sac.update__v1b1_service__cache(v1b1_service_map, reh, log)
		//spew.Dump(v1b1_service_map)
		v1b1_endpoint_map := saaras_ir_slice__to__v1b1_endpoint_map(&obj, log)
		sac.update__v1b1__endpoint_cache(v1b1_endpoint_map, et, log)
		v1_secret_map := saaras_ir_slice__to__v1_secret(&obj, log)
		sac.update__v1__secret_cache(v1_secret_map, reh, log)
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
	if err := FetchConfig(QIngressRoute3, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
		// If we failed reaching the route, an empty IngressRoute is received.
		// Bail here or it'll clear the cache
		return
	}

	var gr DataPayloadSaarasApp2
	if err := json.NewDecoder(&buf).Decode(&gr); err != nil {
		errors.Wrap(err, "decoding response")
		log.Errorf("Error when decoding json [%v]\n", err)
	}
	sdb_spew_dump := bytes.NewBufferString(`Saaras_db_proxy_service`)
	//spew.Fdump(sdb_spew_dump, gr.Data.Saaras_db_application)
	spew.Fdump(sdb_spew_dump, gr)
	//log.Debugf("-> %s", sdb_spew_dump.String())
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
