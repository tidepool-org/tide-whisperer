package store

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	tpMongo "github.com/tidepool-org/go-common/clients/mongo"
)

const (
	dataCollectionName          = "deviceData"
	dataStoreAPIPrefix          = "api/data/store "
	portalDb                    = "portal"
	parametersHistoryCollection = "patient_parameters_history"
)

type (
	// StorageIterator - Interface for the query iterator
	StorageIterator interface {
		Next(ctx context.Context) bool
		Close(ctx context.Context) error
		Decode(val interface{}) error
	}
	// Storage - Interface for our storage layer
	Storage interface {
		EnsureIndexes() error
		WithContext(ctx context.Context) Storage
		Close() error
		Ping() error
		GetDeviceData(p *Params) (StorageIterator, error)
		GetDexcomDataSource(userID string) (bson.M, error)
		GetDiabeloopParametersHistory(userID string, levels []int) (bson.M, error)
		GetLoopableMedtronicDirectUploadIdsAfter(userID string, date string) ([]string, error)
		GetDeviceModel(userID string) (string, error)
		GetTimeInRangeData(p *AggParams, logQuery bool) (StorageIterator, error)
		HasMedtronicDirectData(userID string) (bool, error)
		HasMedtronicLoopDataAfter(userID string, date string) (bool, error)
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
		LevelFilter        []int
	}

	// Date struct
	Date struct {
		Start string
		End   string
	}
)

func InArray(needle string, arr []string) bool {
	for _, n := range arr {
		if needle == n {
			return true
		}
	}
	return false
}

// NewMongoStoreClient creates a new MongoStoreClient
func NewMongoStoreClient(config *tpMongo.Config) *MongoStoreClient {
	connectionString, err := config.ToConnectionString()
	if err != nil {
		log.Fatal(dataStoreAPIPrefix, fmt.Sprintf("Invalid MongoDB configuration: %s", err))
	}

	clientOptions := options.Client().ApplyURI(connectionString)
	mongoClient, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(dataStoreAPIPrefix, fmt.Sprintf("Invalid MongoDB connection string: %s", err))
	}

	return &MongoStoreClient{
		client:   mongoClient,
		context:  context.Background(),
		database: config.Database,
	}
}

// WithContext returns a shallow copy of c with its context changed
// to ctx. The provided ctx must be non-nil.
func (c *MongoStoreClient) WithContext(ctx context.Context) Storage {
	if ctx == nil {
		panic("nil context")
	}
	c2 := new(MongoStoreClient)
	*c2 = *c
	c2.context = ctx
	return c2
}

// EnsureIndexes exist for the MongoDB collection. EnsureIndexes uses the Background() context, in order
// to pass back the MongoDB errors, rather than any context errors.
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
		{
			Keys: bson.D{{Key: "deviceId", Value: 1}, {Key: "time", Value: -1}, {Key: "type", Value: 1}},
			Options: options.Index().
				SetName("DeviceId").
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

	if _, err := dataCollection(c).Indexes().CreateMany(context.Background(), indexes); err != nil {
		log.Fatal(dataStoreAPIPrefix, fmt.Sprintf("Unable to create indexes: %s", err))
	}

	return nil
}

func dataCollection(msc *MongoStoreClient) *mongo.Collection {
	return msc.client.Database(msc.database).Collection(dataCollectionName)
}
func mgoParametersHistoryCollection(msc *MongoStoreClient) *mongo.Collection {
	return msc.client.Database(portalDb).Collection(parametersHistoryCollection)
}

// generateMongoQuery takes in a number of parameters and constructs a mongo query
// to retrieve objects from the Tidepool database. It is used by the router.Add("GET", "/{userID}"
// endpoint, which implements the Tide-whisperer API. See that function for further documentation
// on parameters
func generateMongoQuery(p *Params) bson.M {

	finalQuery := bson.M{}
	skipParamsQuery := false
	groupDataQuery := bson.M{
		"_userId":        p.UserID,
		"_active":        true,
		"_schemaVersion": bson.M{"$gte": p.SchemaVersion.Minimum, "$lte": p.SchemaVersion.Maximum}}

	//if optional parameters are present, then add them to the query
	if len(p.Types) > 0 && p.Types[0] != "" {
		groupDataQuery["type"] = bson.M{"$in": p.Types}
		if !InArray("deviceEvent", p.Types) {
			skipParamsQuery = true
		}
	}

	if len(p.SubTypes) > 0 && p.SubTypes[0] != "" {
		groupDataQuery["subType"] = bson.M{"$in": p.SubTypes}
		if !InArray("deviceParameter", p.SubTypes) {
			skipParamsQuery = true
		}
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
		skipParamsQuery = true
	}

	// If we have an explicit upload ID to filter by, we don't need or want to apply any further
	// data source-based filtering
	if p.UploadID != "" {
		groupDataQuery["uploadId"] = p.UploadID
		finalQuery = groupDataQuery
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
			finalQuery = groupDataQuery
		} else if skipParamsQuery || len(p.LevelFilter) == 0 {
			finalQuery = groupDataQuery
		} else {
			paramQuery := []bson.M{}
			// create the level filter as string
			levelFilterAsString := []string{}
			for value := range p.LevelFilter {
				levelFilterAsString = append(levelFilterAsString, strconv.Itoa(value))
			}
			paramQuery = append(paramQuery, groupDataQuery)

			deviceParametersQuery := bson.M{}
			deviceParametersQuery["type"] = "deviceEvent"
			deviceParametersQuery["subType"] = "deviceParameter"
			deviceParametersQuery["level"] = bson.M{"$in": levelFilterAsString}
			otherDataQuery := bson.M{}
			otherDataQuery["subType"] = bson.M{"$ne": "deviceParameter"}

			orQuery := []bson.M{}
			orQuery = append(orQuery, deviceParametersQuery)
			orQuery = append(orQuery, otherDataQuery)

			paramQuery = append(paramQuery, bson.M{"$or": orQuery})
			finalQuery = bson.M{"$and": paramQuery}
		}
	}

	return finalQuery
}

