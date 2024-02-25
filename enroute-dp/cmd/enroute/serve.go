// Copyright © 2019 Heptio
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
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	clientset "github.com/saarasio/enroute/enroute-dp/apis/generated/clientset/versioned"
	enrouteinformers "github.com/saarasio/enroute/enroute-dp/apis/generated/informers/externalversions"
	"github.com/saarasio/enroute/enroute-dp/internal/contour"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/internal/debug"
	"github.com/saarasio/enroute/enroute-dp/internal/grpc"
	"github.com/saarasio/enroute/enroute-dp/internal/httpsvc"
	"github.com/saarasio/enroute/enroute-dp/internal/k8s"
	"github.com/saarasio/enroute/enroute-dp/internal/metrics"
	"github.com/saarasio/enroute/enroute-dp/internal/workgroup"
	"github.com/saarasio/enroute/enroute-dp/saaras"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	coreinformers "k8s.io/client-go/informers"
	gwinformers "sigs.k8s.io/gateway-api/pkg/client/informers/externalversions"
	gwclientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

// registerServe registers the serve subcommand and flags
// with the Application provided.
func registerServe(app *kingpin.Application) (*kingpin.CmdClause, *serveContext) {
	serve := app.Command("serve", "Serve xDS API traffic")

	// The precedence of configuration for contour serve is as follows:
	// config file, overridden by env vars, overridden by cli flags.
	// however, as -c is a cli flag, we don't know its valye til cli flags
	// have been parsed. To correct this ordering we assign a post parse
	// action to -c, then parse cli flags twice (see main.main). On the second
	// parse our action will return early, resulting in the precedence order
	// we want.
	var (
		configFile string
		parsed     bool
		ctx        serveContext
	)

	parseConfig := func(_ *kingpin.ParseContext) error {
		if parsed || configFile == "" {
			// if there is no config file supplied, or we've
			// already parsed it, return immediately.
			return nil
		}
		f, err := os.Open(configFile)
		if err != nil {
			return err
		}
		defer f.Close()
		dec := json.NewDecoder(f)
		parsed = true
		return dec.Decode(&ctx)
	}
	serve.Flag("config-path", "path to base configuration").Short('c').Action(parseConfig).ExistingFileVar(&configFile)

	serve.Flag("incluster", "use in cluster configuration.").BoolVar(&ctx.InCluster)
	serve.Flag("kubeconfig", "path to kubeconfig (if not in running inside a cluster)").Default(filepath.Join(os.Getenv("HOME"), ".kube", "config")).StringVar(&ctx.Kubeconfig)

	serve.Flag("http-address", "address the metrics http endpoint will bind to").Default("0.0.0.0").StringVar(&ctx.metricsAddr)
	serve.Flag("http-port", "port the metrics http endpoint will bind to").Default("8000").IntVar(&ctx.metricsPort)

	serve.Flag("xds-address", "xDS gRPC API address").Default("127.0.0.1").StringVar(&ctx.xdsAddr)
	serve.Flag("xds-port", "xDS gRPC API port").Default("8001").IntVar(&ctx.xdsPort)

	serve.Flag("stats-address", "Envoy /stats interface address").Default("0.0.0.0").StringVar(&ctx.statsAddr)
	serve.Flag("stats-port", "Envoy /stats interface port").Default("8002").IntVar(&ctx.statsPort)

	serve.Flag("rl-address", "Rate Limit gRPC API address").Default("127.0.0.1").StringVar(&ctx.rlAddr)
	serve.Flag("rl-port", "Rate Limit gRPC API port").Default("8003").IntVar(&ctx.rlPort)


	serve.Flag("debug-http-address", "address the debug http endpoint will bind to").Default("127.0.0.1").StringVar(&ctx.debugAddr)
	serve.Flag("debug-http-port", "port the debug http endpoint will bind to").Default("6060").IntVar(&ctx.debugPort)

	serve.Flag("contour-cafile", "CA bundle file name for serving gRPC with TLS").Envar("CONTOUR_CAFILE").StringVar(&ctx.caFile)
	serve.Flag("contour-cert-file", "Contour certificate file name for serving gRPC over TLS").Envar("CONTOUR_CERT_FILE").StringVar(&ctx.contourCert)
	serve.Flag("contour-key-file", "Contour key file name for serving gRPC over TLS").Envar("CONTOUR_KEY_FILE").StringVar(&ctx.contourKey)

	serve.Flag("gatewayhost-root-namespaces", "Restrict contour to searching these namespaces for root gateway hosts").StringVar(&ctx.rootNamespaces)

	serve.Flag("ingress-class-name", "EnRoute IngressClass name").StringVar(&ctx.ingressClass)

	serve.Flag("envoy-http-access-log", "Envoy HTTP access log").Default(contour.DEFAULT_HTTP_ACCESS_LOG).StringVar(&ctx.httpAccessLog)
	serve.Flag("envoy-https-access-log", "Envoy HTTPS access log").Default(contour.DEFAULT_HTTPS_ACCESS_LOG).StringVar(&ctx.httpsAccessLog)
	serve.Flag("envoy-service-http-address", "Kubernetes Service address for HTTP requests").Default("0.0.0.0").StringVar(&ctx.httpAddr)
	serve.Flag("envoy-service-https-address", "Kubernetes Service address for HTTPS requests").Default("0.0.0.0").StringVar(&ctx.httpsAddr)
	serve.Flag("envoy-service-http-port", "Kubernetes Service port for HTTP requests").Default("8080").IntVar(&ctx.httpPort)
	serve.Flag("envoy-service-https-port", "Kubernetes Service port for HTTPS requests").Default("8443").IntVar(&ctx.httpsPort)
	serve.Flag("use-proxy-protocol", "Use PROXY protocol for all listeners").BoolVar(&ctx.useProxyProto)

	serve.Flag("enroute-cp-ip", "IP address of enroute control plane").StringVar(&saaras.ENROUTE_CP_SERVER_IP)
	serve.Flag("enroute-cp-port", "Port of enroute control plane").StringVar(&saaras.ENROUTE_CP_SERVER_PORT)
	serve.Flag("enroute-name", "Name of this enroute instance - used to query its configuration").StringVar(&saaras.ENROUTE_NAME)
	serve.Flag("enroute-cp-proto", "Specify protocol to use - valid options are HTTP/HTTPS").StringVar(&saaras.ENROUTE_CP_PROTO)

	serve.Flag("mode-ingress", "Set to true to run enroute in ingress mode").BoolVar(&ctx.modeIngress)
	serve.Flag("enable-ratelimit", "Set to true to enable ratelimit").BoolVar(&ctx.ratelimitEnabled)
	serve.Flag("enable-acl", "Set to true to enable ratelimit").Default("true").BoolVar(&ctx.aclEnabled)

	return serve, &ctx
}

