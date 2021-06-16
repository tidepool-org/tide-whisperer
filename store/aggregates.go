package store

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"

	// For debug only pretty printing requests
	"encoding/json"
	"log"

	"github.com/tidepool-org/go-common/clients/mongo"
)

type (
	cbgUserPref struct {
		VeryLow     float32
		Low         float32
		High        float32
		VeryHigh    float32
		CgmInterval int
	}
	// AggParams struct
	AggParams struct {
		UserIDs []string
		Date
		*SchemaVersion
	}
)

func defaultCbgUserPref() cbgUserPref {
	return cbgUserPref{
		VeryLow:     3.0,
		Low:         3.9,
		High:        10.0,
		VeryHigh:    13.9,
		CgmInterval: 5,
	}
}

func dateFromString(field string) bson.M {
	return bson.M{"$dateFromString": bson.M{"dateString": field}}
}

func generateTirAggregateQuery(p *AggParams) []bson.M {
	finalQuery := []bson.M{}
	/*
		1st step of pipeline $match:
			filtering cbg data for userIds
			with time between now and now- 1 day
	*/
	matchCbgDataForUserIds := bson.M{
		"$and": []bson.M{
			{"type": "cbg"},
			{"_userId": bson.M{"$in": p.UserIDs}},
			{"time": bson.M{"$gte": p.Date.Start}},
			{"time": bson.M{"$lte": p.Date.End}},
		},
	}
	finalQuery = append(finalQuery, bson.M{"$match": matchCbgDataForUserIds})

	/*
		2nd step of pipeline $project:
			projecting  only used fields i.e userId/time/value
			casting string time to real Date
			cbg category based on the value and default thresholds
	*/
	castDate := dateFromString("$time")
	prefs := defaultCbgUserPref()
	projectPrefsQuery := bson.M{
		"userId": "$_userId",
		"time":   castDate,
		"value":  "$value",
		"cbgCategory": bson.M{
			"$switch": bson.M{
				"branches": []bson.M{
					{
						"case": bson.M{"$lt": []interface{}{"$value", prefs.VeryLow}},
						"then": "veryLow",
					},
					{
						"case": bson.M{
							"$and": []bson.M{
								{"$gte": []interface{}{"$value", prefs.VeryLow}},
								{"$lt": []interface{}{"$value", prefs.Low}},
							},
						},
						"then": "low",
					},
					{
						"case": bson.M{
							"$and": []bson.M{
								{"$gt": []interface{}{"$value", prefs.High}},
								{"$lte": []interface{}{"$value", prefs.VeryHigh}},
							},
						},
						"then": "high",
					},
					{
						"case": bson.M{"$gt": []interface{}{"$value", prefs.VeryHigh}},
						"then": "veryHigh",
					},
				},
				"default": "target",
			},
		},
	}
	finalQuery = append(finalQuery, bson.M{"$project": projectPrefsQuery})

	/*
		3rd step of pipeline $group:
			grouping data by userId/category/cgmInterval  with:
				max time for all categories (lastCbgTime)
				count of veryLow category
				count of low category
				count of target category
				count of high category
				count of veryHigh category
				max time of veryLow category
				max time of low category
				max time of target category
				max time of high category
				max time of veryHigh category
	*/
	countQuery := bson.M{
		"_id": bson.M{
			"userId":   "$userId",
			"category": "$cbgCategory",
		},
		"lastCbgTime": bson.M{"$max": "$time"},
	}
	thresholdNames := []string{"veryLow", "low", "target", "high", "veryHigh"}

	for _, threshold := range thresholdNames {
		countQuery[threshold+"Count"] = bson.M{
			"$sum": bson.M{
				"$switch": bson.M{
					"branches": []bson.M{
						{
							"case": bson.M{"$eq": []string{"$cbgCategory", threshold}},
							"then": 1,
						},
					},
					"default": 0,
				},
			},
		}
		countQuery[threshold+"Time"] = bson.M{
			"$max": bson.M{
				"$switch": bson.M{
					"branches": []bson.M{
						{
							"case": bson.M{"$eq": []string{"$cbgCategory", threshold}},
							"then": "$time",
						},
					},
					"default": nil,
				},
			},
		}
	}

	finalQuery = append(finalQuery, bson.M{"$group": countQuery})
	/*
		4th step of pipeline $group:
			grouping data by userId:
				max of lastCbgTime (already a max unique value per userId)
				max of veryLow category (only one line per userId as a value <> 0)
				max of low category (only one line per userId as a value <> 0)
				max of target category (only one line per userId as a value <> 0)
				max of high category (only one line per userId as a value <> 0)
				max of veryHigh category (only one line per userId as a value <> 0)
				max time of veryLow category (only one line per userId as a value <> nil)
				max time of low category (only one line per userId as a value <> nil)
				max time of target category (only one line per userId as a value <> nil)
				max time of high category (only one line per userId as a value <> nil)
				max time of veryHigh category (only one line per userId as a value <> nil)
	*/
	finalGroupQuery := bson.M{
		"_id":         "$_id.userId",
		"lastCbgTime": bson.M{"$max": "$lastCbgTime"},
	}
	for _, threshold := range thresholdNames {
		finalGroupQuery[threshold+"Count"] = bson.M{
			"$max": ("$" + threshold + "Count"),
		}
		finalGroupQuery[threshold+"Time"] = bson.M{
			"$max": ("$" + threshold + "Time"),
		}
	}
	finalQuery = append(finalQuery, bson.M{"$group": finalGroupQuery})
	/*
		5th step of pipeline $projecting:
			projecting data for output with following structure:
				userId
				lastCbgTime
				count : // number of cbg per category //
					veryLow
					low
					target
					high
					veryHigh
				lastTime :  // max of cbg time per category //
					veryLow
					low
					target
					high
					veryHigh
				rate : // percentage of (cbg events)/ (total events) per category //
					veryLow
					low
					target
					high
					veryHigh
				totalTime: // (number of cbg) * (cgm time interval) per catgeory //
					veryLow
					low
					target
					high
					veryHigh
	*/

	finalProjectQuery := bson.M{
		"_id":         0,
		"userId":      "$_id",
		"lastCbgTime": "$lastCbgTime",
	}
	counts := bson.M{}
	lastTimes := bson.M{}
	rates := bson.M{}
	totalTimes := bson.M{}
	fieldsTotal := make([]string, len(thresholdNames))
	for idx, threshold := range thresholdNames {
		counts[threshold] = "$" + threshold + "Count"
		lastTimes[threshold] = "$" + threshold + "Time"
		rates[threshold] = bson.M{
			"$multiply": []interface{}{
				bson.M{"$divide": []string{"$" + threshold + "Count", "$$total"}},
				100,
			},
		}
		totalTimes[threshold] = bson.M{
			"$multiply": []string{"$" + threshold + "Count", "$$cgmInterval"},
		}
		fieldsTotal[idx] = "$" + threshold + "Count"
	}
	finalProjectQuery["count"] = counts
	finalProjectQuery["lastTime"] = lastTimes
	finalProjectQuery["rate"] = bson.M{
		"$let": bson.M{
			"vars": bson.M{
				"total": bson.M{"$add": fieldsTotal},
			},
			"in": rates,
		},
	}
	finalProjectQuery["totalTime"] = bson.M{
		"$let": bson.M{
			"vars": bson.M{
				"cgmInterval": bson.M{"$literal": prefs.CgmInterval},
			},
			"in": totalTimes,
		},
	}
	finalQuery = append(finalQuery, bson.M{"$project": finalProjectQuery})

	return finalQuery
}

func (c *Client) GetTimeInRangeData(ctx context.Context, p *AggParams, logQuery bool) (mongo.StorageIterator, error) {
	query := generateTirAggregateQuery(p)
	// For debug purpose pretty printing mongo requests:
	if logQuery {
		if bytes, err := json.MarshalIndent(query, "", "    "); err != nil {
			log.Printf("json marshalled value error %s", err)
		} else {
			log.Printf("TIR aggregate request : %s", bytes)
		}
	}
	cursor, err := dataCollection(c).Aggregate(ctx, query)
	return cursor, err
}
