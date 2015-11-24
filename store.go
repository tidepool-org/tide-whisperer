package main

import (
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/tidepool-org/go-common/clients/mongo"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

const (
	data_collection = "deviceData"
)

type (
	//Interface for the query iterator
	StorageIterator interface {
		Next(result interface{}) bool
		Close() error
	}
	//Interface for our storage layer
	Storage interface {
		Close()
		Ping() error
		GetDeviceData(p *params) StorageIterator
	}
	//Mongo Storage Client
	MongoStoreClient struct {
		session *mgo.Session
	}

	params struct {
		active bool
		//userId comes from the request
		userId string
		//groupId is resolved from the incoming userid and if used storage in queries
		groupId  string
		types    []string
		subTypes []string
		date
		schemaVersion *SchemaVersion
	}

	date struct {
		start string
		end   string
	}
)

func cleanDateString(dateString string) (string, error) {
	if dateString == "" {
		return "", nil
	}
	date, err := time.Parse(time.RFC3339Nano, dateString)
	if err != nil {
		return "", err
	}
	return date.Format(time.RFC3339Nano), nil
}

func getParams(q url.Values, schema *SchemaVersion) (*params, error) {

	startStr, err := cleanDateString(q.Get("startdate"))
	if err != nil {
		return nil, err
	}

	endStr, err := cleanDateString(q.Get("enddate"))
	if err != nil {
		return nil, err
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

	return p, nil

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
	return d.session.Ping()
}

func (d MongoStoreClient) GetDeviceData(p *params) StorageIterator {

	cpy := d.session.Copy()
	//NOTE: We are not defering the close here as we
	//use the iterator to process the data to return

	removeFieldsForReturn := bson.M{"_id": 0, "_groupId": 0, "_version": 0, "_active": 0, "_schemaVersion": 0, "createdTime": 0, "modifiedTime": 0}

	//NOTE: We use an iterator to protect against very large queries
	return mgoDataCollection(cpy).
		Find(generateMongoQuery(p)).
		Select(removeFieldsForReturn).
		Iter()
}
