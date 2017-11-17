package main

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	httpgzip "github.com/daaku/go.httpgzip"
	"github.com/gorilla/pat"
	uuid "github.com/satori/go.uuid"

	common "github.com/tidepool-org/go-common"
	"github.com/tidepool-org/go-common/clients"
	"github.com/tidepool-org/go-common/clients/disc"
	"github.com/tidepool-org/go-common/clients/hakken"
	"github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/go-common/clients/shoreline"
	"github.com/tidepool-org/tide-whisperer/store"
)

type (
	Config struct {
		clients.Config
		Service             disc.ServiceListing `json:"service"`
		Mongo               mongo.Config        `json:"mongo"`
		store.SchemaVersion `json:"schemaVersion"`
	}

	// so we can wrap and marshal the detailed error
	detailedError struct {
		Status int `json:"status"`
		//provided to user so that we can better track down issues
		Id              string `json:"id"`
		Code            string `json:"code"`
		Message         string `json:"message"`
		InternalMessage string `json:"-"` //used only for logging so we don't want to serialize it out
	}
	//generic type as device data can be comprised of many things
	deviceData map[string]interface{}
)

var (
	viewPermissonError = detailedError{Status: http.StatusForbidden, Code: "data_cant_view", Message: "user is not authorized to view data"}
	tokenError         = detailedError{Status: http.StatusUnauthorized, Code: "no_token", Message: "no token was given"}
	//TODO: 500 doesn't seem correct??
	invalidParametersError = detailedError{Status: http.StatusInternalServerError, Code: "invalid_parameters", Message: "one or more parameters are invalid"}

	error_status_check   = detailedError{Status: http.StatusInternalServerError, Code: "data_status_check", Message: "checking of the status endpoint showed an error"}
	error_no_permissons  = detailedError{Status: http.StatusInternalServerError, Code: "data_perms_error", Message: "error finding permissons for user"}
	error_running_query  = detailedError{Status: http.StatusInternalServerError, Code: "data_store_error", Message: "internal server error"}
	error_loading_events = detailedError{Status: http.StatusInternalServerError, Code: "data_marshal_error", Message: "internal server error"}

	storage store.Storage

	serviceLog = log.New(os.Stdout, "api/data: ", log.Lshortfile)
)

//set the intenal message that we will use for logging
func (d detailedError) setInternalMessage(internal error) detailedError {
	d.InternalMessage = internal.Error()
	return d
}

