#! /bin/bash -eu

export GOPATH=`pwd`

PROJECT=tide-whisperer
PACKAGE=github.com/tidepool-org/${PROJECT}
OUTDIR=dist
rm -rf ${OUTDIR}
mkdir ${OUTDIR}

jd get ${PACKAGE} || true
go build -o ${OUTDIR}/${PROJECT} ${PACKAGE}

cp start.sh ${OUTDIR}/