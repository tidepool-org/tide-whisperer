package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"github.com/tidepool-org/go-common/clients/mongo"
)

var testingConfig = &mongo.Config{ConnectionString: "mongodb://localhost/data_test"}

const (
	typeCursor    = "BtreeCursor type_1"
	subTypeCursor = "BtreeCursor subType_1"
	baseCursor    = "BtreeCursor _groupId_1__active_1__schemaVersion_1"
)

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

// FIXME: Mismatch between new MongoDB version 3.0.7 and old mgo package causes
// the Indexes() command to silently fail. Not worth the effort at this point to
// bump the old mgo package
// func TestStore_IndexesExist(t *testing.T) {
//
// 	const (
// 		//index names based on feilds used
// 		subtype_idx = "subType_1"
// 		type_idx    = "type_1"
// 		basic_idx   = "_groupId_1__active_1__schemaVersion_1"
// 		id_idx      = "_id_"
// 	)
//
// 	store := before(t)
//
// 	sCopy := store.session
// 	defer sCopy.Close()
//
// 	if idxs, err := mgoDataCollection(sCopy).Indexes(); err != nil {
// 		t.Error("TestStore_IndexesExist unexpected error ", err.Error())
// 	} else {
// 		// there are the two we have added and also the standard index
// 		if len(idxs) != 4 {
// 			t.Fatalf("TestStore_IndexesExist should be THREE but found [%d] ", len(idxs))
// 		}
//
// 		if idxs[3].Name != type_idx {
// 			t.Errorf("TestStore_IndexesExist expected [%s] got [%s] ", type_idx, idxs[3].Name)
// 		}
//
// 		if idxs[2].Name != subtype_idx {
// 			t.Errorf("TestStore_IndexesExist expected [%s] got [%s] ", subtype_idx, idxs[2].Name)
// 		}
//
// 		if idxs[1].Name != id_idx {
// 			t.Errorf("TestStore_IndexesExist expected [%s] got [%s] ", id_idx, idxs[1].Name)
// 		}
//
// 		if idxs[0].Name != basic_idx {
// 			t.Errorf("TestStore_IndexesExist expected [%s] got [%s] ", basic_idx, idxs[0].Name)
// 		}
//
// 	}
//
// }

func basicQuery() bson.M {
	qParams := &params{
		groupId:       "123",
		userId:        "321",
		schemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
	}

	return generateMongoQuery(qParams)
}

func allParamsQuery() bson.M {
	qParams := &params{
		groupId:       "123ggf",
		userId:        "abc123",
		schemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		date:          date{"2015-10-07T15:00:00.000Z", "2015-10-11T15:00:00.000Z"},
		types:         []string{"smbg", "cbg"},
		subTypes:      []string{"stuff"},
		carelink:      true,
	}

	return generateMongoQuery(qParams)
}

func dateAndTypeQuery() bson.M {
	qParams := &params{
		groupId:       "123ggf",
		userId:        "abc123",
		schemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		date:          date{"2015-10-07T15:00:00.000Z", "2015-10-11T15:00:00.000Z"},
		types:         []string{"smbg", "cbg", "bolus", "basal"},
	}
	return generateMongoQuery(qParams)
}

func typeAndSubtypeQuery() bson.M {
	qParams := &params{
		groupId:       "123ggf",
		userId:        "abc123",
		schemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		types:         []string{"smbg", "cbg"},
		subTypes:      []string{"stuff"},
	}
	return generateMongoQuery(qParams)
}

func dateQuery() bson.M {
	qParams := &params{
		groupId:       "123",
		userId:        "321",
		schemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		date:          date{"2015-10-07T15:00:00.000Z", "2015-10-11T15:00:00.000Z"},
	}

	return generateMongoQuery(qParams)
}

