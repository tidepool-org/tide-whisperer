export TIDEPOOL_TIDE_WHISPERER_ENV="{
      \"auth\": {\"address\":\"http://localhost:9222\", \"serviceSecret\":\"Service secret used for interservice requests with the auth service\", \"userAgent\":\"Tidepool-TideWhisperer\"},
      \"gatekeeper\": {\"serviceSpec\": {\"type\": \"static\", \"hosts\": [\"http://localhost:9123\"]}},
      \"hakken\": {\"host\": \"localhost:8000\",\"skipHakken\":true},
      \"seagull\": {\"serviceSpec\": {\"type\": \"static\", \"hosts\": [\"http://localhost:9120\"]}},
      \"shoreline\": {
          \"name\": \"tide-whisperer\",
          \"secret\": \"This needs to be the same secret everywhere. YaHut75NsK1f9UKUXuWqxNN0RUwHFBCy\",
          \"serviceSpec\": {\"type\": \"static\", \"hosts\": [\"http://localhost:9107\"]},
          \"tokenRefreshInterval\": \"1h\"
      }
  }"

export TIDEPOOL_TIDE_WHISPERER_SERVICE="{
      \"mongo\": {
          \"connectionString\": \"mongodb://medical:password@localhost:27017/data?authSource=admin&ssl=false\"
      },
      \"schemaVersion\": {
          \"maximum\": 99,
          \"minimum\": 1
      },
      \"service\": {
          \"certFile\": \"config/cert.pem\",
          \"host\": \"localhost:9127\",
          \"keyFile\": \"config/key.pem\",
          \"protocol\": \"http\",
          \"service\": \"tide-whisperer\"
      }
  }
"
export TIDEPOOL_STORE_ADDRESSES="localhost:27017"
export TIDEPOOL_STORE_DATABASE="data"
export TIDEPOOL_STORE_USERNAME="medical"
export TIDEPOOL_STORE_PASSWORD="password"
export TIDEPOOL_STORE_OPT_PARAMS="authSource=admin&ssl=false"

export OPA_HOST="http://coastguard:8181"
export SERVICE_NAME="tide-whisperer"