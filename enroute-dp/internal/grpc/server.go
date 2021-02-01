// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

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

// Package grpc provides a gRPC implementation of the Envoy v2 xDS API.
package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	"github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	"github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	"github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v3"
	"github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	"github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"

	rl "github.com/envoyproxy/go-control-plane/envoy/service/ratelimit/v2"
	"github.com/sirupsen/logrus"
)

const (
	// somewhat arbitrary limit to handle many, many, EDS streams
	grpcMaxConcurrentStreams = 1 << 20
)

// NewAPI returns a *grpc.Server which responds to the Envoy v2 xDS gRPC API.
func NewAPI(log logrus.FieldLogger, resources map[string]Resource) *grpc.Server {
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
	s := &grpcServer{
		xdsHandler{
			FieldLogger: log,
			resources:   resources,
		},
	}

	rls := &ratelimitServer{}

	envoy_service_cluster_v3.RegisterClusterDiscoveryServiceServer(g, s)
	envoy_service_endpoint_v3.RegisterEndpointDiscoveryServiceServer(g, s)
	envoy_service_listener_v3.RegisterListenerDiscoveryServiceServer(g, s)
	envoy_service_route_v3.RegisterRouteDiscoveryServiceServer(g, s)
	envoy_service_secret_v3.RegisterSecretDiscoveryServiceServer(g, s)
	rl.RegisterRateLimitServiceServer(g, rls)
	return g
}

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
	rls := &ratelimitServer{}
	rl.RegisterRateLimitServiceServer(g, rls)
	return g
}

// grpcServer implements the LDS, RDS, CDS, and EDS, gRPC endpoints.
type grpcServer struct {
	xdsHandler
}

type ratelimitServer struct {
	rl.RateLimitServiceServer
}

func (s *grpcServer) FetchClusters(_ context.Context, req *envoy_service_discovery_v3.DiscoveryRequest) (*envoy_service_discovery_v3.DiscoveryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "FetchClusters unimplemented")
}

func (s *grpcServer) FetchEndpoints(_ context.Context, req *envoy_service_discovery_v3.DiscoveryRequest) (*envoy_service_discovery_v3.DiscoveryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "FetchEndpoints unimplemented")
}

func (s *grpcServer) DeltaEndpoints(envoy_service_endpoint_v3.EndpointDiscoveryService_DeltaEndpointsServer) error {
	return status.Errorf(codes.Unimplemented, "DeltaEndpoints unimplemented")
}

func (s *grpcServer) FetchListeners(_ context.Context, req *envoy_service_discovery_v3.DiscoveryRequest) (*envoy_service_discovery_v3.DiscoveryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "FetchListeners unimplemented")
}

func (s *grpcServer) DeltaListeners(envoy_service_listener_v3.ListenerDiscoveryService_DeltaListenersServer) error {
	return status.Errorf(codes.Unimplemented, "DeltaListeners unimplemented")
}

func (s *grpcServer) FetchRoutes(_ context.Context, req *envoy_service_discovery_v3.DiscoveryRequest) (*envoy_service_discovery_v3.DiscoveryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "FetchRoutes unimplemented")
}

func (s *grpcServer) FetchSecrets(_ context.Context, req *envoy_service_discovery_v3.DiscoveryRequest) (*envoy_service_discovery_v3.DiscoveryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "FetchSecrets unimplemented")
}

func (s *grpcServer) DeltaSecrets(envoy_service_secret_v3.SecretDiscoveryService_DeltaSecretsServer) error {
	return status.Errorf(codes.Unimplemented, "DeltaSecrets unimplemented")
}

func (s *grpcServer) StreamClusters(srv envoy_service_cluster_v3.ClusterDiscoveryService_StreamClustersServer) error {
	return s.stream(srv)
}

func (s *grpcServer) StreamEndpoints(srv envoy_service_endpoint_v3.EndpointDiscoveryService_StreamEndpointsServer) error {
	return s.stream(srv)
}

func (s *grpcServer) StreamLoadStats(srv envoy_service_load_stats_v3.LoadReportingService_StreamLoadStatsServer) error {
	return status.Errorf(codes.Unimplemented, "StreamLoadStats unimplemented")
}

func (s *grpcServer) DeltaClusters(envoy_service_cluster_v3.ClusterDiscoveryService_DeltaClustersServer) error {
	return status.Errorf(codes.Unimplemented, "IncrementalClusters unimplemented")
}

func (s *grpcServer) DeltaRoutes(envoy_service_route_v3.RouteDiscoveryService_DeltaRoutesServer) error {
	return status.Errorf(codes.Unimplemented, "IncrementalRoutes unimplemented")
}

func (s *grpcServer) StreamListeners(srv envoy_service_listener_v3.ListenerDiscoveryService_StreamListenersServer) error {
	return s.stream(srv)
}

func (s *grpcServer) StreamRoutes(srv envoy_service_route_v3.RouteDiscoveryService_StreamRoutesServer) error {
	return s.stream(srv)
}

func (s *grpcServer) StreamSecrets(srv envoy_service_secret_v3.SecretDiscoveryService_StreamSecretsServer) error {
	return s.stream(srv)
}

func (s *ratelimitServer) getRateLimit(requestsPerUnit uint32, unit rl.RateLimitResponse_RateLimit_Unit) *rl.RateLimitResponse_RateLimit {
	return &rl.RateLimitResponse_RateLimit{RequestsPerUnit: requestsPerUnit, Unit: unit}
}

func (s *ratelimitServer) rateLimitDescriptor() *rl.RateLimitResponse_DescriptorStatus {
	l := s.getRateLimit(10, rl.RateLimitResponse_RateLimit_SECOND)
	return &rl.RateLimitResponse_DescriptorStatus{Code: rl.RateLimitResponse_OK, CurrentLimit: l, LimitRemaining: 5}
}

//func (this *rateLimitServer) DoLimit(
//    request *rl.RateLimitRequest) []*rl.RateLimitResponse_DescriptorStatus {
//}

//func (this *service) shouldRateLimitWorker(
//    ctx context.Context, request *pb.RateLimitRequest) *pb.RateLimitResponse {
//
//    checkServiceErr(request.Domain != "", "rate limit domain must not be empty")
//    checkServiceErr(len(request.Descriptors) != 0, "rate limit descriptor list must not be empty")
//
//    snappedConfig := this.GetCurrentConfig()
//    checkServiceErr(snappedConfig != nil, "no rate limit configuration loaded")
//
//    limitsToCheck := make([]*config.RateLimit, len(request.Descriptors))
//    for i, descriptor := range request.Descriptors {
//        limitsToCheck[i] = snappedConfig.GetLimit(ctx, request.Domain, descriptor)
//    }
//
//    responseDescriptorStatuses := this.cache.DoLimit(ctx, request, limitsToCheck)
//    assert.Assert(len(limitsToCheck) == len(responseDescriptorStatuses))
//
//    response := &pb.RateLimitResponse{}
//    response.Statuses = make([]*pb.RateLimitResponse_DescriptorStatus, len(request.Descriptors))
//    finalCode := pb.RateLimitResponse_OK
//    for i, descriptorStatus := range responseDescriptorStatuses {
//        response.Statuses[i] = descriptorStatus
//        if descriptorStatus.Code == pb.RateLimitResponse_OVER_LIMIT {
//            finalCode = descriptorStatus.Code
//        }
//    }
//
//    response.OverallCode = finalCode
//    return response
//}

func (s *ratelimitServer) ShouldRateLimit(c context.Context, req *rl.RateLimitRequest) (*rl.RateLimitResponse, error) {
	fmt.Printf("Received rate limit request +[%v]\n", req)
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
