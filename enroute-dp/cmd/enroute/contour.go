// Copyright © 2017 Heptio
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

package main

import (
	"fmt"
	"os"

	resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	clientset "github.com/saarasio/enroute/enroute-dp/apis/generated/clientset/versioned"
	gwclientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	"github.com/saarasio/enroute/enroute-dp/internal/logger"
	_ "github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func init() {
	logger.EL = logger.EnrouteLogger{}
	logger.EL.Initialize()
}

func main() {

	// Use our logger
	el := logger.EL


	app := kingpin.New("enroute", "enroute agent to control envoy proxy")

	bootstrap, bootstrapCtx := registerBootstrap(app)

	certgenApp, certgenConfig := registerCertGen(app)

	cli := app.Command("cli", "A CLI client for enroute agent")
	var client Client
	cli.Flag("enroute", "enroute host:port.").Default("127.0.0.1:8001").StringVar(&client.ContourAddr)
	cli.Flag("cafile", "CA bundle file for connecting to a TLS-secured Contour").Envar("CLI_CAFILE").StringVar(&client.CAFile)
	cli.Flag("cert-file", "Client certificate file for connecting to a TLS-secured Contour").Envar("CLI_CERT_FILE").StringVar(&client.ClientCert)
	cli.Flag("key-file", "Client key file for connecting to a TLS-secured Contour").Envar("CLI_KEY_FILE").StringVar(&client.ClientKey)

	var resources []string
	cds := cli.Command("cds", "watch services.")
	cds.Arg("resources", "CDS resource filter").StringsVar(&resources)
	eds := cli.Command("eds", "watch endpoints.")
	eds.Arg("resources", "EDS resource filter").StringsVar(&resources)
	lds := cli.Command("lds", "watch listerners.")
	lds.Arg("resources", "LDS resource filter").StringsVar(&resources)
	rds := cli.Command("rds", "watch routes.")
	rds.Arg("resources", "RDS resource filter").StringsVar(&resources)
	sds := cli.Command("sds", "watch secrets.")
	sds.Arg("resources", "SDS resource filter").StringsVar(&resources)

	serve, serveCtx := registerServe(app)

	args := os.Args[1:]
	switch kingpin.MustParse(app.Parse(args)) {
	case bootstrap.FullCommand():
		doBootstrap(bootstrapCtx)
	case certgenApp.FullCommand():
		doCertgen(certgenConfig)
	case cds.FullCommand():
		stream := client.ClusterStream()
		watchstream(stream, resource.ClusterType, resources)
	case eds.FullCommand():
		stream := client.EndpointStream()
		watchstream(stream, resource.EndpointType, resources)
	case lds.FullCommand():
		stream := client.ListenerStream()
		watchstream(stream, resource.ListenerType, resources)
	case rds.FullCommand():
		stream := client.RouteStream()
		watchstream(stream, resource.RouteType, resources)
	case sds.FullCommand():
		stream := client.RouteStream()
		watchstream(stream, resource.SecretType, resources)
	case serve.FullCommand():
		// parse args a second time so cli flags are applied
		// on top of any values sourced from -c's config file.
		_, err := app.Parse(args)
		check(err)
		el.ELogger.Infof("args: %v", args)
		doServe(el.ELogger, serveCtx)
	default:
		app.Usage(args)
		os.Exit(2)
	}
}

func newClient(kubeconfig string, inCluster bool) (*kubernetes.Clientset, *clientset.Clientset, *gwclientset.Clientset) {
	var err error
	var config *rest.Config
	if kubeconfig != "" && !inCluster {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		check(err)
	} else {
		config, err = rest.InClusterConfig()
		check(err)
	}

	client, err := kubernetes.NewForConfig(config)
	check(err)
	enrouteClient, err := clientset.NewForConfig(config)
	check(err)
	gwClient, err := gwclientset.NewForConfig(config)
	check(err)
	return client, enrouteClient, gwClient
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
