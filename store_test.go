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

func before(t *testing.T) *MongoStoreClient {

	store := NewMongoStoreClient(testingConfig)

	//INIT THE TEST - we use a clean copy of the collection before we start
	cpy := store.session.Copy()
	defer cpy.Close()

	//just drop and don't worry about any errors
	mgoDataCollection(cpy).DropCollection()

	if err := mgoDataCollection(cpy).Create(&mgo.CollectionInfo{}); err != nil {
		t.Error("We couldn't created the deviceData collection for these tests ", err)
	}
	return NewMongoStoreClient(testingConfig)
}

func getErrString(mongoQuery, expectedQuery bson.M) string {
	exp, _ := json.MarshalIndent(expectedQuery, "", "  ")
	mq, _ := json.MarshalIndent(mongoQuery, "", "  ")
	return "expected:\n" + string(exp) + "\ndid not match returned query\n" + string(mq)
}

func TestStore_IndexesExist(t *testing.T) {

	const (
		//index names based on feilds used
		subtype_idx = "subType_1"
		type_idx    = "type_1"
		basic_idx   = "_groupId_1__active_1__schemaVersion_1"
		id_idx      = "_id_"
	)

	store := before(t)

	sCopy := store.session
	defer sCopy.Close()

	if idxs, err := mgoDataCollection(sCopy).Indexes(); err != nil {
		t.Error("TestStore_IndexesExist unexpected error ", err.Error())
	} else {
		// there are the two we have added and also the standard index
		if len(idxs) != 4 {
			t.Fatalf("TestStore_IndexesExist should be THREE but found [%d] ", len(idxs))
		}

		if idxs[3].Name != type_idx {
			t.Errorf("TestStore_IndexesExist expected [%s] got [%s] ", type_idx, idxs[3].Name)
		}

		if idxs[2].Name != subtype_idx {
			t.Errorf("TestStore_IndexesExist expected [%s] got [%s] ", subtype_idx, idxs[2].Name)
		}

		if idxs[1].Name != id_idx {
			t.Errorf("TestStore_IndexesExist expected [%s] got [%s] ", id_idx, idxs[1].Name)
		}

		if idxs[0].Name != basic_idx {
			t.Errorf("TestStore_IndexesExist expected [%s] got [%s] ", basic_idx, idxs[0].Name)
		}

	}

}

func basicQuery() bson.M {
	qParams := &params{
		active:        true,
		groupId:       "123",
		userId:        "321",
		schemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
	}

	return generateMongoQuery(qParams)
}

func allParamsQuery() bson.M {
	qParams := &params{
		active:        true,
		groupId:       "123ggf",
		userId:        "abc123",
		schemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		date:          date{"2015-10-07T15:00:00.000Z", "2015-10-11T15:00:00.000Z"},
		types:         []string{"smbg", "cbg"},
		subTypes:      []string{"stuff"},
	}

	return generateMongoQuery(qParams)
}

func noDatesQuery() bson.M {
	qParams := &params{
		active:        true,
		groupId:       "123ggf",
		userId:        "abc123",
		schemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		types:         []string{"smbg", "cbg"},
		subTypes:      []string{"stuff"},
	}
	return generateMongoQuery(qParams)
}

func dateRangedQuery() bson.M {
	qParams := &params{
		active:        true,
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
		"_schemaVersion": bson.M{"$gte": 0, "$lte": 2}}

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

	query := noDatesQuery()

	expectedQuery := bson.M{
		"_groupId":       "123ggf",
		"_active":        true,
		"type":           bson.M{"$in": strings.Split("smbg,cbg", ",")},
		"subType":        bson.M{"$in": strings.Split("stuff", ",")},
		"_schemaVersion": bson.M{"$gte": 0, "$lte": 2}}

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

func TestStore_IndexUse(t *testing.T) {

	/*store := before(t)

	sCopy := store.session
	defer sCopy.Close()

	query := basicQuery()*/

}
