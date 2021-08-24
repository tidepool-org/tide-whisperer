package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	httpgzip "github.com/daaku/go.httpgzip"
	"github.com/google/uuid"
	"github.com/gorilla/pat"

	common "github.com/tidepool-org/go-common"
	"github.com/tidepool-org/go-common/clients"
	"github.com/tidepool-org/go-common/clients/disc"
	"github.com/tidepool-org/go-common/clients/hakken"
	"github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/go-common/clients/shoreline"

	"github.com/tidepool-org/tide-whisperer/auth"
	"github.com/tidepool-org/tide-whisperer/store"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type (
	// Config holds the configuration for the `tide-whisperer` service
	Config struct {
		clients.Config
		Auth                *auth.Config        `json:"auth"`
		Service             disc.ServiceListing `json:"service"`
		Mongo               mongo.Config        `json:"mongo"`
		store.SchemaVersion `json:"schemaVersion"`
	}

	// so we can wrap and marshal the detailed error
	detailedError struct {
		Status int `json:"status"`
		//provided to user so that we can better track down issues
		ID              string `json:"id"`
		Code            string `json:"code"`
		Message         string `json:"message"`
		InternalMessage string `json:"-"` //used only for logging so we don't want to serialize it out
	}
	//generic type as device data can be comprised of many things
	deviceData map[string]interface{}
)

var (
	errorStatusCheck       = detailedError{Status: http.StatusInternalServerError, Code: "data_status_check", Message: "checking of the status endpoint showed an error"}
	errorNoViewPermission  = detailedError{Status: http.StatusForbidden, Code: "data_cant_view", Message: "user is not authorized to view data"}
	errorNoPermissions     = detailedError{Status: http.StatusInternalServerError, Code: "data_perms_error", Message: "error finding permissions for user"}
	errorRunningQuery      = detailedError{Status: http.StatusInternalServerError, Code: "data_store_error", Message: "internal server error"}
	errorLoadingEvents     = detailedError{Status: http.StatusInternalServerError, Code: "data_marshal_error", Message: "internal server error"}
	errorInvalidParameters = detailedError{Status: http.StatusInternalServerError, Code: "invalid_parameters", Message: "one or more parameters are invalid"}

	storage store.Storage

	slowDataCheckCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "tidepool_tide_whisperer_slow_data_check_count",
		Help: "Counts slow device data checks.",
	}, []string{"manufacturer", "data_access_type"})

	mongoErrorCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "tidepool_tide_mongo_error_count",
		Help: "Counts Mongo errors.",
	}, []string{"type"})
)

const (
	dataAPIPrefix             = "api/data "
	medtronicLoopBoundaryDateUnix = 1504238400
	slowQueryDuration         = 0.1 // seconds
	DeviceTimeFormat          = "2006-01-02T15:04:05"
)

//set the intenal message that we will use for logging
func (d detailedError) setInternalMessage(internal error) detailedError {
	d.InternalMessage = internal.Error()
	return d
}

