package data

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/mdblp/tide-whisperer-v2/schema"
	"github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/tide-whisperer/store"
)

type (
	apiDataParams struct {
		dates               store.Date
		user                string
		traceID             string
		includePumpSettings bool
		writer              writeFromIter
	}
)

func getDataV1Params(res *httpResponseWriter) (*apiDataParams, *detailedError) {
	var err error
	// Mongo iterators
	userID := res.VARS["userID"]

	query := res.URL.Query()
	startDate := query.Get("startDate")
	endDate := query.Get("endDate")
	withPumpSettings := query.Get("withPumpSettings") == "true"

	// Check startDate & endDate parameter
	if startDate != "" || endDate != "" {
		var logError *detailedError
		var startTime time.Time
		var endTime time.Time
		var timeRange int64 = 1 // endDate - startDate in seconds, initialized to 1 to avoid trigger an error, see below

		if startDate != "" {
			startTime, err = time.Parse(time.RFC3339Nano, startDate)
		}
		if err == nil && endDate != "" {
			endTime, err = time.Parse(time.RFC3339Nano, endDate)
		}

		if err == nil && startDate != "" && endDate != "" {
			timeRange = endTime.Unix() - startTime.Unix()
		}

		if timeRange > 0 {
			// Make an estimated guessed about the amount of data we need to send
			// to help our buffer, since we may send ten or so megabytes of JSON
			// I saw ~ 1.15 byte per second in my test
			// fmt.Printf("Grow: %d * 1.15 -> %d\n", timeRange, int(math.Round(float64(timeRange)*1.15)))
			res.Grow(int(math.Round(float64(timeRange) * 1.15)))
		} else {
			err = fmt.Errorf("startDate is after endDate")
		}

		if err != nil {
			logError = &detailedError{
				Status:          errorInvalidParameters.Status,
				Code:            errorInvalidParameters.Code,
				Message:         errorInvalidParameters.Message,
				InternalMessage: err.Error(),
			}
			return nil, logError
		}
	}
	params := apiDataParams{
		dates: store.Date{
			Start: startDate,
			End:   endDate,
		},
		user:                userID,
		traceID:             res.TraceID,
		includePumpSettings: withPumpSettings,
		writer: writeFromIter{
			res:       res,
			uploadIDs: make([]string, 0, 16),
		},
	}
	return &params, nil

}

func (a *API) getLatestPumpSettings(ctx context.Context, traceID string, userID string, writer *writeFromIter) (mongo.StorageIterator, *detailedError) {
	// Initial query to fetch for this user, the client wants the
	// latest pumpSettings
	timeIt(ctx, "getLastPumpSettings")
	iterPumpSettings, err := a.store.GetLatestPumpSettingsV1(ctx, traceID, userID)
	if err != nil {
		logError := &detailedError{
			Status:          errorRunningQuery.Status,
			Code:            errorRunningQuery.Code,
			Message:         errorRunningQuery.Message,
			InternalMessage: err.Error(),
		}
		return nil, logError
	}
	timeEnd(ctx, "getLastPumpSettings")

	// Fetch parameters history from portal:
	timeIt(ctx, "getParamHistory")
	writer.parametersHistory, err = a.store.GetDiabeloopParametersHistory(ctx, userID, parameterLevelFilter[:])
	if err != nil {
		// Just log the problem, don't crash the query
		writer.parametersHistory = nil
		a.logger.Printf("{%s} - {GetDiabeloopParametersHistory:\"%s\"}", traceID, err)
	}
	timeEnd(ctx, "getParamHistory")
	return iterPumpSettings, nil
}

func (a *API) writeDataV1(
	ctx context.Context,
	res *httpResponseWriter,
	includePumpSettings bool,
	iterPumpSettings mongo.StorageIterator,
	iterUploads mongo.StorageIterator,
	iterData mongo.StorageIterator,
	tideV2Data []schema.CbgBucket,
	writeParams *writeFromIter,
) error {
	timeIt(ctx, "writeData")
	defer timeEnd(ctx, "writeData")
	// We return a JSON array, first charater is: '['
	err := res.WriteString("[\n")
	if err != nil {
		return err
	}

	if includePumpSettings && iterPumpSettings != nil {
		writeParams.iter = iterPumpSettings
		err = writeFromIterV1(ctx, writeParams)
		if err != nil {
			return err
		}
	}

	timeIt(ctx, "writeDataMain")
	writeParams.iter = iterData
	err = writeFromIterV1(ctx, writeParams)
	if err != nil {
		return err
	}
	timeEnd(ctx, "writeDataMain")

	if len(tideV2Data) > 0 {
		timeIt(ctx, "writeDataV2")
		writeParams.dataV2 = tideV2Data
		err = writeDataV2(ctx, writeParams)
		if err != nil {
			return err
		}
		timeEnd(ctx, "writeDataV2")
	}
	// Fetch uploads
	if len(writeParams.uploadIDs) > 0 {
		timeIt(ctx, "getUploads")
		iterUploads, err = a.store.GetDataFromIDV1(ctx, res.TraceID, writeParams.uploadIDs)
		if err != nil {
			// Just log the problem, don't crash the query
			writeParams.parametersHistory = nil
			a.logger.Printf("{%s} - {GetDataFromIDV1:\"%s\"}", res.TraceID, err)
		} else {
			defer iterUploads.Close(ctx)
			writeParams.iter = iterUploads
			err = writeFromIterV1(ctx, writeParams)
			if err != nil {
				timeEnd(ctx, "getUploads")
				return err
			}
		}
		timeEnd(ctx, "getUploads")
	}

	// Silently failed theses error to the client, but record them to the log
	if writeParams.decode.firstError != nil {
		a.logger.Printf("{%s} - {nErrors:%d,MongoDecode:\"%s\"}", res.TraceID, writeParams.decode.numErrors, writeParams.decode.firstError)
	}
	if writeParams.jsonError.firstError != nil {
		a.logger.Printf("{%s} - {nErrors:%d,jsonMarshall:\"%s\"}", res.TraceID, writeParams.jsonError.numErrors, writeParams.jsonError.firstError)
	}

	// Last JSON array charater:
	return res.WriteString("]\n")
}

// Mapping V2 Bucket schema to expected V1 schema + write to output
func writeDataV2(ctx context.Context, p *writeFromIter) error {
	var err error
	for _, bucket := range p.dataV2 {
		for i, sample := range bucket.Samples {
			datum := make(map[string]interface{})
			// Building a fake id (bucket.Id/range index)
			datum["id"] = fmt.Sprintf("%s_%d", bucket.Id, i)
			datum["type"] = "cbg"
			datum["time"] = sample.Timestamp
			datum["timezone"] = sample.Timezone
			datum["units"] = sample.Units
			datum["value"] = sample.Value
			jsonDatum, err := json.Marshal(datum)
			if err != nil {
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
		}
	}
	return err
}
