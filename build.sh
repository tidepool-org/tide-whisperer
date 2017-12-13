#!/bin/sh -eu

rm -rf dist
mkdir dist
go build -o dist/tide-whisperer tide-whisperer.go
cp start.sh dist/
cp env.sh dist/
