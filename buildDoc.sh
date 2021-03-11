#!/bin/sh -eu
# Generate OpenAPI documentation
GOPATH=${GOPATH:-~/go}
echo "Using GOPATH: ${GOPATH}"

if [ ! -x "$GOPATH/bin/swag" ]; then
  echo "Getting swag..."
  go get -u github.com/swaggo/swag/cmd/swag
fi

$GOPATH/bin/swag --version
$GOPATH/bin/swag init --parseDependency --generalInfo tide-whisperer.go --output docs

# When tag is present, openapi doc is renamed before being deployed to S3
# It is stored in a new directory that will be used as source by the Travis deploy step
if [ -n "${TRAVIS_TAG:-}" ]; then
    APP="tide-whisperer"
    APP_TAG="${APP}-${TRAVIS_TAG/dblp./}"
    mkdir docs/openapi
    mv docs/swagger.json docs/openapi/${APP_TAG}-swagger.json
    # If this is not a release candidate but a "true" release, we consider this doc is the latest
    # we create a copy named "latest" to be consumed by documentation website using SwaggerUI
    if echo ${TRAVIS_TAG} | grep -Eq '[0-9]+\.[0-9]+\.[0-9]+'; then
      cp docs/openapi/${APP_TAG}-swagger.json docs/openapi/${APP}-latest-swagger.json
    fi
fi
