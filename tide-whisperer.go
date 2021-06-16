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

// @securityDefinitions.apikey TidepoolAuth
// @in header
// @name x-tidepool-session-token
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

	common "github.com/tidepool-org/go-common"
	"github.com/tidepool-org/go-common/clients"
	"github.com/tidepool-org/go-common/clients/disc"
	"github.com/tidepool-org/go-common/clients/hakken"
	"github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/go-common/clients/opa"
	"github.com/tidepool-org/go-common/clients/shoreline"
	"github.com/tidepool-org/tide-whisperer/auth"
	"github.com/tidepool-org/tide-whisperer/data"
	"github.com/tidepool-org/tide-whisperer/store"
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
		logger.Fatal(err)
	}

	hakkenClient := hakken.NewHakkenBuilder().
		WithConfig(&config.HakkenConfig).
		Build()

	if !config.HakkenConfig.SkipHakken {
		if err := hakkenClient.Start(); err != nil {
			logger.Fatal(err)
		}
		defer func() {
			if err := hakkenClient.Close(); err != nil {
				logger.Panic("Error closing hakkenClient, panicing to get stacks: ", err)
			}
		}()
	} else {
		logger.Print("skipping hakken service")
	}

	shorelineClient := shoreline.NewShorelineClientBuilder().
		WithHostGetter(config.ShorelineConfig.ToHostGetter(hakkenClient)).
		WithHttpClient(httpClient).
		WithConfig(&config.ShorelineConfig.ShorelineClientConfig).
		Build()

	permsClient := opa.NewClientFromEnv(httpClient)

	if err := shorelineClient.Start(); err != nil {
		logger.Fatal(err)
	}
	storage, err := store.NewStore(&config.Mongo, logger)
	if err != nil {
		logger.Fatal(err)
	}
	defer storage.Close()
	storage.Start()
	rtr := mux.NewRouter()

	/*
	 * Data-Api setup
	 */

	dataapi := data.InitAPI(storage, shorelineClient, authClient, permsClient, config.SchemaVersion, logger)
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
	hakkenClient.Publish(&config.Service)

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
