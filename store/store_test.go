package store

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	"github.com/tidepool-org/go-common/clients/mongo"
)

var testingConfig = &mongo.Config{ConnectionString: "mongodb://127.0.0.1/data_test"}

func before(t *testing.T, docs ...interface{}) *MongoStoreClient {

	store := NewMongoStoreClient(testingConfig)

	//INIT THE TEST - we use a clean copy of the collection before we start
	cpy := store.session.Copy()
	defer cpy.Close()

	//just drop and don't worry about any errors
	mgoDataCollection(cpy).DropCollection()

	if err := mgoDataCollection(cpy).Create(&mgo.CollectionInfo{}); err != nil {
		t.Error("We couldn't created the deviceData collection for these tests ", err)
	}

	if len(docs) > 0 {
		if err := mgoDataCollection(cpy).Insert(docs...); err != nil {
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
		UserId:        "abc123",
		SchemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		Dexcom:        true,
		Medtronic:     true,
	}

	return generateMongoQuery(qParams)
}

func allParams() *Params {
	earliestDataTime, _ := time.Parse(time.RFC3339, "2015-10-07T15:00:00Z")
	latestDataTime, _ := time.Parse(time.RFC3339, "2016-12-13T02:00:00Z")

	return &Params{
		UserId:        "abc123",
		DeviceId:      "device123",
		SchemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		Date:          Date{"2015-10-07T15:00:00.000Z", "2015-10-11T15:00:00.000Z"},
		Types:         []string{"smbg", "cbg"},
		SubTypes:      []string{"stuff"},
		Carelink:      true,
		Dexcom:        false,
		DexcomDataSource: bson.M{
			"dataSetIds":       []string{"123", "456"},
			"earliestDataTime": earliestDataTime,
			"latestDataTime":   latestDataTime,
		},
		Latest:             false,
		Medtronic:          false,
		MedtronicDate:      "2017-01-01T00:00:00Z",
		MedtronicUploadIds: []string{"555666777", "888999000"},
	}
}

func allParamsQuery() bson.M {
	return generateMongoQuery(allParams())
}

func allParamsIncludingUploadIdQuery() bson.M {
	qParams := allParams()
	qParams.UploadId = "xyz123"

	return generateMongoQuery(qParams)
}

func typeAndSubtypeQuery() bson.M {
	qParams := &Params{
		UserId:             "abc123",
		SchemaVersion:      &SchemaVersion{Maximum: 2, Minimum: 0},
		Types:              []string{"smbg", "cbg"},
		SubTypes:           []string{"stuff"},
		Dexcom:             true,
		Medtronic:          false,
		MedtronicDate:      "2017-01-01T00:00:00Z",
		MedtronicUploadIds: []string{"555666777", "888999000"},
	}
	return generateMongoQuery(qParams)
}

func uploadIdQuery() bson.M {
	qParams := &Params{
		UserId:        "abc123",
		SchemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		UploadId:      "xyz123",
	}
	return generateMongoQuery(qParams)
}

func testDataForLatestTests() map[string]bson.M {
	testData := map[string]bson.M{
		"upload1": bson.M{
			"_active":        true,
			"_userId":        "abc123",
			"_schemaVersion": 1,
			"time":           "2019-03-15T01:24:28.000Z",
			"type":           "upload",
			"deviceId":       "dev123",
			"uploadId":       "9244bb16e27c4973c2f37af81784a05d",
		},
		"cbg1": bson.M{
			"_active":        true,
			"_userId":        "abc123",
			"_schemaVersion": 1,
			"time":           "2019-03-15T00:42:51.902Z",
			"type":           "cbg",
			"units":          "mmol/L",
			"deviceId":       "dev123",
			"uploadId":       "9244bb16e27c4973c2f37af81784a05d",
			"value":          12.82223,
		},
		"upload2": bson.M{
			"_active":        true,
			"_userId":        "abc123",
			"_schemaVersion": 1,
			"time":           "2019-03-14T01:24:28.000Z",
			"type":           "upload",
			"deviceId":       "dev456",
			"uploadId":       "zzz4bb16e27c4973c2f37af81784a05d",
		},
		"cbg2": bson.M{
			"_active":        true,
			"_userId":        "abc123",
			"_schemaVersion": 1,
			"time":           "2019-03-14T00:42:51.902Z",
			"type":           "cbg",
			"units":          "mmol/L",
			"uploadId":       "zzz4bb16e27c4973c2f37af81784a05d",
			"deviceId":       "dev456",
			"value":          9.7213,
		},
		"upload3": bson.M{
			"_active":        true,
			"_userId":        "xyz123",
			"_schemaVersion": 1,
			"time":           "2019-03-19T01:24:28.000Z",
			"type":           "upload",
			"deviceId":       "dev789",
			"uploadId":       "xxx4bb16e27c4973c2f37af81784a05d",
		},
		"cbg3": bson.M{
			"_active":        true,
			"_userId":        "xyz123",
			"_schemaVersion": 1,
			"time":           "2019-03-19T00:42:51.902Z",
			"type":           "cbg",
			"units":          "mmol/L",
			"uploadId":       "xxx4bb16e27c4973c2f37af81784a05d",
			"deviceId":       "dev789",
			"value":          7.1237,
		},
	}

	return testData
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

func TestStore_generateMongoQuery_basic(t *testing.T) {

	time.Now()
	query := basicQuery()

	expectedQuery := bson.M{
		"_userId":        "abc123",
		"_active":        true,
		"_schemaVersion": bson.M{"$gte": 0, "$lte": 2},
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

	expectedQuery := bson.M{
		"_userId":        "abc123",
		"deviceId":       "device123",
		"_active":        true,
		"_schemaVersion": bson.M{"$gte": 0, "$lte": 2},
		"type":           bson.M{"$in": strings.Split("smbg,cbg", ",")},
		"subType":        bson.M{"$in": strings.Split("stuff", ",")},
		"time": bson.M{
			"$gte": "2015-10-07T15:00:00.000Z",
			"$lte": "2015-10-11T15:00:00.000Z"},
		"$and": []bson.M{
			{"$or": []bson.M{
				{"type": bson.M{"$ne": "cbg"}},
				{"uploadId": bson.M{"$in": []string{"123", "456"}}},
				{"time": bson.M{"$lt": "2015-10-07T15:00:00Z"}},
				{"time": bson.M{"$gt": "2016-12-13T02:00:00Z"}},
			}},
			{"$or": []bson.M{
				{"time": bson.M{"$lt": "2017-01-01T00:00:00Z"}},
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

	query := allParamsIncludingUploadIdQuery()

	expectedQuery := bson.M{
		"_userId":        "abc123",
		"deviceId":       "device123",
		"_active":        true,
		"_schemaVersion": bson.M{"$gte": 0, "$lte": 2},
		"type":           bson.M{"$in": strings.Split("smbg,cbg", ",")},
		"subType":        bson.M{"$in": strings.Split("stuff", ",")},
		"uploadId":       "xyz123",
		"time": bson.M{
			"$gte": "2015-10-07T15:00:00.000Z",
			"$lte": "2015-10-11T15:00:00.000Z"},
	}

	eq := reflect.DeepEqual(query, expectedQuery)
	if !eq {
		t.Error(getErrString(query, expectedQuery))
	}
}

func TestStore_generateMongoQuery_uploadId(t *testing.T) {

	query := uploadIdQuery()

	expectedQuery := bson.M{
		"_userId":        "abc123",
		"_active":        true,
		"_schemaVersion": bson.M{"$gte": 0, "$lte": 2},
		"uploadId":       "xyz123",
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

	expectedQuery := bson.M{
		"_userId":        "abc123",
		"_active":        true,
		"type":           bson.M{"$in": strings.Split("smbg,cbg", ",")},
		"subType":        bson.M{"$in": strings.Split("stuff", ",")},
		"_schemaVersion": bson.M{"$gte": 0, "$lte": 2},
		"source": bson.M{
			"$ne": "carelink",
		},
		"$and": []bson.M{
			{"$or": []bson.M{
				{"time": bson.M{"$lt": "2017-01-01T00:00:00Z"}},
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

	dateStr, err := cleanDateString("")

	if dateStr != "" {
		t.Error("the returned dateStr should have been empty but got ", dateStr)
	}
	if err != nil {
		t.Error("didn't expect an error but got ", err.Error())
	}

}

func TestStore_cleanDateString_nonsensical(t *testing.T) {

	dateStr, err := cleanDateString("blah")

	if dateStr != "" {
		t.Error("the returned dateStr should have been empty but got ", dateStr)
	}
	if err == nil {
		t.Error("we should have been given an error")
	}

}

func TestStore_cleanDateString_wrongFormat(t *testing.T) {

	dateStr, err := cleanDateString("2006-20-02T3:04pm")

	if dateStr != "" {
		t.Error("the returned dateStr should have been empty but got ", dateStr)
	}
	if err == nil {
		t.Error("we should have been given an error")
	}

}

func TestStore_cleanDateString(t *testing.T) {

	dateStr, err := cleanDateString("2015-10-10T15:00:00.000Z")

	if dateStr == "" {
		t.Error("the returned dateStr should not be empty")
	}
	if err != nil {
		t.Error("we should have no error but go ", err.Error())
	}

}

func TestStore_GetParams_Empty(t *testing.T) {
	query := url.Values{
		":userID": []string{"1122334455"},
	}
	schema := &SchemaVersion{Minimum: 1, Maximum: 3}

	expectedParams := &Params{
		UserId:        "1122334455",
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
		UserId:        "1122334455",
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
		UserId:        "1122334455",
		SchemaVersion: schema,
		Types:         []string{""},
		SubTypes:      []string{""},
		UploadId:      "xyz123",
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
	store := before(t, bson.M{
		"_userId":             "0000000000",
		"type":                "upload",
		"_state":              "closed",
		"_active":             true,
		"deviceManufacturers": "Medtronic",
		"index":               "0",
	}, bson.M{
		"_userId":             "1234567890",
		"type":                "upload",
		"_state":              "closed",
		"_active":             true,
		"deviceManufacturers": "Medtronic",
		"index":               "1",
	}, bson.M{
		"_userId":             "1234567890",
		"type":                "upload",
		"_state":              "open",
		"_active":             true,
		"deviceManufacturers": "Medtronic",
		"index":               "2",
	}, bson.M{
		"_userId":             "1234567890",
		"type":                "upload",
		"_state":              "closed",
		"_active":             true,
		"deviceManufacturers": "Medtronic",
		"index":               "3",
	}, bson.M{
		"_userId":             "1234567890",
		"type":                "upload",
		"_state":              "closed",
		"_active":             true,
		"deletedTime":         "2017-05-17T20:13:32.064-0700",
		"deviceManufacturers": "Medtronic",
		"index":               "4",
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
	store := before(t, bson.M{
		"_userId":             "1234567890",
		"type":                "upload",
		"_state":              "closed",
		"_active":             true,
		"deletedTime":         "2017-05-17T20:10:26.607-0700",
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
	store := before(t, bson.M{
		"_userId":             "0000000000",
		"type":                "upload",
		"_state":              "closed",
		"_active":             true,
		"deviceManufacturers": "Medtronic",
		"index":               "0",
	}, bson.M{
		"_userId":             "1234567890",
		"type":                "cgm",
		"_state":              "closed",
		"_active":             true,
		"deviceManufacturers": "Medtronic",
		"index":               "1",
	}, bson.M{
		"_userId":             "1234567890",
		"type":                "upload",
		"_state":              "open",
		"_active":             true,
		"deviceManufacturers": "Medtronic",
		"index":               "2",
	}, bson.M{
		"_userId":             "1234567890",
		"type":                "upload",
		"_state":              "closed",
		"_active":             false,
		"deviceManufacturers": "Medtronic",
		"index":               "3",
	}, bson.M{
		"_userId":             "1234567890",
		"type":                "upload",
		"_state":              "closed",
		"_active":             true,
		"deletedTime":         "2017-05-17T20:13:32.064-0700",
		"deviceManufacturers": "Medtronic",
		"index":               "4",
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
	store := before(t, bson.M{
		"_active":        true,
		"_userId":        "0000000000",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"origin":         bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}},
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"origin":         bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Animas"}}},
	}, bson.M{
		"_active":        false,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"origin":         bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}},
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 0,
		"time":           "2018-02-03T04:05:06Z",
		"origin":         bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}},
	})

	hasMedtronicLoopDataAfter, err := store.HasMedtronicLoopDataAfter("1234567890", "2017-01-01T00:00:00Z")

	if err != nil {
		t.Error("failure querying HasMedtronicLoopDataAfter", err)
	}
	if hasMedtronicLoopDataAfter {
		t.Error("should not have Medtronic Loop Data After, but got some")
	}
}

func TestStore_HasMedtronicLoopDataAfter_NotFound_Time(t *testing.T) {
	store := before(t, bson.M{
		"_active":        true,
		"_userId":        "0000000000",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"origin":         bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}},
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2016-12-31T23:59:59Z",
		"origin":         bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}},
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"origin":         bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Animas"}}},
	}, bson.M{
		"_active":        false,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"origin":         bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}},
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 0,
		"time":           "2018-02-03T04:05:06Z",
		"origin":         bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}},
	})

	hasMedtronicLoopDataAfter, err := store.HasMedtronicLoopDataAfter("1234567890", "2017-01-01T00:00:00Z")

	if err != nil {
		t.Error("failure querying HasMedtronicLoopDataAfter", err)
	}
	if hasMedtronicLoopDataAfter {
		t.Error("should not have Medtronic Loop Data After, but got some")
	}
}

func TestStore_HasMedtronicLoopDataAfter_Found(t *testing.T) {
	store := before(t, bson.M{
		"_active":        true,
		"_userId":        "0000000000",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"origin":         bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}},
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2016-12-31T23:59:59Z",
		"origin":         bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}},
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"origin":         bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}},
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"origin":         bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Animas"}}},
	}, bson.M{
		"_active":        false,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"origin":         bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}},
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 0,
		"time":           "2018-02-03T04:05:06Z",
		"origin":         bson.M{"payload": bson.M{"device": bson.M{"manufacturer": "Medtronic"}}},
	})

	hasMedtronicLoopDataAfter, err := store.HasMedtronicLoopDataAfter("1234567890", "2017-01-01T00:00:00Z")

	if err != nil {
		t.Error("failure querying HasMedtronicLoopDataAfter", err)
	}
	if !hasMedtronicLoopDataAfter {
		t.Error("should have Medtronic Loop Data After, but got none")
	}
}

