module github.com/tidepool-org/tide-whisperer

go 1.12

replace github.com/tidepool-org/shoreline => ./

replace github.com/tidepool-org/go-common => github.com/mdblp/go-common v0.3.0

require (
	github.com/daaku/go.httpgzip v0.0.0-20180202095102-86d27ccd810b
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8
	github.com/gorilla/context v1.1.1
	github.com/gorilla/mux v1.7.2
	github.com/gorilla/pat v0.0.0-20180118221401-71e7b868be7b
	github.com/satori/go.uuid v1.2.0
	github.com/tidepool-org/go-common v0.0.0-00010101000000-000000000000
)