func main() {
	var config Config

	medtronicLoopBoundaryDate := time.Unix(medtronicLoopBoundaryDateUnix, 0).UTC()

	log.SetPrefix(dataAPIPrefix)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if err := common.LoadEnvironmentConfig(
		[]string{"TIDEPOOL_TIDE_WHISPERER_SERVICE", "TIDEPOOL_TIDE_WHISPERER_ENV"},
		&config,
	); err != nil {
		log.Fatal(dataAPIPrefix, "Problem loading config: ", err)
	}

	// server secret may be passed via a separate env variable to accommodate easy secrets injection via Kubernetes
	serverSecret, found := os.LookupEnv("SERVER_SECRET")
	if found {
		config.ShorelineConfig.Secret = serverSecret
	}
	authSecret, found := os.LookupEnv("AUTH_SECRET")
	if found {
		config.Auth.ServiceSecret = authSecret
	}

	config.Mongo.FromEnv()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	authClient, err := auth.NewClient(config.Auth, httpClient)
	if err != nil {
		log.Fatal(dataAPIPrefix, err)
	}

	hakkenClient := hakken.NewHakkenBuilder().
		WithConfig(&config.HakkenConfig).
		Build()

	if !config.HakkenConfig.SkipHakken {
		if err := hakkenClient.Start(); err != nil {
			log.Fatal(dataAPIPrefix, err)
		}
		defer func() {
			if err := hakkenClient.Close(); err != nil {
				log.Panic(dataAPIPrefix, "Error closing hakkenClient, panicing to get stacks: ", err)
			}
		}()
	} else {
		log.Print("skipping hakken service")
	}

	shorelineClient := shoreline.NewShorelineClientBuilder().
		WithHostGetter(config.ShorelineConfig.ToHostGetter(hakkenClient)).
		WithHttpClient(httpClient).
		WithConfig(&config.ShorelineConfig.ShorelineClientConfig).
		Build()

	gatekeeperClient := clients.NewGatekeeperClientBuilder().
		WithHostGetter(config.GatekeeperConfig.ToHostGetter(hakkenClient)).
		WithHttpClient(httpClient).
		WithTokenProvider(shorelineClient).
		Build()

	userCanViewData := func(authenticatedUserID string, targetUserID string) bool {
		if authenticatedUserID == targetUserID {
			return true
		}

		perms, err := gatekeeperClient.UserInGroup(authenticatedUserID, targetUserID)
		if err != nil {
			log.Println(dataAPIPrefix, "Error looking up user in group", err)
			return false
		}

		log.Println(perms)
		return !(perms["root"] == nil && perms["view"] == nil)
	}

	//log error detail and write as application/json
	jsonError := func(res http.ResponseWriter, err detailedError, startedAt time.Time) {

		err.ID = uuid.New().String()

		log.Println(dataAPIPrefix, fmt.Sprintf("[%s][%s] failed after [%.3f]secs with error [%s][%s] ", err.ID, err.Code, time.Now().Sub(startedAt).Seconds(), err.Message, err.InternalMessage))

		jsonErr, _ := json.Marshal(err)

		res.Header().Add("content-type", "application/json")
		res.WriteHeader(err.Status)
		res.Write(jsonErr)
	}

	if err := shorelineClient.Start(); err != nil {
		log.Fatal(err)
	}

	storage := store.NewMongoStoreClient(&config.Mongo)
	defer storage.Disconnect()
	storage.EnsureIndexes()

	router := pat.New()

	router.Handle("/metrics", promhttp.Handler())

	router.Add("GET", "/data/status", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		start := time.Now()
		if err := storage.Ping(); err != nil {
			jsonError(res, errorStatusCheck.setInternalMessage(err), start)
			return
		}
		res.Write([]byte("OK\n"))
		return
	}))

	router.Add("GET", "/status", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		start := time.Now()
		if err := storage.WithContext(req.Context()).Ping(); err != nil {
			jsonError(res, errorStatusCheck.setInternalMessage(err), start)
			return
		}
		res.Write([]byte("OK\n"))
		return
	}))

	f := httpgzip.NewHandler(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		start := time.Now()

		storageWithCtx := storage.WithContext(req.Context())

		queryParams, err := store.GetParams(req.URL.Query(), &config.SchemaVersion)

		if err != nil {
			log.Println(dataAPIPrefix, fmt.Sprintf("Error parsing query params: %s", err))
			jsonError(res, errorInvalidParameters, start)
			return
		}

		var td *shoreline.TokenData
		if sessionToken := req.Header.Get("x-tidepool-session-token"); sessionToken != "" {
			td = shorelineClient.CheckToken(sessionToken)
		} else if restrictedTokens, found := req.URL.Query()["restricted_token"]; found && len(restrictedTokens) == 1 {
			restrictedToken, restrictedTokenErr := authClient.GetRestrictedToken(req.Context(), restrictedTokens[0])
			if restrictedTokenErr == nil && restrictedToken != nil && restrictedToken.Authenticates(req) {
				td = &shoreline.TokenData{UserID: restrictedToken.UserID}
			}
		}

		userID := queryParams.UserID
		if td == nil || !(td.IsServer || td.UserID == userID || userCanViewData(td.UserID, userID)) {
			log.Printf("userid %v", userID)
			jsonError(res, errorNoViewPermission, start)
			return
		}

		requestID := NewRequestID()
		queryStart := time.Now()
		if _, ok := req.URL.Query()["carelink"]; !ok {
			if hasMedtronicDirectData, medtronicErr := storageWithCtx.HasMedtronicDirectData(queryParams.UserID); medtronicErr != nil {
				log.Printf("%s request %s user %s HasMedtronicDirectData returned error: %s", dataAPIPrefix, requestID, userID, medtronicErr)
				jsonError(res, errorRunningQuery, start)
				return
			} else if !hasMedtronicDirectData {
				queryParams.Carelink = true
			}
			if queryDuration := time.Now().Sub(queryStart).Seconds(); queryDuration > slowQueryDuration {
				slowDataCheckCount.WithLabelValues("medtronic", "direct").Inc()
				log.Printf("%s request %s user %s HasMedtronicDirectData took %.3fs", dataAPIPrefix, requestID, userID, queryDuration)
			}
			queryStart = time.Now()
		}
		if !queryParams.Dexcom {
			dexcomDataSource, dexcomErr := storageWithCtx.GetDexcomDataSource(queryParams.UserID)
			if dexcomErr != nil {
				log.Printf("%s request %s user %s GetDexcomDataSource returned error: %s", dataAPIPrefix, requestID, userID, dexcomErr)
				jsonError(res, errorRunningQuery, start)
				return
			}
			queryParams.DexcomDataSource = dexcomDataSource

			if queryDuration := time.Now().Sub(queryStart).Seconds(); queryDuration > slowQueryDuration {
				slowDataCheckCount.WithLabelValues("dexcom", "datasource").Inc()
				log.Printf("%s request %s user %s GetDexcomDataSource took %.3fs", dataAPIPrefix, requestID, userID, queryDuration)
			}
			queryStart = time.Now()
		}
		if _, ok := req.URL.Query()["medtronic"]; !ok {
			hasMedtronicLoopData, medtronicErr := storageWithCtx.HasMedtronicLoopDataAfter(queryParams.UserID, medtronicLoopBoundaryDate)
			if medtronicErr != nil {
				log.Printf("%s request %s user %s HasMedtronicLoopDataAfter returned error: %s", dataAPIPrefix, requestID, userID, medtronicErr)
				jsonError(res, errorRunningQuery, start)
				return
			}
			if !hasMedtronicLoopData {
				queryParams.Medtronic = true
			}
			if queryDuration := time.Now().Sub(queryStart).Seconds(); queryDuration > slowQueryDuration {
				slowDataCheckCount.WithLabelValues("medtronic", "loop_data").Inc()
				log.Printf("%s request %s user %s HasMedtronicLoopDataAfter took %.3fs", dataAPIPrefix, requestID, userID, queryDuration)
			}
			queryStart = time.Now()
		}
		if !queryParams.Medtronic {
			medtronicUploadIds, medtronicErr := storageWithCtx.GetLoopableMedtronicDirectUploadIdsAfter(queryParams.UserID, medtronicLoopBoundaryDate)
			if medtronicErr != nil {
				log.Printf("%s request %s user %s GetLoopableMedtronicDirectUploadIdsAfter returned error: %s", dataAPIPrefix, requestID, userID, medtronicErr)
				jsonError(res, errorRunningQuery, start)
				return
			}
			queryParams.MedtronicDate = medtronicLoopBoundaryDate
			queryParams.MedtronicUploadIds = medtronicUploadIds

			if queryDuration := time.Now().Sub(queryStart).Seconds(); queryDuration > slowQueryDuration {
				slowDataCheckCount.WithLabelValues("medtronic", "loop_direct_upload_ids").Inc()
				log.Printf("%s request %s user %s GetLoopableMedtronicDirectUploadIdsAfter took %.3fs", dataAPIPrefix, requestID, userID, queryDuration)
			}
			queryStart = time.Now()
		}

		iter, err := storageWithCtx.GetDeviceData(queryParams)
		if err != nil {
			mongoErrorCount.WithLabelValues(err.Error()).Inc()
			log.Printf("%s request %s user %s Mongo Query returned error: %s", dataAPIPrefix, requestID, userID, err)
		}

		defer iter.Close(req.Context())

		var writeCount int

		res.Header().Add("Content-Type", "application/json")

		res.Write([]byte("["))

		for iter.Next(req.Context()) {
			var results map[string]interface{}
			err := iter.Decode(&results)
			if err != nil {
				mongoErrorCount.WithLabelValues("decode").Inc()
				log.Printf("%s request %s user %s Mongo Decode returned error: %s", dataAPIPrefix, requestID, userID, err)
			}

			if len(results) > 0 {
				// HACK convert deviceTime to string before marshal, to avoid modifying the results map above
				switch v := results["deviceTime"].(type) {
					case primitive.DateTime:
						results["deviceTime"] = v.Time().Format(DeviceTimeFormat)
					case string:
						// do nothing
				}

				if bytes, err := json.Marshal(results); err != nil {
					mongoErrorCount.WithLabelValues("marshal").Inc()
					log.Printf("%s request %s user %s Marshal returned error: %s", dataAPIPrefix, requestID, userID, err)
				} else {
					if writeCount > 0 {
						res.Write([]byte(","))
					}
					res.Write([]byte("\n"))
					res.Write(bytes)
					writeCount++
				}
			}
		}

		if writeCount > 0 {
			res.Write([]byte("\n"))
		}
		res.Write([]byte("]"))

		if queryDuration := time.Now().Sub(queryStart).Seconds(); queryDuration > slowQueryDuration {
			// XXX use metrics
			//log.Printf("%s request %s user %s GetDeviceData took %.3fs", DATA_API_PREFIX, requestID, userID, queryDuration)
		}
		log.Printf("%s request %s user %s took %.3fs returned %d records", dataAPIPrefix, requestID, userID, time.Now().Sub(start).Seconds(), writeCount)
	}))

	// The /data/userId endpoint retrieves device/health data for a user based on a set of parameters
	// userid: the ID of the user you want to retrieve data for
	// uploadId (optional) : Search for Tidepool data by uploadId. Only objects with a uploadId field matching the specified uploadId param will be returned.
	// deviceId (optional) : Search for Tidepool data by deviceId. Only objects with a deviceId field matching the specified uploadId param will be returned.
	// type (optional) : The Tidepool data type to search for. Only objects with a type field matching the specified type param will be returned.
	//					can be /userid?type=smbg or a comma seperated list e.g /userid?type=smgb,cbg . If is a comma seperated
	//					list, then objects matching any of the sub types will be returned
	// subType (optional) : The Tidepool data subtype to search for. Only objects with a subtype field matching the specified subtype param will be returned.
	//					can be /userid?subtype=physicalactivity or a comma seperated list e.g /userid?subtypetype=physicalactivity,steps . If is a comma seperated
	//					list, then objects matching any of the types will be returned
	// startDate (optional) : Only objects with 'time' field equal to or greater than start date will be returned.
	//					Must be in ISO date/time format e.g. 2015-10-10T15:00:00.000Z
	// endDate (optional) : Only objects with 'time' field less than to or equal to start date will be returned.
	//					Must be in ISO date/time format e.g. 2015-10-10T15:00:00.000Z
	// latest (optional) : Returns only the most recent results for each `type` matching the results filtered by the other query parameters
	router.Add("GET", "/data/{userID}", f)
	router.Add("GET", "/{userID}", f)

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
		log.Fatal(dataAPIPrefix, err)
	}
	hakkenClient.Publish(&config.Service)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			sig := <-signals

			log.Printf(dataAPIPrefix+" Got signal [%s]", sig)
			server.Close()
			done <- true
		}
	}()

	<-done
}

// NewRequestID returns a new random hexadecimal ID
func NewRequestID() string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes) // In case of failure, do not fail request, just use default bytes (zero)
	return hex.EncodeToString(bytes)
}
