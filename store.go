package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tidepool-org/go-common/clients/mongo"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

type StorageIterator interface {
	Next(result interface{}) bool
	Close() error
}

type Storage interface {
	Close()
	Ping() error
	GetDeviceData(p *params, res http.ResponseWriter)
}

const (
	data_collection = "deviceData"
)

type params struct {
	active bool
	userId string
	//userId is resolved to the groupId for use in queries
	groupId  string
	types    []string
	subTypes []string
	date
	schemaVersion *SchemaVersion
}

type date struct {
	start string
	end   string
}

func getParams(q url.Values, schema *SchemaVersion) (*params, error) {

	startStr := q.Get("startdate")
	endStr := q.Get("enddate")

	if startStr != "" {
		startDate, err := time.Parse(time.RFC3339Nano, startStr)
		if err != nil {
			return nil, err
		}
		startStr = startDate.Format(time.RFC3339Nano)
	}
	if endStr != "" {
		endDate, err := time.Parse(time.RFC3339Nano, endStr)
		if err != nil {
			return nil, err
		}
		endStr = endDate.Format(time.RFC3339Nano)
	}

	p := &params{
		userId: q.Get(":userID"),
		active: true,
		//the query params for type and subtype can contain multiple values seperated by a comma e.g. "type=smbg,cbg"
		//so split them out into an array of values
		types:         strings.Split(q.Get("type"), ","),
		subTypes:      strings.Split(q.Get("subtype"), ","),
		date:          date{startStr, endStr},
		schemaVersion: schema,
	}

	log.Println(DATA_API_PREFIX, fmt.Sprintf("****Params: %v", p))

	return p, nil

}

type MongoStoreClient struct {
	session *mgo.Session
}

func NewMongoStoreClient(config *mongo.Config) *MongoStoreClient {

	mongoSession, err := mongo.Connect(config)
	if err != nil {
		log.Fatal(DATA_API_PREFIX, err)
	}

	//index based on sort and where keys
	index := mgo.Index{
		Key:        []string{"_groupId", "_active", "_schemaVersion"},
		Background: true,
	}
	err = mgoDataCollection(mongoSession).EnsureIndex(index)
	if err != nil {
		log.Panic("Setting up base index", err.Error())
	}

	//index on type
	typeIndex := mgo.Index{
		Key:        []string{"type"},
		Background: true,
	}
	err = mgoDataCollection(mongoSession).EnsureIndex(typeIndex)
	if err != nil {
		log.Panic("Setting up type index", err.Error())
	}

	//index on subType
	subTypeIndex := mgo.Index{
		Key:        []string{"subType"},
		Background: true,
	}
	err = mgoDataCollection(mongoSession).EnsureIndex(subTypeIndex)
	if err != nil {
		log.Panic("Setting up subType index", err.Error())
	}

	return &MongoStoreClient{
		session: mongoSession,
	}
}

func mgoDataCollection(cpy *mgo.Session) *mgo.Collection {
	return cpy.DB("").C(data_collection)
}

// generateMongoQuery takes in a number of parameters and constructs a mongo query
// to retrieve objects from the Tidepool database. It is used by the router.Add("GET", "/{userID}"
// endpoint, which implements the Tide-whisperer API. See that function for further documentation
// on parameters
func generateMongoQuery(p *params) bson.M {

	groupDataQuery := bson.M{
		"_groupId":       p.groupId,
		"_active":        p.active,
		"_schemaVersion": bson.M{"$gte": p.schemaVersion.Minimum, "$lte": p.schemaVersion.Maximum}}

	//if optional parameters are present, then add them to the query
	if len(p.types) > 0 && p.types[0] != "" {
		groupDataQuery["type"] = bson.M{"$in": p.types}
	}

	if len(p.subTypes) > 0 && p.subTypes[0] != "" {
		groupDataQuery["subType"] = bson.M{"$in": p.subTypes}
	}

	if p.date.start != "" && p.date.end != "" {
		groupDataQuery["time"] = bson.M{"$gte": p.date.start, "$lte": p.date.end}
	} else if p.date.start != "" {
		groupDataQuery["time"] = bson.M{"$gte": p.date.start}
	} else if p.date.end != "" {
		groupDataQuery["time"] = bson.M{"$lte": p.date.end}
	}

	return groupDataQuery
}

func (d MongoStoreClient) Close() {
	log.Print(DATA_API_PREFIX, "Close the session")
	d.session.Close()
	return
}

func (d MongoStoreClient) Ping() error {
	// do we have a store session
	if err := d.session.Ping(); err != nil {
		return err
	}
	return nil
}

func (d MongoStoreClient) GetDeviceData(p *params, res http.ResponseWriter) {

	//process the found data and send the appropriate response
	processResults := func(res http.ResponseWriter, iter StorageIterator, startedAt time.Time) {
		var results map[string]interface{}
		found := 0
		first := false

		log.Println(DATA_API_PREFIX, fmt.Sprintf("mongo processing started after [%.5f]secs", time.Now().Sub(startedAt).Seconds()))

		for iter.Next(&results) {

			found = found + 1

			bytes, err := json.Marshal(results)
			if err != nil {

				res.WriteHeader(http.StatusInternalServerError)
				//jsonError(res, error_loading_events.setInternalMessage(err), startedAt)
				return
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

		log.Println(DATA_API_PREFIX, fmt.Sprintf("mongo processing finished after [%.5f]secs and returned [%d] records", time.Now().Sub(startedAt).Seconds(), found))

		if err := iter.Close(); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			//jsonError(res, error_running_query.setInternalMessage(err), startedAt)
			return
		}

		res.Write([]byte("]"))
		return
	}

	cpy := d.session.Copy()
	defer cpy.Close()

	started := time.Now()

	query := generateMongoQuery(p)

	removeFieldsForReturn := bson.M{"_id": 0, "_groupId": 0, "_version": 0, "_active": 0, "_schemaVersion": 0, "createdTime": 0, "modifiedTime": 0}

	//use an iterator to protect against very large queries
	iter := mgoDataCollection(cpy).
		Find(query).
		Select(removeFieldsForReturn).
		Iter()

	processResults(res, iter, started)
	return
}
