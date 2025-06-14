package store

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	tpMongo "github.com/tidepool-org/go-common/clients/mongo"
)

const (
	dataCollectionName     = "deviceData"
	dataSetsCollectionName = "deviceDataSets" // all datum with type == "upload" go in this collection as opposed to the deviceData collection. These act as the root/parent for all other data.
	dataStoreAPIPrefix     = "api/data/store "
	RFC3339NanoSortable    = "2006-01-02T15:04:05.00000000Z07:00"
	medtronicDateFormat    = "2006-01-02"
	medtronicIndexDate     = "2017-09-01"
)

type (
	// StorageIterator - Interface for the query iterator
	StorageIterator interface {
		Next(context.Context) bool
		Decode(interface{}) error
		Close(context.Context) error
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

	// FieldFilter is a map with field names and a list of values to filter by
	FieldFilter map[string][]string

	// TypeFieldFilter is a map with types to which to apply field filters
	TypeFieldFilter map[string]FieldFilter

	// Params struct
	Params struct {
		UserID          string
		Types           []string
		SubTypes        []string
		TypeFieldFilter TypeFieldFilter
		Date
		*SchemaVersion
		Carelink              bool
		CBGFilter             bool
		CBGCloudDataSources   []bson.M
		DeviceID              string
		Latest                bool
		Medtronic             bool
		MedtronicDate         string
		MedtronicUploadIds    []string
		UploadID              string
		SampleIntervalMinimum int
	}

	// Date struct
	Date struct {
		Start time.Time
		End   time.Time
	}

	latestIterator struct {
		results []bson.Raw
		pos     int
	}

	// multiStorageIterator is a StorageIterator reads from multiple iterators
	// until there is no more data this is needed in the case that we are
	// reading multiple types and need to read both uploads and data.
	multiStorageIterator struct {
		iters          []StorageIterator
		currentIterIdx int
	}
)

var AllowedFieldFilters = TypeFieldFilter{
	"dosingDecision": FieldFilter{
		"reason": nil,
	},
}

func cleanDateString(dateString string) (time.Time, error) {
	date := time.Time{}

	if dateString == "" {
		return date, nil
	}

	date, err := time.Parse(time.RFC3339Nano, dateString)
	if err != nil {
		return date, err
	}

	return date, nil
}

// GetParams parses a URL to set parameters
func GetParams(q url.Values, schema *SchemaVersion) (*Params, error) {
	startDate, err := cleanDateString(q.Get("startDate"))
	if err != nil {
		return nil, err
	}

	endDate, err := cleanDateString(q.Get("endDate"))
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

	cbgFilter := true
	if values, ok := q["cbgFilter"]; ok {
		if len(values) < 1 {
			return nil, errors.New("cbgFilter parameter not valid")
		}
		cbgFilter, err = strconv.ParseBool(values[len(values)-1])
		if err != nil {
			return nil, errors.New("cbgFilter parameter not valid")
		}
	} else if values, ok := q["dexcom"]; ok { // Legacy
		if len(values) < 1 {
			return nil, errors.New("dexcom parameter not valid")
		}
		dexcom, err := strconv.ParseBool(values[len(values)-1])
		if err != nil {
			return nil, errors.New("dexcom parameter not valid")
		}
		cbgFilter = !dexcom // Inverted logic for backwards compatibility
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

	var sampleIntervalMinimum int
	if values, ok := q["sampleIntervalMinimum"]; ok {
		if len(values) < 1 {
			return nil, errors.New("sampleIntervalMinimum parameter not valid")
		}
		value, err := strconv.ParseInt(values[len(values)-1], 10, 32)
		if err != nil {
			return nil, errors.New("sampleIntervalMinimum parameter not valid")
		}
		sampleIntervalMinimum = int(value)
	}

	p := &Params{
		UserID:   q.Get(":userID"),
		DeviceID: q.Get("deviceId"),
		UploadID: q.Get("uploadId"),
		//the query params for type and subtype can contain multiple values separated
		//by a comma e.g. "type=smbg,cbg" so split them out into an array of values
		Types:                 strings.Split(q.Get("type"), ","),
		SubTypes:              strings.Split(q.Get("subType"), ","),
		TypeFieldFilter:       TypeFieldFilter{},
		Date:                  Date{startDate, endDate},
		SchemaVersion:         schema,
		Carelink:              carelink,
		CBGFilter:             cbgFilter,
		Latest:                latest,
		Medtronic:             medtronic,
		SampleIntervalMinimum: sampleIntervalMinimum,
	}

	// Parse the allowed filters to further restrict the result set,
	// e.g. "dosingDecision.reason=normalBolus,simpleBolus,watchBolus" to filter out dosing decisions
	// which have a 'reason' field other than [normalBolus,simpleBolus,watchBolus]
	for typ, fields := range AllowedFieldFilters {
		for field, _ := range fields {
			key := fmt.Sprintf("%s.%s", typ, field)
			value := q.Get(key)
			if len(value) > 0 {
				f, ok := p.TypeFieldFilter[typ]
				if !ok {
					f = FieldFilter{}
				}
				f[field] = strings.Split(value, ",")
				p.TypeFieldFilter[typ] = f
			}

		}
	}

	return p, nil

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
func (c *MongoStoreClient) WithContext(ctx context.Context) *MongoStoreClient {
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
	medtronicIndexDateTime, _ := time.Parse(medtronicDateFormat, medtronicIndexDate)
	dataIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "_userId", Value: 1}, {Key: "origin.payload.device.manufacturer", Value: 1}, {Key: "fakefield", Value: 1}},
			Options: options.Index().
				SetName("HasMedtronicLoopDataAfter_v2_DateTime").
				SetPartialFilterExpression(
					bson.D{
						{Key: "_active", Value: true},
						{Key: "origin.payload.device.manufacturer", Value: "Medtronic"},
						{Key: "time", Value: bson.M{
							"$gte": medtronicIndexDateTime,
						}},
					},
				),
		},
	}

	if _, err := dataCollection(c).Indexes().CreateMany(context.Background(), dataIndexes); err != nil {
		log.Fatal(dataStoreAPIPrefix, fmt.Sprintf("Unable to create indexes: %s", err))
	}

	// Not sure if all these indexes need to also be on the deviceDataSets collection.
	dataSetsIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "_userId", Value: 1}, {Key: "deviceModel", Value: 1}, {Key: "fakefield", Value: 1}},
			Options: options.Index().
				SetName("GetLoopableMedtronicDirectUploadIdsAfter_v2_DateTime").
				SetPartialFilterExpression(
					bson.D{
						{Key: "_active", Value: true},
						{Key: "type", Value: "upload"},
						{Key: "deviceModel", Value: bson.M{
							"$exists": true,
						}},
						{Key: "time", Value: bson.M{
							"$gte": medtronicIndexDateTime,
						}},
					},
				),
		},
		{
			Keys: bson.D{
				{Key: "_userId", Value: 1},
				{Key: "deviceManufacturers", Value: 1},
				{Key: "type", Value: 1},
				{Key: "deletedTime", Value: 1},
			},
			Options: options.Index().
				SetName("HasMedtronicDirectData").
				SetPartialFilterExpression(
					bson.D{
						{Key: "_active", Value: true},
						{Key: "_state", Value: "closed"},
					},
				),
		},
	}

	if _, err := dataSetsCollection(c).Indexes().CreateMany(context.Background(), dataSetsIndexes); err != nil {
		log.Fatal(dataStoreAPIPrefix, fmt.Sprintf("Unable to create indexes: %s", err))
	}

	return nil
}