type serveContext struct {
	// contour's kubernetes client parameters
	InCluster  bool   `json:"incluster"`
	Kubeconfig string `json:"kubeconfig"`

	// contour's xds service parameters
	xdsAddr                         string
	xdsPort                         int
	caFile, contourCert, contourKey string

	// enroute's rate-limit service parameters
	rlAddr string
	rlPort int

	aclAddr string
	aclPort int

	// contour's debug handler parameters
	debugAddr string
	debugPort int

	// contour's metrics handler parameters
	metricsAddr string
	metricsPort int

	// gatewayhost root namespaces
	rootNamespaces string

	// ingress class
	ingressClass string

	// envoy's stats listener parameters
	statsAddr string
	statsPort int

	// envoy's listener parameters
	useProxyProto bool

	// envoy's http listener parameters
	httpAddr      string
	httpPort      int
	httpAccessLog string

	// envoy's https listener parameters
	httpsAddr      string
	httpsPort      int
	httpsAccessLog string

	modeIngress      bool
	ratelimitEnabled bool
	aclEnabled       bool
}

// tlsconfig returns a new *tls.Config. If the context is not properly configured
// for tls communication, tlsconfig returns nil.
func (ctx *serveContext) tlsconfig() *tls.Config {
	if ctx.caFile == "" && ctx.contourCert == "" && ctx.contourKey == "" {
		// tls not enabled
		return nil
	}
	// If one of the three TLS commands is not empty, they all must be not empty
	if !(ctx.caFile != "" && ctx.contourCert != "" && ctx.contourKey != "") {
		log.Fatal("You must supply all three TLS parameters - --contour-cafile, --contour-cert-file, --contour-key-file, or none of them.")
	}

	cert, err := tls.LoadX509KeyPair(ctx.contourCert, ctx.contourKey)
	check(err)

	ca, err := ioutil.ReadFile(ctx.caFile)
	check(err)

	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Fatalf("unable to append certificate in %s to CA pool", ctx.caFile)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
		Rand:         rand.Reader,
	}
}

