package store

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	tpMongo "github.com/tidepool-org/go-common/clients/mongo"
	"go.mongodb.org/mongo-driver/bson"
)

var testingConfig = &tpMongo.Config{ConnectionString: "mongodb://127.0.0.1/data_test", Database: "data_test"}

func ptr[T any](v T) *T {
	return &v
}

type TestDataSchema struct {
	Active              *bool      `bson:"_active,omitempty"`
	UserId              *string    `bson:"_userId,omitempty"`
	State               *string    `bson:"_state,omitempty"`
	DeviceManufacturers *string    `bson:"deviceManufacturers,omitempty"`
	DeviceModel         *string    `bson:"deviceModel,omitempty"`
	Index               *string    `bson:"index,omitempty"`
	SchemaVersion       *int       `bson:"_schemaVersion,omitempty"`
	Time                *time.Time `bson:"time,omitempty"`
	DeletedTime         *time.Time `bson:"deletedTime,omitempty"`
	Type                *string    `bson:"type,omitempty"`
	Units               *string    `bson:"units,omitempty"`
	DeviceId            *string    `bson:"deviceId,omitempty"`
	UploadId            *string    `bson:"uploadId,omitempty"`
	Value               *float64   `bson:"value,omitempty"`
	Origin              *bson.M    `bson:"origin,omitempty"`
}

func before(t *testing.T, docs ...interface{}) *MongoStoreClient {

	store := NewMongoStoreClient(testingConfig)

	//INIT THE TEST - we use a clean copy of the collection before we start
	//just drop and don't worry about any errors
	dataCollection(store).Drop(context.TODO())

	if len(docs) > 0 {
		if _, err := dataCollection(store).InsertMany(store.context, docs); err != nil {
			t.Error("Unable to insert documents", err)
		}
	}

	return NewMongoStoreClient(testingConfig)
}

func getErrString(mongoQuery, expectedQuery bson.M) string {
	return "expected:\n" + formatForReading(expectedQuery) + "\ndid not match returned query\n" + formatForReading(mongoQuery)
}

func formatForReading(toFormat interface{}) string {
	formatted, _ := json.MarshalIndent(toFormat, "", "  ")
	return string(formatted)
}

func getCursors(exPlans interface{}) []string {
	var cursors []string

	if exPlans != nil {

		plans := exPlans.([]interface{})

		if plans != nil {
			for i := range plans {
				p := plans[i].(map[string]interface{})
				cursors = append(cursors, p["cursor"].(string))
			}
		}
	}
	return cursors
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if strings.Contains(a, e) {
			return true
		}
	}
	return false
}

func basicQuery() bson.M {
	qParams := &Params{
		UserID:        "abc123",
		SchemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		Dexcom:        true,
		Medtronic:     true,
	}

	return generateMongoQuery(qParams)
}

func allParams() *Params {
	earliestDataTime, _ := time.Parse(time.RFC3339, "2015-10-07T15:00:00Z")
	latestDataTime, _ := time.Parse(time.RFC3339, "2016-12-13T02:00:00Z")

	dateStart, _ := time.Parse(time.RFC3339, "2015-10-07T15:00:00.000Z")
	dateEnd, _ := time.Parse(time.RFC3339, "2015-10-11T15:00:00.000Z")

	return &Params{
		UserID:        "abc123",
		DeviceID:      "device123",
		SchemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		Date:          Date{dateStart, dateEnd},
		Types:         []string{"smbg", "cbg"},
		SubTypes:      []string{"stuff"},
		Carelink:      true,
		Dexcom:        false,
		DexcomDataSource: bson.M{
			"dataSetIds":       []string{"123", "456"},
			"earliestDataTime": primitive.NewDateTimeFromTime(earliestDataTime),
			"latestDataTime":   primitive.NewDateTimeFromTime(latestDataTime),
		},
		Latest:             false,
		Medtronic:          false,
		MedtronicDate:      "2017-01-01",
		MedtronicUploadIds: []string{"555666777", "888999000"},
	}
}

func allParamsQuery() bson.M {
	return generateMongoQuery(allParams())
}

func allParamsIncludingUploadIDQuery() bson.M {
	qParams := allParams()
	qParams.UploadID = "xyz123"

	return generateMongoQuery(qParams)
}

func typeAndSubtypeQuery() bson.M {
	qParams := &Params{
		UserID:             "abc123",
		SchemaVersion:      &SchemaVersion{Maximum: 2, Minimum: 0},
		Types:              []string{"smbg", "cbg"},
		SubTypes:           []string{"stuff"},
		Dexcom:             true,
		Medtronic:          false,
		MedtronicDate:      "2017-01-01",
		MedtronicUploadIds: []string{"555666777", "888999000"},
	}
	return generateMongoQuery(qParams)
}

func uploadIDQuery() bson.M {
	qParams := &Params{
		UserID:        "abc123",
		SchemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		UploadID:      "xyz123",
	}
	return generateMongoQuery(qParams)
}

