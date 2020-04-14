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
if [ -n "${TRAVIS_TAG:-}" ]; then
    APP="${TRAVIS_REPO_SLUG#*/}"
    APP_TAG="${APP}-${TRAVIS_TAG}"
    mkdir docs/openapi
    mv docs/swagger.json docs/openapi/${APP_TAG}-swagger.json
fi
