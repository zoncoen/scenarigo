package proto

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/google/go-cmp/cmp"
	testpb "github.com/zoncoen/scenarigo/testdata/gen/pb/test"
)

func TestReflectionClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr := startServer(t)
	cc, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}
	defer cc.Close()

	client := NewReflectionClient(ctx, cc)
	names, err := client.ListServices()
	if err != nil {
		t.Fatalf("failed to get services: %s", err)
	}
	if diff := cmp.Diff([]protoreflect.FullName{
		"grpc.reflection.v1.ServerReflection",
		"grpc.reflection.v1alpha.ServerReflection",
		"scenarigo.testdata.test.Test",
	}, names); diff != "" {
		t.Fatalf("request differs (-want +got):\n%s", diff)
	}

	if _, err := client.ResolveService(protoreflect.FullName("scenarigo.testdata.test.Test")); err != nil {
		t.Fatalf("failed to get service: %s", err)
	}
}

func startServer(t *testing.T) string {
	t.Helper()
	s := grpc.NewServer()
	testpb.RegisterTestServer(s, nil)
	reflection.Register(s)

	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	t.Cleanup(func() {
		ln.Close()
	})

	go func() {
		_ = s.Serve(ln)
	}()
	t.Cleanup(func() {
		s.Stop()
	})

	return ln.Addr().String()
}
