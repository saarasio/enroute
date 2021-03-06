// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package saaras

import (
	"github.com/saarasio/enroute/enroute-dp/internal/contour"
	"github.com/saarasio/enroute/enroute-dp/internal/workgroup"
	"github.com/sirupsen/logrus"
	"time"
)

const oneSecond time.Duration = 1
const cloudPollIntervalSeconds int = 10

type Service struct {
	logrus.FieldLogger
}

func WatchCloudGatewayHost(g *workgroup.Group,
	log logrus.FieldLogger,
	reh *contour.ResourceEventHandler,
	et *contour.EndpointsTranslator,
	pct *contour.GlobalConfigTranslator,
	scc *SaarasCloudCache) {

	g.Add(func(stop <-chan struct{}) error {
		log.Println("started")
		defer log.Println("stopped")

		count := 0

		for {
			time.Sleep(oneSecond * 1000 * time.Millisecond)
			count = count + 1

			// Poll for GatewayHost every cloudPollIntervalSeconds seconds
			if (count % cloudPollIntervalSeconds) == 0 {
				log.Infoln("Fetch-and-Apply configuration from cloud")
				FetchGatewayHost(reh, et, pct, scc, log)
			}
		}

		return nil
	})
}

// Start fulfills the g.Start contract.
// When stop is closed the http server will shutdown.
func (svc *Service) Start(stop <-chan struct{}) error {
	svc.Println("cloud fetcher started")
	return nil
}
