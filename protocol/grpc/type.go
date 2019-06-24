package grpc

import (
	"context"
	"reflect"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
)

var (
	typeContext  = reflect.TypeOf((*context.Context)(nil)).Elem()
	typeMessage  = reflect.TypeOf((*proto.Message)(nil)).Elem()
	typeCallOpts = reflect.TypeOf([]grpc.CallOption(nil))
	typeError    = reflect.TypeOf((*error)(nil)).Elem()
)