func testDataForLatestTests() map[string]TestDataSchema {
	date1, _ := time.Parse(time.RFC3339, "2019-03-15T01:24:28.000Z")
	date2, _ := time.Parse(time.RFC3339, "2019-03-15T00:42:51.902Z")
	date3, _ := time.Parse(time.RFC3339, "2019-03-14T01:24:28.000Z")
	date4, _ := time.Parse(time.RFC3339, "2019-03-14T00:42:51.902Z")
	date5, _ := time.Parse(time.RFC3339, "2019-03-19T01:24:28.000Z")
	date6, _ := time.Parse(time.RFC3339, "2019-03-19T00:42:51.902Z")

	// We keep _schemaVersion in the test data until BACK-1281 is completed.
	testData := map[string]TestDataSchema{
		"upload1": {
			Active:        ptr(true),
			UserId:        ptr("abc123"),
			SchemaVersion: ptr(1),
			Time:          ptr(date1),
			Type:          ptr("upload"),
			DeviceId:      ptr("dev123"),
			UploadId:      ptr("9244bb16e27c4973c2f37af81784a05d"),
		},
		"cbg1": {
			Active:        ptr(true),
			UserId:        ptr("abc123"),
			SchemaVersion: ptr(1),
			Time:          ptr(date2),
			Type:          ptr("cbg"),
			Units:         ptr("mmol/L"),
			DeviceId:      ptr("dev123"),
			UploadId:      ptr("9244bb16e27c4973c2f37af81784a05d"),
			Value:         ptr(12.82223),
		},
		"upload2": {
			Active:        ptr(true),
			UserId:        ptr("abc123"),
			SchemaVersion: ptr(1),
			Time:          ptr(date3),
			Type:          ptr("upload"),
			DeviceId:      ptr("dev456"),
			UploadId:      ptr("zzz4bb16e27c4973c2f37af81784a05d"),
		},
		"cbg2": {
			Active:        ptr(true),
			UserId:        ptr("abc123"),
			SchemaVersion: ptr(1),
			Time:          ptr(date4),
			Type:          ptr("cbg"),
			Units:         ptr("mmol/L"),
			DeviceId:      ptr("dev456"),
			UploadId:      ptr("zzz4bb16e27c4973c2f37af81784a05d"),
			Value:         ptr(9.7213),
		},
		"upload3": {
			Active:        ptr(true),
			UserId:        ptr("xyz123"),
			SchemaVersion: ptr(1),
			Time:          ptr(date5),
			Type:          ptr("upload"),
			DeviceId:      ptr("dev789"),
			UploadId:      ptr("xxx4bb16e27c4973c2f37af81784a05d"),
		},
		"cbg3": {
			Active:        ptr(true),
			UserId:        ptr("xyz123"),
			SchemaVersion: ptr(1),
			Time:          ptr(date6),
			Type:          ptr("cbg"),
			Units:         ptr("mmol/L"),
			DeviceId:      ptr("dev789"),
			UploadId:      ptr("xxx4bb16e27c4973c2f37af81784a05d"),
			Value:         ptr(7.1237),
		},
	}

	return testData
}

func dropInternalKeys(inputData TestDataSchema) TestDataSchema {
	// NOTE we do not deep copy here, as we don't need the internal data to not be the same
	// we just need the ability to set some of them to nil independently of the source
	outputData := inputData
	outputData.UserId = nil
	outputData.Active = nil
	outputData.SchemaVersion = nil
	outputData.State = nil

	return outputData
}

func storeDataForLatestTests() []interface{} {
	testData := testDataForLatestTests()

	storeData := make([]interface{}, len(testData))
	index := 0
	for _, v := range testData {
		storeData[index] = v
		index++
	}

	return storeData
}

func TestStore_EnsureIndexes(t *testing.T) {
	store := before(t)
	err := store.EnsureIndexes()
	if err != nil {
		t.Error("Failed to run EnsureIndexes()")
	}

	indexView := dataCollection(store).Indexes()
	cursor, err := indexView.List(context.TODO())
	if err != nil {
		t.Error("Unexpected error fetching indexes")
	}

	defer cursor.Close(context.Background())
	type mongoIndex struct {
		Key                     bson.D
		Name                    string
		Background              bool
		Unique                  bool
		PartialFilterExpression bson.D
	}
	var indexes []mongoIndex

	for cursor.Next(context.Background()) {
		r := mongoIndex{}

		if err = cursor.Decode(&r); err != nil {
			break
		}
		indexes = append(indexes, r)
	}
	if err != nil {
		t.Error("Unexpected error decoding indexes")
	}

	makeKeySlice := func(mgoList ...string) bson.D {
		keySlice := bson.D{}
		for _, key := range mgoList {
			order := int32(1)
			if key[0] == '-' {
				order = int32(-1)
				key = key[1:]
			}
			keySlice = append(keySlice, bson.E{Key: key, Value: order})
		}
		return keySlice
	}

	medtronicIndexDateTime, _ := time.Parse(medtronicDateFormat, medtronicIndexDate)

	expectedIndexes := []mongoIndex{
		{
			Key:  makeKeySlice("_id"),
			Name: "_id_",
		},
		{
			Key:        makeKeySlice("_userId", "deviceModel", "fakefield"),
			Background: true,
			PartialFilterExpression: bson.D{
				{Key: "_active", Value: true},
				{Key: "type", Value: "upload"},
				{Key: "deviceModel", Value: bson.D{
					{Key: "$exists", Value: true},
				}},
				{Key: "time", Value: bson.D{
					{Key: "$gte", Value: primitive.NewDateTimeFromTime(medtronicIndexDateTime)},
				}},
			},
			Name: "GetLoopableMedtronicDirectUploadIdsAfter_v2_DateTime",
		},
		{
			Key:        makeKeySlice("_userId", "origin.payload.device.manufacturer", "fakefield"),
			Background: true,
			PartialFilterExpression: bson.D{
				{Key: "_active", Value: true},
				{Key: "origin.payload.device.manufacturer", Value: "Medtronic"},
				{Key: "time", Value: bson.D{
					{Key: "$gte", Value: primitive.NewDateTimeFromTime(medtronicIndexDateTime)},
				}},
			},
			Name: "HasMedtronicLoopDataAfter_v2_DateTime",
		},
		{
			Key:        makeKeySlice("_userId", "-time", "type"),
			Background: true,
			PartialFilterExpression: bson.D{
				{Key: "_active", Value: true},
			},
			Name: "UserIdTimeWeighted_v2",
		},
	}

	eq := reflect.DeepEqual(indexes, expectedIndexes)
	if !eq {
		t.Error(fmt.Sprintf("expected:\n%+#v\ngot:\n%+#v\n", expectedIndexes, indexes))
	}
}

