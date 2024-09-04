package proto

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/grpcreflect"
)

// ReflectionClient is a client to access reflection service.
type ReflectionClient struct {
	client *grpcreflect.Client
}

// NewReflectionClient creates a new reflection client.
func NewReflectionClient(ctx context.Context, cc grpc.ClientConnInterface) *ReflectionClient {
	return &ReflectionClient{
		client: grpcreflect.NewClientAuto(ctx, cc),
	}
}

// ListServices lists all service names.
func (c *ReflectionClient) ListServices() ([]protoreflect.FullName, error) {
	resp, err := c.client.ListServices()
	if err != nil {
		return nil, err
	}
	names := make([]protoreflect.FullName, len(resp))
	for i, s := range resp {
		names[i] = protoreflect.FullName(s)
	}
	return names, nil
}

// ResolveService resolves a service descriptor by the given name.
func (c *ReflectionClient) ResolveService(name protoreflect.FullName) (protoreflect.ServiceDescriptor, error) {
	fd, err := c.client.ResolveService(string(name))
	if err != nil {
		return nil, err
	}
	return fd.UnwrapService(), nil
}
