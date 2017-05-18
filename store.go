package main

import (
	"errors"
	"log"
	"net/url"
	"strconv"
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
		//userId comes from the request
		userId string
		//groupId is resolved from the incoming userid and if used storage in queries
		groupId  string
		types    []string
		subTypes []string
		date
		schemaVersion *SchemaVersion
		carelink      bool
	}

	date struct {
		start string
		end   string
	}

	ClosingSessionIterator struct {
		*mgo.Session
		*mgo.Iter
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

	startStr, err := cleanDateString(q.Get("startDate"))
	if err != nil {
		return nil, err
	}

	endStr, err := cleanDateString(q.Get("endDate"))
	if err != nil {
		return nil, err
	}

	carelink := false
	if values, ok := q["carelink"]; ok {
		if len(values) < 1 {
			return nil, errors.New("carelink parameter not valid")
		}
		carelink, err = strconv.ParseBool(values[len(values)-1])
		if err != nil {
			return nil, errors.New("carelink parameter not valid")
		}
	}

	p := &params{
		userId: q.Get(":userID"),
		//the query params for type and subtype can contain multiple values seperated
		//by a comma e.g. "type=smbg,cbg" so split them out into an array of values
		types:         strings.Split(q.Get("type"), ","),
		subTypes:      strings.Split(q.Get("subType"), ","),
		date:          date{startStr, endStr},
		schemaVersion: schema,
		carelink:      carelink,
	}

	return p, nil

}

func NewMongoStoreClient(config *mongo.Config) *MongoStoreClient {

	mongoSession, err := mongo.Connect(config)
	if err != nil {
		log.Fatal(DATA_API_PREFIX, err)
	}

	deviceDataCollection := mgoDataCollection(mongoSession)

	//index based on sort and where keys
	index := mgo.Index{
		Key:        []string{"_groupId", "_active", "_schemaVersion"},
		Background: true,
	}
	err = deviceDataCollection.EnsureIndex(index)
	if err != nil {
		log.Panic("Setting up base index", err.Error())
	}

	//index on type
	typeIndex := mgo.Index{
		Key:        []string{"type"},
		Background: true,
	}
	err = deviceDataCollection.EnsureIndex(typeIndex)
	if err != nil {
		log.Panic("Setting up type index", err.Error())
	}

	//index on subType
	subTypeIndex := mgo.Index{
		Key:        []string{"subType"},
		Background: true,
	}
	err = deviceDataCollection.EnsureIndex(subTypeIndex)
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
		"_active":        true,
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

	if !p.carelink {
		groupDataQuery["source"] = bson.M{"$ne": "carelink"}
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

func (d MongoStoreClient) HasMedtronicDirectData(userID string) (bool, error) {
	if userID == "" {
		return false, errors.New("user id is missing")
	}

	session := d.session.Copy()
	defer session.Close()

	query := bson.M{
		"_userId": userID,
		"type":    "upload",
		"_state":  "closed",
		"_active": true,
		"deletedTime": bson.M{
			"$exists": false,
		},
		"deviceManufacturers": "Medtronic",
	}

	count, err := mgoDataCollection(session).Find(query).Limit(1).Count()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (d MongoStoreClient) GetDeviceData(p *params) StorageIterator {

	removeFieldsForReturn := bson.M{"_id": 0, "_userId": 0, "_groupId": 0, "_version": 0, "_active": 0, "_schemaVersion": 0, "createdTime": 0, "modifiedTime": 0}

	// Note: We do not defer closing the session copy here as the iterator is returned back to be
	// caller for processing. Instead, we wrap the session and iterator in an object that
	// closes session when the iterator is closed. See ClosingSessionIterator below.
	session := d.session.Copy()

	iter := mgoDataCollection(session).
		Find(generateMongoQuery(p)).
		Select(removeFieldsForReturn).
		Iter()

	return &ClosingSessionIterator{session, iter}
}

func (i *ClosingSessionIterator) Next(result interface{}) bool {
	if i.Iter != nil {
		return i.Iter.Next(result)
	}
	return false
}

func (i *ClosingSessionIterator) Close() (err error) {
	if i.Iter != nil {
		err = i.Iter.Close()
		i.Iter = nil
	}
	if i.Session != nil {
		i.Session.Close()
		i.Session = nil
	}
	return err
}
