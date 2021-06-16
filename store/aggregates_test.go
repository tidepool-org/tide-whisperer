package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func basicTirParams() *AggParams {
	return &AggParams{
		UserIDs:       []string{"tir123", "tir456"},
		SchemaVersion: &SchemaVersion{Maximum: 2, Minimum: 0},
		Date: Date{
			Start: "2020-01-01T08:00:00Z",
			End:   "2020-01-02T08:00:00Z",
		},
	}
}
func basicTirQuery() []bson.M {
	qParams := basicTirParams()

	return generateTirAggregateQuery(qParams)
}

func TestStore_generateTirAggregateQuery(t *testing.T) {

	time.Now()
	query := basicTirQuery()
	expectedQuery := expectedBasicTirQuery()
	queryString, _ := json.MarshalIndent(query, "", "    ")
	expectedQuerytring, _ := json.MarshalIndent(expectedQuery, "", "    ")
	if string(queryString) != string(expectedQuerytring) {
		errMessage := "expected:\n" + string(expectedQuerytring) +
			"\ndid not match returned query\n" + string(queryString)
		t.Error(errMessage)
	}

}

func checkDate(t *testing.T, field string, result map[string]interface{}, expected string) {
	expectedTime, _ := time.Parse(time.RFC3339, expected)
	expectedTime = expectedTime.UTC()
	resultTime := result[field].(primitive.DateTime).Time().UTC()
	if resultTime != expectedTime {
		t.Errorf("Unexpected %v given %v expected %v", field, resultTime, expectedTime)
	}
}
func checkTirAggregate(t *testing.T, parentField string, result map[string]interface{}, expected map[string]interface{}) {
	subfield := result[parentField].(map[string]interface{})
	fields := []string{"veryLow", "low", "target", "high", "veryHigh"}
	for _, field := range fields {
		expectedValue := expected[field]
		resultValue := subfield[field]
		err := false
		switch parentField {
		case "count", "totalTime":
			err = resultValue.(int32) != int32(expectedValue.(int))
		case "lastTime":
			if expectedValue == nil {
				err = resultValue != nil
			} else {
				expectedTime, _ := time.Parse(time.RFC3339, expectedValue.(string))
				expectedTime = expectedTime.UTC()
				err = resultValue.(primitive.DateTime).Time().UTC() != expectedTime
			}
		case "rate":
			err = resultValue.(float64) != expectedValue.(float64)
		}
		if err {
			t.Errorf("Unexpected %v.%v given %v expected %v", parentField, field, resultValue, expectedValue)
		}
	}
}
func storeDataForTirTests() []interface{} {
	testData := testDataForTirTests()

	storeData := make([]interface{}, len(testData))
	index := 0
	for _, v := range testData {
		storeData[index] = v
		index++
	}

	return storeData
}
func TestStore_GetTimeInRangeData(t *testing.T) {
	storeData := storeDataForTirTests()

	store := before(t, storeData...)

	qParams := basicTirParams()

	iter, err := store.GetTimeInRangeData(context.Background(), qParams, false)
	if err != nil {
		t.Error("Error querying Mongo")
	}

	resultCount := 0
	for iter.Next(context.Background()) {
		var result map[string]interface{}
		err := iter.Decode(&result)
		if err != nil {
			t.Error("Mongo Decode error")
		}
		switch userID := result["userId"]; userID {
		case "tir123":
			checkDate(t, "lastCbgTime", result, "2020-01-01T08:45:00Z")

			expectedCounts := map[string]interface{}{
				"veryLow": 2, "low": 2, "target": 2,
				"high": 2, "veryHigh": 2,
			}
			checkTirAggregate(t, "count", result, expectedCounts)

			expectedLastTimes := map[string]interface{}{
				"veryLow":  "2020-01-01T08:05:00Z",
				"low":      "2020-01-01T08:15:00Z",
				"target":   "2020-01-01T08:25:00Z",
				"high":     "2020-01-01T08:35:00Z",
				"veryHigh": "2020-01-01T08:45:00Z",
			}
			checkTirAggregate(t, "lastTime", result, expectedLastTimes)

			expectedRates := map[string]interface{}{
				"veryLow":  20.0,
				"low":      20.0,
				"target":   20.0,
				"high":     20.0,
				"veryHigh": 20.0,
			}
			checkTirAggregate(t, "rate", result, expectedRates)

			expectedTotalTimes := map[string]interface{}{
				"veryLow":  10,
				"low":      10,
				"target":   10,
				"high":     10,
				"veryHigh": 10,
			}
			checkTirAggregate(t, "totalTime", result, expectedTotalTimes)
		case "tir456":
			checkDate(t, "lastCbgTime", result, "2020-01-01T08:35:00Z")

			expectedCounts := map[string]interface{}{
				"veryLow": 2, "low": 2, "target": 2,
				"high": 2, "veryHigh": 0,
			}
			checkTirAggregate(t, "count", result, expectedCounts)

			expectedLastTimes := map[string]interface{}{
				"veryLow":  "2020-01-01T08:05:00Z",
				"low":      "2020-01-01T08:15:00Z",
				"target":   "2020-01-01T08:25:00Z",
				"high":     "2020-01-01T08:35:00Z",
				"veryHigh": nil,
			}
			checkTirAggregate(t, "lastTime", result, expectedLastTimes)

			expectedRates := map[string]interface{}{
				"veryLow":  25.0,
				"low":      25.0,
				"target":   25.0,
				"high":     25.0,
				"veryHigh": 0.0,
			}
			checkTirAggregate(t, "rate", result, expectedRates)

			expectedTotalTimes := map[string]interface{}{
				"veryLow":  10,
				"low":      10,
				"target":   10,
				"high":     10,
				"veryHigh": 0,
			}
			checkTirAggregate(t, "totalTime", result, expectedTotalTimes)

		default:
			t.Errorf("Unexpected userId:%v in the aggregate", userID)
		}
		resultCount++
	}

	if resultCount != 2 {
		t.Errorf("Tir aggregate request gave %d results expected %d", resultCount, 2)
	}
}

