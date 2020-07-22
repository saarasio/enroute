// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

// Copyright © 2018 Heptio
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package metrics provides Prometheus metrics for Contour.
package metrics

import (
	"fmt"
	"net/http"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/saarasio/enroute/enroute-dp/internal/httpsvc"
)

// Metrics provide Prometheus metrics for the app
type Metrics struct {
	gatewayHostTotalGauge      *prometheus.GaugeVec
	gatewayHostRootTotalGauge  *prometheus.GaugeVec
	gatewayHostInvalidGauge    *prometheus.GaugeVec
	gatewayHostValidGauge      *prometheus.GaugeVec
	gatewayHostOrphanedGauge   *prometheus.GaugeVec
	gatewayHostDAGRebuildGauge *prometheus.GaugeVec

	CacheHandlerOnUpdateSummary prometheus.Summary
	ResourceEventHandlerSummary *prometheus.SummaryVec

	// Keep a local cache of metrics for comparison on updates
	metricCache *GatewayHostMetric
}

// GatewayHostMetric stores various metrics for GatewayHost objects
type GatewayHostMetric struct {
	Total    map[Meta]int
	Valid    map[Meta]int
	Invalid  map[Meta]int
	Orphaned map[Meta]int
	Root     map[Meta]int
}

// Meta holds the vhost and namespace of a metric object
type Meta struct {
	VHost, Namespace string
}

const (
	GatewayHostTotalGauge      = "enroute_gatewayhost_total"
	GatewayHostRootTotalGauge  = "enroute_gatewayhost_root_total"
	GatewayHostInvalidGauge    = "enroute_gatewayhost_invalid_total"
	GatewayHostValidGauge      = "enroute_gatewayhost_valid_total"
	GatewayHostOrphanedGauge   = "enroute_gatewayhost_orphaned_total"
	GatewayHostDAGRebuildGauge = "enroute_gatewayhost_dagrebuild_timestamp"

	cacheHandlerOnUpdateSummary = "enroute_cachehandler_onupdate_duration_seconds"
	resourceEventHandlerSummary = "enroute_resourceeventhandler_duration_seconds"
)

// NewMetrics creates a new set of metrics and registers them with
// the supplied registry.
func NewMetrics(registry *prometheus.Registry) *Metrics {
	m := Metrics{
		metricCache: &GatewayHostMetric{},
		gatewayHostTotalGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: GatewayHostTotalGauge,
				Help: "Total number of GatewayHosts",
			},
			[]string{"namespace"},
		),
		gatewayHostRootTotalGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: GatewayHostRootTotalGauge,
				Help: "Total number of root GatewayHosts",
			},
			[]string{"namespace"},
		),
		gatewayHostInvalidGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: GatewayHostInvalidGauge,
				Help: "Total number of invalid GatewayHosts",
			},
			[]string{"namespace", "vhost"},
		),
		gatewayHostValidGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: GatewayHostValidGauge,
				Help: "Total number of valid GatewayHosts",
			},
			[]string{"namespace", "vhost"},
		),
		gatewayHostOrphanedGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: GatewayHostOrphanedGauge,
				Help: "Total number of orphaned GatewayHosts",
			},
			[]string{"namespace"},
		),
		gatewayHostDAGRebuildGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: GatewayHostDAGRebuildGauge,
				Help: "Timestamp of the last DAG rebuild",
			},
			[]string{},
		),
		CacheHandlerOnUpdateSummary: prometheus.NewSummary(prometheus.SummaryOpts{
			Name:       cacheHandlerOnUpdateSummary,
			Help:       "Histogram for the runtime of xDS cache regeneration",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),
		ResourceEventHandlerSummary: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Name:       resourceEventHandlerSummary,
			Help:       "Histogram for the runtime of k8s watcher events",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
			[]string{"op"},
		),
	}
	m.register(registry)
	return &m
}