func TestStore_generateMongoQuery_basic(t *testing.T) {

	time.Now()
	query := basicQuery()

	expectedQuery := bson.M{"_groupId": "123",
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

func TestStore_generateMongoQuery_allparams(t *testing.T) {

	query := allParamsQuery()

	expectedQuery := bson.M{
		"_groupId":       "123ggf",
		"_active":        true,
		"_schemaVersion": bson.M{"$gte": 0, "$lte": 2},
		"type":           bson.M{"$in": strings.Split("smbg,cbg", ",")},
		"subType":        bson.M{"$in": strings.Split("stuff", ",")},
		"time": bson.M{
			"$gte": "2015-10-07T15:00:00.000Z",
			"$lte": "2015-10-11T15:00:00.000Z"},
	}

	eq := reflect.DeepEqual(query, expectedQuery)
	if !eq {
		t.Error(getErrString(query, expectedQuery))
	}
}

func TestStore_generateMongoQuery_noDates(t *testing.T) {

	query := typeAndSubtypeQuery()

	expectedQuery := bson.M{
		"_groupId":       "123ggf",
		"_active":        true,
		"type":           bson.M{"$in": strings.Split("smbg,cbg", ",")},
		"subType":        bson.M{"$in": strings.Split("stuff", ",")},
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

func TestStore_Ping(t *testing.T) {

	store := before(t)
	err := store.Ping()

	if err != nil {
		t.Error("there should be no error but got", err.Error())
	}
}

// FIXME: Mismatch between new MongoDB version 3.0.7 and old mgo package causes
// the Indexes() command to silently fail. Not worth the effort at this point to
// bump the old mgo package
// func TestStore_IndexUse_basicQuery(t *testing.T) {
//
// 	const expectedCursor = "BtreeCursor _groupId_1__active_1__schemaVersion_1"
//
// 	store := before(t)
// 	sCopy := store.session
// 	defer sCopy.Close()
//
// 	query := basicQuery()
//
// 	var executedPlan map[string]interface{}
//
// 	err := mgoDataCollection(sCopy).Find(query).Explain(&executedPlan)
//
// 	if err != nil {
// 		t.Error("there should be no error execting the query", err.Error())
// 	}
//
// 	usedCursors := getCursors(executedPlan["allPlans"])
//
// 	if !contains(usedCursors, baseCursor) {
// 		t.Errorf("excpected [%s] to be in  [%v]", baseCursor, usedCursors)
// 	}
//
// 	if contains(usedCursors, typeCursor) {
// 		t.Errorf("didn't expect [%s] to be in  [%v]", typeCursor, usedCursors)
// 	}
//
// }

// FIXME: Mismatch between new MongoDB version 3.0.7 and old mgo package causes
// the Indexes() command to silently fail. Not worth the effort at this point to
// bump the old mgo package
// func TestStore_IndexUse_fullQuery(t *testing.T) {
//
// 	const expectedCursor = "BtreeCursor type_1"
//
// 	store := before(t)
// 	sCopy := store.session
// 	defer sCopy.Close()
//
// 	query := allParamsQuery()
//
// 	var executedPlan map[string]interface{}
//
// 	err := mgoDataCollection(sCopy).Find(query).Explain(&executedPlan)
//
// 	if err != nil {
// 		t.Error("there should be no error execting the query", err.Error())
// 	}
//
// 	usedCursors := getCursors(executedPlan["allPlans"])
//
// 	if !contains(usedCursors, typeCursor) {
// 		t.Errorf("excpected [%s] to be in  [%v]", typeCursor, usedCursors)
// 	}
//
// 	if !contains(usedCursors, subTypeCursor) {
// 		t.Errorf("excpected [%s] to be in  [%v]", subTypeCursor, usedCursors)
// 	}
//
// 	if !contains(usedCursors, baseCursor) {
// 		t.Errorf("excpected [%s] to be in  [%v]", baseCursor, usedCursors)
// 	}
//
// }

// FIXME: Mismatch between new MongoDB version 3.0.7 and old mgo package causes
// the Indexes() command to silently fail. Not worth the effort at this point to
// bump the old mgo package
// func TestStore_IndexUse_typeQuery(t *testing.T) {
//
// 	store := before(t)
// 	sCopy := store.session
// 	defer sCopy.Close()
//
// 	query := dateAndTypeQuery()
//
// 	var executedPlan map[string]interface{}
//
// 	err := mgoDataCollection(sCopy).Find(query).Explain(&executedPlan)
//
// 	if err != nil {
// 		t.Error("there should be no error execting the query", err.Error())
// 	}
//
// 	usedCursors := getCursors(executedPlan["allPlans"])
//
// 	if !contains(usedCursors, typeCursor) {
// 		t.Errorf("excpected [%s] to be in  [%v]", typeCursor, usedCursors)
// 	}
//
// 	if !contains(usedCursors, baseCursor) {
// 		t.Errorf("excpected [%s] to be in  [%v]", baseCursor, usedCursors)
// 	}
//
// 	if contains(usedCursors, subTypeCursor) {
// 		t.Errorf("didn't expect [%s] to be in  [%v]", subTypeCursor, usedCursors)
// 	}
//
// }

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