func TestStore_GetLoopableMedtronicDirectUploadIdsAfter_NotFound_UserID(t *testing.T) {
	store := before(t, bson.M{
		"_active":        true,
		"_userId":        "0000000000",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"type":           "upload",
		"deviceModel":    "523",
	}, bson.M{
		"_active":        false,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"type":           "upload",
		"deviceModel":    "523",
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 0,
		"time":           "2018-02-03T04:05:06Z",
		"type":           "upload",
		"deviceModel":    "523",
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"type":           "cgm",
		"deviceModel":    "523",
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"type":           "upload",
		"deviceModel":    "Another Model",
	})

	loopableMedtronicDirectUploadIdsAfter, err := store.GetLoopableMedtronicDirectUploadIdsAfter("1234567890", "2017-01-01T00:00:00Z")

	if err != nil {
		t.Error("failure querying GetLoopableMedtronicDirectUploadIdsAfter", err)
	}
	if !reflect.DeepEqual(loopableMedtronicDirectUploadIdsAfter, []string{}) {
		t.Error("should not have Loopable Medtronic Direct Upload Ids After, but got some")
	}
}

func TestStore_GetLoopableMedtronicDirectUploadIdsAfter_NotFound_Time(t *testing.T) {
	store := before(t, bson.M{
		"_active":        true,
		"_userId":        "0000000000",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"type":           "upload",
		"deviceModel":    "723",
	}, bson.M{
		"_active":        false,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"type":           "upload",
		"deviceModel":    "523",
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 0,
		"time":           "2018-02-03T04:05:06Z",
		"type":           "upload",
		"deviceModel":    "554",
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"type":           "cgm",
		"deviceModel":    "523",
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"type":           "upload",
		"deviceModel":    "Another Model",
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2016-12-31T23:59:59Z",
		"type":           "upload",
		"deviceModel":    "523",
	})

	loopableMedtronicDirectUploadIdsAfter, err := store.GetLoopableMedtronicDirectUploadIdsAfter("1234567890", "2017-01-01T00:00:00Z")

	if err != nil {
		t.Error("failure querying GetLoopableMedtronicDirectUploadIdsAfter", err)
	}
	if !reflect.DeepEqual(loopableMedtronicDirectUploadIdsAfter, []string{}) {
		t.Error("should not have Loopable Medtronic Direct Upload Ids After, but got some")
	}
}