// register registers the Metrics with the supplied registry.
func (m *Metrics) register(registry *prometheus.Registry) {
	registry.MustRegister(
		m.gatewayHostTotalGauge,
		m.gatewayHostRootTotalGauge,
		m.gatewayHostInvalidGauge,
		m.gatewayHostValidGauge,
		m.gatewayHostOrphanedGauge,
		m.gatewayHostDAGRebuildGauge,
		m.CacheHandlerOnUpdateSummary,
		m.ResourceEventHandlerSummary,
	)
}

// SetDAGLastRebuilt records the last time the DAG was rebuilt.
func (m *Metrics) SetDAGLastRebuilt(ts time.Time) {
	m.gatewayHostDAGRebuildGauge.WithLabelValues().Set(float64(ts.Unix()))
}

// SetGatewayHostMetric sets metric values for a set of GatewayHosts
func (m *Metrics) SetGatewayHostMetric(metrics GatewayHostMetric) {
	// Process metrics
	for meta, value := range metrics.Total {
		m.gatewayHostTotalGauge.WithLabelValues(meta.Namespace).Set(float64(value))
		delete(m.metricCache.Total, meta)
	}
	for meta, value := range metrics.Invalid {
		m.gatewayHostInvalidGauge.WithLabelValues(meta.Namespace, meta.VHost).Set(float64(value))
		delete(m.metricCache.Invalid, meta)
	}
	for meta, value := range metrics.Orphaned {
		m.gatewayHostOrphanedGauge.WithLabelValues(meta.Namespace).Set(float64(value))
		delete(m.metricCache.Orphaned, meta)
	}
	for meta, value := range metrics.Valid {
		m.gatewayHostValidGauge.WithLabelValues(meta.Namespace, meta.VHost).Set(float64(value))
		delete(m.metricCache.Valid, meta)
	}
	for meta, value := range metrics.Root {
		m.gatewayHostRootTotalGauge.WithLabelValues(meta.Namespace).Set(float64(value))
		delete(m.metricCache.Root, meta)
	}

	// All metrics processed, now remove what's left as they are not needed
	for meta := range m.metricCache.Total {
		m.gatewayHostTotalGauge.DeleteLabelValues(meta.Namespace)
	}
	for meta := range m.metricCache.Invalid {
		m.gatewayHostInvalidGauge.DeleteLabelValues(meta.Namespace, meta.VHost)
	}
	for meta := range m.metricCache.Orphaned {
		m.gatewayHostOrphanedGauge.DeleteLabelValues(meta.Namespace)
	}
	for meta := range m.metricCache.Valid {
		m.gatewayHostValidGauge.DeleteLabelValues(meta.Namespace, meta.VHost)
	}
	for meta := range m.metricCache.Root {
		m.gatewayHostRootTotalGauge.DeleteLabelValues(meta.Namespace)
	}

	// copier.Copy(&m.metricCache, metrics)
	m.metricCache = &GatewayHostMetric{
		Total:    metrics.Total,
		Invalid:  metrics.Invalid,
		Valid:    metrics.Valid,
		Orphaned: metrics.Orphaned,
		Root:     metrics.Root,
	}
}

// Service serves various metric and health checking endpoints
type Service struct {
	httpsvc.Service
	*prometheus.Registry
	Client *kubernetes.Clientset
}

// Start fulfills the g.Start contract.
// When stop is closed the http server will shutdown.
func (svc *Service) Start(stop <-chan struct{}) error {

	registerHealthCheck(&svc.ServeMux, svc.Client)
	registerMetrics(&svc.ServeMux, svc.Registry)

	return svc.Service.Start(stop)
}

func registerHealthCheck(mux *http.ServeMux, client *kubernetes.Clientset) {
	healthCheckHandler := func(w http.ResponseWriter, r *http.Request) {
		// Try and lookup Kubernetes server version as a quick and dirty check
		_, err := client.ServerVersion()
		if err != nil {
			msg := fmt.Sprintf("Failed Kubernetes Check: %v", err)
			http.Error(w, msg, http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	}
	mux.HandleFunc("/health", healthCheckHandler)
	mux.HandleFunc("/healthz", healthCheckHandler)
}

func registerMetrics(mux *http.ServeMux, registry *prometheus.Registry) {
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
}
