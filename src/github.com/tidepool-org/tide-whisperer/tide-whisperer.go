package main

import (
	"crypto/tls"
	"encoding/json"
	httpgzip "github.com/daaku/go.httpgzip"
	"github.com/gorilla/pat"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"github.com/tidepool-org/common"
	"github.com/tidepool-org/tide-whisperer/clients"
	"github.com/tidepool-org/tide-whisperer/clients/disc"
	"github.com/tidepool-org/tide-whisperer/clients/hakken"
	"github.com/tidepool-org/tide-whisperer/clients/mongo"
)

type Config struct {
	clients.Config
	Service disc.ServiceListing `json:"service"`
	Mongo   mongo.Config        `json:"mongo"`
}

func main() {
	var config Config
	if err := common.LoadConfig([]string{"./config/env.json", "./config/server.json"}, &config); err != nil {
		log.Fatal("Problem loading config: ", err)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	hakkenClient := hakken.NewHakkenBuilder().
		WithConfig(&config.HakkenConfig).
		Build()

	if err := hakkenClient.Start(); err != nil {
		log.Fatal(err)
	}
	defer hakkenClient.Close()

	userAPI := clients.NewApiClient(
		config.UserApiConfig.ToHostGetter(hakkenClient),
		&config.UserApiConfig.UserApiClientConfig,
		httpClient)

	seagullClient := clients.NewSeagullClientBuilder().
		WithHostGetter(config.SeagullConfig.ToHostGetter(hakkenClient)).
		WithHttpClient(httpClient).
		Build()

	gatekeeperClient := clients.NewGatekeeperClientBuilder().
		WithHostGetter(config.GatekeeperConfig.ToHostGetter(hakkenClient)).
		WithHttpClient(httpClient).
		WithTokenProvider(userAPI).
		Build()

	userCanViewData := func(userID, groupID string) bool {
		if userID == groupID {
			return true
		}

		perms, err := gatekeeperClient.UserInGroup(userID, groupID)
		if err != nil {
			log.Println("Error looking up user in group", err)
			return false
		}

		log.Println(perms)
		return !(perms["root"] == nil && perms["view"] == nil)
	}

	if err := userAPI.Start(); err != nil {
		log.Fatal(err)
	}

	session, err := mongo.Connect(&config.Mongo)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	router := pat.New()
	router.Add("GET", "/{userID}", httpgzip.NewHandler(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		userID := req.URL.Query().Get(":userID")

		token := req.Header.Get("x-tidepool-session-token")
		td := userAPI.CheckToken(token)

		if td == nil || !(td.IsServer || td.UserID == userID || userCanViewData(userID, td.UserID)) {
			res.WriteHeader(403)
			return
		}

		pair := seagullClient.GetPrivatePair(userID, "uploads", userAPI.TokenProvide())
		if pair == nil {
			res.WriteHeader(500)
			return
		}

		groupId := pair.ID

		iter := session.DB("").C("deviceData").
			Find(bson.M{"$or": []bson.M{bson.M{"groupId": groupId}, bson.M{"_groupId": groupId, "_active": true}}}).
			Sort("deviceTime").
			Iter()

		first := false
		var result map[string]interface{}
		for iter.Next(&result) {
			if !first {
				res.Header().Add("content-type", "application/json")
				res.Write([]byte("["))
				first = true
			} else {
				res.Write([]byte(",\n"))
			}
			delete(result, "groupId")
			bytes, err := json.Marshal(result)
			if err != nil {
				log.Fatal(err)
			}
			res.Write(bytes)
		}
		if !first {
			res.WriteHeader(404)
		} else {
			res.Write([]byte("]"))
		}
		if err := iter.Close(); err != nil {
			log.Fatal("Iterator ended with an error", err)
		}
	})))

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
		log.Fatal(err)
	}
	hakkenClient.Publish(&config.Service)

	signals := make(chan os.Signal, 40)
	signal.Notify(signals)
	go func() {
		for {
			sig := <-signals
			log.Printf("Got signal [%s]", sig)

			if sig == syscall.SIGINT || sig == syscall.SIGTERM {
				server.Close()
				done <- true
			}
		}
	}()

	<-done
}