func TestStore_GetLoopableMedtronicDirectUploadIdsAfter_Found(t *testing.T) {
	store := before(t, bson.M{
		"_active":        true,
		"_userId":        "0000000000",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"type":           "upload",
		"deviceModel":    "723",
	}, bson.M{
		"_active":        false,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"type":           "upload",
		"deviceModel":    "523",
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 0,
		"time":           "2018-02-03T04:05:06Z",
		"type":           "upload",
		"deviceModel":    "554",
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"type":           "upload",
		"deviceModel":    "554",
		"uploadId":       "11223344",
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"type":           "cgm",
		"deviceModel":    "523",
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"type":           "upload",
		"deviceModel":    "523K",
		"uploadId":       "55667788",
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2018-02-03T04:05:06Z",
		"type":           "upload",
		"deviceModel":    "Another Model",
	}, bson.M{
		"_active":        true,
		"_userId":        "1234567890",
		"_schemaVersion": 1,
		"time":           "2016-12-31T23:59:59Z",
		"type":           "upload",
		"deviceModel":    "523",
	})

	loopableMedtronicDirectUploadIdsAfter, err := store.GetLoopableMedtronicDirectUploadIdsAfter("1234567890", "2017-01-01T00:00:00Z")

	if err != nil {
		t.Error("failure querying GetLoopableMedtronicDirectUploadIdsAfter", err)
	}
	if !reflect.DeepEqual(loopableMedtronicDirectUploadIdsAfter, []string{"11223344", "55667788"}) {
		t.Error("should not have Loopable Medtronic Direct Upload Ids After, but got some")
	}
}

