// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package saaras

import (
	cfg "github.com/saarasio/enroute/enroute-dp/internal/config"
	"github.com/sirupsen/logrus"
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
