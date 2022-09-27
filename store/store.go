package store

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	goComMgo "github.com/tidepool-org/go-common/clients/mongo"
)

const (
	dataCollectionName          = "deviceData"
	dataStoreAPIPrefix          = "api/data/store "
	portalDb                    = "portal"
	parametersHistoryCollection = "patient_parameters_history"
	idxUserIDTypeTime           = "UserIdTypeTimeWeighted"
)

var unwantedFields = bson.M{
	"_id":                0,
	"_userId":            0,
	"_groupId":           0,
	"_version":           0,
	"_active":            0,
	"_schemaVersion":     0,
	"createdTime":        0,
	"modifiedTime":       0,
	"conversionOffset":   0,
	"clockDriftOffset":   0,
	"timezoneOffset":     0,
	"deviceTime":         0,
	"deviceId":           0,
	"deviceSerialNumber": 0,
	"source":             0,
}

var unwantedPumpSettingsFields = bson.M{
	"_id":                  0,
	"_userId":              0,
	"_groupId":             0,
	"_version":             0,
	"_active":              0,
	"_schemaVersion":       0,
	"createdTime":          0,
	"modifiedTime":         0,
	"conversionOffset":     0,
	"clockDriftOffset":     0,
	"timezoneOffset":       0,
	"basalSchedules":       0,
	"bgTargets":            0,
	"carbRatios":           0,
	"insulinSensitivities": 0,
}

var wantedRangeFields = bson.M{
	"_id":  0,
	"time": 1,
}

var wantedBgFields = bson.M{
	"_id":   0,
	"value": 1,
	"units": 1,
}

var tideWhispererIndexes = map[string][]mongo.IndexModel{
	"deviceData": {
		{
			Keys: bson.D{{Key: "_userId", Value: 1}, {Key: "type", Value: 1}, {Key: "time", Value: -1}},
			Options: options.Index().
				SetName(idxUserIDTypeTime).
				SetWeights(bson.M{"_userId": 10, "type": 5, "time": 1}),
		},
	},
}

