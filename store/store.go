package store

import (
	"context"
	"errors"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	tpMongo "github.com/tidepool-org/go-common/clients/mongo"
)

const (
	dataCollectionName = "deviceData"
	dataStoreAPIPrefix = "api/data/store"
)

type (
	// StorageIterator - Interface for the query iterator
	StorageIterator interface {
		Next(result interface{}) bool
		Close() error
	}
	// Storage - Interface for our storage layer
	Storage interface {
		Close()
		Ping() error
		GetDeviceData(p *Params) StorageIterator
	}
	// MongoStoreClient - Mongo Storage Client
	MongoStoreClient struct {
		client   *mongo.Client
		context  context.Context
		database string
	}

	// SchemaVersion struct
	SchemaVersion struct {
		Minimum int
		Maximum int
	}

	// Params struct
	Params struct {
		UserID   string
		Types    []string
		SubTypes []string
		Date
		*SchemaVersion
		Carelink           bool
		Dexcom             bool
		DexcomDataSource   bson.M
		DeviceID           string
		Latest             bool
		Medtronic          bool
		MedtronicDate      string
		MedtronicUploadIds []string
		UploadID           string
	}

	// Date struct
	Date struct {
		Start string
		End   string
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

// GetParams parses a URL to set parameters
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
		UserID:   q.Get(":userID"),
		DeviceID: q.Get("deviceId"),
		UploadID: q.Get("uploadId"),
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

// NewMongoStoreClient creates a new MongoStoreClient
func NewMongoStoreClient(config *tpMongo.Config) *MongoStoreClient {

	connectionString, err := config.ToConnectionString()
	if err != nil {
		log.Fatal(dataStoreAPIPrefix, err)
	}

	clientOptions := options.Client().ApplyURI(connectionString)
	mongoClient, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(dataStoreAPIPrefix, err)
	}

	return &MongoStoreClient{
		client:   mongoClient,
		context:  context.Background(),
		database: config.Database,
	}
}

// WithContext returns a shallow copy of c with its context changed
// to ctx. The provided ctx must be non-nil.
func (c *MongoStoreClient) WithContext(ctx context.Context) *MongoStoreClient {
	if ctx == nil {
		panic("nil context")
	}
	c2 := new(MongoStoreClient)
	*c2 = *c
	c2.context = ctx
	return c2
}

// EnsureIndexes exist for the MongoDB collection
func (c *MongoStoreClient) EnsureIndexes() error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "_userId", Value: 1}, {Key: "deviceModel", Value: 1}},
			Options: options.Index().
				SetName("GetLoopableMedtronicDirectUploadIdsAfter").
				SetBackground(true).
				SetPartialFilterExpression(
					bson.D{
						{Key: "_active", Value: true},
						{Key: "type", Value: "upload"},
						{Key: "deviceModel", Value: bson.M{
							"$exists": true,
						}},
						{Key: "time", Value: bson.M{
							"$gte": "2017-09-01",
						}},
						{Key: "_schemaVersion", Value: bson.M{
							"$gt": 0,
						}},
					},
				),
		},
		{
			Keys: bson.D{{Key: "_userId", Value: 1}, {Key: "origin.payload.device.manufacturer", Value: 1}},
			Options: options.Index().
				SetName("HasMedtronicLoopDataAfter").
				SetBackground(true).
				SetPartialFilterExpression(
					bson.D{
						{Key: "_active", Value: true},
						{Key: "origin.payload.device.manufacturer", Value: "Medtronic"},
						{Key: "time", Value: bson.M{
							"$gte": "2017-09-01",
						}},
						{Key: "_schemaVersion", Value: bson.M{
							"$gt": 0,
						}},
					},
				),
		},
		{
			Keys: bson.D{{Key: "_userId", Value: 1}, {Key: "time", Value: -1}, {Key: "type", Value: 1}},
			Options: options.Index().
				SetName("UserIdTimeWeighted").
				SetBackground(true).
				SetPartialFilterExpression(
					bson.D{
						{Key: "_schemaVersion", Value: bson.M{
							"$gt": 0,
						}},
						{Key: "_active", Value: true},
					},
				),
		},
	}

	opts := options.CreateIndexes().SetMaxTime(10 * time.Second)

	if _, err := dataCollection(c).Indexes().CreateMany(c.context, indexes, opts); err != nil {
		log.Fatal(dataStoreAPIPrefix, err)
	}

	return nil
}

