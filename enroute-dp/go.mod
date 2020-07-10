module github.com/saarasio/enroute/enroute-dp

go 1.12

require (
	cloud.google.com/go v0.37.4 // indirect
	github.com/client9/misspell v0.3.4
	github.com/davecgh/go-spew v1.1.1
	github.com/envoyproxy/go-control-plane v0.9.0
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/gogo/protobuf v1.2.2-0.20190730201129-28a6bbf47e48
	github.com/golang/protobuf v1.3.2
	github.com/google/go-cmp v0.4.0
	github.com/gordonklaus/ineffassign v0.0.0-20180909121442-1003c8bd00dc
	github.com/gorilla/mux v1.6.2
	github.com/kavu/go_reuseport v1.4.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kisielk/errcheck v1.2.0
	github.com/lyft/goruntime v0.2.3
	github.com/lyft/gostats v0.3.6
	github.com/mdempsky/unconvert v0.0.0-20190325185700-2f5dc3378ed3
	github.com/mediocregopher/radix.v2 v0.0.0-20181115013041-b67df6e626f9
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/sirupsen/logrus v1.4.2
	golang.org/x/net v0.0.0-20191004110552-13f9640d40b9
	google.golang.org/grpc v1.25.1
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.2.7
	honnef.co/go/tools v0.0.0-20190523083050-ea95bdfd59fc
	istio.io/gogo-genproto v0.0.0-20190731221249-06e20ada0df2 // indirect
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.0.0-20190912054826-cd179ad6a269
	mvdan.cc/unparam v0.0.0-20190310220240-1b9ccfa71afe
)

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