func TestStore_generateMongoQuery_basic(t *testing.T) {

	time.Now()
	query := basicQuery()

	expectedQuery := bson.M{
		"_userId": "abc123",
		"_active": true,
		"source": bson.M{
			"$ne": "carelink",
		},
	}

	eq := reflect.DeepEqual(query, expectedQuery)
	if !eq {
		t.Error(getErrString(query, expectedQuery))
	}

}

func TestStore_generateMongoQuery_allParams(t *testing.T) {

	query := allParamsQuery()

	timeStart, _ := time.Parse(time.RFC3339, "2015-10-07T15:00:00.00Z")
	timeEnd, _ := time.Parse(time.RFC3339, "2015-10-11T15:00:00.00Z")

	dexcomStart, _ := time.Parse(time.RFC3339, "2015-10-07T15:00:00Z")
	dexcomEnd, _ := time.Parse(time.RFC3339, "2016-12-13T02:00:00Z")

	medtronicEnd, _ := time.Parse(time.RFC3339, "2017-01-01T00:00:00Z")

	expectedQuery := bson.M{
		"_userId":  "abc123",
		"deviceId": "device123",
		"_active":  true,
		"type":     bson.M{"$in": strings.Split("smbg,cbg", ",")},
		"subType":  bson.M{"$in": strings.Split("stuff", ",")},
		"time":     bson.M{"$gte": timeStart, "$lte": timeEnd},
		"$and": []bson.M{
			{"$or": []bson.M{
				{"type": bson.M{"$ne": "cbg"}},
				{"uploadId": bson.M{"$in": []string{"123", "456"}}},
				{"time": bson.M{"$lt": dexcomStart}},
				{"time": bson.M{"$gt": dexcomEnd}},
			}},
			{"$or": []bson.M{
				{"time": bson.M{"$lt": medtronicEnd}},
				{"type": bson.M{"$nin": []string{"basal", "bolus", "cbg"}}},
				{"uploadId": bson.M{"$nin": []string{"555666777", "888999000"}}},
			}},
		},
	}

	eq := reflect.DeepEqual(query, expectedQuery)
	if !eq {
		t.Error(getErrString(query, expectedQuery))
	}
}

func TestStore_generateMongoQuery_allparamsWithUploadId(t *testing.T) {

	query := allParamsIncludingUploadIDQuery()

	timeStart, _ := time.Parse(time.RFC3339, "2015-10-07T15:00:00.00Z")
	timeEnd, _ := time.Parse(time.RFC3339, "2015-10-11T15:00:00.00Z")

	expectedQuery := bson.M{
		"_userId":  "abc123",
		"deviceId": "device123",
		"_active":  true,
		"type":     bson.M{"$in": strings.Split("smbg,cbg", ",")},
		"subType":  bson.M{"$in": strings.Split("stuff", ",")},
		"uploadId": "xyz123",
		"time":     bson.M{"$gte": timeStart, "$lte": timeEnd},
	}

	eq := reflect.DeepEqual(query, expectedQuery)
	if !eq {
		t.Error(getErrString(query, expectedQuery))
	}
}

func TestStore_generateMongoQuery_uploadId(t *testing.T) {

	query := uploadIDQuery()

	expectedQuery := bson.M{
		"_userId":  "abc123",
		"_active":  true,
		"uploadId": "xyz123",
		"source": bson.M{
			"$ne": "carelink",
		},
	}

	eq := reflect.DeepEqual(query, expectedQuery)
	if !eq {
		t.Error(getErrString(query, expectedQuery))
	}
}

func TestStore_generateMongoQuery_noDates(t *testing.T) {

	query := typeAndSubtypeQuery()

	medtronicEnd, _ := time.Parse(time.RFC3339, "2017-01-01T00:00:00Z")

	expectedQuery := bson.M{
		"_userId": "abc123",
		"_active": true,
		"type":    bson.M{"$in": strings.Split("smbg,cbg", ",")},
		"subType": bson.M{"$in": strings.Split("stuff", ",")},
		"source": bson.M{
			"$ne": "carelink",
		},
		"$and": []bson.M{
			{"$or": []bson.M{
				{"time": bson.M{"$lt": medtronicEnd}},
				{"type": bson.M{"$nin": []string{"basal", "bolus", "cbg"}}},
				{"uploadId": bson.M{"$nin": []string{"555666777", "888999000"}}},
			}},
		},
	}

	eq := reflect.DeepEqual(query, expectedQuery)
	if !eq {
		t.Error(getErrString(query, expectedQuery))
	}
}