// expected query for tir aggregate
func expectedBasicTirQuery() []bson.M {
	return []bson.M{
		bson.M{
			"$match": bson.M{
				"$and": []bson.M{
					{"type": "cbg"},
					{"_userId": bson.M{"$in": []string{"tir123", "tir456"}}},
					{"time": bson.M{"$gte": "2020-01-01T08:00:00Z"}},
					{"time": bson.M{"$lte": "2020-01-02T08:00:00Z"}},
				},
			},
		},
		bson.M{
			"$project": bson.M{
				"userId": "$_userId",
				"time":   bson.M{"$dateFromString": bson.M{"dateString": "$time"}},
				"value":  "$value",
				"cbgCategory": bson.M{
					"$switch": bson.M{
						"branches": []bson.M{
							{
								"case": bson.M{"$lt": []interface{}{"$value", 3.0}},
								"then": "veryLow",
							},
							{
								"case": bson.M{
									"$and": []bson.M{
										{"$gte": []interface{}{"$value", 3.0}},
										{"$lt": []interface{}{"$value", 3.9}},
									},
								},
								"then": "low",
							},
							{
								"case": bson.M{
									"$and": []bson.M{
										{"$gt": []interface{}{"$value", 10.0}},
										{"$lte": []interface{}{"$value", 13.9}},
									},
								},
								"then": "high",
							},
							{
								"case": bson.M{"$gt": []interface{}{"$value", 13.9}},
								"then": "veryHigh",
							},
						},
						"default": "target",
					},
				},
			},
		},
		bson.M{
			"$group": bson.M{
				"_id": bson.M{
					"userId":   "$userId",
					"category": "$cbgCategory",
				},
				"lastCbgTime": bson.M{"$max": "$time"},
				"veryLowCount": bson.M{
					"$sum": bson.M{
						"$switch": bson.M{
							"branches": []bson.M{
								{
									"case": bson.M{"$eq": []string{"$cbgCategory", "veryLow"}},
									"then": 1,
								},
							},
							"default": 0,
						},
					},
				},
				"lowCount": bson.M{
					"$sum": bson.M{
						"$switch": bson.M{
							"branches": []bson.M{
								{
									"case": bson.M{"$eq": []string{"$cbgCategory", "low"}},
									"then": 1,
								},
							},
							"default": 0,
						},
					},
				},
				"targetCount": bson.M{
					"$sum": bson.M{
						"$switch": bson.M{
							"branches": []bson.M{
								{
									"case": bson.M{"$eq": []string{"$cbgCategory", "target"}},
									"then": 1,
								},
							},
							"default": 0,
						},
					},
				},
				"highCount": bson.M{
					"$sum": bson.M{
						"$switch": bson.M{
							"branches": []bson.M{
								{
									"case": bson.M{"$eq": []string{"$cbgCategory", "high"}},
									"then": 1,
								},
							},
							"default": 0,
						},
					},
				},
				"veryHighCount": bson.M{
					"$sum": bson.M{
						"$switch": bson.M{
							"branches": []bson.M{
								{
									"case": bson.M{"$eq": []string{"$cbgCategory", "veryHigh"}},
									"then": 1,
								},
							},
							"default": 0,
						},
					},
				},
				"veryLowTime": bson.M{
					"$max": bson.M{
						"$switch": bson.M{
							"branches": []bson.M{
								{
									"case": bson.M{"$eq": []string{"$cbgCategory", "veryLow"}},
									"then": "$time",
								},
							},
							"default": nil,
						},
					},
				},
				"lowTime": bson.M{
					"$max": bson.M{
						"$switch": bson.M{
							"branches": []bson.M{
								{
									"case": bson.M{"$eq": []string{"$cbgCategory", "low"}},
									"then": "$time",
								},
							},
							"default": nil,
						},
					},
				},
				"targetTime": bson.M{
					"$max": bson.M{
						"$switch": bson.M{
							"branches": []bson.M{
								{
									"case": bson.M{"$eq": []string{"$cbgCategory", "target"}},
									"then": "$time",
								},
							},
							"default": nil,
						},
					},
				},
				"highTime": bson.M{
					"$max": bson.M{
						"$switch": bson.M{
							"branches": []bson.M{
								{
									"case": bson.M{"$eq": []string{"$cbgCategory", "high"}},
									"then": "$time",
								},
							},
							"default": nil,
						},
					},
				},
				"veryHighTime": bson.M{
					"$max": bson.M{
						"$switch": bson.M{
							"branches": []bson.M{
								{
									"case": bson.M{"$eq": []string{"$cbgCategory", "veryHigh"}},
									"then": "$time",
								},
							},
							"default": nil,
						},
					},
				},
			},
		},
		bson.M{
			"$group": bson.M{
				"_id":         "$_id.userId",
				"lastCbgTime": bson.M{"$max": "$lastCbgTime"},
				"veryLowCount": bson.M{
					"$max": "$veryLowCount",
				},
				"lowCount": bson.M{
					"$max": "$lowCount",
				},
				"targetCount": bson.M{
					"$max": "$targetCount",
				},
				"highCount": bson.M{
					"$max": "$highCount",
				},
				"veryHighCount": bson.M{
					"$max": "$veryHighCount",
				},
				"veryLowTime": bson.M{
					"$max": "$veryLowTime",
				},
				"lowTime": bson.M{
					"$max": "$lowTime",
				},
				"targetTime": bson.M{
					"$max": "$targetTime",
				},
				"highTime": bson.M{
					"$max": "$highTime",
				},
				"veryHighTime": bson.M{
					"$max": "$veryHighTime",
				},
			},
		},
		bson.M{
			"$project": bson.M{
				"_id":         0,
				"userId":      "$_id",
				"lastCbgTime": "$lastCbgTime",
				"count": bson.M{
					"veryLow":  "$veryLowCount",
					"low":      "$lowCount",
					"target":   "$targetCount",
					"high":     "$highCount",
					"veryHigh": "$veryHighCount",
				},
				"lastTime": bson.M{
					"veryLow":  "$veryLowTime",
					"low":      "$lowTime",
					"target":   "$targetTime",
					"high":     "$highTime",
					"veryHigh": "$veryHighTime",
				},
				"rate": bson.M{
					"$let": bson.M{
						"vars": bson.M{
							"total": bson.M{
								"$add": []string{
									"$veryLowCount", "$lowCount", "$targetCount",
									"$highCount", "$veryHighCount",
								},
							},
						},
						"in": bson.M{
							"veryLow": bson.M{
								"$multiply": []interface{}{
									bson.M{"$divide": []string{"$veryLowCount", "$$total"}},
									100,
								},
							},
							"low": bson.M{
								"$multiply": []interface{}{
									bson.M{"$divide": []string{"$lowCount", "$$total"}},
									100,
								},
							},
							"target": bson.M{
								"$multiply": []interface{}{
									bson.M{"$divide": []string{"$targetCount", "$$total"}},
									100,
								},
							},
							"high": bson.M{
								"$multiply": []interface{}{
									bson.M{"$divide": []string{"$highCount", "$$total"}},
									100,
								},
							},
							"veryHigh": bson.M{
								"$multiply": []interface{}{
									bson.M{"$divide": []string{"$veryHighCount", "$$total"}},
									100,
								},
							},
						},
					},
				},
				"totalTime": bson.M{
					"$let": bson.M{
						"vars": bson.M{
							"cgmInterval": bson.M{"$literal": 5},
						},
						"in": bson.M{
							"veryLow": bson.M{
								"$multiply": []string{"$veryLowCount", "$$cgmInterval"},
							},
							"low": bson.M{
								"$multiply": []string{"$lowCount", "$$cgmInterval"},
							},
							"target": bson.M{
								"$multiply": []string{"$targetCount", "$$cgmInterval"},
							},
							"high": bson.M{
								"$multiply": []string{"$highCount", "$$cgmInterval"},
							},
							"veryHigh": bson.M{
								"$multiply": []string{"$veryHighCount", "$$cgmInterval"},
							},
						},
					},
				},
			},
		},
	}
}

