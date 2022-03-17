module github.com/tidepool-org/tide-whisperer

go 1.17

// Commit id relative to go-common@dblp.0.9.0
replace github.com/tidepool-org/go-common => github.com/mdblp/go-common v0.7.2-0.20210611071916-2fe8363c8b02

require (
	github.com/google/uuid v1.3.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	// on main branch (release 2.0.0-RC1)
	github.com/mdblp/tide-whisperer-v2 v0.0.0-20220117150717-4b55fbd64eaa
	github.com/prometheus/client_golang v1.11.0
	github.com/stretchr/testify v1.7.0 // indirect
	github.com/tidepool-org/go-common v0.0.0
	gitlab.com/msvechla/mux-prometheus v0.0.2
	go.mongodb.org/mongo-driver v1.8.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/felixge/httpsnoop v1.0.1 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mdblp/go-common v1.2.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.0.2 // indirect
	github.com/xdg-go/stringprep v1.0.2 // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	golang.org/x/crypto v0.0.0-20211215153901-e495a2d5b3d3 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
