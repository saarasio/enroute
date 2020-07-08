// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2019 Saaras Inc.

package saaras

import (
	cfg "github.com/saarasio/enroute/enroute-dp/internal/config"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
)

const QProxyGroupCfg string = `
query get_proxygroup_config_by_name($pgname: String!) {
  saaras_db_proxygroup_config(where: {proxygroup_name: {_eq: $pgname}}) {
    proxygroup_id
    proxygroup_name
    trace_service_ip
    trace_service_port
    create_ts
    update_ts
  }
}
`

type ProxyGroupConfig struct {
	Saaras_db_proxygroup_config []cfg.SaarasProxyGroupConfig
}

type DataPayloadProxyGroup struct {
	Data   ProxyGroupConfig
	Errors []GraphErr
}

func pg_equal(pg1, pg2 *cfg.SaarasProxyGroupConfig) bool {

	if pg1.Proxygroup_id == pg2.Proxygroup_id &&
		pg1.Proxygroup_name == pg2.Proxygroup_name &&
		pg1.Trace_service_ip == pg2.Trace_service_ip &&
		pg1.Trace_service_port == pg2.Trace_service_port {
		return true
	}

	return false
}

func saaras_pg_to_saaras_ms(pg *cfg.SaarasProxyGroupConfig) *SaarasMicroService {
	p, _ := strconv.Atoi(pg.Trace_service_port)
	p_i32 := int32(p)
	return &SaarasMicroService{
		MicroservicesBymicroserviceId: SaarasMicroserviceDetail{
			Microservice_name: pg.Proxygroup_name,
			Port:              p_i32,
			External_ip:       pg.Trace_service_ip,
		},
		Namespace: pg.Proxygroup_name,
	}
}

func saaras_pg_to_v1_ep(pg *cfg.SaarasProxyGroupConfig) *v1.Endpoints {

	p, _ := strconv.Atoi(pg.Trace_service_port)
	p_i32 := int32(p)

	ep_subsets := make([]v1.EndpointSubset, 0)
	ep_subsets_addresses := make([]v1.EndpointAddress, 0)
	ep_subsets_ports := make([]v1.EndpointPort, 0)

	ep_subsets_port := v1.EndpointPort{
		Port: p_i32,
	}
	ep_subsets_ports = append(ep_subsets_ports, ep_subsets_port)

	ep_subsets_address := v1.EndpointAddress{
		IP: pg.Trace_service_ip,
	}
	ep_subsets_addresses = append(ep_subsets_addresses, ep_subsets_address)

	ep_subset := v1.EndpointSubset{
		Addresses: ep_subsets_addresses,
		Ports:     ep_subsets_ports,
	}
	ep_subsets = append(ep_subsets, ep_subset)

	return &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "jaeger-trace",
			Namespace:   pg.Proxygroup_name,
			ClusterName: "jaeger-trace",
		},
		Subsets: ep_subsets,
	}
}

func saaras_cluster_slice_to_map(sc *[]cfg.SaarasProxyGroupConfig, log logrus.FieldLogger) (*[]string, *map[string]*cfg.SaarasProxyGroupConfig) {
	var m map[string]*cfg.SaarasProxyGroupConfig
	m = make(map[string]*cfg.SaarasProxyGroupConfig)

	keys := make([]string, 0)

	for _, pg := range *sc {
		m[pg.Proxygroup_name] = &pg
		keys = append(keys, pg.Proxygroup_name)
	}

	return &keys, &m
}
