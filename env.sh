export TIDEPOOL_TIDEWHISPERER_ENV='{
  "hakken": { "host": "localhost:8000" },
  "gatekeeper": { "serviceSpec": { "type": "static", "hosts": ["http://localhost:9123"] } },
  "seagull": { "serviceSpec": { "type": "static", "hosts": ["http://localhost:9120"] } },
  "shoreline": {
    "serviceSpec": { "type": "static", "hosts": ["http://localhost:9107"] },
    "name": "tide-whisperer-local",
    "secret": "This needs to be the same secret everywhere. YaHut75NsK1f9UKUXuWqxNN0RUwHFBCy",
    "tokenRefreshInterval": "1h"
  }
}'

export TIDEPOOL_TIDEWHISPERER_SERVICE='{
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