func dataCollection(msc *MongoStoreClient) *mongo.Collection {
	return msc.client.Database(msc.database).Collection(dataCollectionName)
}

func dataSetsCollection(msc *MongoStoreClient) *mongo.Collection {
	return msc.client.Database(msc.database).Collection(dataSetsCollectionName)
}

// generateMongoQuery takes in a number of parameters and constructs a mongo query
// to retrieve objects from the Tidepool database. It is used by the router.Add("GET", "/{userID}"
// endpoint, which implements the Tide-whisperer API. See that function for further documentation
// on parameters
func generateMongoQuery(p *Params) bson.M {

	groupDataQuery := bson.M{
		"_userId": p.UserID,
		"_active": true}

	//if optional parameters are present, then add them to the query
	if len(p.Types) > 0 && p.Types[0] != "" {
		groupDataQuery["type"] = bson.M{"$in": p.Types}
	}

	if len(p.SubTypes) > 0 && p.SubTypes[0] != "" {
		groupDataQuery["subType"] = bson.M{"$in": p.SubTypes}
	}

	// The Golang implementation of time.RFC3339Nano does not use a fixed number of digits after the
	// decimal point and therefore is not reliably sortable. And so we use our own custom format for
	// database range queries that will properly sort any data with time stored as an ISO string.
	// See https://github.com/golang/go/issues/19635
	if !p.Date.Start.IsZero() && !p.Date.End.IsZero() {
		groupDataQuery["time"] = bson.M{"$gte": p.Date.Start, "$lte": p.Date.End}
	} else if !p.Date.Start.IsZero() {
		groupDataQuery["time"] = bson.M{"$gte": p.Date.Start}
	} else if !p.Date.End.IsZero() {
		groupDataQuery["time"] = bson.M{"$lte": p.Date.End}
	}

	if !p.Carelink {
		groupDataQuery["source"] = bson.M{"$ne": "carelink"}
	}

	if p.DeviceID != "" {
		groupDataQuery["deviceId"] = p.DeviceID
	}

	andQuery := []bson.M{}

	// If we have an explicit upload ID to filter by, we don't need or want to apply any further
	// data source-based filtering
	if p.UploadID != "" {
		groupDataQuery["uploadId"] = p.UploadID
	} else {
		if p.CBGFilter {
			cloudDataSetIds := primitive.A{}
			cloudDataTimeRanges := []bson.M{}
			for _, dataSource := range p.CBGCloudDataSources {
				if dataSetIds, ok := dataSource["dataSetIds"].(primitive.A); ok && len(dataSetIds) > 0 {
					cloudDataSetIds = append(cloudDataSetIds, dataSetIds...)
					if earliestDataTime, ok := dataSource["earliestDataTime"].(primitive.DateTime); ok {
						if latestDataTime, ok := dataSource["latestDataTime"].(primitive.DateTime); ok {
							cloudDataTimeRanges = append(cloudDataTimeRanges, bson.M{
								"time": bson.M{"$gte": earliestDataTime.Time().UTC(), "$lte": latestDataTime.Time().UTC()},
							})
						}
					}
				}
			}

			cloudQuery := []bson.M{}
			if len(cloudDataSetIds) > 0 {
				cloudQuery = append(cloudQuery, bson.M{"uploadId": bson.M{"$in": cloudDataSetIds}})
			}
			if len(cloudDataTimeRanges) > 0 {
				cloudQuery = append(cloudQuery, bson.M{"$nor": cloudDataTimeRanges})
			}

			if len(cloudQuery) > 0 {
				cloudQuery = append(cloudQuery, bson.M{"type": bson.M{"$ne": "cbg"}})
				andQuery = append(andQuery, bson.M{"$or": cloudQuery})
			}
		}

		if !p.Medtronic && len(p.MedtronicUploadIds) > 0 {
			medtronicDateTime, err := time.Parse(medtronicDateFormat, p.MedtronicDate)
			if err != nil {
				medtronicDateTime, _ = time.Parse(time.RFC3339, p.MedtronicDate)
			}
			medtronicQuery := []bson.M{
				{"time": bson.M{"$lt": medtronicDateTime}},
				{"type": bson.M{"$nin": []string{"basal", "bolus", "cbg"}}},
				{"uploadId": bson.M{"$nin": p.MedtronicUploadIds}},
			}
			andQuery = append(andQuery, bson.M{"$or": medtronicQuery})
		}
	}

	var orQueries []bson.M

	if p.SampleIntervalMinimum > 0 {
		orQueries = append(orQueries, bson.M{
			"$or": []bson.M{
				{"type": bson.M{"$ne": "cbg"}},
				{"sampleInterval": bson.M{"$exists": false}},
				{"sampleInterval": bson.M{"$gte": p.SampleIntervalMinimum}},
			},
		})
	}

	if len(p.TypeFieldFilter) > 0 {
		for typ, fields := range p.TypeFieldFilter {
			for field, values := range fields {
				orQueries = append(orQueries, bson.M{
					"$or": []bson.M{
						{"type": bson.M{"$ne": typ}},
						{field: bson.M{"$in": values}},
					},
				})
			}
		}
	}

	if len(orQueries) > 0 {
		andQuery = append(andQuery, orQueries...)
	}

	if len(andQuery) > 0 {
		groupDataQuery["$and"] = andQuery
	}

	return groupDataQuery
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

	err := dataSetsCollection(c).FindOne(c.context, query).Err()
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return false, err
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}

	return err == nil, err
}