func TestStore_Ping(t *testing.T) {

	store := before(t)
	err := store.Ping()

	if err != nil {
		t.Error("there should be no error but got", err.Error())
	}
}

func TestStore_cleanDateString_empty(t *testing.T) {

	date, err := cleanDateString("")

	if !date.IsZero() {
		t.Error("the returned date should have been zero but got ", date)
	}
	if err != nil {
		t.Error("didn't expect an error but got ", err.Error())
	}

}

func TestStore_cleanDateString_nonsensical(t *testing.T) {

	date, err := cleanDateString("blah")

	if !date.IsZero() {
		t.Error("the returned date should have been empty zero but got ", date)
	}
	if err == nil {
		t.Error("we should have been given an error")
	}

}

func TestStore_cleanDateString_wrongFormat(t *testing.T) {

	date, err := cleanDateString("2006-20-02T3:04pm")

	if !date.IsZero() {
		t.Error("the returned date should have been empty but got ", date)
	}
	if err == nil {
		t.Error("we should have been given an error")
	}

}

func TestStore_cleanDateString(t *testing.T) {

	date, err := cleanDateString("2015-10-10T15:00:00.000Z")
	targetDate, _ := time.Parse(RFC3339NanoSortable, "2015-10-10T15:00:00.00000000Z")

	if date.IsZero() {
		t.Error("the returned date should not be empty")
	}

	if date != targetDate {
		t.Error("the returned date should equal 2015-10-10T15:00:00.00000000Z")
	}

	if err != nil {
		t.Error("we should have no error but got ", err.Error())
	}

}

func TestStore_GetParams_Empty(t *testing.T) {
	query := url.Values{
		":userID": []string{"1122334455"},
	}
	schema := &SchemaVersion{Minimum: 1, Maximum: 3}

	expectedParams := &Params{
		UserID:        "1122334455",
		SchemaVersion: schema,
		Types:         []string{""},
		SubTypes:      []string{""},
	}

	params, err := GetParams(query, schema)

	if err != nil {
		t.Error("should not have received error, but got one")
	}
	if !reflect.DeepEqual(params, expectedParams) {
		t.Error(fmt.Sprintf("params %#v do not equal expected params %#v", params, expectedParams))
	}
}

func TestStore_GetParams_Medtronic(t *testing.T) {
	query := url.Values{
		":userID":   []string{"1122334455"},
		"medtronic": []string{"true"},
	}
	schema := &SchemaVersion{Minimum: 1, Maximum: 3}

	expectedParams := &Params{
		UserID:        "1122334455",
		SchemaVersion: schema,
		Types:         []string{""},
		SubTypes:      []string{""},
		Medtronic:     true,
	}

	params, err := GetParams(query, schema)

	if err != nil {
		t.Error("should not have received error, but got one")
	}
	if !reflect.DeepEqual(params, expectedParams) {
		t.Error(fmt.Sprintf("params %#v do not equal expected params %#v", params, expectedParams))
	}
}

func TestStore_GetParams_UploadId(t *testing.T) {
	query := url.Values{
		":userID":  []string{"1122334455"},
		"uploadId": []string{"xyz123"},
	}
	schema := &SchemaVersion{Minimum: 1, Maximum: 3}

	expectedParams := &Params{
		UserID:        "1122334455",
		SchemaVersion: schema,
		Types:         []string{""},
		SubTypes:      []string{""},
		UploadID:      "xyz123",
	}

	params, err := GetParams(query, schema)

	if err != nil {
		t.Error("should not have received error, but got one")
	}
	if !reflect.DeepEqual(params, expectedParams) {
		t.Error(fmt.Sprintf("params %#v do not equal expected params %#v", params, expectedParams))
	}
}

func TestStore_HasMedtronicDirectData_UserID_Missing(t *testing.T) {
	store := before(t)

	hasMedtronicDirectData, err := store.HasMedtronicDirectData("")

	if err == nil {
		t.Error("should have received error, but got nil")
	}
	if hasMedtronicDirectData {
		t.Error("should not have Medtronic Direct data, but got some")
	}
}

func TestStore_HasMedtronicDirectData_Found(t *testing.T) {
	store := before(t, bson.M{
		"_userId":             "1234567890",
		"type":                "upload",
		"_state":              "closed",
		"_active":             true,
		"deviceManufacturers": "Medtronic",
	})

	hasMedtronicDirectData, err := store.HasMedtronicDirectData("1234567890")

	if err != nil {
		t.Error("failure querying HasMedtronicDirectData", err)
	}
	if !hasMedtronicDirectData {
		t.Error("should have Medtronic Direct data, but got none")
	}
}