func TestStore_LatestNoFilter(t *testing.T) {
	testData := testDataForLatestTests()
	storeData := storeDataForLatestTests()

	store := before(t, storeData...)

	qParams := &Params{
		UserId:        "abc123",
		SchemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		Latest:        true,
	}

	var result bson.M
	iter := store.GetDeviceData(qParams)
	resultCount := 0
	processedResultCount := 0
	for iter.Next(&result) {
		switch dataType := result["type"]; dataType {
		case "cbg":
			delete(result, "_id") // _id is assigned by MongoDB. We don't know it up front
			if !reflect.DeepEqual(result, testData["cbg1"]) {
				t.Error("Unexpected 'cbg' result when requesting latest data")
			}
			processedResultCount++
		case "upload":
			delete(result, "_id") // _id is assigned by MongoDB. We don't know it up front
			if !reflect.DeepEqual(result, testData["upload1"]) {
				t.Error("Unexpected 'upload' result when requesting latest data")
			}
			processedResultCount++
		}
		resultCount++
	}

	if resultCount < 2 || processedResultCount < 2 {
		t.Error("Not enough results when requesting latest data")
	}
}

func TestStore_LatestTypeFilter(t *testing.T) {
	testData := testDataForLatestTests()
	storeData := storeDataForLatestTests()

	store := before(t, storeData...)

	qParams := &Params{
		UserId:        "abc123",
		SchemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		Types:         []string{"cbg"},
		Latest:        true,
	}

	var result bson.M
	iter := store.GetDeviceData(qParams)
	resultCount := 0
	processedResultCount := 0
	for iter.Next(&result) {
		switch dataType := result["type"]; dataType {
		case "cbg":
			delete(result, "_id") // _id is assigned by MongoDB. We don't know it up front
			if !reflect.DeepEqual(result, testData["cbg1"]) {
				t.Error("Unexpected 'cbg' result when requesting latest data")
			}
			processedResultCount++
		}
		resultCount++
	}

	if resultCount < 1 || processedResultCount < 1 {
		t.Error("Not enough results when requesting latest data")
	}
}

