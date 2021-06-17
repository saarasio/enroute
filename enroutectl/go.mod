module github.com/saarasio/enroutectl

require (
	github.com/ghodss/yaml v1.0.0
	github.com/go-openapi/loads v0.19.5
	github.com/go-openapi/spec v0.19.8
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.2
	github.com/google/uuid v1.1.2
	github.com/mitchellh/go-homedir v1.0.0
	github.com/saarasio/enroute/enroute-dp v0.0.0-00010101000000-000000000000
	github.com/saarasio/enroute/enroutectl v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v0.0.3
	github.com/spf13/viper v1.3.1
	k8s.io/apimachinery v0.21.0
)

replace github.com/saarasio/enroute/enroutectl => ./

replace github.com/saarasio/enroute/enroute-dp => ../enroute-dp/

go 1.15
