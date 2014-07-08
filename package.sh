#! /bin/bash -eu

export GOPATH=`pwd`

OUTDIR=dist

rm -rf ${OUTDIR}
mkdir ${OUTDIR}
go build -o ${OUTDIR}/tide-whisperer github.com/tidepool-org/tide-whisperer

cp start.sh ${OUTDIR}/