type (
	// Storage - Interface for our storage layer
	Storage interface {
		goComMgo.Storage
		// WithContext(ctx context.Context) Storage
		// V1 API data functions:
		GetDataRangeV1(ctx context.Context, traceID string, userID string) (*Date, error)
		GetDataV1(ctx context.Context, traceID string, userID string, dates *Date, excludeTypes []string) (goComMgo.StorageIterator, error)
		GetLatestBasalSecurityProfile(ctx context.Context, traceID string, userID string) (*DbProfile, error)
		GetUploadDataV1(ctx context.Context, traceID string, uploadIds []string) (goComMgo.StorageIterator, error)
		GetCbgForSummaryV1(ctx context.Context, traceID string, userID string, startDate string) (goComMgo.StorageIterator, error)
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
	// Client struct
	Client struct {
		*goComMgo.StoreClient
	}

	// Date struct
	Date struct {
		Start string
		End   string
	}

	DbSchedule struct {
		Rate  float64 `bson:"rate,omitempty"`
		Start int64   `bson:"start,omitempty"`
	}

	DbProfile struct {
		Type          string       `bson:"type,omitempty"`
		Time          time.Time    `bson:"time,omitempty"`
		Timezone      string       `bson:"timezone,omitempty"`
		Guid          string       `bson:"guid,omitempty"`
		BasalSchedule []DbSchedule `bson:"basalSchedule,omitempty"`
	}
)

// InArray returns a boolean indicating the presence of a string value in a string array
func InArray(needle string, arr []string) bool {
	for _, n := range arr {
		if needle == n {
			return true
		}
	}
	return false
}

// NewStore creates a new Client
func NewStore(config *goComMgo.Config, logger *log.Logger) (*Client, error) {
	if config != nil {
		config.Indexes = tideWhispererIndexes
	}
	client := Client{}
	store, err := goComMgo.NewStoreClient(config, logger)
	client.StoreClient = store
	return &client, err
}

func dataCollection(c *Client) *mongo.Collection {
	return c.Collection(dataCollectionName)
}
func mgoParametersHistoryCollection(c *Client) *mongo.Collection {
	return c.Collection(parametersHistoryCollection, portalDb)
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

// GetDataRangeV1 returns the time data range
//
// If no data for the requested user, return nil or empty string dates
func (c *Client) GetDataRangeV1(ctx context.Context, traceID string, userID string) (*Date, error) {
	if userID == "" {
		return nil, errors.New("user id is missing")
	}

	dateRange := &Date{
		Start: "",
		End:   "",
	}
	var res map[string]interface{}

	query := bson.M{
		"_userId": userID,
		// Use only diabetes data, excluding upload & pumpSettings
		"type": bson.M{"$not": bson.M{"$in": []string{"upload", "pumpSettings"}}},
	}

	opts := options.FindOne()
	opts.SetHint(idxUserIDTypeTime)
	opts.SetProjection(wantedRangeFields)
	opts.SetComment(traceID)

	// Finding Last time (i.e. findOne with sort time DESC)
	opts.SetSort(bson.D{primitive.E{Key: "time", Value: -1}})
	err := dataCollection(c).FindOne(ctx, query, opts).Decode(&res)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	dateRange.End = res["time"].(string)

	// Finding First time (i.e. findOne with sort time ASC)
	opts.SetSort(bson.D{primitive.E{Key: "time", Value: 1}})
	err = dataCollection(c).FindOne(ctx, query, opts).Decode(&res)
	if err != nil {
		return nil, err
	}
	dateRange.Start = res["time"].(string)

	return dateRange, nil
}

// GetDataV1 v1 api call to fetch diabetes data, excludes "upload" and "pumpSettings"
// and potentially other types
func (c *Client) GetDataV1(ctx context.Context, traceID string, userID string, dates *Date, excludeTypes []string) (goComMgo.StorageIterator, error) {
	if !InArray("upload", excludeTypes) {
		excludeTypes = append(excludeTypes, "upload")
	}
	if !InArray("pumpSettings", excludeTypes) {
		excludeTypes = append(excludeTypes, "pumpSettings")
	}
	query := bson.M{
		"_userId": userID,
		"type":    bson.M{"$not": bson.M{"$in": excludeTypes}},
	}

	if dates.Start != "" && dates.End != "" {
		query["time"] = bson.M{"$gte": dates.Start, "$lt": dates.End}
	} else if dates.Start != "" {
		query["time"] = bson.M{"$gte": dates.Start}
	} else if dates.End != "" {
		query["time"] = bson.M{"$lt": dates.End}
	}

	opts := options.Find()
	opts.SetProjection(unwantedFields)
	opts.SetHint(idxUserIDTypeTime)
	opts.SetComment(traceID)

	return dataCollection(c).Find(ctx, query, opts)
}

func (c *Client) GetLatestBasalSecurityProfile(ctx context.Context, traceID string, userID string) (*DbProfile, error) {
	if userID == "" {
		return nil, errors.New("invalid user id")
	}

	query := bson.M{
		"_userId": userID,
		"type":    "basalSecurity",
	}
	opts := options.FindOne()
	//opts.SetProjection(unwantedPumpSettingsFields) TODO
	opts.SetSort(bson.M{"time": -1})
	opts.SetComment(traceID)
	var result DbProfile
	err := dataCollection(c).FindOne(ctx, query, opts).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return &result, nil
}

// GetUploadDataV1 Fetch upload data from theirs upload ids, using the $in query parameter
func (c *Client) GetUploadDataV1(ctx context.Context, traceID string, uploadIds []string) (goComMgo.StorageIterator, error) {
	query := bson.M{
		"uploadId": bson.M{"$in": uploadIds},
		"type":     "upload",
	}

	opts := options.Find()
	opts.SetProjection(unwantedFields)
	opts.SetComment(traceID)
	return dataCollection(c).Find(ctx, query, opts)
}

// GetCbgForSummaryV1 return the cbg/smbg values for the given user starting at startDate
func (c *Client) GetCbgForSummaryV1(ctx context.Context, traceID string, userID string, startDate string) (goComMgo.StorageIterator, error) {
	query := bson.M{
		"_userId": userID,
		"type":    "cbg",
		"time":    bson.M{"$gt": startDate},
	}

	opts := options.Find()
	opts.SetProjection(wantedBgFields)
	opts.SetComment(traceID)
	opts.SetHint(idxUserIDTypeTime)
	return dataCollection(c).Find(ctx, query, opts)
}
