package main

import (
	"encoding/json"
	"fmt"
	httpgzip "github.com/daaku/go.httpgzip"
	"github.com/gorilla/pat"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"net/url"
	"tidepool.org/tide-whisperer/clients"
	"time"
)

func main() {
	httpClient := http.DefaultClient

	userApiHost, err := url.Parse("http://localhost:9107")
	if err != nil {
		log.Fatal(err)
	}

	userAPI := clients.NewApiClient(
		clients.HostGetterFunc(func() url.URL { return *userApiHost }),
		clients.UserApiConfig(
			"tide-whisperer",
			"This needs to be the same secret everywhere. YaHut75NsK1f9UKUXuWqxNN0RUwHFBCy",
			1*time.Hour),
		httpClient)
	err = userAPI.Start()
	if err != nil {
		log.Fatal(err)
	}

	seagullHost, err := url.Parse("http://localhost:9120")
	if err != nil {
		log.Fatal(err)
	}
	seagullClient := clients.NewSeagullClientBuilder().
		WithHostGetter(clients.HostGetterFunc(func() url.URL { return *seagullHost })).
		WithHttpClient(httpClient).
		Build()

	gatekeeperHost, err := url.Parse("http://localhost:9123")
	if err != nil {
		log.Fatal(err)
	}
	gatekeeperClient := clients.NewGatekeeperClientBuilder().
		WithHostGetter(clients.HostGetterFunc(func() url.URL { return *gatekeeperHost })).
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

		enc := json.NewEncoder(res)

		iter := session.DB("").C("deviceData").
			Find(bson.M{"groupId": groupId}).
			Sort("-deviceTime").
			Iter()

		first := false
		res.Write([]byte("["))
		var result map[string]interface{}
		for iter.Next(&result) {
			if !first {
				first = true
			} else {
				res.Write([]byte(","))
			}
			fmt.Println(result)
			enc.Encode(result)
		}
		res.Write([]byte("]"))
		if err := iter.Close(); err != nil {
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
