# go-common Makefile

ifeq ($(CI),)
GO_BUILD_FLAGS =
GO_TEST_FLAGS =
else
GO_BUILD_FLAGS = -v
GO_TEST_FLAGS = -v
endif

.PHONY: build
build:
	go build $(GO_BUILD_FLAGS) ./...

.PHONY: test
test:
	go test $(GO_TEST_FLAGS) ./...

.PHONY: test-cover
test-cover:
	go test -coverprofile cover.out $(GO_TEST_FLAGS) ./...

# Generates client files
generate:
	swagger-cli bundle ../TidepoolApi/reference/summary.v1.yaml -o ./spec/summary.v1.yaml -t yaml
	oapi-codegen -package=api -generate=types spec/summary.v1.yaml > clients/summary/types.go
	oapi-codegen -package=api -generate=client spec/summary.v1.yaml > clients/summary/client.go
	cd clients/summary && go generate ./...