// gatewayHostRootNamespaces returns a slice of namespaces restricting where
// contour should look for gatewayhost roots.
func (ctx *serveContext) gatewayHostRootNamespaces() []string {
	if strings.TrimSpace(ctx.rootNamespaces) == "" {
		return nil
	}
	var ns []string
	for _, s := range strings.Split(ctx.rootNamespaces, ",") {
		ns = append(ns, strings.TrimSpace(s))
	}
	return ns
}

// doServe runs the contour serve subcommand.
func doServe(log logrus.FieldLogger, ctx *serveContext) error {

	var mode_ingress bool
	mode_ingress = false

	// Read env to determine how we wish to run this
	// mode_ingress_env := os.Getenv("MODE_INGRESS")

	mode_ingress = ctx.modeIngress

	var client *kubernetes.Clientset
	var enrouteClient *clientset.Clientset
	var gwClient *gwclientset.Clientset

	client = nil

	// step 1. establish k8s client connection
	if mode_ingress {
		client, enrouteClient, gwClient = newClient(ctx.Kubeconfig, ctx.InCluster)
	}

	// step 2. create informers
	// note: 0 means resync timers are disabled

	var coreInformers coreinformers.SharedInformerFactory
	var enrouteInformers enrouteinformers.SharedInformerFactory
	var gwInformers gwinformers.SharedInformerFactory

	if mode_ingress {
		coreInformers = coreinformers.NewSharedInformerFactory(client, 0)
		enrouteInformers = enrouteinformers.NewSharedInformerFactory(enrouteClient, 0)
		gwInformers = gwinformers.NewSharedInformerFactory(gwClient, 0)
	}

	c := make(chan string)
	c2 := make(chan string)

	// Create a set of SharedInformerFactories for each root-gatewayhost namespace (if defined)
	var namespacedInformers []coreinformers.SharedInformerFactory
	for _, namespace := range ctx.gatewayHostRootNamespaces() {
		inf := coreinformers.NewSharedInformerFactoryWithOptions(client, 0, coreinformers.WithNamespace(namespace))
		namespacedInformers = append(namespacedInformers, inf)
	}

	// step 3. establish our (poorly named) gRPC cache handler.
	ch := contour.CacheHandler{
		ListenerVisitorConfig: contour.ListenerVisitorConfig{
			UseProxyProto:  ctx.useProxyProto,
			HTTPAddress:    ctx.httpAddr,
			HTTPPort:       ctx.httpPort,
			HTTPAccessLog:  ctx.httpAccessLog,
			HTTPSAddress:   ctx.httpsAddr,
			HTTPSPort:      ctx.httpsPort,
			HTTPSAccessLog: ctx.httpsAccessLog,
		},
		ListenerCache:     contour.NewListenerCache(ctx.statsAddr, ctx.statsPort),
		FieldLogger:       log.WithField("context", "CacheHandler"),
		GatewayHostStatus: &k8s.GatewayHostStatus{
			Client: enrouteClient,
			GatewayClient: gwClient,
		},
	}


	// step 4. wrap the gRPC cache handler in a k8s resource event handler.
	reh := contour.ResourceEventHandler{
		Notifier: &contour.HoldoffNotifier{
			Notifier:    &ch,
			FieldLogger: log.WithField("context", "HoldoffNotifier"),
		},
		KubernetesCache: dag.KubernetesCache{
			GatewayHostRootNamespaces: ctx.gatewayHostRootNamespaces(),
			FieldLogger:               log.WithField("context", "KubernetesCache"),
		},
		IngressClass: ctx.ingressClass,
		FieldLogger:  log.WithField("context", "resourceEventHandler"),
	}

	// step 5. register out resource event handler with the k8s informers.
	if mode_ingress {
		coreInformers.Core().V1().Services().Informer().AddEventHandler(&reh)
		coreInformers.Core().V1().Secrets().Informer().AddEventHandler(&reh)
		coreInformers.Networking().V1().Ingresses().Informer().AddEventHandler(&reh)
		enrouteInformers.Enroute().V1().GatewayHosts().Informer().AddEventHandler(&reh)
		enrouteInformers.Enroute().V1().ServiceRoutes().Informer().AddEventHandler(&reh)
		coreInformers.Core().V1().Secrets().Informer().AddEventHandler(&reh)

//		// Add informers for each root-gatewayhost namespaces
//		for _, inf := range namespacedInformers {
//			inf.Core().V1().Secrets().Informer().AddEventHandler(&reh)
//		}
//		// If root-gatewayhosts are not defined, then add the informer for all namespaces
//		if len(namespacedInformers) == 0 {
//			coreInformers.Core().V1().Secrets().Informer().AddEventHandler(&reh)
//		}
	}

	// step 5.5 register resource event handler with k8s informers
	if mode_ingress {
		enrouteInformers.Enroute().V1().RouteFilters().Informer().AddEventHandler(&reh)
		enrouteInformers.Enroute().V1().HttpFilters().Informer().AddEventHandler(&reh)
		enrouteInformers.Enroute().V1().TLSCertificateDelegations().Informer().AddEventHandler(&reh)
		enrouteInformers.Enroute().V1().GlobalConfigs().Informer().AddEventHandler(&reh)
	}

	gateway_api := true

	if gateway_api {
		// pkg/mod/sigs.k8s.io/gateway-api@v0.5.1/apis/v1beta1/gateway_types.go
		// https://github.com/kubernetes-sigs/gateway-api/blob/8a57d9a71583dba2e7c433bfbd04e8f4a98bf84f/pkg/client/informers/externalversions/generic.go
		// https://pkg.go.dev/sigs.k8s.io/gateway-api@v0.5.1/apis/v1beta1
		gwInformers.Gateway().V1beta1().Gateways().Informer().AddEventHandler(&reh)
		gwInformers.Gateway().V1beta1().GatewayClasses().Informer().AddEventHandler(&reh)
		gwInformers.Gateway().V1beta1().HTTPRoutes().Informer().AddEventHandler(&reh)
		gwInformers.Gateway().V1alpha2().UDPRoutes().Informer().AddEventHandler(&reh)
		//gwInformers.Gateway().V1alpha2().ReferenceGrants().Informer().AddEventHandler(&reh)
		gwInformers.Gateway().V1beta1().ReferenceGrants().Informer().AddEventHandler(&reh)
		//gwInformers.Gateway().V1alpha2().ReferencePolicies().Informer().AddEventHandler(&reh)
	}

	// step 6. endpoints updates are handled directly by the EndpointsTranslator
	// due to their high update rate and their orthogonal nature.
	et := &contour.EndpointsTranslator{
		FieldLogger: log.WithField("context", "endpointstranslator"),
	}

	pct := &contour.GlobalConfigTranslator{
		FieldLogger: log.WithField("context", "proxyconfigtranslator"),
		C:           c,
		C2:          c2,
	}

	if mode_ingress {
		coreInformers.Core().V1().Endpoints().Informer().AddEventHandler(et)
	}

	// step 6.5
	if mode_ingress {
		enrouteInformers.Enroute().V1().GlobalConfigs().Informer().AddEventHandler(pct)
	}

	// step 7. setup workgroup runner and register informers.
	var g workgroup.Group
	if mode_ingress {
		g.Add(startInformer(coreInformers, log.WithField("context", "coreinformers")))
		g.Add(startInformer(enrouteInformers, log.WithField("context", "enrouteinformers")))
		for _, inf := range namespacedInformers {
			g.Add(startInformer(inf, log.WithField("context", "corenamespacedinformers")))
		}

		if gateway_api {
			g.Add(startInformer(gwInformers, log.WithField("context", "gatewayinformers")))
		}
	}

	// step 8. setup prometheus registry and register base metrics.
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	registry.MustRegister(prometheus.NewGoCollector())

	// step 9. create metrics service and register with workgroup.
	metricsvc := metrics.Service{
		Service: httpsvc.Service{
			Addr:        ctx.metricsAddr,
			Port:        ctx.metricsPort,
			FieldLogger: log.WithField("context", "metricsvc"),
		},
		Client:   client,
		Registry: registry,
	}
	g.Add(metricsvc.Start)

	// step 10. create debug service and register with workgroup.
	debugsvc := debug.Service{
		Service: httpsvc.Service{
			Addr:        ctx.debugAddr,
			Port:        ctx.debugPort,
			FieldLogger: log.WithField("context", "debugsvc"),
		},
		KubernetesCache: &reh.KubernetesCache,
	}
	g.Add(debugsvc.Start)

	// step 11. register our custom metrics and plumb into cache handler
	// and resource event handler.
	metrics := metrics.NewMetrics(registry)
	ch.Metrics = metrics
	reh.Metrics = metrics

	// step 12. create grpc handler and register with workgroup.
	g.Add(func(stop <-chan struct{}) error {
		log := log.WithField("context", "grpc")
		addr := net.JoinHostPort(ctx.xdsAddr, strconv.Itoa(ctx.xdsPort))

		var l net.Listener
		var err error
		tlsconfig := ctx.tlsconfig()
		if tlsconfig != nil {
			log.Info("Setting up TLS for gRPC")
			l, err = tls.Listen("tcp", addr, tlsconfig)
			if err != nil {
				return err
			}
		} else {
			l, err = net.Listen("tcp", addr)
			if err != nil {
				return err
			}
		}

		s := grpc.NewAPI(log, map[string]grpc.Resource{
			ch.ClusterCache.TypeURL():  &ch.ClusterCache,
			ch.RouteCache.TypeURL():    &ch.RouteCache,
			ch.ListenerCache.TypeURL(): &ch.ListenerCache,
			et.TypeURL():               et,
			ch.SecretCache.TypeURL():   &ch.SecretCache,
		})
		log.Println("started")
		defer log.Println("stopped")
		return s.Serve(l)
	})

	if ctx.ratelimitEnabled {
		SetupRateLimit(&g, log, ctx, c)
	}

	if !mode_ingress {
		wl := log.WithField("context", "saaras")
		saarasCloudCache := saaras.SaarasCloudCache{}
		saaras.WatchCloudGatewayHost(&g, wl, &reh, et, pct, &saarasCloudCache)
	}

	// step 13. GO!
	return g.Run()
}

type informer interface {
	WaitForCacheSync(stopCh <-chan struct{}) map[reflect.Type]bool
	Start(stopCh <-chan struct{})
}

func startInformer(inf informer, log logrus.FieldLogger) func(stop <-chan struct{}) error {
	return func(stop <-chan struct{}) error {
		log.Println("waiting for cache sync")
		inf.WaitForCacheSync(stop)

		log.Println("started")
		defer log.Println("stopping")
		inf.Start(stop)
		<-stop
		return nil
	}
}
