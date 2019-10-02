package config

import (
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/saarasio/enroute/enroute-dp/internal/envoy"
	"time"
)

type SaarasProxyGroupConfig struct {
	Proxygroup_id      string
	Proxygroup_name    string
	Trace_service_ip   string
	Trace_service_port string
	Create_ts          string
	Update_ts          string
}

// High level data structure that holds configuration fetched from the cloud
// This is how non-CloudConfig state propagates from cloud to envoy
//  (Cloud) -> (KubernetesCache) -> (builder) -> (DAG) -> (CacheHandler) -> (envoy config)
// For CloudConfig state it propages from the cloud as follows
//  (Cloud) -> (CloudConfig) -> (envoy config)

type GlobalCloudConfig struct {
	Sdbpg *map[string]*SaarasProxyGroupConfig
}

var GCC GlobalCloudConfig

const JAEGER_TRACING_CLUSTER string = "jaeger-trace"
const EDS_CONFIG_CLUSTER string = "contour"

func edsconfig2(cluster string, pgc *SaarasProxyGroupConfig) *v2.Cluster_EdsClusterConfig {
	return &v2.Cluster_EdsClusterConfig{
		EdsConfig:   envoy.ConfigSource(cluster),
		ServiceName: pgc.Proxygroup_name + "/" + JAEGER_TRACING_CLUSTER,
	}
}

func (pgc *SaarasProxyGroupConfig) cluster2() *v2.Cluster {
	c := &v2.Cluster{
		Name:        JAEGER_TRACING_CLUSTER,
		AltStatName: JAEGER_TRACING_CLUSTER,
		//Type:             v2.Cluster_EDS,
		EdsClusterConfig: edsconfig2("contour", pgc),
		ConnectTimeout:   250 * time.Millisecond,
		LbPolicy:         v2.Cluster_ROUND_ROBIN,
		CommonLbConfig:   envoy.ClusterCommonLBConfig(),
	}
	return c
}

func (gcc GlobalCloudConfig) traceCluster() *v2.Cluster {
	if GCC.Sdbpg != nil {
		for _, v := range *(GCC.Sdbpg) {
			return v.cluster2()
		}
	}

	return nil
}

func (gcc GlobalCloudConfig) AddTraceClusterFromCloudConfig(cm *map[string]*v2.Cluster) {
	tc := gcc.traceCluster()
	if tc != nil {
		(*cm)[JAEGER_TRACING_CLUSTER] = tc
	}
}