func main() {
	var config Config
	if err := common.LoadConfig([]string{"./config/env.json", "./config/server.json"}, &config); err != nil {
		serviceLog.Fatal("Problem loading config: ", err)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	hakkenClient := hakken.NewHakkenBuilder().
		WithConfig(&config.HakkenConfig).
		Build()

	if err := hakkenClient.Start(); err != nil {
		serviceLog.Fatal(err)
	}
	defer func() {
		if err := hakkenClient.Close(); err != nil {
			serviceLog.Panic("Error closing hakkenClient, panicing to get stacks: ", err)
		}
	}()

	shorelineClient := shoreline.NewShorelineClientBuilder().
		WithHostGetter(config.ShorelineConfig.ToHostGetter(hakkenClient)).
		WithHttpClient(httpClient).
		WithConfig(&config.ShorelineConfig.ShorelineClientConfig).
		Build()

	seagullClient := clients.NewSeagullClientBuilder().
		WithHostGetter(config.SeagullConfig.ToHostGetter(hakkenClient)).
		WithHttpClient(httpClient).
		Build()

	gatekeeperClient := clients.NewGatekeeperClientBuilder().
		WithHostGetter(config.GatekeeperConfig.ToHostGetter(hakkenClient)).
		WithHttpClient(httpClient).
		WithTokenProvider(shorelineClient).
		Build()

	userCanViewData := func(tokenData *shoreline.TokenData, groupID string) bool {
		if tokenData.IsServer {
			return true
		}
		if tokenData.UserID == groupID {
			return true
		}

		perms, err := gatekeeperClient.UserInGroup(tokenData.UserID, groupID)
		if err != nil {
			serviceLog.Println("Error looking up user in group", err)
			return false
		}
		return !(perms["root"] == nil && perms["view"] == nil)
	}

	//log error detail and write as application/json
	jsonError := func(res http.ResponseWriter, err detailedError, startedAt time.Time) {

		err.Id = uuid.NewV4().String()

		serviceLog.Printf("[%s][%s] failed after [%.5f]secs with error [%s][%s] ", err.Id, err.Code, time.Now().Sub(startedAt).Seconds(), err.Message, err.InternalMessage)

		jsonErr, _ := json.Marshal(err)

		res.Header().Add("content-type", "application/json")
		res.WriteHeader(err.Status)
		res.Write(jsonErr)
	}

	processResults := func(response http.ResponseWriter, iterator store.StorageIterator, startTime time.Time) {
		var writeCount int

		serviceLog.Printf("mongo processing started after %.5f seconds", time.Now().Sub(startTime).Seconds())

		response.Header().Add("Content-Type", "application/json")
		response.Write([]byte("["))

		var results map[string]interface{}
		for iterator.Next(&results) {
			if len(results) > 0 {
				if bytes, err := json.Marshal(results); err != nil {
					serviceLog.Printf("failed to marshal mongo result with error: %s", err)
				} else {
					if writeCount > 0 {
						response.Write([]byte(","))
					}
					response.Write([]byte("\n"))
					response.Write(bytes)
					writeCount++
				}
			}
		}

		if writeCount > 0 {
			response.Write([]byte("\n"))
		}
		response.Write([]byte("]"))

		serviceLog.Printf("mongo processing finished after %.5f seconds and returned %d records", time.Now().Sub(startTime).Seconds(), writeCount)
	}

	getToken := func(r *http.Request) string {
		var token string
		if authorization := r.Header.Get("Authorization"); authorization != "" {
			if parts := strings.Split(authorization, " "); len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				token = parts[1]
			}
		}
		if token == "" {
			token = r.Header.Get("X-Tidepool-Session-Token")
		}
		return token
	}

	authorize := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			if token := getToken(r); token != "" {
				if tokenData := shorelineClient.CheckToken(token); tokenData != nil {
					queryParams, err := store.GetParams(r.URL.Query(), &config.SchemaVersion)
					if err != nil {
						serviceLog.Println(err.Error())
						jsonError(w, invalidParametersError, start)
						return
					}
					if userCanViewData(tokenData, queryParams.UserId) {
						h.ServeHTTP(w, r)
						return
					}
				}
				//we have a token but it is invalid
				jsonError(w, viewPermissonError, start)
				return
			}
			//we have no token at all
			jsonError(w, tokenError, start)
		})
	}

	if err := shorelineClient.Start(); err != nil {
		serviceLog.Fatal(err)
	}

	storage := store.NewMongoStoreClient(&config.Mongo)

	router := pat.New()

	router.Add("GET", "/status", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		start := time.Now()
		if err := storage.Ping(); err != nil {
			jsonError(res, error_status_check.setInternalMessage(err), start)
			return
		}
		res.Write([]byte("OK\n"))
		return
	}))

	// The /data/userId endpoint retrieves device/health data for a user based on a set of parameters
	// userid: the ID of the user you want to retrieve data for
	// type (optional) : The Tidepool data type to search for. Only objects with a type field matching the specified type param will be returned.
	//					can be /userid?type=smbg or a comma seperated list e.g /userid?type=smgb,cbg . If is a comma seperated
	//					list, then objects matching any of the sub types will be returned
	// subType (optional) : The Tidepool data subtype to search for. Only objects with a subtype field matching the specified subtype param will be returned.
	//					can be /userid?subtype=physicalactivity or a comma seperated list e.g /userid?subtypetype=physicalactivity,steps . If is a comma seperated
	//					list, then objects matching any of the types will be returned
	// startDate (optional) : Only objects with 'time' field equal to or greater than start date will be returned .
	//						  Must be in ISO date/time format e.g. 2015-10-10T15:00:00.000Z
	// endDate (optional) : Only objects with 'time' field less than to or equal to start date will be returned .
	//						  Must be in ISO date/time format e.g. 2015-10-10T15:00:00.000Z
	router.Add("GET", "/{userID}", authorize(httpgzip.NewHandler(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		start := time.Now()
		queryParams, err := store.GetParams(req.URL.Query(), &config.SchemaVersion)

		if err != nil {
			serviceLog.Printf("Error parsing date: %s", err)
			jsonError(res, invalidParametersError, start)
			return
		}
		//TODO: If the user has a legitimate token but no data storage
		// account we should be returning a `StatusNotFound` 404
		if _, ok := req.URL.Query()["carelink"]; !ok {
			if hasMedtronicDirectData, err := storage.HasMedtronicDirectData(queryParams.UserId); err != nil {
				serviceLog.Printf("Error while querying for Medtronic Direct data: %s", err)
				jsonError(res, error_running_query, start)
				return
			} else if !hasMedtronicDirectData {
				queryParams.Carelink = true
			}
		}

		pair := seagullClient.GetPrivatePair(queryParams.UserId, "uploads", shorelineClient.TokenProvide())
		if pair == nil {
			jsonError(res, error_no_permissons, start)
			return
		}

		queryParams.GroupId = pair.ID
		started := time.Now()

		iter := storage.GetDeviceData(queryParams)
		defer iter.Close()

		processResults(res, iter, started)
	}))))

	done := make(chan bool)
	server := common.NewServer(&http.Server{
		Addr:    config.Service.GetPort(),
		Handler: router,
	})

	var start func() error
	if config.Service.Scheme == "https" {
		sslSpec := config.Service.GetSSLSpec()
		start = func() error { return server.ListenAndServeTLS(sslSpec.CertFile, sslSpec.KeyFile) }
	} else {
		start = func() error { return server.ListenAndServe() }
	}
	if err := start(); err != nil {
		serviceLog.Fatal(err)
	}
	hakkenClient.Publish(&config.Service)

	signals := make(chan os.Signal, 40)
	signal.Notify(signals)
	go func() {
		for {
			sig := <-signals
			serviceLog.Printf(" Got signal [%s]", sig)

			if sig == syscall.SIGINT || sig == syscall.SIGTERM {
				server.Close()
				done <- true
			}
		}
	}()

	<-done
}
