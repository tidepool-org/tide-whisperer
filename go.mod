module github.com/tidepool-org/tide-whisperer

go 1.17

// Commit id relative to go-common@dblp.0.9.0
replace github.com/tidepool-org/go-common => github.com/mdblp/go-common v0.7.2-0.20210611071916-2fe8363c8b02

require (
	github.com/google/uuid v1.3.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/prometheus/client_golang v1.13.0
	github.com/tidepool-org/go-common v0.9.0
	gitlab.com/msvechla/mux-prometheus v0.0.2
	go.mongodb.org/mongo-driver v1.10.2
)

require (
	github.com/mdblp/go-common v1.5.0
	github.com/mdblp/portal-api-v2 v0.2.1-0.20220923163225-577456cdac2b
	github.com/mdblp/shoreline v1.9.2
	github.com/mdblp/tide-whisperer-v2/v2 v2.6.2-0.20220923164521-70a6c0fe05c1
	github.com/sirupsen/logrus v1.9.0
	github.com/stretchr/testify v1.8.0
)

require (
	github.com/auth0/go-jwt-middleware/v2 v2.0.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/felixge/httpsnoop v1.0.1 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/klauspost/compress v1.15.10 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mdblp/go-routers v0.5.6 // indirect
	github.com/montanaflynn/stats v0.6.6 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/rogpeppe/go-internal v1.8.0 // indirect
	github.com/stretchr/objx v0.4.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.1 // indirect
	github.com/xdg-go/stringprep v1.0.3 // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	golang.org/x/crypto v0.0.0-20220829220503-c86fa9a7ed90 // indirect
	golang.org/x/sync v0.0.0-20220907140024-f12130a52804 // indirect
	golang.org/x/sys v0.0.0-20220919091848-fb04ddd9f9c8 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
