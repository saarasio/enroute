module github.com/saarasio/enroute/examples/helloworld

go 1.12

require (
	github.com/golang/protobuf v1.3.0
	google.golang.org/grpc v1.19.0
)

replace github.com/saarasio/enroute/examples/ => ../examples
