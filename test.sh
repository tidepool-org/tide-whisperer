#!/bin/sh -eu

for D in $(find . -name '*_test.go' ! -path './.gvm_local/*' ! -path './vendor/*' | cut -f2 -d'/' | uniq); do
    (cd ${D}; go test -v)
done
