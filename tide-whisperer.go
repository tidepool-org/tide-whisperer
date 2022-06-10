// @title Tide-Whisperer API
// @version 0.7.4
// @description Data access API for Diabeloop's diabetes data as used by Blip
// @license.name BSD 2-Clause "Simplified" License
// @host api.android-qa.your-loops.dev
// @BasePath /data
// @accept json
// @produce json
// @schemes https
// @contact.name Diabeloop
// #contact.url https://www.diabeloop.com
// @contact.email platforms@diabeloop.fr

// @securityDefinitions.apikey Auth0
// @in header
// @name Authorization
package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	muxprom "gitlab.com/msvechla/mux-prometheus/pkg/middleware"

	"github.com/mdblp/go-common/clients/auth"
	tideV2Client "github.com/mdblp/tide-whisperer-v2/v2/client/tidewhisperer"
	common "github.com/tidepool-org/go-common"
	"github.com/tidepool-org/go-common/clients"
	"github.com/tidepool-org/go-common/clients/disc"
	"github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/go-common/clients/opa"
	"github.com/tidepool-org/tide-whisperer/data"
	"github.com/tidepool-org/tide-whisperer/store"
)

type (
	// Config holds the configuration for the `tide-whisperer` service
	Config struct {
		clients.Config
		Service             disc.ServiceListing `json:"service"`
		Mongo               mongo.Config        `json:"mongo"`
		store.SchemaVersion `json:"schemaVersion"`
	}
)

func main() {
	var config Config
	logger := log.New(os.Stdout, data.DataAPIPrefix, log.LstdFlags|log.Lshortfile)

	if err := common.LoadEnvironmentConfig(
		[]string{"TIDEPOOL_TIDE_WHISPERER_SERVICE", "TIDEPOOL_TIDE_WHISPERER_ENV"},
		&config,
	); err != nil {
		logger.Fatal("Problem loading config: ", err)
	}
	authSecret, found := os.LookupEnv("API_SECRET")
	if !found || authSecret == "" {
		logger.Fatal("Env var API_SECRET is not provided or empty")
	}
	authClient, err := auth.NewClient(authSecret)
	config.Mongo.FromEnv()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	if err != nil {
		logger.Fatal(err)
	}

	permsClient := opa.NewClientFromEnv(httpClient)

	tideV2Client := tideV2Client.NewTideWhispererClientFromEnv(httpClient)

	/*
	 * Instrumentation setup
	 */
	instrumentation := muxprom.NewCustomInstrumentation(true, "dblp", "tidewhisperer", prometheus.DefBuckets, nil, prometheus.DefaultRegisterer)

	storage, err := store.NewStore(&config.Mongo, logger)
	if err != nil {
		logger.Fatal(err)
	}
	defer storage.Close()
	storage.Start()
	rtr := mux.NewRouter()

	rtr.Use(instrumentation.Middleware)
	rtr.Path("/metrics").Handler(promhttp.Handler())

	/*
	 * Data-Api setup
	 */

	dataapi := data.InitAPI(storage, authClient, permsClient, config.SchemaVersion, logger, tideV2Client)
	dataapi.SetHandlers("", rtr)

	// ability to return compressed (gzip/deflate) responses if client browser accepts it
	// this is interesting to minimise network traffic especially if we expect to have long
	// responses such as what the GetData() route here can return
	gzipHandler := handlers.CompressHandler(rtr)

	done := make(chan bool)
	server := common.NewServer(&http.Server{
		Addr:    config.Service.GetPort(),
		Handler: gzipHandler,
	})

	var start func() error
	if config.Service.Scheme == "https" {
		sslSpec := config.Service.GetSSLSpec()
		start = func() error { return server.ListenAndServeTLS(sslSpec.CertFile, sslSpec.KeyFile) }
	} else {
		start = func() error { return server.ListenAndServe() }
	}
	if err := start(); err != nil {
		logger.Fatal(err)
	}

	// Wait for SIGINT (Ctrl+C) or SIGTERM to stop the service
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			<-sigc
			storage.Close()
			server.Close()
			done <- true
		}
	}()

	<-done
}
