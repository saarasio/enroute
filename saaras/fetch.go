package saaras

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	//"github.com/saarasio/enroute/saaras"
	"time"
)

const cloudPollIntervalSeconds time.Duration = 5
const SAARAS_GRAPHQL_SERVER_URL = "http://51.158.75.43/v1alpha1/graphql"

type Service struct {
	logrus.FieldLogger
}

var qApplicatons string = `
query
  get_applications($oname: String!) {

    saaras_db_application (
      where: {

        _and:
        [
            { orgByorgId: {org_name: {_eq :$oname}} },
            { orgByorgId: {org_name: {_eq :$oname}} }
        ]

      }
    )
      {
        app_id
        app_name
        fqdn
        create_ts
        update_ts
        orgByorgId {
          org_name
        }
        applicationMicroservicessByappId {
          create_ts
          update_ts
          load_percentage
          microservicesBymicroserviceId {
            microservice_name
            clusterByclusterId {
              cluster_name
            }
          }
        }
      }
  }
`

const sample_response string = `
    "data": {
        "saaras_db_application": [
            {
                "app_id": "1",
                "app_name": "trial_app_1",
                "applicationMicroservicessByappId": [
                    {
                        "microservicesBymicroserviceId": {
                            "clusterByclusterId": {
                                "cluster_name": "trial_cluster_1"
                            },
                            "microservice_name": "trial_microservice_1"
                        }
                    },
                    {
                        "microservicesBymicroserviceId": {
                            "clusterByclusterId": {
                                "cluster_name": "trial_cluster_2"
                            },
                            "microservice_name": "trial_microservice_1"
                        }
                    }
                ],
                "create_ts": "2019-01-12T21:30:23.258264+00:00",
                "orgByorgId": {
                    "org_name": "trial_org_1"
                }
            }
        ]
    }
`

// Note: Either the cluster members should start with uppercase or
// json tags should be defined
// Else decoding of the message would fail.
// Also the best way to setup the structs is by starting with the
// output and then defining/replacing members with structs
type cByClusterId struct {
	Cluster_name string
}

type mBymicroserviceId struct {
	ClusterByclusterId cByClusterId
	Microservice_name  string
}

type miBymicroserviceId struct {
	Create_ts                     string
	Update_ts                     string
	Load_percentage               int
	MicroservicesBymicroserviceId mBymicroserviceId
}

type oByorgId struct {
	Org_name string
}

type sdba struct {
	App_id                           string
	App_name                         string
	Fqdn                             string
	Route                            string
	ApplicationMicroservicessByappId []miBymicroserviceId
	Create_ts                        string
	Update_ts                        string
	OrgByorgId                       oByorgId
}

type sdba2 struct {
	Saaras_db_application []sdba
}

type dataPayloadApp struct {
	Data   sdba2
	Errors []GraphErr
}

type dataPayload struct {
	Data   interface{}
	Errors []GraphErr
}

func fetchConfig(query string) {
	client := NewClient(SAARAS_GRAPHQL_SERVER_URL)
	client.Log = func(s string) { fmt.Printf(" [%s]\n", s) }
	req := NewRequest(qApplicatons)
	req.Var("oname", "trial_org_1")

	ctx := context.Background()

	var gr dataPayloadApp
	//var gr dataPayload
	var buf bytes.Buffer
	if err := client.Run(ctx, req, &buf); err != nil {
		fmt.Printf("Error when running http request [%v]\n", err)
	}
	fmt.Printf("Received buf [%s]\n", buf.String())
	if err := json.NewDecoder(&buf).Decode(&gr); err != nil {
		errors.Wrap(err, "decoding response")
		fmt.Printf("Error when decoding json [%v]\n", err)
	}
	// fmt.Printf("Return value [%+v]\n", gr)
	spew.Dump(gr)
}

const sample_response2 string = `
{
  "data": {
    "saaras_db_application": [
      {
        "app_name": "trial_app_1",
        "fqdn": "www.example.com",
        "create_ts": "2019-01-12T21:30:23.258264+00:00",
        "update_ts": "2019-01-12T21:30:23.258264+00:00",
        "routessByappId": [
          {
            "route_prefix": "/",
            "create_ts": "2019-02-07T21:53:02.998742+00:00",
            "update_ts": "2019-02-07T21:53:02.998742+00:00",
            "routeMssByrouteId": [
              {
                "microservicesBymicroserviceId": {
                  "microservice_name": "trial_microservice_1",
                  "port": 443,
                  "create_ts": "2019-01-12T21:29:52.87347+00:00",
                  "update_ts": "2019-01-12T21:29:52.87347+00:00",
                  "clusterByclusterId": {
                    "cluster_name": "trial_cluster_1",
                    "external_ip": "1.1.1.1",
                    "create_ts": "2019-01-12T21:29:11.371402+00:00",
                    "update_ts": "2019-01-12T21:29:11.371402+00:00"
                  }
                }
              },
              {
                "microservicesBymicroserviceId": {
                  "microservice_name": "trial_microservice_5",
                  "port": 443,
                  "create_ts": "2019-01-13T01:16:55.074408+00:00",
                  "update_ts": "2019-01-13T01:16:55.074408+00:00",
                  "clusterByclusterId": {
                    "cluster_name": "trial_cluster_4",
                    "external_ip": "2.2.2.2",
                    "create_ts": "2019-01-13T01:12:14.097374+00:00",
                    "update_ts": "2019-01-13T01:12:14.097374+00:00"
                  }
                }
              }
            ]
          }
        ]
      }
    ]
  }
}
`

type cByclusterId struct {
	Cluster_name string
	External_ip  string
	Create_ts    string
	Update_ts    string
}

type mBymicroserviceId2 struct {
	Microservice_name  string
	Port               string
	Create_ts          string
	Update_ts          string
	ClusterByclusterId cByclusterId
}

type oneRouteMssByrouteId struct {
	MicroservicesBymicroserviceId mBymicroserviceId2
}

type oneRByappId struct {
	Route_prefix      string
	Create_ts         string
	Update_ts         string
	RouteMssByrouteId []oneRouteMssByrouteId
}

type oneSdbApp struct {
	App_name       string
	Fqdn           string
	Create_ts      string
	Update_ts      string
	RoutessByappId []oneRByappId
}

type dataSaarasDbApp struct {
	Saaras_db_application []oneSdbApp
}

const sample_output3 string = `
{
  data dataSaarasDbApp
}
`

// Start fulfills the g.Start contract.
// When stop is closed the http server will shutdown.
func (svc *Service) Start(stop <-chan struct{}) error {
	svc.Println("cloud fetcher started")
	for {
		time.Sleep(cloudPollIntervalSeconds * 1000 * time.Millisecond)
		svc.Println("Fetching configuration from cloud")
		fetchConfig(qApplicatons)
	}
}