func TestStore_HasMedtronicDirectData_Found_Multiple(t *testing.T) {
	date1, _ := time.Parse(time.RFC3339, "2017-05-17T20:13:32.064-0700")
	store := before(t, TestDataSchema{
		UserId:              ptr("0000000000"),
		Type:                ptr("upload"),
		State:               ptr("closed"),
		Active:              ptr(true),
		DeviceManufacturers: ptr("Medtronic"),
		Index:               ptr("0"),
	}, TestDataSchema{
		UserId:              ptr("1234567890"),
		Type:                ptr("upload"),
		State:               ptr("closed"),
		Active:              ptr(true),
		DeviceManufacturers: ptr("Medtronic"),
		Index:               ptr("1"),
	}, TestDataSchema{
		UserId:              ptr("1234567890"),
		Type:                ptr("upload"),
		State:               ptr("open"),
		Active:              ptr(true),
		DeviceManufacturers: ptr("Medtronic"),
		Index:               ptr("2"),
	}, TestDataSchema{
		UserId:              ptr("1234567890"),
		Type:                ptr("upload"),
		State:               ptr("closed"),
		Active:              ptr(true),
		DeviceManufacturers: ptr("Medtronic"),
		Index:               ptr("3"),
	}, TestDataSchema{
		UserId:              ptr("1234567890"),
		Type:                ptr("upload"),
		State:               ptr("closed"),
		Active:              ptr(true),
		DeletedTime:         ptr(date1),
		DeviceManufacturers: ptr("Medtronic"),
		Index:               ptr("4"),
	})

	hasMedtronicDirectData, err := store.HasMedtronicDirectData("1234567890")

	if err != nil {
		t.Error("failure querying HasMedtronicDirectData", err)
	}
	if !hasMedtronicDirectData {
		t.Error("should have Medtronic Direct data, but got none")
	}
}

func TestStore_HasMedtronicDirectData_NotFound_UserID(t *testing.T) {
	store := before(t, bson.M{
		"_userId":             "0000000000",
		"type":                "upload",
		"_state":              "closed",
		"_active":             true,
		"deviceManufacturers": "Medtronic",
	})

	hasMedtronicDirectData, err := store.HasMedtronicDirectData("1234567890")

	if err != nil {
		t.Error("failure querying HasMedtronicDirectData", err)
	}
	if hasMedtronicDirectData {
		t.Error("should not have Medtronic Direct data, but got some")
	}
}

func TestStore_HasMedtronicDirectData_NotFound_Type(t *testing.T) {
	store := before(t, bson.M{
		"_userId":             "1234567890",
		"type":                "cgm",
		"_state":              "closed",
		"_active":             true,
		"deviceManufacturers": "Medtronic",
	})

	hasMedtronicDirectData, err := store.HasMedtronicDirectData("1234567890")

	if err != nil {
		t.Error("failure querying HasMedtronicDirectData", err)
	}
	if hasMedtronicDirectData {
		t.Error("should not have Medtronic Direct data, but got some")
	}
}

func TestStore_HasMedtronicDirectData_NotFound_State(t *testing.T) {
	store := before(t, bson.M{
		"_userId":             "1234567890",
		"type":                "upload",
		"_state":              "open",
		"_active":             true,
		"deviceManufacturers": "Medtronic",
	})

	hasMedtronicDirectData, err := store.HasMedtronicDirectData("1234567890")

	if err != nil {
		t.Error("failure querying HasMedtronicDirectData", err)
	}
	if hasMedtronicDirectData {
		t.Error("should not have Medtronic Direct data, but got some")
	}
}

func TestStore_HasMedtronicDirectData_NotFound_Active(t *testing.T) {
	store := before(t, bson.M{
		"_userId":             "1234567890",
		"type":                "upload",
		"_state":              "closed",
		"_active":             false,
		"deviceManufacturers": "Medtronic",
	})

	hasMedtronicDirectData, err := store.HasMedtronicDirectData("1234567890")

	if err != nil {
		t.Error("failure querying HasMedtronicDirectData", err)
	}
	if hasMedtronicDirectData {
		t.Error("should not have Medtronic Direct data, but got some")
	}
}

func TestStore_HasMedtronicDirectData_NotFound_DeletedTime(t *testing.T) {
	date1, _ := time.Parse(time.RFC3339, "2017-05-17T20:10:26.607-0700")
	store := before(t, bson.M{
		"_userId":             "1234567890",
		"type":                "upload",
		"_state":              "closed",
		"_active":             true,
		"deletedTime":         date1,
		"deviceManufacturers": "Medtronic",
	})

	hasMedtronicDirectData, err := store.HasMedtronicDirectData("1234567890")

	if err != nil {
		t.Error("failure querying HasMedtronicDirectData", err)
	}
	if hasMedtronicDirectData {
		t.Error("should not have Medtronic Direct data, but got some")
	}
}

func TestStore_HasMedtronicDirectData_NotFound_DeviceManufacturer(t *testing.T) {
	store := before(t, bson.M{
		"_userId":             "1234567890",
		"type":                "upload",
		"_state":              "closed",
		"_active":             true,
		"deviceManufacturers": "Acme",
	})

	hasMedtronicDirectData, err := store.HasMedtronicDirectData("1234567890")

	if err != nil {
		t.Error("failure querying HasMedtronicDirectData", err)
	}
	if hasMedtronicDirectData {
		t.Error("should not have Medtronic Direct data, but got some")
	}
}

