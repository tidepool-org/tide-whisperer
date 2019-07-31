#!/bin/sh -eu

rm -rf dist
mkdir dist

echo "Run dep ensure"
$GOPATH/bin/dep ensure
$GOPATH/bin/dep check

echo "Build tide-whisperer"
go build -o dist/tide-whisperer tide-whisperer.go
cp start.sh dist/
cp env.sh dist/
