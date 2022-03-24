package data

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/mdblp/tide-whisperer-v2/v2/schema"
	"github.com/tidepool-org/go-common/clients/mongo"
)

type (
	// errorCounter to record only the first error to avoid spamming the log and takes too much time
	errorCounter struct {
		firstError error
		numErrors  int
	}
	// writeFromIter struct to pass to the function which write the http result from the mongo iterator for diabetes data
	writeFromIter struct {
		res    *httpResponseWriter
		iter   mongo.StorageIterator
		cbgs   []schema.CbgBucket
		basals []schema.BasalBucket
		// parametersHistory fetched from portal database
		parametersHistory map[string]interface{}
		// basalSecurityProfile
		basalSecurityProfile interface{}
		// uploadIDs encountered during the operation
		uploadIDs []string
		// writeCount the number of data written
		writeCount int
		// datum decode errors
		decode errorCounter
		// datum JSON marshall errors
		jsonError errorCounter
	}
	// SummaryResultV1 returned by the summary v1 route
	SummaryResultV1 struct {
		// The userID of this summary
		UserID string `json:"userId"`
		// First upload data date (ISO-8601 datetime)
		RangeStart string `json:"rangeStart"`
		// Last upload data date (ISO-8601 datetime)
		RangeEnd string `json:"rangeEnd"`
		// Number of days used to compute the TIR & TBR
		ComputeDays int `json:"computeDays"`
		// % of cbg/smbg in range (TIR)
		PercentTimeInRange int `json:"percentTimeInRange"`
		// % of cbg/smbg below range (TBR)
		PercentTimeBelowRange int `json:"percentTimeBelowRange"`
		// Number of bg values used to compute the TIR & TBR (if 0, the percent values are meaningless)
		NumBgValues int `json:"numBgValues"`
		// The Hypo limit used to compute TIR & TBR
		GlyHypoLimit float64 `json:"glyHypoLimit"`
		// The Hyper limit used to compute TIR & TBR
		GlyHyperLimit float64 `json:"glyHyperLimit"`
		// The unit of hypo/hyper values
		GlyUnit string `json:"glyUnit"`
	}
	simplifiedBgDatum struct {
		Value float64 `json:"value" bson:"value"`
		Unit  string  `json:"units" bson:"units"`
	}
	deviceParameter struct {
		Level int    `json:"level" bson:"level"`
		Name  string `json:"name" bson:"name"`
		Unit  string `json:"unit" bson:"unit"`
		Value string `json:"value" bson:"value"`
	}
	pumpSettingsPayload struct {
		Parameters []deviceParameter `json:"parameters" bson:"parameters"`
		// Uncomment & fill if needed:
		// Device     map[string]string `json:"device" bson:"device"`
		// CGM        map[string]string `json:"cgm" bson:"cgm"`
		// Pump       map[string]string `json:"pump" bson:"pump"`
	}
	// PumpSettings datum to get a specific device parameter
	PumpSettings struct {
		ID      string              `json:"id" bson:"id"`
		Type    string              `json:"type" bson:"type"`
		Time    string              `json:"time" bson:"time"`
		Payload pumpSettingsPayload `json:"payload" bson:"payload"`
	}
)

// Parameters level to keep in api response
var parameterLevelFilter = [...]int{1, 2}

func (a *API) setHandlesV1(prefix string, rtr *mux.Router) {
	// rtr.HandleFunc(prefix+"/status", a.requestLogger(a.getStatus)).Methods("GET")
	rtr.HandleFunc(prefix+"/range/{userID}", a.middlewareV1(a.getRangeV1, true, "userID")).Methods("GET")
	rtr.HandleFunc(prefix+"/summary/{userID}", a.middlewareV1(a.getDataSummaryV1, true, "userID")).Methods("GET")
	rtr.HandleFunc(prefix+"/data/{userID}", a.middlewareV1(a.getDataV1, true, "userID")).Methods("GET")
	rtr.HandleFunc(prefix+"/dataV2/{userID}", a.middlewareV1(a.getDataV2, true, "userID")).Methods("GET")
	rtr.HandleFunc(prefix+"/{.*}", a.middlewareV1(a.getNotFoundV1, false)).Methods("GET")
}

// getNotFoundV1 should it be version free?
func (a *API) getNotFoundV1(ctx context.Context, res *httpResponseWriter) error {
	res.WriteHeader(http.StatusNotFound)
	return nil
}