func TestStore_HasMedtronicDirectData_NotFound_Multiple(t *testing.T) {
	date1, _ := time.Parse(time.RFC3339Nano, "2017-05-17T20:13:32.064-07:00")
	store := before(t, TestDataSchema{
		UserId:              ptr("0000000000"),
		Type:                ptr("upload"),
		State:               ptr("closed"),
		Active:              ptr(true),
		DeviceManufacturers: ptr("Medtronic"),
		Index:               ptr("0"),
	}, TestDataSchema{
		UserId:              ptr("1234567890"),
		Type:                ptr("cgm"),
		State:               ptr("closed"),
		Active:              ptr(true),
		DeviceManufacturers: ptr("Medtronic"),
		Index:               ptr("1"),
	}, TestDataSchema{
		UserId:              ptr("1234567890"),
		Type:                ptr("upload"),
		State:               ptr("open"),
		Active:              ptr(true),
		DeviceManufacturers: ptr("Medtronic"),
		Index:               ptr("2"),
	}, TestDataSchema{
		UserId:              ptr("1234567890"),
		Type:                ptr("upload"),
		State:               ptr("closed"),
		Active:              ptr(false),
		DeviceManufacturers: ptr("Medtronic"),
		Index:               ptr("3"),
	}, TestDataSchema{
		UserId:              ptr("1234567890"),
		Type:                ptr("upload"),
		State:               ptr("closed"),
		Active:              ptr(true),
		DeletedTime:         ptr(date1),
		DeviceManufacturers: ptr("Medtronic"),
		Index:               ptr("4"),
	})

	hasMedtronicDirectData, err := store.HasMedtronicDirectData("1234567890")

	if err != nil {
		t.Error("failure querying HasMedtronicDirectData", err)
	}
	if hasMedtronicDirectData {
		t.Error("should not have Medtronic Direct data, but got some")
	}
}

func TestStore_HasMedtronicLoopDataAfter_NotFound_UserID(t *testing.T) {
	date1, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date2, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date3, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	// We keep _schemaVersion in the test data until BACK-1281 is completed.
	store := before(t, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("0000000000"),
		SchemaVersion: ptr(1),
		Time:          ptr(date1),
		Origin:        ptr(bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}}),
	}, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date2),
		Origin:        ptr(bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Animas"}}}),
	}, TestDataSchema{
		Active:        ptr(false),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date3),
		Origin:        ptr(bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}}),
	})

	// we need indexes here as the following queries rely on working index hints for performance
	err := store.EnsureIndexes()
	if err != nil {
		t.Error("Failed to run EnsureIndexes()")
	}

	hasMedtronicLoopDataAfter, err := store.HasMedtronicLoopDataAfter("1234567890", "2017-01-01T00:00:00Z")

	if err != nil {
		t.Error("failure querying HasMedtronicLoopDataAfter", err)
	}
	if hasMedtronicLoopDataAfter {
		t.Error("should not have Medtronic Loop Data After, but got some")
	}
}

func TestStore_HasMedtronicLoopDataAfter_NotFound_Time(t *testing.T) {
	date1, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date2, _ := time.Parse(time.RFC3339, "2016-12-31T23:59:59Z")
	date3, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date4, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	// We keep _schemaVersion in the test data until BACK-1281 is completed.
	store := before(t, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("0000000000"),
		SchemaVersion: ptr(1),
		Time:          ptr(date1),
		Origin:        ptr(bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}}),
	}, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date2),
		Origin:        ptr(bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}}),
	}, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date3),
		Origin:        ptr(bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Animas"}}}),
	}, TestDataSchema{
		Active:        ptr(false),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date4),
		Origin:        ptr(bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}}),
	})

	// we need indexes here as the following queries rely on working index hints for performance
	err := store.EnsureIndexes()
	if err != nil {
		t.Error("Failed to run EnsureIndexes()")
	}

	hasMedtronicLoopDataAfter, err := store.HasMedtronicLoopDataAfter("1234567890", "2017-01-01T00:00:00Z")

	if err != nil {
		t.Error("failure querying HasMedtronicLoopDataAfter", err)
	}
	if hasMedtronicLoopDataAfter {
		t.Error("should not have Medtronic Loop Data After, but got some")
	}
}

func TestStore_HasMedtronicLoopDataAfter_Found(t *testing.T) {
	date1, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date2, _ := time.Parse(time.RFC3339, "2016-12-31T23:59:59Z")
	date3, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date4, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date5, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	// We keep _schemaVersion in the test data until BACK-1281 is completed.
	store := before(t, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("0000000000"),
		SchemaVersion: ptr(1),
		Time:          ptr(date1),
		Origin:        ptr(bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}}),
	}, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date2),
		Origin:        ptr(bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}}),
	}, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date3),
		Origin:        ptr(bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}}),
	}, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date4),
		Origin:        ptr(bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Animas"}}}),
	}, TestDataSchema{
		Active:        ptr(false),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date5),
		Origin:        ptr(bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}}),
	})

	// we need indexes here as the following queries rely on working index hints for performance
	err := store.EnsureIndexes()
	if err != nil {
		t.Error("Failed to run EnsureIndexes()")
	}

	hasMedtronicLoopDataAfter, err := store.HasMedtronicLoopDataAfter("1234567890", "2017-01-01T00:00:00Z")

	if err != nil {
		t.Error("failure querying HasMedtronicLoopDataAfter", err)
	}
	if !hasMedtronicLoopDataAfter {
		t.Error("should have Medtronic Loop Data After, but got none")
	}
}

