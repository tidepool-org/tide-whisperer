# go-common Makefile

# Generates client files
generate:
	swagger-cli bundle ../TidepoolApi/reference/summary.v1.yaml -o ./spec/summary.v1.yaml -t yaml
	oapi-codegen -package=api -generate=types spec/summary.v1.yaml > clients/summary/types.go
	oapi-codegen -package=api -generate=client spec/summary.v1.yaml > clients/summary/client.go
	cd clients/summary && go generate ./...
