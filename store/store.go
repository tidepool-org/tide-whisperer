package store

import (
	"errors"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	"github.com/tidepool-org/go-common/clients/mongo"
)

const (
	data_collection       = "deviceData"
	DATA_STORE_API_PREFIX = "api/data/store"
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
		GetDeviceData(p *Params) StorageIterator
	}
	//Mongo Storage Client
	MongoStoreClient struct {
		session *mgo.Session
	}

	SchemaVersion struct {
		Minimum int
		Maximum int
	}

	Params struct {
		UserId   string
		Types    []string
		SubTypes []string
		Date
		*SchemaVersion
		Carelink           bool
		Dexcom             bool
		DexcomDataSource   bson.M
		DeviceId           string
		Latest             bool
		Medtronic          bool
		MedtronicDate      string
		MedtronicUploadIds []string
		UploadId           string
	}

	Date struct {
		Start string
		End   string
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

func GetParams(q url.Values, schema *SchemaVersion) (*Params, error) {

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

	dexcom := false
	if values, ok := q["dexcom"]; ok {
		if len(values) < 1 {
			return nil, errors.New("dexcom parameter not valid")
		}
		dexcom, err = strconv.ParseBool(values[len(values)-1])
		if err != nil {
			return nil, errors.New("dexcom parameter not valid")
		}
	}

	latest := false
	if values, ok := q["latest"]; ok {
		if len(values) < 1 {
			return nil, errors.New("latest parameter not valid")
		}
		latest, err = strconv.ParseBool(values[len(values)-1])
		if err != nil {
			return nil, errors.New("latest parameter not valid")
		}
	}

	medtronic := false
	if values, ok := q["medtronic"]; ok {
		if len(values) < 1 {
			return nil, errors.New("medtronic parameter not valid")
		}
		medtronic, err = strconv.ParseBool(values[len(values)-1])
		if err != nil {
			return nil, errors.New("medtronic parameter not valid")
		}
	}

	p := &Params{
		UserId:   q.Get(":userID"),
		DeviceId: q.Get("deviceId"),
		UploadId: q.Get("uploadId"),
		//the query params for type and subtype can contain multiple values seperated
		//by a comma e.g. "type=smbg,cbg" so split them out into an array of values
		Types:         strings.Split(q.Get("type"), ","),
		SubTypes:      strings.Split(q.Get("subType"), ","),
		Date:          Date{startStr, endStr},
		SchemaVersion: schema,
		Carelink:      carelink,
		Dexcom:        dexcom,
		Latest:        latest,
		Medtronic:     medtronic,
	}

	return p, nil

}

func NewMongoStoreClient(config *mongo.Config) *MongoStoreClient {

	mongoSession, err := mongo.Connect(config)
	if err != nil {
		log.Fatal(DATA_STORE_API_PREFIX, err)
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
func generateMongoQuery(p *Params) bson.M {

	groupDataQuery := bson.M{
		"_userId":        p.UserId,
		"_active":        true,
		"_schemaVersion": bson.M{"$gte": p.SchemaVersion.Minimum, "$lte": p.SchemaVersion.Maximum}}

	//if optional parameters are present, then add them to the query
	if len(p.Types) > 0 && p.Types[0] != "" {
		groupDataQuery["type"] = bson.M{"$in": p.Types}
	}

	if len(p.SubTypes) > 0 && p.SubTypes[0] != "" {
		groupDataQuery["subType"] = bson.M{"$in": p.SubTypes}
	}

	if p.Date.Start != "" && p.Date.End != "" {
		groupDataQuery["time"] = bson.M{"$gte": p.Date.Start, "$lte": p.Date.End}
	} else if p.Date.Start != "" {
		groupDataQuery["time"] = bson.M{"$gte": p.Date.Start}
	} else if p.Date.End != "" {
		groupDataQuery["time"] = bson.M{"$lte": p.Date.End}
	}

	if !p.Carelink {
		groupDataQuery["source"] = bson.M{"$ne": "carelink"}
	}

	if p.DeviceId != "" {
		groupDataQuery["deviceId"] = p.DeviceId
	}

	// If we have an explicit upload ID to filter by, we don't need or want to apply any further
	// data source-based filtering
	if p.UploadId != "" {
		groupDataQuery["uploadId"] = p.UploadId
	} else {
		andQuery := []bson.M{}
		if !p.Dexcom && p.DexcomDataSource != nil {
			dexcomQuery := []bson.M{
				{"type": bson.M{"$ne": "cbg"}},
				{"uploadId": bson.M{"$in": p.DexcomDataSource["dataSetIds"]}},
			}
			if earliestDataTime, ok := p.DexcomDataSource["earliestDataTime"].(time.Time); ok {
				dexcomQuery = append(dexcomQuery, bson.M{"time": bson.M{"$lt": earliestDataTime.Format(time.RFC3339)}})
			}
			if latestDataTime, ok := p.DexcomDataSource["latestDataTime"].(time.Time); ok {
				dexcomQuery = append(dexcomQuery, bson.M{"time": bson.M{"$gt": latestDataTime.Format(time.RFC3339)}})
			}
			andQuery = append(andQuery, bson.M{"$or": dexcomQuery})
		}

		if !p.Medtronic && len(p.MedtronicUploadIds) > 0 {
			medtronicQuery := []bson.M{
				{"time": bson.M{"$lt": p.MedtronicDate}},
				{"type": bson.M{"$nin": []string{"basal", "bolus", "cbg"}}},
				{"uploadId": bson.M{"$nin": p.MedtronicUploadIds}},
			}
			andQuery = append(andQuery, bson.M{"$or": medtronicQuery})
		}

		if len(andQuery) > 0 {
			groupDataQuery["$and"] = andQuery
		}
	}

	return groupDataQuery
}

func (d MongoStoreClient) Close() {
	log.Print(DATA_STORE_API_PREFIX, "Close the session")
	d.session.Close()
	return
}

func (d MongoStoreClient) Ping() error {
	session := d.session.Copy()
	defer session.Close()
	// do we have a store session
	return session.Ping()
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

func (d MongoStoreClient) GetDexcomDataSource(userID string) (bson.M, error) {
	if userID == "" {
		return nil, errors.New("user id is missing")
	}

	session := d.session.Copy()
	defer session.Close()

	query := bson.M{
		"userId":       userID,
		"providerType": "oauth",
		"providerName": "dexcom",
		"dataSetIds": bson.M{
			"$exists": true,
			"$not": bson.M{
				"$size": 0,
			},
		},
		"earliestDataTime": bson.M{
			"$exists": true,
		},
		"latestDataTime": bson.M{
			"$exists": true,
		},
	}

	dataSources := []bson.M{}
	err := session.DB("tidepool").C("data_sources").Find(query).Limit(1).All(&dataSources)
	if err != nil {
		return nil, err
	} else if len(dataSources) == 0 {
		return nil, nil
	}

	return dataSources[0], nil
}

func (d MongoStoreClient) HasMedtronicLoopDataAfter(userID string, date string) (bool, error) {
	if userID == "" {
		return false, errors.New("user id is missing")
	}
	if date == "" {
		return false, errors.New("date is missing")
	}

	session := d.session.Copy()
	defer session.Close()

	query := bson.M{
		"_active":                            true,
		"_userId":                            userID,
		"_schemaVersion":                     bson.M{"$gt": 0},
		"time":                               bson.M{"$gte": date},
		"origin.payload.device.manufacturer": "Medtronic",
	}

	count, err := mgoDataCollection(session).Find(query).Limit(1).Count()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (d MongoStoreClient) GetLoopableMedtronicDirectUploadIdsAfter(userID string, date string) ([]string, error) {
	if userID == "" {
		return nil, errors.New("user id is missing")
	}
	if date == "" {
		return nil, errors.New("date is missing")
	}

	session := d.session.Copy()
	defer session.Close()

	query := bson.M{
		"_active":        true,
		"_userId":        userID,
		"_schemaVersion": bson.M{"$gt": 0},
		"time":           bson.M{"$gte": date},
		"type":           "upload",
		"deviceModel":    bson.M{"$in": []string{"523", "523K", "554", "723", "723K", "754"}},
	}

	objects := []struct {
		UploadID string `bson:"uploadId"`
	}{}
	err := mgoDataCollection(session).Find(query).Select(bson.M{"_id": 0, "uploadId": 1}).All(&objects)
	if err != nil {
		return nil, err
	}

	uploadIds := make([]string, len(objects))
	for index, object := range objects {
		uploadIds[index] = object.UploadID
	}

	return uploadIds, nil
}

func (d MongoStoreClient) GetDeviceData(p *Params) StorageIterator {

	removeFieldsForReturn := bson.M{"_id": 0, "_userId": 0, "_groupId": 0, "_version": 0, "_active": 0, "_schemaVersion": 0, "createdTime": 0, "modifiedTime": 0}

	// Note: We do not defer closing the session copy here as the iterator is returned back to be
	// caller for processing. Instead, we wrap the session and iterator in an object that
	// closes session when the iterator is closed. See ClosingSessionIterator below.
	session := d.session.Copy()

	var iter *mgo.Iter

	if p.Latest {
		// Create an $aggregate query to return the latest of each `type` requested
		// that matches the query parameters
		pipeline := []bson.M{
			{
				"$match": generateMongoQuery(p),
			},
			{
				"$sort": bson.M{
					"type": 1,
					"time": -1,
				},
			},
			{
				"$group": bson.M{
					"_id": bson.M{
						"type": "$type",
					},
					"groupId": bson.M{
						"$first": "$_id",
					},
				},
			},
			{
				"$lookup": bson.M{
					"from":         "deviceData",
					"localField":   "groupId",
					"foreignField": "_id",
					"as":           "latest_doc",
				},
			},
			{
				"$unwind": "$latest_doc",
			},
			/*
				// TODO: we can only use this code once we upgrade to MongoDB 3.4+
				// We would also need to update the corresponding code in `tide-whisperer.go`
				// (search for "latest_doc")
				{
					"$replaceRoot": bson.M{
						"newRoot": "$latest_doc"
					},
				},
			*/
		}
		pipe := mgoDataCollection(session).Pipe(pipeline)
		iter = pipe.Iter()
	} else {
		iter = mgoDataCollection(session).
			Find(generateMongoQuery(p)).
			Select(removeFieldsForReturn).
			Iter()
	}

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