func TestStore_GetLoopableMedtronicDirectUploadIdsAfter_NotFound_UserID(t *testing.T) {
	date1, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date2, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date3, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date4, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	// We keep _schemaVersion in the test data until BACK-1281 is completed.
	store := before(t, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("0000000000"),
		SchemaVersion: ptr(1),
		Time:          ptr(date1),
		Type:          ptr("upload"),
		DeviceModel:   ptr("523"),
	}, TestDataSchema{
		Active:        ptr(false),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date2),
		Type:          ptr("upload"),
		DeviceModel:   ptr("523"),
	}, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date3),
		Type:          ptr("cgm"),
		DeviceModel:   ptr("523"),
	}, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date4),
		Type:          ptr("upload"),
		DeviceModel:   ptr("Another Model"),
	})

	// we need indexes here as the following queries rely on working index hints for performance
	err := store.EnsureIndexes()
	if err != nil {
		t.Error("Failed to run EnsureIndexes()")
	}

	loopableMedtronicDirectUploadIdsAfter, err := store.GetLoopableMedtronicDirectUploadIdsAfter("1234567890", "2017-01-01T00:00:00Z")

	if err != nil {
		t.Error("failure querying GetLoopableMedtronicDirectUploadIdsAfter", err)
	}
	if !reflect.DeepEqual(loopableMedtronicDirectUploadIdsAfter, []string{}) {
		t.Error("should not have Loopable Medtronic Direct Upload Ids After, but got some")
	}
}

func TestStore_GetLoopableMedtronicDirectUploadIdsAfter_NotFound_Time(t *testing.T) {
	date1, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date2, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date3, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date4, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date5, _ := time.Parse(time.RFC3339, "2016-12-31T23:59:59Z")
	// We keep _schemaVersion in the test data until BACK-1281 is completed.
	store := before(t, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("0000000000"),
		SchemaVersion: ptr(1),
		Time:          ptr(date1),
		Type:          ptr("upload"),
		DeviceModel:   ptr("723"),
	}, TestDataSchema{
		Active:        ptr(false),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date2),
		Type:          ptr("upload"),
		DeviceModel:   ptr("523"),
	}, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date3),
		Type:          ptr("cgm"),
		DeviceModel:   ptr("523"),
	}, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date4),
		Type:          ptr("upload"),
		DeviceModel:   ptr("Another Model"),
	}, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date5),
		Type:          ptr("upload"),
		DeviceModel:   ptr("523"),
	})

	// we need indexes here as the following queries rely on working index hints for performance
	err := store.EnsureIndexes()
	if err != nil {
		t.Error("Failed to run EnsureIndexes()")
	}

	loopableMedtronicDirectUploadIdsAfter, err := store.GetLoopableMedtronicDirectUploadIdsAfter("1234567890", "2017-01-01T00:00:00Z")

	if err != nil {
		t.Error("failure querying GetLoopableMedtronicDirectUploadIdsAfter", err)
	}
	if !reflect.DeepEqual(loopableMedtronicDirectUploadIdsAfter, []string{}) {
		t.Error("should not have Loopable Medtronic Direct Upload Ids After, but got some")
	}
}

func TestStore_GetLoopableMedtronicDirectUploadIdsAfter_Found(t *testing.T) {
	date1, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date2, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date3, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date4, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date5, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date6, _ := time.Parse(time.RFC3339, "2018-02-03T04:05:06Z")
	date7, _ := time.Parse(time.RFC3339, "2016-12-31T23:59:59Z")
	// We keep _schemaVersion in the test data until BACK-1281 is completed.
	store := before(t, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("0000000000"),
		SchemaVersion: ptr(1),
		Time:          ptr(date1),
		Type:          ptr("upload"),
		DeviceModel:   ptr("723"),
	}, TestDataSchema{
		Active:        ptr(false),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date2),
		Type:          ptr("upload"),
		DeviceModel:   ptr("523"),
	}, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date3),
		Type:          ptr("upload"),
		DeviceModel:   ptr("554"),
		UploadId:      ptr("11223344"),
	}, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date4),
		Type:          ptr("cgm"),
		DeviceModel:   ptr("523"),
	}, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date5),
		Type:          ptr("upload"),
		DeviceModel:   ptr("523K"),
		UploadId:      ptr("55667788"),
	}, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date6),
		Type:          ptr("upload"),
		DeviceModel:   ptr("Another Model"),
	}, TestDataSchema{
		Active:        ptr(true),
		UserId:        ptr("1234567890"),
		SchemaVersion: ptr(1),
		Time:          ptr(date7),
		Type:          ptr("upload"),
		DeviceModel:   ptr("523"),
	})

	// we need indexes here as the following queries rely on working index hints for performance
	err := store.EnsureIndexes()
	if err != nil {
		t.Error("Failed to run EnsureIndexes()")
	}

	loopableMedtronicDirectUploadIdsAfter, err := store.GetLoopableMedtronicDirectUploadIdsAfter("1234567890", "2017-01-01T00:00:00Z")

	if err != nil {
		t.Error("failure querying GetLoopableMedtronicDirectUploadIdsAfter", err)
	}
	if !reflect.DeepEqual(loopableMedtronicDirectUploadIdsAfter, []string{"55667788", "11223344"}) {
		t.Error("should not have Loopable Medtronic Direct Upload Ids After, but got some")
	}
}

