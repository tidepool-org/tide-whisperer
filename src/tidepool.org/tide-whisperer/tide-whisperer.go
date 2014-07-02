package main

import (
	"encoding/json"
	httpgzip "github.com/daaku/go.httpgzip"
	"github.com/gorilla/pat"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"tidepool.org/tide-whisperer/clients"
	"tidepool.org/tide-whisperer/clients/hakken"
	"time"
)

func main() {
	httpClient := http.DefaultClient

	hakkenClient := hakken.NewHakkenBuilder().
		WithHost("localhost:8000").
		Build()

	err := hakkenClient.Start()
	if err != nil {
		log.Fatal(err)
	}
	defer hakkenClient.Close()

	userAPI := clients.NewApiClient(
		hakkenClient.Watch("user-api-local").Random(),
		clients.UserApiConfig(
			"tide-whisperer",
			"This needs to be the same secret everywhere. YaHut75NsK1f9UKUXuWqxNN0RUwHFBCy",
			1*time.Hour),
		httpClient)

	seagullClient := clients.NewSeagullClientBuilder().
		WithHostGetter(hakkenClient.Watch("seagull-local").Random()).
		WithHttpClient(httpClient).
		Build()

	gatekeeperClient := clients.NewGatekeeperClientBuilder().
		WithHostGetter(hakkenClient.Watch("gatekeeper-local").Random()).
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

		return !(perms["root"] == nil && perms["view"] == nil)
	}

	err = userAPI.Start()
	if err != nil {
		log.Fatal(err)
	}

	session, err := mgo.Dial("mongodb://localhost/streams")
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
			Find(bson.M{"groupId": groupId}).
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
		res.Write([]byte("]"))
		if err := iter.Close(); err != nil {
			log.Println("HUH?")
			log.Fatal(err)
		}
	})))

	server := &http.Server{
		Addr:    ":17071",
		Handler: router,
	}
	log.Print("Starting server at ", server.Addr)
	log.Fatal(server.ListenAndServe())
}
