// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package cfg

type SaarasProxyGroupConfig struct {
	Proxygroup_id      string
	Proxygroup_name    string
	Trace_service_ip   string
	Trace_service_port string
	Create_ts          string
	Update_ts          string
}

type GlobalCloudConfig struct {
	Sdbpg *map[string]*SaarasProxyGroupConfig
}

var GCC GlobalCloudConfig