func dataCollection(msc *MongoStoreClient) *mongo.Collection {
	return msc.client.Database(msc.database).Collection(dataCollectionName)
}

// generateMongoQuery takes in a number of parameters and constructs a mongo query
// to retrieve objects from the Tidepool database. It is used by the router.Add("GET", "/{userID}"
// endpoint, which implements the Tide-whisperer API. See that function for further documentation
// on parameters
func generateMongoQuery(p *Params) bson.M {

	groupDataQuery := bson.M{
		"_userId":        p.UserID,
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

	if p.DeviceID != "" {
		groupDataQuery["deviceId"] = p.DeviceID
	}

	// If we have an explicit upload ID to filter by, we don't need or want to apply any further
	// data source-based filtering
	if p.UploadID != "" {
		groupDataQuery["uploadId"] = p.UploadID
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

// Ping the MongoDB database
func (c *MongoStoreClient) Ping() error {
	// do we have a store session
	return c.client.Ping(c.context, nil)
}

// HasMedtronicDirectData - check whether the userID has Medtronic data that has been uploaded via Uploader
func (c *MongoStoreClient) HasMedtronicDirectData(userID string) (bool, error) {
	if userID == "" {
		return false, errors.New("user id is missing")
	}

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

	opts := options.Count().SetLimit(1)
	count, err := dataCollection(c).CountDocuments(c.context, query, opts)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetDexcomDataSource - get
func (c *MongoStoreClient) GetDexcomDataSource(userID string) (bson.M, error) {
	if userID == "" {
		return nil, errors.New("user id is missing")
	}

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
	opts := options.Find().SetLimit(1)
	cursor, err := c.client.Database("tidepool").Collection("data_sources").Find(c.context, query, opts)
	if err != nil {
		return nil, err
	}

	defer cursor.Close(c.context)
	if err = cursor.All(c.context, &dataSources); err != nil {
		return nil, err
	} else if len(dataSources) == 0 {
		return nil, nil
	}

	return dataSources[0], nil
}

// HasMedtronicLoopDataAfter checks the database to see if Loop data exists for `userID` that originated
// from a Medtronic device after `date`
func (c *MongoStoreClient) HasMedtronicLoopDataAfter(userID string, date string) (bool, error) {
	if userID == "" {
		return false, errors.New("user id is missing")
	}
	if date == "" {
		return false, errors.New("date is missing")
	}

	query := bson.D{
		{Key: "_active", Value: true},
		{Key: "_userId", Value: userID},
		{Key: "_schemaVersion", Value: bson.D{{Key: "$gt", Value: 0}}},
		{Key: "time", Value: bson.D{{Key: "$gte", Value: date}}},
		{Key: "origin.payload.device.manufacturer", Value: "Medtronic"},
	}

	count, err := dataCollection(c).CountDocuments(c.context, query)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetLoopableMedtronicDirectUploadIdsAfter returns all Upload IDs for `userID` where Loop data was found
// for a Medtronic device after `date`.
func (c *MongoStoreClient) GetLoopableMedtronicDirectUploadIdsAfter(userID string, date string) ([]string, error) {
	if userID == "" {
		return nil, errors.New("user id is missing")
	}
	if date == "" {
		return nil, errors.New("date is missing")
	}

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

	opts := options.Find().SetProjection(bson.M{"_id": 0, "uploadId": 1})
	cursor, err := dataCollection(c).Find(c.context, query, opts)
	if err != nil {
		return nil, err
	}

	defer cursor.Close(c.context)
	err = cursor.All(c.context, &objects)

	if err != nil {
		return nil, err
	}

	uploadIds := make([]string, len(objects))
	for index, object := range objects {
		uploadIds[index] = object.UploadID
	}

	return uploadIds, nil
}

// GetDeviceData returns all of the device data for a user
func (c *MongoStoreClient) GetDeviceData(p *Params) (*mongo.Cursor, error) {

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
		return dataCollection(c).Aggregate(c.context, pipeline)
	}

	opts := options.Find().
		SetProjection(bson.M{"_id": 0, "_userId": 0, "_groupId": 0, "_version": 0, "_active": 0, "_schemaVersion": 0, "createdTime": 0, "modifiedTime": 0})
	return dataCollection(c).
		Find(c.context, generateMongoQuery(p), opts)
}