// GetCBGCloudDataSources - get
func (c *MongoStoreClient) GetCBGCloudDataSources(userID string) ([]bson.M, error) {
	if userID == "" {
		return nil, errors.New("user id is missing")
	}

	// `earliestDataTime` and `latestDataTime` are bson.Date fields. Internally, they are int64's
	// so if they exist, the must be set to something, even if 0 (ie Unix epoch)
	query := bson.M{
		"userId": userID,
		"dataSetIds": bson.M{
			"$exists": true,
			"$not": bson.M{
				"$size": 0,
			},
		},
	}

	cursor, err := c.client.Database("tidepool").Collection("data_sources").Find(c.context, query)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	dataSources := []bson.M{}
	if err = cursor.All(c.context, &dataSources); err != nil {
		return nil, err
	}

	return dataSources, nil
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

	dateTime, err := time.Parse(medtronicDateFormat, date)
	if err != nil {
		dateTime, err = time.Parse(time.RFC3339, date)
	}
	if err != nil {
		return false, errors.New("date is invalid")
	}

	opts := options.FindOne()
	query := bson.M{
		"_active":                            true,
		"_userId":                            userID,
		"time":                               bson.M{"$gte": dateTime},
		"origin.payload.device.manufacturer": "Medtronic",
	}

	err = dataCollection(c).FindOne(c.context, query, opts).Err()
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return false, err
	}

	return err == nil, nil
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

	dateTime, err := time.Parse(medtronicDateFormat, date)
	if err != nil {
		dateTime, err = time.Parse(time.RFC3339, date)
	}
	if err != nil {
		return nil, errors.New("date is invalid")
	}

	opts := options.Find()
	opts.SetHint("GetLoopableMedtronicDirectUploadIdsAfter_v2_DateTime")
	opts.SetProjection(bson.M{"_id": 0, "uploadId": 1})

	query := bson.M{
		"_active":     true,
		"_userId":     userID,
		"time":        bson.M{"$gte": dateTime},
		"type":        "upload", // redundant since all types in collection is deviceDataSets is upload but just leaving the original query here.
		"deviceModel": bson.M{"$in": []string{"523", "523K", "554", "723", "723K", "754"}},
	}

	var objects []struct {
		UploadID string `bson:"uploadId"`
	}

	cursor, err := dataSetsCollection(c).Find(c.context, query, opts)
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

