module github.com/tidepool-org/tide-whisperer

go 1.15

// Commit id relative to latest commit in feature/opa-client branch from go-common
replace github.com/tidepool-org/go-common => github.com/mdblp/go-common v0.7.1-0.20210309192313-12f55f1fff3b

require (
	github.com/google/uuid v1.1.2
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/kr/pretty v0.2.0 // indirect
	github.com/mdblp/go-common v0.7.1 // indirect
	github.com/tidepool-org/go-common v0.0.0
	github.com/xdg/stringprep v1.0.0 // indirect
	go.mongodb.org/mongo-driver v1.4.2
	golang.org/x/net v0.0.0-20200625001655-4c5254603344 // indirect
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
)
