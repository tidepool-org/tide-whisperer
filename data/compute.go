package data

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/tide-whisperer/store"
)

type (
	// LoggerInfo struct for grouping fields used in log functions
	LoggerInfo struct {
		UserID       string
		UserIDs      []string
		requestID    string
		apiCallStart time.Time
		queryStart   time.Time
	}
	// only documented for swaggo
	cbgCounts struct {
		veryLow  int
		low      int
		target   int
		high     int
		veryHigh int
	}
	// only documented for swaggo
	cbgLastTimes struct {
		veryLow  time.Time
		low      time.Time
		target   time.Time
		high     time.Time
		veryHigh time.Time
	}
	// only documented for swaggo
	cbgRates struct {
		veryLow  float32
		low      float32
		target   float32
		high     float32
		veryHigh float32
	}
	// only documented for swaggo
	cbgTotalTimes struct {
		veryLow  int
		low      int
		target   int
		high     int
		veryHigh int
	}
	// TirResult only documented for swaggo
	TirResult struct {
		userId      string
		lastCbgTime time.Time
		count       cbgCounts
		lastTime    cbgLastTimes
		rate        cbgRates
		totalTime   cbgTotalTimes
	}
)

const (
	slowAggregateQueryDuration = 0.5 // seconds
)

func writeMongoJSONResponse(res http.ResponseWriter, req *http.Request, cursor mongo.StorageIterator, logData *LoggerInfo) {
	res.Header().Add("Content-Type", "application/json")
	res.Write([]byte("["))

	var writeCount int
	var results map[string]interface{}
	for cursor.Next(req.Context()) {
		err := cursor.Decode(&results)
		if err != nil {
			logIndicatorError(logData, "Mongo Decode", err)
		}
		if len(results) > 0 {
			if bytes, err := json.Marshal(results); err != nil {
				logIndicatorError(logData, "Marshal", err)
			} else {
				if writeCount > 0 {
					res.Write([]byte(","))
				}
				res.Write([]byte("\n"))
				res.Write(bytes)
				writeCount++
			}
		}
	}
	if writeCount > 0 {
		res.Write([]byte("\n"))
	}
	res.Write([]byte("]"))
}
func logIndicatorError(logData *LoggerInfo, message string, err error) {
	log.Printf("%s request %s users %s %s returned error: %s", DataAPIPrefix, logData.requestID, logData.UserIDs, message, err)
}
func logIndicatorSlowQuery(logData *LoggerInfo, message string) {
	if queryDuration := time.Now().Sub(logData.queryStart).Seconds(); queryDuration > slowAggregateQueryDuration {
		log.Printf("%s SlowQuery: request %s users %s %s took %.3fs", DataAPIPrefix, logData.requestID, logData.UserIDs, message, queryDuration)
	}
}

func (a *API) parseIndicatorParams(q url.Values) (*store.AggParams, error) {
	endStr, err := cleanDateString(q.Get("endDate"))
	if err != nil {
		return nil, err
	}
	var endTime time.Time
	if endStr == "" {
		endTime = time.Now()
	} else {
		endTime, _ = time.Parse(time.RFC3339, endStr)
	}
	endTime.UTC()
	startStr := endTime.Add(-(time.Hour * 24)).Format(time.RFC3339)
	endStr = endTime.Format(time.RFC3339)

	userIds := strings.Split(q.Get("userIds"), ",")

	p := &store.AggParams{
		UserIDs:       userIds,
		Date:          store.Date{startStr, endStr},
		SchemaVersion: &a.schemaVersion,
	}

	return p, nil

}

// GetTimeInRange API function for time in range indicators
// @Summary Get time in range indicators for the given user ids
// @Description Get the api status
// @ID tide-whisperer-api-gettimeinrange
// @Accept json
// @Produce json
// @Param userIds query []string true "List of user ids to fetch" collectionFormat(csv)
// @Param endDate query string false "End date to get indicators" format(date-time)
// @Security TidepoolAuth
// @Success 200 {array} TirResult
// @Failure 403 {object} data.detailedError
// @Failure 500 {object} data.detailedError
// @Router /indicators/tir [get]
func (a *API) GetTimeInRange(res http.ResponseWriter, req *http.Request) {
	logInfo := &LoggerInfo{
		requestID:    newRequestID(),
		apiCallStart: time.Now(),
	}
	params, err := a.parseIndicatorParams(req.URL.Query())
	if err != nil {
		logIndicatorError(logInfo, "store.GetAggParams", err)
		jsonError(res, errorInvalidParameters, logInfo.apiCallStart)
		return
	}

	logInfo.UserIDs = params.UserIDs
	if !(a.isAuthorized(req, params.UserIDs)) {
		jsonError(res, errorNoViewPermission, logInfo.apiCallStart)
		return
	}

	ctx := req.Context()

	logInfo.queryStart = time.Now()
	iter, err := a.store.GetTimeInRangeData(ctx, params, false)
	if err != nil {
		logIndicatorError(logInfo, "Mongo Query", err)
		jsonError(res, errorRunningQuery, logInfo.apiCallStart)
	}
	logIndicatorSlowQuery(logInfo, "GetTimeInRangeData")

	defer iter.Close(req.Context())
	writeMongoJSONResponse(res, req, iter, logInfo)
}