// Ping the MongoDB database
func (c *MongoStoreClient) Ping() error {
	// do we have a store session
	return c.client.Ping(c.context, nil)
}

// Disconnect from the MongoDB database
func (c *MongoStoreClient) Disconnect() error {
	return c.client.Disconnect(c.context)
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
func (c *MongoStoreClient) GetDeviceData(p *Params) (StorageIterator, error) {

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

func (c *MongoStoreClient) GetDiabeloopParametersHistory(userID string, levels []int) (bson.M, error) {
	if userID == "" {
		return nil, errors.New("user id is missing")
	}
	if levels == nil {
		levels = make([]int, 1)
		levels[0] = 1
	}

	var bsonLevels = make([]interface{}, len(levels))
	for i, d := range levels {
		bsonLevels[i] = d
	}

	// session := d.session.Copy()
	// defer session.Close()

	query := []bson.M{
		// Filtering on userid
		{
			"$match": bson.M{"userid": userID},
		},
		// unnesting history array (keeping index for future grouping)
		{
			"$unwind": bson.M{"path": "$history", "includeArrayIndex": "historyIdx"},
		},
		// unnesting history.parameters array
		{
			"$unwind": "$history.parameters",
		},
		// filtering level parameters
		{
			"$match": bson.M{
				"history.parameters.level": bson.M{"$in": bsonLevels},
			},
		},
		// removing unnecessary fields
		{
			"$project": bson.M{
				"userid":     1,
				"historyIdx": 1,
				"_id":        0,
				"parameters": bson.M{
					"changeType": "$history.parameters.changeType", "name": "$history.parameters.name",
					"value": "$history.parameters.value", "unit": "$history.parameters.unit",
					"level": "$history.parameters.level", "effectiveDate": "$history.parameters.effectiveDate",
				},
			},
		},
		// grouping by change
		{
			"$group": bson.M{
				"_id":        bson.M{"historyIdx": "$historyIdx", "userid": "$userid"},
				"parameters": bson.M{"$addToSet": "$parameters"},
				"changeDate": bson.M{"$max": "$parameters.effectiveDate"},
			},
		},
		// grouping all changes in one array
		{
			"$group": bson.M{
				"_id":     bson.M{"userid": "$userid"},
				"history": bson.M{"$addToSet": bson.M{"parameters": "$parameters", "changeDate": "$changeDate"}},
			},
		},
		// removing unnecessary fields
		{
			"$project": bson.M{"_id": 0},
		},
	}
	dataSources := []bson.M{}
	cursor, err := mgoParametersHistoryCollection(c).Aggregate(c.context, query)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(c.context)
	err = cursor.All(c.context, &dataSources)
	if err != nil {
		return nil, err
	} else if len(dataSources) == 0 {
		return nil, nil
	}

	return dataSources[0], nil
}
func (c *MongoStoreClient) GetDeviceModel(userID string) (string, error) {

	if userID == "" {
		return "", errors.New("user id is missing")
	}

	var payLoadDeviceNameQuery = make([]interface{}, 2)
	payLoadDeviceNameQuery[0] = bson.M{"payload.device.name": bson.M{"$exists": true}}
	payLoadDeviceNameQuery[1] = bson.M{"payload.device.name": bson.M{"$ne": nil}}

	query := bson.M{
		"_userId":        userID,
		"type":           "pumpSettings",
		"_schemaVersion": bson.M{"$gt": 0},
		"_active":        true,
		"$and":           payLoadDeviceNameQuery,
	}

	var res map[string]interface{}
	opts := options.FindOne()
	opts.SetSort(bson.D{{"time", -1}})
	opts.SetProjection(bson.M{"payload.device.name": 1})

	err := dataCollection(c).FindOne(c.context, query, opts).Decode(&res)
	if err != nil {
		return "", err
	}

	device := res["payload"].(map[string]interface{})["device"].(map[string]interface{})
	return device["name"].(string), err
}

func (c *MongoStoreClient) Close() error {
	return c.client.Disconnect(c.context)
}