func TestStore_LatestUploadIdFilter(t *testing.T) {
	testData := testDataForLatestTests()
	storeData := storeDataForLatestTests()

	store := before(t, storeData...)

	qParams := &Params{
		UserId:        "abc123",
		SchemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		UploadId:      "zzz4bb16e27c4973c2f37af81784a05d",
		Latest:        true,
	}

	var result bson.M
	iter := store.GetDeviceData(qParams)
	resultCount := 0
	processedResultCount := 0
	for iter.Next(&result) {
		switch dataType := result["type"]; dataType {
		case "cbg":
			delete(result, "_id") // _id is assigned by MongoDB. We don't know it up front
			if !reflect.DeepEqual(result, testData["cbg2"]) {
				t.Error("Unexpected 'cbg' result when requesting latest data")
			}
			processedResultCount++
		case "upload":
			delete(result, "_id") // _id is assigned by MongoDB. We don't know it up front
			if !reflect.DeepEqual(result, testData["upload2"]) {
				t.Error("Unexpected 'upload' result when requesting latest data")
			}
			processedResultCount++
		}
		resultCount++
	}

	if resultCount < 2 || processedResultCount < 2 {
		t.Error("Not enough results when requesting latest data")
	}
}

func TestStore_LatestDeviceIdFilter(t *testing.T) {
	testData := testDataForLatestTests()
	storeData := storeDataForLatestTests()

	store := before(t, storeData...)

	qParams := &Params{
		UserId:        "xyz123",
		DeviceId:      "dev789",
		SchemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		Latest:        true,
	}

	var result bson.M
	iter := store.GetDeviceData(qParams)
	resultCount := 0
	processedResultCount := 0
	for iter.Next(&result) {
		switch dataType := result["type"]; dataType {
		case "cbg":
			delete(result, "_id") // _id is assigned by MongoDB. We don't know it up front
			if !reflect.DeepEqual(result, testData["cbg3"]) {
				t.Error("Unexpected 'cbg' result when requesting latest data")
			}
			processedResultCount++
		case "upload":
			delete(result, "_id") // _id is assigned by MongoDB. We don't know it up front
			if !reflect.DeepEqual(result, testData["upload3"]) {
				t.Error("Unexpected 'upload' result when requesting latest data")
			}
			processedResultCount++
		}
		resultCount++
	}

	if resultCount < 2 || processedResultCount < 2 {
		t.Error("Not enough results when requesting latest data")
	}
}
