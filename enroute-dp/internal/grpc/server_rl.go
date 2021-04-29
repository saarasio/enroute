// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

package grpc

import (
	"context"

	"google.golang.org/grpc"

	rl "github.com/envoyproxy/go-control-plane/envoy/service/ratelimit/v2"
	"github.com/sirupsen/logrus"
)

func NewAPIRateLimit(log logrus.FieldLogger, c chan string) *grpc.Server {
	opts := []grpc.ServerOption{
		// By default the Go grpc library defaults to a value of ~100 streams per
		// connection. This number is likely derived from the HTTP/2 spec:
		// https://http2.github.io/http2-spec/#SettingValues
		// We need to raise this value because Envoy will open one EDS stream per
		// CDS entry. There doesn't seem to be a penalty for increasing this value,
		// so set it the limit similar to envoyproxy/go-control-plane#70.
		grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams),
	}
	g := grpc.NewServer(opts...)
	rls := &ratelimitServer{FieldLogger: log}
	rl.RegisterRateLimitServiceServer(g, rls)
	return g
}

type ratelimitServer struct {
	rl.RateLimitServiceServer
	logrus.FieldLogger
}

func (s *ratelimitServer) getRateLimit(requestsPerUnit uint32, unit rl.RateLimitResponse_RateLimit_Unit) *rl.RateLimitResponse_RateLimit {
	return &rl.RateLimitResponse_RateLimit{RequestsPerUnit: requestsPerUnit, Unit: unit}
}

func (s *ratelimitServer) rateLimitDescriptor() *rl.RateLimitResponse_DescriptorStatus {
	l := s.getRateLimit(10, rl.RateLimitResponse_RateLimit_SECOND)
	return &rl.RateLimitResponse_DescriptorStatus{Code: rl.RateLimitResponse_OK, CurrentLimit: l, LimitRemaining: 5}
}

func (s *ratelimitServer) ShouldRateLimit(c context.Context, req *rl.RateLimitRequest) (*rl.RateLimitResponse, error) {
	s.Debugf("Received rate limit request +[%v]\n", req)
	response := &rl.RateLimitResponse{}
	response.Statuses = make([]*rl.RateLimitResponse_DescriptorStatus, len(req.Descriptors))
	finalCode := rl.RateLimitResponse_OK
	for i := range req.Descriptors {
		descriptorStatus := s.rateLimitDescriptor()
		response.Statuses[i] = descriptorStatus
		if descriptorStatus.Code == rl.RateLimitResponse_OVER_LIMIT {
			finalCode = descriptorStatus.Code
		}
	}

	response.OverallCode = finalCode
	return response, nil
}
