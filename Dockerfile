FROM golang:1.9.1-alpine

# Common ENV
ENV API_SECRET="This is a local API secret for everyone. BsscSHqSHiwrBMJsEGqbvXiuIUPAjQXU" \
    SERVER_SECRET="This needs to be the same secret everywhere. YaHut75NsK1f9UKUXuWqxNN0RUwHFBCy" \
    LONGTERM_KEY="abcdefghijklmnopqrstuvwxyz" \
    DISCOVERY_HOST=hakken:8000 \
    PUBLISH_HOST=hakken \
    METRICS_SERVICE="{ \"type\": \"static\", \"hosts\": [{ \"protocol\": \"http\", \"host\": \"highwater:9191\" }] }" \
    USER_API_SERVICE="{ \"type\": \"static\", \"hosts\": [{ \"protocol\": \"http\", \"host\": \"shoreline:9107\" }] }" \
    SEAGULL_SERVICE="{ \"type\": \"static\", \"hosts\": [{ \"protocol\": \"http\", \"host\": \"seagull:9120\" }] }" \
    GATEKEEPER_SERVICE="{ \"type\": \"static\", \"hosts\": [{ \"protocol\": \"http\", \"host\": \"gatekeeper:9123\" }] }" \
# Container specific ENV
    TIDEPOOL_TIDE_WHISPERER_ENV="{\"hakken\": { \"host\": \"hakken:8000\" },\"gatekeeper\": { \"serviceSpec\": { \"type\": \"static\", \"hosts\": [\"http://gatekeeper:9123\"] } },\"seagull\": { \"serviceSpec\": { \"type\": \"static\", \"hosts\": [\"http://seagull:9120\"] } },\"shoreline\": {\"serviceSpec\": { \"type\": \"static\", \"hosts\": [\"http://shoreline:9107\"] },\"name\": \"tide-whisperer-local\",\"secret\": \"This needs to be the same secret everywhere. YaHut75NsK1f9UKUXuWqxNN0RUwHFBCy\",\"tokenRefreshInterval\": \"1h\"}}" \
    TIDEPOOL_TIDE_WHISPERER_SERVICE="{\"service\": {\"service\": \"tide-whisperer-local\",\"protocol\": \"http\",\"host\": \"localhost:9127\",\"keyFile\": \"config/key.pem\",\"certFile\": \"config/cert.pem\"},\"mongo\": {\"connectionString\": \"mongodb://mongo/data\"},\"schemaVersion\": {\"minimum\": 1,\"maximum\": 99}}"

WORKDIR /go/src/github.com/tidepool-org/tide-whisperer

COPY . /go/src/github.com/tidepool-org/tide-whisperer

RUN  ./build.sh && rm -rf .git .gitignore

CMD ["./dist/tide-whisperer"]