// @Summary Get the data dates range for a specific patient
//
// @Description Get the data dates range for a specific patient, returning a JSON array of two ISO 8601 strings: ["2021-01-01T10:00:00.430Z", "2021-02-10T10:18:27.430Z"]
//
// @ID tide-whisperer-api-v1-getrange
// @Produce json
// @Success 200 {array} string "Array of two ISO 8601 datetime"
// @Failure 400 {object} data.detailedError
// @Failure 403 {object} data.detailedError
// @Failure 404 {object} data.detailedError
// @Failure 500 {object} data.detailedError
// @Param userID path string true "The ID of the user to search data for"
// @Param x-tidepool-trace-session header string false "Trace session uuid" format(uuid)
// @Security TidepoolAuth
// @Router /v1/range/{userID} [get]
func (a *API) getRangeV1(ctx context.Context, res *httpResponseWriter) error {
	userID := res.VARS["userID"]

	dates, err := a.store.GetDataRangeV1(ctx, res.TraceID, userID)
	if err != nil {
		logError := &detailedError{
			Status:          errorRunningQuery.Status,
			Code:            errorRunningQuery.Code,
			Message:         errorRunningQuery.Message,
			InternalMessage: err.Error(),
		}
		return res.WriteError(logError)
	}

	if dates == nil || dates.Start == "" || dates.End == "" {
		return res.WriteError(&errorNotfound)
	}

	result := make([]string, 2)
	result[0] = dates.Start
	result[1] = dates.End

	jsonResult, err := json.Marshal(result)
	if err != nil {
		logError := &detailedError{
			Status:          http.StatusInternalServerError,
			Code:            "json_marshall_error",
			Message:         "internal server error",
			InternalMessage: err.Error(),
		}
		return res.WriteError(logError)
	}

	return res.Write(jsonResult)
}

// @Summary Get the data summary for a specific patient
//
// @Description Return summary information for a patient (TIR/TBR/...)
//
// @ID tide-whisperer-api-v1-getsummary
// @Produce json
//
// @Param userID path string true "The ID of the user"
//
// @Param days query integer false "The number of days used to compute TIR & TBR" default(1)
//
// @Success 200 {object} data.SummaryResultV1
// @Failure 400 {object} data.detailedError
// @Failure 403 {object} data.detailedError
// @Failure 404 {object} data.detailedError
// @Failure 500 {object} data.detailedError
//
// @Param x-tidepool-trace-session header string false "Trace session uuid" format(uuid)
// @Security TidepoolAuth
//
// @Router /v1/summary/{userID} [get]
func (a *API) getDataSummaryV1(ctx context.Context, res *httpResponseWriter) error {
	var err error
	var numDays int64 = 1

	userID := res.VARS["userID"]
	query := res.URL.Query()
	daysStr := query.Get("days")

	if daysStr != "" && daysStr != "1" {
		numDays, err = strconv.ParseInt(daysStr, 10, 8)
		if err != nil || numDays < 1 || numDays > 31 {
			logError := &detailedError{
				Status:          errorInvalidParameters.Status,
				Code:            errorInvalidParameters.Code,
				Message:         "invalid days parameter",
				InternalMessage: err.Error(),
			}
			return res.WriteError(logError)
		}
	}

	// First get the data range
	dates, startTime, err := a.getDataSummaryRangeV1(ctx, res.TraceID, userID, numDays)
	if err != nil {
		var logError *detailedError
		if err.Error() == errorNotfound.Code {
			logError = &errorNotfound
		} else {
			logError = &detailedError{
				Status:          errorRunningQuery.Status,
				Code:            errorRunningQuery.Code,
				Message:         errorRunningQuery.Message,
				InternalMessage: err.Error(),
			}
		}
		return res.WriteError(logError)
	}

	// Get the current device parameters (to get hypo/hyper values)
	glyLimits, err := a.getDataSummaryGlyLimits(ctx, res.TraceID, userID)
	if err != nil {
		logError := &detailedError{
			Status:          errorRunningQuery.Status,
			Code:            errorRunningQuery.Code,
			Message:         errorRunningQuery.Message,
			InternalMessage: err.Error(),
		}
		return res.WriteError(logError)
	}

	// Get the BG percent below/in range
	tresholds, err := a.getDataSummaryThresholds(ctx, res.TraceID, userID, startTime, glyLimits)
	if err != nil {
		logError := &detailedError{
			Status:          errorRunningQuery.Status,
			Code:            errorRunningQuery.Code,
			Message:         errorRunningQuery.Message,
			InternalMessage: err.Error(),
		}
		return res.WriteError(logError)
	}

	result := &SummaryResultV1{
		UserID:                userID,
		RangeStart:            dates.Start,
		RangeEnd:              dates.End,
		ComputeDays:           int(numDays),
		NumBgValues:           tresholds.totalNumBgValues,
		PercentTimeBelowRange: tresholds.percentTimeBelowRange,
		PercentTimeInRange:    tresholds.percentTimeInRange,
		GlyUnit:               glyLimits.unit,
	}
	if glyLimits.unit == unitMgdL {
		result.GlyHypoLimit = glyLimits.hypoMgdl
		result.GlyHyperLimit = glyLimits.hyperMgdl
	} else {
		result.GlyHypoLimit = glyLimits.hypoMmoll
		result.GlyHyperLimit = glyLimits.hyperMmoll
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		logError := &detailedError{
			Status:          http.StatusInternalServerError,
			Code:            "json_marshall_error",
			Message:         "internal server error",
			InternalMessage: err.Error(),
		}
		return res.WriteError(logError)
	}

	return res.Write(jsonResult)
}

