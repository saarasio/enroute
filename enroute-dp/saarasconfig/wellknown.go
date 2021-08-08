package saarasconfig

import (
	_ "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1"
)

const (
    HTTPFilterCompressorGzip    = "type.googleapis.com/envoy.extensions.compression.gzip.compressor.v3.Gzip"
    HTTPFilterRouter  = "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
    HTTPFilterGrpcWeb = "type.googleapis.com/envoy.extensions.filters.http.grpc_web.v3.GrpcWeb"
)