// data for time in range test
func testDataForTirTests() map[string]bson.M {
	testData := map[string]bson.M{
		// user1
		// userid ok | type ok | time ok | veryLow value
		"user1cbgVeryLow1": bson.M{
			"_userId": "tir123",
			"time":    "2020-01-01T08:00:00Z",
			"type":    "cbg",
			"value":   2.8,
		},
		"user1cbgVeryLow2": bson.M{
			"_userId": "tir123",
			"time":    "2020-01-01T08:05:00Z",
			"type":    "cbg",
			"value":   2.99,
		},
		// userid ok | type ok | time ok | low value
		"user1cbgLow1": bson.M{
			"_userId": "tir123",
			"time":    "2020-01-01T08:10:00Z",
			"type":    "cbg",
			"value":   3.01,
		},
		"user1cbgLow2": bson.M{
			"_userId": "tir123",
			"time":    "2020-01-01T08:15:00Z",
			"type":    "cbg",
			"value":   3.89,
		},
		// userid ok | type ok | time ok | target value
		"user1cbgTarget1": bson.M{
			"_userId": "tir123",
			"time":    "2020-01-01T08:20:00Z",
			"type":    "cbg",
			"value":   3.91,
		},
		"user1cbgTarget2": bson.M{
			"_userId": "tir123",
			"time":    "2020-01-01T08:25:00Z",
			"type":    "cbg",
			"value":   9.99,
		},
		// userid ok | type ok | time ok | high value
		"user1cbgHigh1": bson.M{
			"_userId": "tir123",
			"time":    "2020-01-01T08:30:00Z",
			"type":    "cbg",
			"value":   10.01,
		},
		"user1cbgHigh2": bson.M{
			"_userId": "tir123",
			"time":    "2020-01-01T08:35:00Z",
			"type":    "cbg",
			"value":   13.89,
		},
		// userid ok | type ok | time ok | veryhigh value
		"user1cbgVeryHigh1": bson.M{
			"_userId": "tir123",
			"time":    "2020-01-01T08:40:00Z",
			"type":    "cbg",
			"value":   13.91,
		},
		"user1cbgVeryHigh2": bson.M{
			"_userId": "tir123",
			"time":    "2020-01-01T08:45:00Z",
			"type":    "cbg",
			"value":   14.5,
		},
		// userid ok | type ok | time ko
		"user1CbgOutOfTime": bson.M{
			"_userId": "tir123",
			"time":    "2020-01-03T00:00:00Z",
			"type":    "cbg",
			"value":   7.1237,
		},
		// userid ok | type ko | time ok
		"user1CbgWrongType": bson.M{
			"_userId": "tir123",
			"time":    "2020-01-01T08:50:00Z",
			"type":    "smbg",
			"value":   7.1237,
		},
		// user2
		// userid ok | type ok | time ok | veryLow value
		"user2cbgVeryLow1": bson.M{
			"_userId": "tir456",
			"time":    "2020-01-01T08:00:00Z",
			"type":    "cbg",
			"value":   2.8,
		},
		"user2cbgVeryLow2": bson.M{
			"_userId": "tir456",
			"time":    "2020-01-01T08:05:00Z",
			"type":    "cbg",
			"value":   2.99,
		},
		// userid ok | type ok | time ok | low value
		"user2cbgLow1": bson.M{
			"_userId": "tir456",
			"time":    "2020-01-01T08:10:00Z",
			"type":    "cbg",
			"value":   3.01,
		},
		"user2cbgLow2": bson.M{
			"_userId": "tir456",
			"time":    "2020-01-01T08:15:00Z",
			"type":    "cbg",
			"value":   3.89,
		},
		// userid ok | type ok | time ok | target value
		"user2cbgTarget1": bson.M{
			"_userId": "tir456",
			"time":    "2020-01-01T08:20:00Z",
			"type":    "cbg",
			"value":   3.91,
		},
		"user2cbgTarget2": bson.M{
			"_userId": "tir456",
			"time":    "2020-01-01T08:25:00Z",
			"type":    "cbg",
			"value":   9.99,
		},
		// userid ok | type ok | time ok | high value
		"user2cbgHigh1": bson.M{
			"_userId": "tir456",
			"time":    "2020-01-01T08:30:00Z",
			"type":    "cbg",
			"value":   10.01,
		},
		"user2cbgHigh2": bson.M{
			"_userId": "tir456",
			"time":    "2020-01-01T08:35:00Z",
			"type":    "cbg",
			"value":   13.89,
		},
		// userid ok | type ok | time ko
		"user2CbgOutOfTime": bson.M{
			"_userId": "tir456",
			"time":    "2019-12-31T08:00:00Z",
			"type":    "cbg",
			"value":   7.1237,
		},
		// userid ok | type ko | time ok
		"user2CbgWrongType": bson.M{
			"_userId": "tir456",
			"time":    "2020-01-01T08:50:00Z",
			"type":    "food",
			"value":   7.1237,
		},
		// userid ko | type ok | time ok
		"user3cbg": bson.M{
			"_userId": "xyz123",
			"time":    "2020-01-01T08:30:00Z",
			"type":    "cbg",
			"value":   7.1237,
		},
	}

	return testData
}
