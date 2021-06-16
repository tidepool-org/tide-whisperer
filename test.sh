#!/bin/sh

which go-junit-report
if [ "$?" != "0" ]; then
  go get -u "github.com/jstemmer/go-junit-report"
fi

go test -v -race ./... 2>&1 > test-result.txt
RET=$?
cat test-result.txt
cat test-result.txt | go-junit-report > test-report.xml
cat test-report.xml

exit $RET