// @Summary Get the data for a specific patient
//
// @Description Get the data for a specific patient, returning a JSON array of objects
//
// @ID tide-whisperer-api-v1-getdata
// @Produce json
//
// @Success 200 {array} string "Array of objects"
// @Failure 400 {object} data.detailedError
// @Failure 403 {object} data.detailedError
// @Failure 404 {object} data.detailedError
// @Failure 500 {object} data.detailedError
//
// @Param userID path string true "The ID of the user to search data for"
//
// @Param startDate query string false "ISO Date time (RFC3339) for search lower limit" format(date-time)
//
// @Param endDate query string false "ISO Date time (RFC3339) for search upper limit" format(date-time)
//
// @Param withPumpSettings query string false "true to include the pump settings in the results" format(boolean)
//
// @Param x-tidepool-trace-session header string false "Trace session uuid" format(uuid)
// @Security TidepoolAuth
//
// @Router /v1/data/{userID} [get]
func (a *API) getDataV1(ctx context.Context, res *httpResponseWriter) error {
	params, logError := getDataV1Params(res)
	if logError != nil {
		return res.WriteError(logError)
	}
	var err error
	// Mongo iterators
	var iterData mongo.StorageIterator
	var iterPumpSettings mongo.StorageIterator
	var iterUploads mongo.StorageIterator

	dates := &params.dates

	writeParams := &params.writer

	if params.includePumpSettings {
		// Fetch LastestPumpSettings
		iterPumpSettings, logError = a.getLatestPumpSettings(ctx, params.traceID, params.user, writeParams)
		if logError != nil {
			return res.WriteError(logError)
		}
		defer iterPumpSettings.Close(ctx)
	}

	// Fetch normal data:
	timeIt(ctx, "getData")
	iterData, err = a.store.GetDataV1(ctx, params.traceID, params.user, dates, []string{})
	if err != nil {
		logError := &detailedError{
			Status:          errorRunningQuery.Status,
			Code:            errorRunningQuery.Code,
			Message:         errorRunningQuery.Message,
			InternalMessage: err.Error(),
		}
		return res.WriteError(logError)
	}
	timeEnd(ctx, "getData")

	defer iterData.Close(ctx)
	return a.writeDataV1(
		ctx,
		res,
		params.includePumpSettings,
		iterPumpSettings,
		iterUploads,
		iterData,
		[]schema.CbgBucket{},
		[]schema.BasalBucket{},
		writeParams,
	)
}

// writeFromIterV1 Common code to write
func writeFromIterV1(ctx context.Context, p *writeFromIter) error {
	var err error

	iter := p.iter
	p.iter = nil

	for iter.Next(ctx) {
		var jsonDatum []byte
		var datum map[string]interface{}

		err = iter.Decode(&datum)
		if err != nil {
			p.decode.numErrors++
			if p.decode.firstError == nil {
				p.decode.firstError = err
			}
			continue
		}
		if len(datum) > 0 {
			datumID, haveID := datum["id"].(string)
			if !haveID {
				// Ignore datum with no id, should never happend
				continue
			}
			datumType, haveType := datum["type"].(string)
			if !haveType {
				// Ignore datum with no type, should never happend
				continue
			}
			uploadID, haveUploadID := datum["uploadId"].(string)
			if !haveUploadID {
				// No upload ID, abnormal situation
				continue
			}
			if datumType == "deviceEvent" {
				datumSubType, haveSubType := datum["subType"].(string)
				if haveSubType && datumSubType == "deviceParameter" {
					datumLevel, haveLevel := datum["level"]
					if haveLevel {
						intLevel, err := strconv.Atoi(fmt.Sprintf("%v", datumLevel))
						if err == nil && !containsInt(parameterLevelFilter[:], intLevel) {
							continue
						}
					}
				}
			}
			// Record the uploadID
			if !(datumType == "upload" && uploadID == datumID) {
				if !contains(p.uploadIDs, uploadID) {
					p.uploadIDs = append(p.uploadIDs, uploadID)
				}
			}
			
			if datumType == "pumpSettings" && (p.parametersHistory != nil || p.basalSecurityProfile != nil) {
				payload := datum["payload"].(map[string]interface{})
				
				// Add the parameter history to the pump settings
				if p.parametersHistory != nil {
					payload["history"] = p.parametersHistory["history"]
				}

				// Add the basal security profile to the pump settings
				if p.basalSecurityProfile != nil {
					payload["basalsecurityprofile"] = p.basalSecurityProfile
				}

				datum["payload"] = payload
			}

			// Create the JSON string for this datum
			if jsonDatum, err = json.Marshal(datum); err != nil {
				if p.jsonError.firstError == nil {
					p.jsonError.firstError = err
				}
				p.jsonError.numErrors++
				continue
			}

			if p.writeCount > 0 {
				// Add the coma and line return (for readability)
				err = p.res.WriteString(",\n")
				if err != nil {
					return err
				}
			}
			err = p.res.Write(jsonDatum)
			if err != nil {
				return err
			}
			p.writeCount++
		} // else ignore
	}
	return nil
}