// GetDeviceData returns all the device data for a user
func (c *MongoStoreClient) GetDeviceData(p *Params) (StorageIterator, error) {

	// _schemaVersion is still in the list of fields to remove. Although we don't query for it, data can still exist for it
	// until BACK-1281 is done.
	removeFieldsForReturn := bson.M{"_id": 0, "_userId": 0, "_groupId": 0, "_version": 0, "_active": 0, "_schemaVersion": 0, "createdTime": 0, "modifiedTime": 0, "_migrationMarker": 0, "provenance": 0}

	if p.Latest {
		latest := &latestIterator{pos: -1}

		var typeRanges []string
		if len(p.Types) > 0 && p.Types[0] != "" {
			typeRanges = p.Types
		} else {
			typeRanges = []string{"physicalActivity", "basal", "cbg", "smbg", "bloodKetone", "bolus", "wizard", "deviceEvent", "food", "insulin", "cgmSettings", "pumpSettings", "reportedState", "upload"}
		}

		var err error

		for _, theType := range typeRanges {
			query := generateMongoQuery(p)
			query["type"] = theType
			opts := options.FindOne().SetProjection(removeFieldsForReturn).SetSort(bson.M{"time": -1})
			// collections to search. stop at first collection that has data.
			collection := dataCollection(c)
			if theType == "upload" {
				// Uploads are only in the deviceDataSets collection after migration completes.
				collection = dataSetsCollection(c)
			}
			result, resultErr := collection.
				FindOne(c.context, query, opts).
				DecodeBytes()
			if resultErr != nil {
				if resultErr == mongo.ErrNoDocuments {
					continue
				}
				err = resultErr
				break
			}

			latest.results = append(latest.results, result)
		}
		return latest, err
	}

	opts := options.Find().SetProjection(removeFieldsForReturn)

	mongoQuery := generateMongoQuery(p)

	// If query only needs to read from one collection use the collection directly.
	switch {
	case len(p.Types) == 1 && p.Types[0] == "upload":
		return dataSetsCollection(c).Find(c.context, mongoQuery, opts)
	// Have to check for empty string as sometimes that is the type sent.
	case len(p.Types) > 0 && !contains("upload", p.Types) && p.Types[0] != "":
		return dataCollection(c).Find(c.context, mongoQuery, opts)
	}

	// Otherwise query needs to read from both deviceData and deviceDataSets collection.
	dataIter, err := dataCollection(c).Find(c.context, mongoQuery, opts)
	if err != nil {
		return nil, err
	}
	dataSetIter, err := dataSetsCollection(c).Find(c.context, mongoQuery, opts)
	if err != nil {
		return nil, err
	}
	return &multiStorageIterator{
		iters: []StorageIterator{
			dataIter,
			dataSetIter,
		},
	}, nil
}

func (l *latestIterator) Next(context.Context) bool {
	l.pos++
	return l.pos < len(l.results)
}

func (l *latestIterator) Decode(result interface{}) error {
	return bson.Unmarshal(l.results[l.pos], result)
}

func (l *latestIterator) Close(context.Context) error {
	return nil
}

func (l *multiStorageIterator) Next(ctx context.Context) bool {
	if l.currentIterIdx >= len(l.iters) {
		return false
	}
	hasNext := l.iters[l.currentIterIdx].Next(ctx)
	if hasNext {
		return true
	}
	l.currentIterIdx++
	return l.Next(ctx)
}

func (l *multiStorageIterator) Decode(result interface{}) error {
	if l.currentIterIdx >= len(l.iters) {
		return io.EOF
	}

	return l.iters[l.currentIterIdx].Decode(result)
}

func (l *multiStorageIterator) Close(ctx context.Context) error {
	for _, iter := range l.iters {
		if err := iter.Close(ctx); err != nil {
			return err
		}
	}
	return nil
}

func contains(needle string, haystack []string) bool {
	for _, x := range haystack {
		if needle == x {
			return true
		}
	}
	return false
}
