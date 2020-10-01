#!/bin/sh -e

wget -q -O artifact_go.sh 'https://raw.githubusercontent.com/tidepool-org/tools/master/artifact/artifact.sh'
chmod +x artifact_go.sh

./artifact_go.sh go
