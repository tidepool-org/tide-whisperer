FROM golang:1.7.1-alpine

# Common ENV
ENV API_SECRET="This is a local API secret for everyone. BsscSHqSHiwrBMJsEGqbvXiuIUPAjQXU" \
    SERVER_SECRET="This needs to be the same secret everywhere. YaHut75NsK1f9UKUXuWqxNN0RUwHFBCy" \
    LONGTERM_KEY="abcdefghijklmnopqrstuvwxyz" \
    DISCOVERY_HOST=hakken:8000 \
    PUBLISH_HOST=hakken \
    METRICS_SERVICE="{ \"type\": \"static\", \"hosts\": [{ \"protocol\": \"http\", \"host\": \"highwater:9191\" }] }" \
    USER_API_SERVICE="{ \"type\": \"static\", \"hosts\": [{ \"protocol\": \"http\", \"host\": \"shoreline:9107\" }] }" \
    SEAGULL_SERVICE="{ \"type\": \"static\", \"hosts\": [{ \"protocol\": \"http\", \"host\": \"seagull:9120\" }] }" \
    GATEKEEPER_SERVICE="{ \"type\": \"static\", \"hosts\": [{ \"protocol\": \"http\", \"host\": \"gatekeeper:9123\" }] }"

WORKDIR /go/src/app

COPY . /go/src/app

# Get come_deps.sh
RUN apk --no-cache add bash curl bzr git \
 && curl --remote-name https://raw.githubusercontent.com/tidepool-org/tools/master/come_deps.sh \
 && chmod a+x come_deps.sh \
# Update config to work with Docker hostnames
 && sed -i -e 's/mongodb:\/\/localhost\/data/mongodb:\/\/mongo\/data/g' config/server.json \
 && sed -i -e 's/localhost:8000/hakken:8000/g' \
           -e 's/localhost:9123/gatekeeper:9123/g' \
           -e 's/localhost:9120/seagull:9120/g' \
           -e's/localhost:9107/shoreline:9107/g' config/env.json \
# Build
 && PATH=${PATH}:. ./build \
# Remove packages needed to build
 && apk del bash curl bzr git \
# Remove files no longer needed after the build to reduce fs layer size
 && rm -rf .git .gitignore come_deps.sh

CMD ["./dist/tide-whisperer"]
