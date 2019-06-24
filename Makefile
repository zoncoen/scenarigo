SHELL := /bin/bash

ROOT_DIR := $(realpath $(dir $(lastword $(MAKEFILE_LIST))))
BIN_DIR := $(ROOT_DIR)/.bin
PROTO_DIR := $(ROOT_DIR)/testdata/proto
GEN_PB_DIR := $(ROOT_DIR)/testdata/gen/pb

PROTOC_OPTION := -I$(PROTO_DIR)
PROTOC_GO_OPTION := $(PROTOC_OPTION) --plugin=${BIN_DIR}/protoc-gen-go --go_out=plugins=grpc:${GEN_PB_DIR}

${BIN_DIR}/protoc-gen-go:
	@mkdir -p ${BIN_DIR}
	@go build -o ${BIN_DIR}/protoc-gen-go github.com/golang/protobuf/protoc-gen-go

${BIN_DIR}/goprotoyamltag:
	@mkdir -p ${BIN_DIR}
	@go build -o ${BIN_DIR}/goprotoyamltag github.com/zoncoen/goprotoyamltag

${BIN_DIR}/gotypenames:
	@mkdir -p ${BIN_DIR}
	@go build -o ${BIN_DIR}/gotypenames github.com/zoncoen/gotypenames

${BIN_DIR}/mockgen:
	@mkdir -p ${BIN_DIR}
	@go build -o ${BIN_DIR}/mockgen github.com/golang/mock/mockgen

.PHONY: gen
gen: gen-proto

.PHONY: gen-proto
gen-proto: ${BIN_DIR}/protoc-gen-go
	@rm -rf $(GEN_PB_DIR)
	@mkdir -p $(GEN_PB_DIR)
	@find $(PROTO_DIR) -name '*.proto' | xargs -P8 protoc $(PROTOC_GO_OPTION)
	@make add-yaml-tag
	@make gen-mock

.PHONY: add-yaml-tag
add-yaml-tag: ${BIN_DIR}/goprotoyamltag
	@for file in $$(find $(GEN_PB_DIR) -name '*.pb.go'); do \
		echo "add yaml tag $$file"; \
		${BIN_DIR}/goprotoyamltag --filename $$file -w; \
	done

.PHONY: gen-mock
gen-mock: ${BIN_DIR}/gotypenames ${BIN_DIR}/mockgen
	@for file in $$(find $(GEN_PB_DIR) -name '*.pb.go'); do \
		echo "generate mock for $$file"; \
		${BIN_DIR}/gotypenames --filename $$file --only-exported --types interface | xargs -ISTRUCT -L1 -P8 ${BIN_DIR}/mockgen -source $$file -package $$(basename $$(dirname $$file)) -destination $$(dirname $$file)/$$(basename $${file%.pb.go})_mock.go; \
	done
