export TIDEPOOL_TIDE_WHISPERER_ENV='{
  "auth": { 
    "address": "http://localhost:9222", 
    "serviceSecret": "AMSTRAMGRAM", 
    "userAgent": "Tidepool-TideWhisperer"
  }
  "hakken": { "host": "localhost:8000", "skipHakken\":true },
  "gatekeeper": { "serviceSpec": { "type": "static", "hosts": ["http://localhost:8181"] } },
  "seagull": { "serviceSpec": { "type": "static", "hosts": ["http://localhost:9120"] } },
  "shoreline": {
    "serviceSpec": { "type": "static", "hosts": ["http://localhost:9107"] },
    "name": "tide-whisperer-local",
    "secret": "This needs to be the same secret everywhere. YaHut75NsK1f9UKUXuWqxNN0RUwHFBCy",
    "tokenRefreshInterval": "1h"
  }
}'

export TIDEPOOL_TIDE_WHISPERER_SERVICE='{
  "service": {
    "service": "tide-whisperer-local",
    "protocol": "http",
    "host": "localhost:9127",
    "keyFile": "config/key.pem",
    "certFile": "config/cert.pem"
  },
  "mongo": {
    "connectionString": "mongodb://localhost/data"
  },
  "schemaVersion": {
    "minimum": 1,
    "maximum": 99
  }
}'

export TIDEPOOL_STORE_ADDRESSES="localhost:27017"
export TIDEPOOL_STORE_DATABASE="data"
export TIDEPOOL_STORE_USERNAME="medical"
export TIDEPOOL_STORE_PASSWORD="password"
export TIDEPOOL_STORE_OPT_PARAMS="authSource=admin&ssl=false"

export OPA_HOST="http://localhost:8181"
export SERVICE_NAME="tide-whisperer"
export AUTH_SECRET="This is a local API secret for everyone. BsscSHqSHiwrBMJsEGqbvXiuIUPAjQXU"
export AUTH0_URL="https://yourloops-dev.eu.auth0.com"