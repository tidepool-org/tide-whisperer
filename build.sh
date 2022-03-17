#!/bin/sh -e

rm -rf dist
mkdir dist

TARGETPLATFORM=$1
if ["$TARGETPLATFORM"="linux/arm64"]; then
    export GOOS=darwin
    export GOARCH=arm64
    export CGO_ENABLED=0
else
    export CGO_ENABLED=1
fi

# generate version number
if [ -n "${TRAVIS_TAG:-}" ]; then
    VERSION_BASE=${TRAVIS_TAG}  
else 
    VERSION_BASE=$(git describe --abbrev=0 --tags 2> /dev/null || echo 'dblp.0.0.0')
fi
VERSION_SHORT_COMMIT=$(git rev-parse --short HEAD)
VERSION_FULL_COMMIT=$(git rev-parse HEAD)

GO_COMMON_PATH="github.com/tidepool-org/go-common"

echo "Build tide-whisperer $VERSION_BASE+$VERSION_FULL_COMMIT"
go mod tidy
go build -ldflags "-X $GO_COMMON_PATH/clients/version.ReleaseNumber=$VERSION_BASE \
    -X $GO_COMMON_PATH/clients/version.FullCommit=$VERSION_FULL_COMMIT \
    -X $GO_COMMON_PATH/clients/version.ShortCommit=$VERSION_SHORT_COMMIT" \
    -o dist/tide-whisperer tide-whisperer.go
cp start.sh dist/
cp env.sh dist/
