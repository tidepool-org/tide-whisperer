package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
)

type (
	Config struct {
		clients.Config
		Service disc.ServiceListing `json:"service"`
		Mongo   mongo.Config        `json:"mongo"`
	}
	//generic type as device data can be comprised of many things
	deviceData map[string]interface{}
)

const (
	DATA_API_PREFIX = "api/data"

	error_no_view_permisson = "user is not authorized to view data"
	error_no_permissons     = "permissons not found"
	error_running_query     = "error running query"
	error_marshalling_event = "failed to marshall event"

	query_no_data = "no data found"
)

//these fields are removed from the data to be returned by the API
func (data deviceData) filterUnwantedFields() {
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
		log.Fatal(DATA_API_PREFIX, "Problem loading config: ", err)
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
			log.Panic(DATA_API_PREFIX, "Error closing hakkenClient, panicing to get stacks: ", err)
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
			log.Println(DATA_API_PREFIX, "Error looking up user in group", err)
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
	router.Add("GET", "/status", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if err := session.Ping(); err != nil {
			log.Println(DATA_API_PREFIX, req.URL.Path, err.Error())
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.WriteHeader(200)
		res.Write([]byte("OK\n"))
		return
	}))
	router.Add("GET", "/{userID}", httpgzip.NewHandler(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		start := time.Now()

		userToView := req.URL.Query().Get(":userID")

		token := req.Header.Get("x-tidepool-session-token")
		td := shorelineClient.CheckToken(token)

		if td == nil || !(td.IsServer || td.UserID == userToView || userCanViewData(td.UserID, userToView)) {
			log.Println(DATA_API_PREFIX, req.URL.Path, fmt.Sprintf("failed after [%.5f] secs", time.Now().Sub(start).Seconds()))
			log.Println(DATA_API_PREFIX, req.URL.Path, error_no_view_permisson)
			http.Error(res, error_no_view_permisson, http.StatusForbidden)
			return
		}

		pair := seagullClient.GetPrivatePair(userToView, "uploads", shorelineClient.TokenProvide())
		if pair == nil {
			log.Println(DATA_API_PREFIX, req.URL.Path, fmt.Sprintf("failed after [%.5f] secs", time.Now().Sub(start).Seconds()))
			log.Println(DATA_API_PREFIX, req.URL.Path, error_no_permissons)
			http.Error(res, error_no_permissons, http.StatusInternalServerError)
			return
		}

		groupId := pair.ID

		mongoSession := session.Copy()
		defer mongoSession.Close()

		// the Sort here has to use the same fields in the same order
		// as the index, or the index won't apply
		iter := mongoSession.DB("").C(deviceDataCollection).
			Find(bson.M{"$or": []bson.M{
			bson.M{"groupId": groupId},
			bson.M{"_groupId": groupId, "_active": true}}}).
			Sort("groupId", "_groupId", "time").
			Iter()

		first := false
		var result deviceData //generic type as device data can be comprised of many things
		for iter.Next(&result) {
			result.filterUnwantedFields()

			bytes, err := json.Marshal(result)
			if err != nil {
				//log and continue
				log.Println(DATA_API_PREFIX, req.URL.Path, error_marshalling_event, result, err.Error())
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
			log.Println(DATA_API_PREFIX, req.URL.Path, fmt.Sprintf("failed after [%.5f] secs", time.Now().Sub(start).Seconds()))
			log.Println(DATA_API_PREFIX, req.URL.Path, error_running_query, err.Error())
			http.Error(res, error_running_query, http.StatusInternalServerError)
			return
		}

		if !first {
			log.Println(DATA_API_PREFIX, req.URL.Path, fmt.Sprintf("completed in [%.5f] secs", time.Now().Sub(start).Seconds()))
			log.Println(DATA_API_PREFIX, req.URL.Path, query_no_data)
			http.Error(res, query_no_data, http.StatusNotFound)
			return
		} else {
			res.Write([]byte("]"))
			log.Println(DATA_API_PREFIX, req.URL.Path, fmt.Sprintf("completed in [%.5f] secs", time.Now().Sub(start).Seconds()))
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
