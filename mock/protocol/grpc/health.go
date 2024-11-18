package grpc

import (
	"context"

	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type healthServer struct{}

// Check implements healthpb.HealthServer interface.
func (s *healthServer) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{
		Status: healthpb.HealthCheckResponse_SERVING,
	}, nil
}

// Watch implements healthpb.HealthServer interface.
func (s *healthServer) Watch(req *healthpb.HealthCheckRequest, streams grpc.ServerStreamingServer[healthpb.HealthCheckResponse]) error {
	return streams.SendMsg(&healthpb.HealthCheckResponse{
		Status: healthpb.HealthCheckResponse_SERVING,
	})
}
