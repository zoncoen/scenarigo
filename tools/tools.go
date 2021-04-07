// +build tools

package tools

import (
	_ "github.com/Songmu/gocredits/cmd/gocredits"
	_ "github.com/git-chglog/git-chglog/cmd/git-chglog"
	_ "github.com/golang/mock/mockgen"
	_ "github.com/google/go-licenses"
	_ "github.com/x-motemen/gobump/cmd/gobump"
	_ "github.com/zoncoen/goprotoyamltag"
	_ "github.com/zoncoen/gotypenames"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
