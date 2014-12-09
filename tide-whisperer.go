package main

import (
	"crypto/tls"
	"encoding/json"
	httpgzip "github.com/daaku/go.httpgzip"
	"github.com/gorilla/pat"
	common "github.com/tidepool-org/go-common"
	"github.com/tidepool-org/go-common/clients"
	"github.com/tidepool-org/go-common/clients/disc"
	"github.com/tidepool-org/go-common/clients/hakken"
	"github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/go-common/clients/shoreline"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type (
	Config struct {
		clients.Config
		Service disc.ServiceListing `json:"service"`
		Mongo   mongo.Config        `json:"mongo"`
	}
	//generic return type as device data can be comprised of many things
	DeviceData map[string]interface{}
)

//these fields are removed from the data to be returned by the API
func (data DeviceData) filterUnwantedFields() {
	delete(data, "groupId")
	delete(data, "_id")
	delete(data, "_groupId")
	delete(data, "_version")
	delete(data, "_active")
	delete(data, "createdTime")
	delete(data, "modifiedTime")
}

func main() {
	const deviceDataCollection = "deviceData"
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
	defer func() {
		if err := hakkenClient.Close(); err != nil {
			log.Panic("Error closing hakkenClient, panicing to get stacks: ", err)
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

	if err := shorelineClient.Start(); err != nil {
		log.Fatal(err)
	}

	session, err := mongo.Connect(&config.Mongo)
	if err != nil {
		log.Fatal(err)
	}
	//index based on sort and where kys
	index := mgo.Index{
		Key:        []string{"groupId", "_groupId", "time"},
		Background: true,
	}
	_ = session.DB("").C(deviceDataCollection).EnsureIndex(index)

	defer session.Close()

	router := pat.New()
	router.Add("GET", "/status", http.HandlerFunc(func(res http.ResponseWriter, _ *http.Request) {
		if err := session.Ping(); err != nil {
			res.WriteHeader(500)
			res.Write([]byte(err.Error()))
			return
		}
		res.WriteHeader(200)
		res.Write([]byte("OK\n"))
		return
	}))
	router.Add("GET", "/{userID}", httpgzip.NewHandler(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		userToView := req.URL.Query().Get(":userID")

		token := req.Header.Get("x-tidepool-session-token")
		td := shorelineClient.CheckToken(token)

		if td == nil || !(td.IsServer || td.UserID == userToView || userCanViewData(td.UserID, userToView)) {
			res.WriteHeader(403)
			return
		}

		pair := seagullClient.GetPrivatePair(userToView, "uploads", shorelineClient.TokenProvide())
		if pair == nil {
			res.WriteHeader(500)
			return
		}

		groupId := pair.ID

		mongoSession := session.Copy()
		defer mongoSession.Close()

		iter := mongoSession.DB("").C(deviceDataCollection).
			Find(bson.M{"$or": []bson.M{bson.M{"groupId": groupId}, bson.M{"_groupId": groupId, "_active": true}}}).
			Sort("time").
			Iter()

		failureReturnCode := 404
		first := false
		var result DeviceData
		for iter.Next(&result) {
			result.filterUnwantedFields()

			bytes, err := json.Marshal(result)
			if err != nil {
				log.Print("Failed to marshall event: ", result, err)
			} else {
				if !first {
					res.Header().Add("content-type", "application/json")
					res.Write([]byte("["))
					first = true
				} else {
					res.Write([]byte(",\n"))
				}
				res.Write(bytes)
			}
		}
		if err := iter.Close(); err != nil {
			log.Print("Iterator ended with an error: ", err)
			failureReturnCode = 500
		}

		if !first {
			res.WriteHeader(failureReturnCode)
		} else {
			res.Write([]byte("]"))
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
