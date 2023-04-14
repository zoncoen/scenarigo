package grpc

import (
	"context"
	"reflect"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

var (
	typeContext  = reflect.TypeOf((*context.Context)(nil)).Elem()
	typeMessage  = reflect.TypeOf((*proto.Message)(nil)).Elem()
	typeCallOpts = reflect.TypeOf([]grpc.CallOption(nil))
)