func TestStore_LatestNoFilter(t *testing.T) {
	testData := testDataForLatestTests()
	storeData := storeDataForLatestTests()

	store := before(t, storeData...)

	qParams := &Params{
		UserID:        "abc123",
		SchemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		Latest:        true,
	}

	iter, err := store.GetDeviceData(qParams)
	if err != nil {
		t.Error("Error querying Mongo")
	}

	processedResults := struct {
		cbg    bool
		upload bool
	}{}

	for iter.Next(store.context) {
		var result TestDataSchema
		err := iter.Decode(&result)
		if err != nil {
			t.Error("Mongo Decode error")
		}
		switch dataType := *result.Type; dataType {
		case "cbg":
			compareResult := dropInternalKeys(testData["cbg1"])
			diff := cmp.Diff(compareResult, result)
			if diff != "" {
				t.Errorf("Unexpected 'cbg' result when requesting latest data (-want +have):\n%s", diff)
			}
			processedResults.cbg = true
		case "upload":
			compareResult := dropInternalKeys(testData["upload1"])
			diff := cmp.Diff(compareResult, result)
			if diff != "" {
				t.Errorf("Unexpected 'upload' result when requesting latest data (-want +have):\n%s", diff)
			}
			processedResults.upload = true
		}
	}

	if processedResults.cbg == false || processedResults.upload == false {
		t.Error("Not enough results when requesting latest data")
	}
}

func TestStore_LatestTypeFilter(t *testing.T) {
	testData := testDataForLatestTests()
	storeData := storeDataForLatestTests()

	store := before(t, storeData...)

	qParams := &Params{
		UserID:        "abc123",
		SchemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		Types:         []string{"cbg"},
		Latest:        true,
	}

	iter, err := store.GetDeviceData(qParams)
	if err != nil {
		t.Error("Error querying Mongo")
	}

	processedResults := struct {
		cbg bool
	}{}

	for iter.Next(store.context) {
		var result TestDataSchema
		err := iter.Decode(&result)
		if err != nil {
			t.Error("Mongo Decode error")
		}
		switch dataType := *result.Type; dataType {
		case "cbg":
			compareResult := dropInternalKeys(testData["cbg1"])
			diff := cmp.Diff(compareResult, result)
			if diff != "" {
				t.Errorf("Unexpected 'cbg' result when requesting latest data (-want +have):\n%s", diff)
			}
			processedResults.cbg = true
		}
	}

	if processedResults.cbg == false {
		t.Error("Not enough results when requesting latest data")
	}
}

func TestStore_LatestUploadIdFilter(t *testing.T) {
	testData := testDataForLatestTests()
	storeData := storeDataForLatestTests()

	store := before(t, storeData...)

	qParams := &Params{
		UserID:        "abc123",
		SchemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		UploadID:      "zzz4bb16e27c4973c2f37af81784a05d",
		Latest:        true,
	}

	iter, err := store.GetDeviceData(qParams)
	if err != nil {
		t.Error("Error querying Mongo")
	}

	processedResults := struct {
		cbg    bool
		upload bool
	}{}

	for iter.Next(store.context) {
		var result TestDataSchema
		err := iter.Decode(&result)
		if err != nil {
			t.Error("Mongo Decode error")
		}
		switch dataType := *result.Type; dataType {
		case "cbg":
			compareResult := dropInternalKeys(testData["cbg2"])
			diff := cmp.Diff(compareResult, result)
			if diff != "" {
				t.Errorf("Unexpected 'cbg' result when requesting latest data (-want +have):\n%s", diff)
			}
			processedResults.cbg = true
		case "upload":
			compareResult := dropInternalKeys(testData["upload2"])
			diff := cmp.Diff(compareResult, result)
			if diff != "" {
				t.Errorf("Unexpected 'upload' result when requesting latest data (-want +have):\n%s", diff)
			}
			processedResults.upload = true
		}
	}

	if processedResults.cbg == false || processedResults.upload == false {
		t.Error("Not enough results when requesting latest data")
	}
}

func TestStore_LatestDeviceIdFilter(t *testing.T) {
	testData := testDataForLatestTests()
	storeData := storeDataForLatestTests()

	store := before(t, storeData...)

	qParams := &Params{
		UserID:        "xyz123",
		DeviceID:      "dev789",
		SchemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		Latest:        true,
	}

	iter, err := store.GetDeviceData(qParams)
	if err != nil {
		t.Error("Error querying Mongo")
	}

	processedResults := struct {
		cbg    bool
		upload bool
	}{}

	for iter.Next(store.context) {
		var result TestDataSchema
		err := iter.Decode(&result)
		if err != nil {
			t.Error("Mongo Decode error")
		}
		switch dataType := *result.Type; dataType {
		case "cbg":
			compareResult := dropInternalKeys(testData["cbg3"])
			diff := cmp.Diff(compareResult, result)
			if diff != "" {
				t.Errorf("Unexpected 'cbg' result when requesting latest data (-want +have):\n%s", diff)
			}
			processedResults.cbg = true
		case "upload":
			compareResult := dropInternalKeys(testData["upload3"])
			diff := cmp.Diff(compareResult, result)
			if diff != "" {
				t.Errorf("Unexpected 'upload' result when requesting latest data (-want +have):\n%s", diff)
			}
			processedResults.upload = true
		}
	}

	if processedResults.cbg == false || processedResults.upload == false {
		t.Error("Not enough results when requesting latest data")
	}
}
