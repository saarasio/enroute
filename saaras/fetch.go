package saaras

import (
	"github.com/sirupsen/logrus"
	"github.com/saarasio/enroute/internal/contour"
	"github.com/saarasio/enroute/internal/workgroup"
	"time"
)

const oneSecond time.Duration = 1
const cloudPollIntervalSeconds int = 10

type Service struct {
	logrus.FieldLogger
}

func WatchCloudIngressRoute(g *workgroup.Group,
	log logrus.FieldLogger,
	reh *contour.ResourceEventHandler,
	et *contour.EndpointsTranslator,
	scc *SaarasCloudCache) {

	g.Add(func(stop <-chan struct{}) error {
		log.Println("started")
		defer log.Println("stopped")

		count := 0

		for {
			time.Sleep(oneSecond * 1000 * time.Millisecond)
			count = count + 1

			// Poll for IngressRoute every cloudPollIntervalSeconds seconds
			if (count % cloudPollIntervalSeconds) == 0 {
				log.Infoln("Fetch-and-Apply configuration from cloud")
				FetchIngressRoute(reh, et, scc, log)
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
