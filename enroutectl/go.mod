module github.com/saarasio/enroutectl

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/envoyproxy/go-control-plane v0.9.5
	github.com/ghodss/yaml v1.0.0
	github.com/go-openapi/analysis v0.19.10
	github.com/go-openapi/loads v0.19.5
	github.com/go-openapi/spec v0.19.8
	github.com/go-openapi/swag v0.19.9
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/golang/protobuf v1.4.2
	github.com/google/go-cmp v0.4.0
	github.com/google/uuid v1.1.1
	github.com/hashicorp/go-multierror v1.1.0
	github.com/kr/pretty v0.1.0
	github.com/mitchellh/go-homedir v1.0.0
	github.com/pkg/errors v0.8.1
	github.com/saarasio/enroute/enroute-dp v0.0.0-00010101000000-000000000000
	github.com/saarasio/enroute/enroutectl v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.3
	github.com/spf13/viper v1.3.1
	github.com/stretchr/testify v1.4.0
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v0.17.0
)

replace github.com/saarasio/enroute/enroutectl => ./

replace github.com/saarasio/enroute/enroute-dp => ../enroute-dp/

go 1.13
