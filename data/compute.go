package data

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
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
	// summaryGlyLimits is the return struct of getDataSummaryGlyLimits()
	summaryGlyLimits struct {
		hypoMgdl   float64
		hyperMgdl  float64
		hypoMmoll  float64
		hyperMmoll float64
		// The unit found in the parameters
		unit string
	}
	// summaryTresholds is the return struct of getDataSummaryThresholds()
	summaryTresholds struct {
		totalNumBgValues      int
		percentTimeInRange    int
		percentTimeBelowRange int
	}
)

const (
	slowAggregateQueryDuration         = 0.5 // seconds
	patientGlyHypoLimitMgdl    float64 = 70.0
	patientGlyHyperLimitMgdl   float64 = 180
	patientGlyHypoLimitMmoll   float64 = 3.9
	patientGlyHyperLimitMmoll  float64 = 10.0
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
		Date:          store.Date{Start: startStr, End: endStr},
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
		a.jsonError(res, errorInvalidParameters, logInfo.apiCallStart)
		return
	}

	logInfo.UserIDs = params.UserIDs
	if !(a.isAuthorized(req, params.UserIDs)) {
		a.jsonError(res, errorNoViewPermission, logInfo.apiCallStart)
		return
	}

	ctx := req.Context()

	logInfo.queryStart = time.Now()
	iter, err := a.store.GetTimeInRangeData(ctx, params, false)
	if err != nil {
		logIndicatorError(logInfo, "Mongo Query", err)
		a.jsonError(res, errorRunningQuery, logInfo.apiCallStart)
	}
	logIndicatorSlowQuery(logInfo, "GetTimeInRangeData")

	defer iter.Close(req.Context())
	writeMongoJSONResponse(res, req, iter, logInfo)
}

func getParameterLimitValue(p deviceParameter) (mgdl float64, mmoll float64, unit string, err error) {
	limitValue, err := strconv.ParseFloat(p.Value, 64)
	if err != nil {
		return 0, 0, "", err
	}
	if p.Unit == unitMgdL {
		mgdl = limitValue
		mmoll, err = convertBG(mgdl, unitMgdL)
	} else if p.Unit == unitMmolL {
		mmoll = limitValue
		mgdl, err = convertBG(mgdl, unitMmolL)
	}
	return mgdl, mmoll, p.Unit, err
}

// Get the first / last data time for the specified user
func (a *API) getDataSummaryRangeV1(ctx context.Context, traceID string, userID string, numDays int64) (*store.Date, string, error) {
	timeIt(ctx, "getDataRange")
	defer timeEnd(ctx, "getDataRange")

	dates, err := a.store.GetDataRangeV1(ctx, traceID, userID)
	if err != nil {
		return nil, "", err
	}

	// If no data, we can stop here
	if dates == nil || dates.Start == "" || dates.End == "" {
		return nil, "", errors.New(errorNotfound.Code)
	}

	endRange, err := time.Parse(time.RFC3339Nano, dates.End)
	if err != nil {
		return nil, "", err
	}
	subDaysDuration := time.Duration(-24 * int64(time.Hour) * numDays)
	startTime := endRange.Add(subDaysDuration)

	return dates, startTime.Format(time.RFC3339), nil
}

// Get the current device parameters (to get hypo/hyper values)
func (a *API) getDataSummaryGlyLimits(ctx context.Context, traceID, userID string) (*summaryGlyLimits, error) {
	var err error
	var glyLimits *summaryGlyLimits = &summaryGlyLimits{
		hypoMgdl:   patientGlyHypoLimitMgdl,
		hyperMgdl:  patientGlyHyperLimitMgdl,
		hypoMmoll:  patientGlyHypoLimitMmoll,
		hyperMmoll: patientGlyHyperLimitMmoll,
		unit:       unitMgdL,
	}
	var iterPumpSettings mongo.StorageIterator

	timeIt(ctx, "getLastPumpSettings")
	iterPumpSettings, err = a.store.GetLatestPumpSettingsV1(ctx, traceID, userID)
	if err != nil {
		return nil, err
	}
	defer iterPumpSettings.Close(ctx)

	// This algorithm assume that the parameters units are the same for hypo and hyper
	if iterPumpSettings.Next(ctx) {
		var pumpSettings PumpSettings
		err = iterPumpSettings.Decode(&pumpSettings)
		if err == nil {
			haveHypo := false
			haveHyper := false
			for _, parameter := range pumpSettings.Payload.Parameters {
				if !haveHypo && parameter.Name == "PATIENT_GLY_HYPO_LIMIT" {
					haveHypo = true
					glyLimits.hypoMgdl, glyLimits.hypoMmoll, glyLimits.unit, err = getParameterLimitValue(parameter)
				} else if !haveHyper && parameter.Name == "PATIENT_GLY_HYPER_LIMIT" {
					haveHyper = true
					glyLimits.hyperMgdl, glyLimits.hyperMmoll, glyLimits.unit, err = getParameterLimitValue(parameter)
				}
				if err != nil {
					a.logger.Printf("{%s} - getDataSummaryGlyLimits for %s: Parse device parameter value error: %s (%v)", traceID, userID, err.Error(), parameter)
					break
				}
				if haveHypo && haveHyper {
					break
				}
			}
		} else {
			a.logger.Printf("{%s} - getDataSummaryGlyLimits for %s: Decode pump settings error: %s", traceID, userID, err.Error())
		}
	} // else: No pump settings (Use the default values)!
	timeEnd(ctx, "getLastPumpSettings")

	return glyLimits, nil
}

// Get the TIR & TBR percent, and the number of BG results used
func (a *API) getDataSummaryThresholds(ctx context.Context, traceID string, userID string, startTime string, glyLimits *summaryGlyLimits) (*summaryTresholds, error) {
	var totalNumBgValues int = 0
	var totalNumInRange int = 0
	var totalNumBelowRange int = 0
	var totalWrongUnit int = 0
	var tresholds *summaryTresholds = &summaryTresholds{}

	timeIt(ctx, "getBG")
	iterBG, err := a.store.GetCbgForSummaryV1(ctx, traceID, userID, startTime)
	if err != nil {
		return nil, err
	}
	defer iterBG.Close(ctx)
	timeEnd(ctx, "getBG")

	timeIt(ctx, "computeBG")
	for iterBG.Next(ctx) {
		bgValue := simplifiedBgDatum{}
		err = iterBG.Decode(&bgValue)
		if err != nil {
			return nil, err
		}
		totalNumBgValues++
		if bgValue.Unit == unitMgdL {
			if bgValue.Value < glyLimits.hypoMgdl {
				totalNumBelowRange++
			} else if bgValue.Value < glyLimits.hyperMgdl {
				totalNumInRange++
			}
		} else if bgValue.Unit == unitMmolL {
			if bgValue.Value < glyLimits.hypoMmoll {
				totalNumBelowRange++
			} else if bgValue.Value < glyLimits.hyperMmoll {
				totalNumInRange++
			}
		} else {
			totalWrongUnit++
		}
	}
	timeEnd(ctx, "computeBG")

	if totalNumBgValues > 0 {
		tresholds.totalNumBgValues = totalNumBgValues
		tresholds.percentTimeBelowRange = int(math.Round(100.0 * float64(totalNumBelowRange) / float64(totalNumBgValues)))
		tresholds.percentTimeInRange = int(math.Round(100.0 * float64(totalNumInRange) / float64(totalNumBgValues)))
	}
	if totalWrongUnit > 0 {
		a.logger.Printf("Found %d wrong unit in CBG data for user %s", totalWrongUnit, userID)
	}

	return tresholds, nil
}
