#!/bin/sh -eu

rm -rf dist
mkdir dist
go get gopkg.in/mgo.v2
go build -o dist/tide-whisperer tide-whisperer.go
cp start.sh dist/
cp env.sh dist/
