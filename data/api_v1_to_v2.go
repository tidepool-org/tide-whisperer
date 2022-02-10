package data

import (
	"context"
	"sync"
	"time"

	"github.com/mdblp/tide-whisperer-v2/schema"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/tide-whisperer/store"
)

var dataFromStoreTimer = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:      "data_from_store_time",
	Help:      "A histogram for getDataFromStore execution time (ms)",
	Buckets:   prometheus.LinearBuckets(20, 20, 300),
	Subsystem: "tidewhisperer",
	Namespace: "dblp",
})

var dataFromTideV2Timer = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:      "data_from_tidev2_time",
	Help:      "A histogram for dataFromTideV2Timer execution time (ms)",
	Buckets:   prometheus.LinearBuckets(20, 20, 300),
	Subsystem: "tidewhisperer",
	Namespace: "dblp",
})

func (a *API) getDataFromStore(ctx context.Context, wg *sync.WaitGroup, traceID string, userID string, dates *store.Date, excludes []string, iterData chan mongo.StorageIterator, logError chan *detailedError) {
	defer wg.Done()
	start := time.Now()
	data, err := a.store.GetDataV1(ctx, traceID, userID, dates, excludes)
	if err != nil {
		logError <- &detailedError{
			Status:          errorRunningQuery.Status,
			Code:            errorRunningQuery.Code,
			Message:         errorRunningQuery.Message,
			InternalMessage: err.Error(),
		}
		iterData <- nil
	} else {
		logError <- nil
		iterData <- data
	}
	elapsed_time := time.Now().Sub(start).Milliseconds()
	dataFromStoreTimer.Observe(float64(elapsed_time))
}
func (a *API) getCbgFromTideV2(ctx context.Context, wg *sync.WaitGroup, userID string, sessionToken string, dates *store.Date, tideV2Data chan []schema.CbgBucket, logErrorDataV2 chan *detailedError) {
	defer wg.Done()
	start := time.Now()
	data, err := a.tideV2Client.GetCbgV2(userID, sessionToken, dates.Start, dates.End)
	if err != nil {
		logErrorDataV2 <- &detailedError{
			Status:          errorTideV2Http.Status,
			Code:            errorTideV2Http.Code,
			Message:         errorTideV2Http.Message,
			InternalMessage: err.Error(),
		}
		tideV2Data <- nil
	} else {
		tideV2Data <- data
		logErrorDataV2 <- nil
	}
	elapsed_time := time.Now().Sub(start).Milliseconds()
	dataFromTideV2Timer.Observe(float64(elapsed_time))
}

func (a *API) getBasalFromTideV2(ctx context.Context, wg *sync.WaitGroup, userID string, sessionToken string, dates *store.Date, v2Data chan []schema.BasalBucket, logErrorDataV2 chan *detailedError) {
	defer wg.Done()
	start := time.Now()
	data, err := a.tideV2Client.GetBasalV2(userID, sessionToken, dates.Start, dates.End)
	if err != nil {
		logErrorDataV2 <- &detailedError{
			Status:          errorTideV2Http.Status,
			Code:            errorTideV2Http.Code,
			Message:         errorTideV2Http.Message,
			InternalMessage: err.Error(),
		}
		v2Data <- nil
	} else {
		v2Data <- data
		logErrorDataV2 <- nil
	}
	elapsed_time := time.Now().Sub(start).Milliseconds()
	dataFromTideV2Timer.Observe(float64(elapsed_time))
}

// @Summary Get the data for a specific patient using new bucket api
//
// @Description Get the data for a specific patient, returning a JSON array of objects
//
// @ID tide-whisperer-api-v1V2-getdata
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
// @Param cbgBucket query string false "no parameter or not equal to true to get cbg from buckets" format(boolean)
//
// @Param basalBucket query string false "true to get basals from buckets, if the parameter is not there or not equal to true the basals are from deviceData" format(boolean)
//
// @Param x-tidepool-trace-session header string false "Trace session uuid" format(uuid)
// @Security TidepoolAuth
//
// @Router /v1/dataV2/{userID} [get]
func (a *API) getDataV2(ctx context.Context, res *httpResponseWriter) error {
	params, logError := getDataV1Params(res)
	if logError != nil {
		return res.WriteError(logError)
	}
	// Mongo iterators
	var iterPumpSettings mongo.StorageIterator
	var iterUploads mongo.StorageIterator
	var chanApiCbgs chan []schema.CbgBucket
	var chanApiBasals chan []schema.BasalBucket
	var chanApiCbgError, chanApiBasalError chan *detailedError
	var logErrorDataV2 *detailedError
	var wg sync.WaitGroup

	var exclusions = map[string]string{
		"cbgBucket":   "cbg",
		"basalBucket": "basal",
	}
	var exclusionList []string
	groups := 0
	for key, value := range params.source {
		if value {
			groups++
			if _, ok := exclusions[key]; ok {
				exclusionList = append(exclusionList, exclusions[key])
			}
		}
	}
	dates := &params.dates

	writeParams := &params.writer

	if params.includePumpSettings {
		iterPumpSettings, logError = a.getLatestPumpSettings(ctx, res.TraceID, params.user, writeParams)
		if logError != nil {
			return res.WriteError(logError)
		}
		defer iterPumpSettings.Close(ctx)
	}

	// Fetch data from store and V2 API (for cbg)
	sessionToken := res.Header.Get("x-tidepool-session-token")
	chanStoreError := make(chan *detailedError, 1)
	defer close(chanStoreError)
	chanMongoIter := make(chan mongo.StorageIterator, 1)
	defer close(chanMongoIter)

	// Parallel routines
	wg.Add(groups)
	go a.getDataFromStore(ctx, &wg, res.TraceID, params.user, dates, exclusionList, chanMongoIter, chanStoreError)

	if params.source["cbgBucket"] {
		chanApiCbgs = make(chan []schema.CbgBucket, 1)
		defer close(chanApiCbgs)
		chanApiCbgError = make(chan *detailedError, 1)
		defer close(chanApiCbgError)
		go a.getCbgFromTideV2(ctx, &wg, params.user, sessionToken, dates, chanApiCbgs, chanApiCbgError)
	}
	if params.source["basalBucket"] {
		chanApiBasals = make(chan []schema.BasalBucket, 1)
		defer close(chanApiBasals)
		chanApiBasalError = make(chan *detailedError, 1)
		defer close(chanApiBasalError)
		go a.getBasalFromTideV2(ctx, &wg, params.user, sessionToken, dates, chanApiBasals, chanApiBasalError)
	}

	wg.Wait()

	logErrorStore := <-chanStoreError
	if logErrorStore != nil {
		return res.WriteError(logErrorStore)
	}
	if params.source["cbgBucket"] {
		logErrorDataV2 = <-chanApiCbgError
		if logErrorDataV2 != nil {
			return res.WriteError(logErrorDataV2)
		}
	}
	if params.source["basalBucket"] {
		logErrorDataV2 = <-chanApiBasalError
		if logErrorDataV2 != nil {
			return res.WriteError(logErrorDataV2)
		}
	}
	iterData := <-chanMongoIter
	var Cbgs []schema.CbgBucket
	if params.source["cbgBucket"] {
		Cbgs = <-chanApiCbgs
	}
	var Basals []schema.BasalBucket
	if params.source["basalBucket"] {
		Basals = <-chanApiBasals
	}
	defer iterData.Close(ctx)

	return a.writeDataV1(
		ctx,
		res,
		params.includePumpSettings,
		iterPumpSettings,
		iterUploads,
		iterData,
		Cbgs,
		Basals,
		writeParams,
	)
}
