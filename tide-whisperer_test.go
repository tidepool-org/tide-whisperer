package main
import (
	"testing"
	"reflect"
	"labix.org/v2/mgo/bson"
	"encoding/json"
	"strings"
)

func TestGenerateMongoQuery_basic(t *testing.T) {
	userId := "abc123"
	minSV := 0
	maxSV := 1
	startDate := ""
	endDate := ""
	types := ""
	subTypes := ""

	mongoQuery, err := generateMongoQuery(userId, minSV, maxSV, startDate, endDate, types, subTypes)
	if err != nil {
		t.Fatal(err)
	}

	expectedQuery := bson.M{"_groupId": userId, 
		"_active": true, 
		"_schemaVersion": bson.M{"$gte": minSV, "$lte": maxSV }}

	eq := reflect.DeepEqual(mongoQuery, expectedQuery)
	if !eq {
	    t.Fatal(getErrString(mongoQuery, expectedQuery))
	}
}

func TestGenerateMongoQuery_allparams(t *testing.T) {
	userId := "abc123"
	minSV := 0
	maxSV := 1
	startDate := "2015-10-08T15:00:00.000Z"
	endDate := "2015-10-11T15:00:00.000Z"
	types := "smbg"
	subTypes := "stype"

	mongoQuery, err := generateMongoQuery(userId, minSV, maxSV, startDate, endDate, types, subTypes)
	if err != nil {
		t.Fatal(err)
	}

	expectedQuery := bson.M{
		"_groupId": userId, 
		"_active": true, 
		"type":bson.M{"$in":strings.Split("smbg",",")},
		"subType":bson.M{"$in":strings.Split("stype",",")},
		"time": bson.M{"$gte": "2015-10-08T15:00:00Z", "$lte": "2015-10-11T15:00:00Z"},
		"_schemaVersion": bson.M{"$gte": minSV, "$lte": maxSV }}

	eq := reflect.DeepEqual(mongoQuery, expectedQuery)
	if !eq {
	    t.Fatal(getErrString(mongoQuery, expectedQuery))
	}
}

func TestGenerateMongoQuery_badDates(t *testing.T) {
	userId := "abc123"
	minSV := 0
	maxSV := 1
	startDate := "123"
	endDate := ""
	types := "smbg"
	subTypes := "stype"

	mongoQuery, err := generateMongoQuery(userId, minSV, maxSV, startDate, endDate, types, subTypes)
	if err == nil {
		t.Fatal("should have failed to parse start date")
	}

	startDate = "2015-10-11T15:00:00.000Z"
	endDate = "2015-10-11"
	mongoQuery, err = generateMongoQuery(userId, minSV, maxSV, startDate, endDate, types, subTypes)
	if err == nil {
		t.Fatal("Should have failed to parse end date")
	}

	if mongoQuery == nil{}
}

func TestGenerateMongoQuery_multipleTypesAndSubTypes(t *testing.T) {
	userId := "abc123"
	minSV := 0
	maxSV := 1
	startDate := "2015-10-08T15:00:00.000Z"
	endDate := "2015-10-11T15:00:00.000Z"
	types := "smbg,physicalActivity"
	subTypes := "stype1,stype2"

	mongoQuery, err := generateMongoQuery(userId, minSV, maxSV, startDate, endDate, types, subTypes)
	if err != nil {
		t.Fatal(err)
	}

	expectedQuery := bson.M{
		"_groupId": userId, 
		"_active": true, 
		"type":bson.M{"$in":strings.Split(types,",")},
		"subType":bson.M{"$in":strings.Split(subTypes,",")},
		"time": bson.M{"$gte": "2015-10-08T15:00:00Z", "$lte": "2015-10-11T15:00:00Z"},
		"_schemaVersion": bson.M{"$gte": minSV, "$lte": maxSV }}

	eq := reflect.DeepEqual(mongoQuery, expectedQuery)
	if !eq {
	    t.Fatal(getErrString(mongoQuery, expectedQuery))
	}
}

func getErrString(mongoQuery, expectedQuery bson.M) string {
	exp, err1 := json.MarshalIndent(expectedQuery, "", "  ")
	mq, err2 := json.MarshalIndent(mongoQuery, "", "  ")
	errStr := "expected:\n"+string(exp)+"\ndid not match returned query\n"+string(mq)
	if err1 == nil && err2 == nil {}
	return errStr

}




