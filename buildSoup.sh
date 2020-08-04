#!/bin/bash -eu
# Generate Soups

DEPLOY_DOC=${DEPLOY_DOC:-docs/soup}
TRAVIS_REPO_SLUG=${TRAVIS_REPO_SLUG:-owner/repo}
APP="${TRAVIS_REPO_SLUG#*/}"
GO111MODULE=on

if [ -z "${TRAVIS_TAG+x}" ]; then
  TRAVIS_TAG=0.0.0
fi
if [ -d ${DEPLOY_DOC} ]; then
  rm -rf ./${DEPLOY_DOC}
fi

VERSION=${TRAVIS_TAG/dblp./}
echo "# SOUPs List for ${APP}@${VERSION}" > soup.md 

go list -f '## {{printf "%s \n\t* description: \n\t* version: %s\n\t* webSite: https://%s\n\t* sources:" .Path .Version .Path}}' -m all >> soup.md && \
	mkdir -p ./${DEPLOY_DOC}/${APP} && \
	mv soup.md ./${DEPLOY_DOC}/${APP}/${APP}-${VERSION}-soup.md
