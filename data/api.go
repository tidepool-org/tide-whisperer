package data

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/gorilla/mux"
	"github.com/tidepool-org/go-common/clients"
	"github.com/tidepool-org/go-common/clients/shoreline"
	"github.com/tidepool-org/go-common/clients/status"
	"github.com/tidepool-org/tide-whisperer/auth"
	"github.com/tidepool-org/tide-whisperer/store"

	"strconv"
	"strings"
)

type (
	API struct {
		store           store.Storage
		shorelineClient shoreline.Client
		authClient      auth.ClientInterface
		perms           clients.Gatekeeper
		schemaVersion   store.SchemaVersion
	}

	varsHandler func(http.ResponseWriter, *http.Request, map[string]string)

	// so we can wrap and marshal the detailed error
	detailedError struct {
		Status int `json:"status"`
		//provided to user so that we can better track down issues
		ID              string `json:"id"`
		Code            string `json:"code"`
		Message         string `json:"message"`
		InternalMessage string `json:"-"` //used only for logging so we don't want to serialize it out
	}

	//generic type as device data can be comprised of many things
	deviceData map[string]interface{}
)

const (
	//api logging prefix
	DataAPIPrefix             = "api/data "
	medtronicLoopBoundaryDate = "2017-09-01"
	slowQueryDuration         = 0.1 // seconds
)

var (
	errorStatusCheck       = detailedError{Status: http.StatusInternalServerError, Code: "data_status_check", Message: "checking of the status endpoint showed an error"}
	errorNoViewPermission  = detailedError{Status: http.StatusForbidden, Code: "data_cant_view", Message: "user is not authorized to view data"}
	errorNoPermissions     = detailedError{Status: http.StatusInternalServerError, Code: "data_perms_error", Message: "error finding permissions for user"}
	errorRunningQuery      = detailedError{Status: http.StatusInternalServerError, Code: "data_store_error", Message: "internal server error"}
	errorLoadingEvents     = detailedError{Status: http.StatusInternalServerError, Code: "data_marshal_error", Message: "internal server error"}
	errorInvalidParameters = detailedError{Status: http.StatusInternalServerError, Code: "invalid_parameters", Message: "one or more parameters are invalid"}
)

func InitApi(storage store.Storage, shoreline shoreline.Client, auth auth.ClientInterface, permsClient clients.Gatekeeper, schemaV store.SchemaVersion) *API {
	return &API{
		store:           storage,
		shorelineClient: shoreline,
		authClient:      auth,
		perms:           permsClient,
		schemaVersion:   schemaV,
	}
}

func (a *API) SetHandlers(prefix string, rtr *mux.Router) {
	/*
	 Gloo performs autodiscovery by trying certain paths,
	 including /swagger, /v1, and v2.  Unfortunately, tide-whisperer
	 interprets those paths as userids.  To avoid misleading
	 error messages, we catch these calls and return an error
	 code.
	*/
	rtr.HandleFunc("/swagger", a.Get501).Methods("GET")
	rtr.HandleFunc("/v1", a.Get501).Methods("GET")
	rtr.HandleFunc("/v2", a.Get501).Methods("GET")
	rtr.HandleFunc("/status", a.GetStatus).Methods("GET")
	rtr.HandleFunc("/compute/tir", a.GetTimeInRange).Methods("GET")
	rtr.Handle("/{userID}", varsHandler(a.GetData)).Methods("GET")
}

func (h varsHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	h(res, req, vars)
}

func (a *API) Get501(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(501)
	return
}

// @Summary Get the api status
// @Description Get the api status
// @ID tide-whisperer-api-getstatus
// @Produce json
// @Success 200 {object} status.ApiStatus
// @Failure 500 {object} status.ApiStatus
// @Router /status [get]
func (a *API) GetStatus(res http.ResponseWriter, req *http.Request) {
	start := time.Now()
	var s status.ApiStatus
	if err := a.store.Ping(); err != nil {
		errorLog := errorStatusCheck.setInternalMessage(err)
		logError(&errorLog, start)
		s = status.NewApiStatus(errorLog.Status, err.Error())
	} else {
		s = status.NewApiStatus(http.StatusOK, "OK")
	}
	if jsonDetails, err := json.Marshal(s); err != nil {
		jsonError(res, errorLoadingEvents.setInternalMessage(err), start)
	} else {
		res.Header().Add("content-type", "application/json")
		res.WriteHeader(s.Status.Code)
		res.Write(jsonDetails)
	}
	return
}

// @Summary Get device/health data for a user based on a set of parameters
// @Description Get device/health data for a user based on a set of parameters
// @ID tide-whisperer-api-getdata
// @Produce json
// @Success 200 {array} deviceData "List of user data objects"
// @Failure 500 {object} data.detailedError
// @Failure 403 {object} data.detailedError
// @Param userID path string true "The ID of the user to search data for"
// @Param type query []string false "Type of data to search for - can be a list of types separated by commas" collectionFormat(csv)
// @Param subType query []string false "Subtype of data to search for - can be a list of subtypes separated by commas" collectionFormat(csv)
// @Param deviceId query string false "ID of the device to search data for"
// @Param uploadId query string false "ID of the upload to search data for"
// @Param startDate query string false "ISO Date time for search lower limit" format(date-time)
// @Param endDate query string false "ISO Date time for search upper limit" format(date-time)
// @Param carelink query bool false "N/A - Unused for diabeloop devices"
// @Param dexcom query bool false "N/A - Unused for diabeloop devices"
// @Param medtronic query bool false "N/A - Unused for diabeloop devices"
// @Param latest query bool false "To return only the most recent results for each `type` matching the results filtered by the other query parameters"
// @Security TidepoolAuth
// @Router /{userID} [get]
func (a *API) GetData(res http.ResponseWriter, req *http.Request, vars map[string]string) {

	start := time.Now()

	ctx := req.Context()

	queryValues := url.Values{":userID": []string{vars["userID"]}}
	for k, v := range req.URL.Query() {
		queryValues[k] = v
	}

	queryParams, err := a.parseDataParams(ctx, queryValues)

	if err != nil {
		log.Println(DataAPIPrefix, fmt.Sprintf("Error parsing query params: %s", err))
		jsonError(res, errorInvalidParameters, start)
		return
	}

	userIDs := []string{queryParams.UserID}
	if !(a.isAuthorized(req, userIDs)) {
		log.Printf("userid %v", queryParams.UserID)
		jsonError(res, errorNoViewPermission, start)
		return
	}

	requestID := newRequestID()
	queryStart := time.Now()
	if _, ok := req.URL.Query()["carelink"]; !ok {
		if hasMedtronicDirectData, medtronicErr := a.store.HasMedtronicDirectData(ctx, queryParams.UserID); medtronicErr != nil {
			log.Printf("%s request %s user %s HasMedtronicDirectData returned error: %s", DataAPIPrefix, requestID, queryParams.UserID, medtronicErr)
			jsonError(res, errorRunningQuery, start)
			return
		} else if !hasMedtronicDirectData {
			queryParams.Carelink = true
		}
		if queryDuration := time.Now().Sub(queryStart).Seconds(); queryDuration > slowQueryDuration {
			// XXX replace with metrics
			//log.Printf("%s request %s user %s HasMedtronicDirectData took %.3fs", DataAPIPrefix, requestID, userID, queryDuration)
		}
		queryStart = time.Now()
	}
	if !queryParams.Dexcom {
		dexcomDataSource, dexcomErr := a.store.GetDexcomDataSource(ctx, queryParams.UserID)
		if dexcomErr != nil {
			log.Printf("%s request %s user %s GetDexcomDataSource returned error: %s", DataAPIPrefix, requestID, queryParams.UserID, dexcomErr)
			jsonError(res, errorRunningQuery, start)
			return
		}
		queryParams.DexcomDataSource = dexcomDataSource

		if queryDuration := time.Now().Sub(queryStart).Seconds(); queryDuration > slowQueryDuration {
			log.Printf("%s SlowQuery: request %s user %s GetDexcomDataSource took %.3fs", DataAPIPrefix, requestID, queryParams.UserID, queryDuration)
		}
		queryStart = time.Now()
	}
	if _, ok := req.URL.Query()["medtronic"]; !ok {
		hasMedtronicLoopData, medtronicErr := a.store.HasMedtronicLoopDataAfter(ctx, queryParams.UserID, medtronicLoopBoundaryDate)
		if medtronicErr != nil {
			log.Printf("%s request %s user %s HasMedtronicLoopDataAfter returned error: %s", DataAPIPrefix, requestID, queryParams.UserID, medtronicErr)
			jsonError(res, errorRunningQuery, start)
			return
		}
		if !hasMedtronicLoopData {
			queryParams.Medtronic = true
		}
		if queryDuration := time.Now().Sub(queryStart).Seconds(); queryDuration > slowQueryDuration {
			log.Printf("%s SlowQuery: request %s user %s HasMedtronicLoopDataAfter took %.3fs", DataAPIPrefix, requestID, queryParams.UserID, queryDuration)
		}
		queryStart = time.Now()
	}
	if !queryParams.Medtronic {
		medtronicUploadIds, medtronicErr := a.store.GetLoopableMedtronicDirectUploadIdsAfter(ctx, queryParams.UserID, medtronicLoopBoundaryDate)
		if medtronicErr != nil {
			log.Printf("%s request %s user %s GetLoopableMedtronicDirectUploadIdsAfter returned error: %s", DataAPIPrefix, requestID, queryParams.UserID, medtronicErr)
			jsonError(res, errorRunningQuery, start)
			return
		}
		queryParams.MedtronicDate = medtronicLoopBoundaryDate
		queryParams.MedtronicUploadIds = medtronicUploadIds

		if queryDuration := time.Now().Sub(queryStart).Seconds(); queryDuration > slowQueryDuration {
			// XXX replace with metrics
			//log.Printf("%s request %s user %s GetLoopableMedtronicDirectUploadIdsAfter took %.3fs", DataAPIPrefix, requestID, userID, queryDuration)
		}
		queryStart = time.Now()
	}

	iter, err := a.store.GetDeviceData(ctx, queryParams)
	if err != nil {
		log.Printf("%s request %s user %s Mongo Query returned error: %s", DataAPIPrefix, requestID, queryParams.UserID, err)
	}

	defer iter.Close(ctx)

	var parametersHistory map[string]interface{}
	var parametersHistoryErr error
	if store.InArray("pumpSettings", queryParams.Types) || (len(queryParams.Types) == 1 && queryParams.Types[0] == "") {
		log.Printf("Calling GetDiabeloopParametersHistory")

		if parametersHistory, parametersHistoryErr = a.store.GetDiabeloopParametersHistory(ctx, queryParams.UserID, queryParams.LevelFilter); parametersHistoryErr != nil {
			log.Printf("%s request %s user %s GetDiabeloopParametersHistory returned error: %s", DataAPIPrefix, requestID, queryParams.UserID, parametersHistoryErr)
			jsonError(res, errorRunningQuery, start)
			return
		}
	}
	var writeCount int

	res.Header().Add("Content-Type", "application/json")

	res.Write([]byte("["))

	for iter.Next(ctx) {
		var results map[string]interface{}
		err := iter.Decode(&results)
		if err != nil {
			log.Printf("%s request %s user %s Mongo Decode returned error: %s", DataAPIPrefix, requestID, queryParams.UserID, err)
		}

		if queryParams.Latest {
			// If we're using the `latest` parameter, then we ran an `$aggregate` query to get only the latest data.
			// Since we use Mongo 3.2, we can't use the $replaceRoot function, so we need to manually extract the
			// latest subdocument here. When we move to MongoDB 3.4+ and can use $replaceRoot, we can get rid of this
			// conditional block. We'd also need to fix the corresponding code in `store.go`
			results = results["latest_doc"].(map[string]interface{})
		}
		if len(results) > 0 {
			if results["type"].(string) == "pumpSettings" && parametersHistory != nil {
				payload := results["payload"].(map[string]interface{})
				payload["history"] = parametersHistory["history"]
				results["payload"] = payload
			}
			if bytes, err := json.Marshal(results); err != nil {
				log.Printf("%s request %s user %s Marshal returned error: %s", DataAPIPrefix, requestID, queryParams.UserID, err)
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

	if queryDuration := time.Now().Sub(queryStart).Seconds(); queryDuration > slowQueryDuration {
		// XXX use metrics
		//log.Printf("%s request %s user %s GetDeviceData took %.3fs", DataAPIPrefix, requestID, userID, queryDuration)
	}
	log.Printf("%s request %s user %s took %.3fs returned %d records", DataAPIPrefix, requestID, queryParams.UserID, time.Now().Sub(start).Seconds(), writeCount)
}

func cleanDateString(dateString string) (string, error) {
	if dateString == "" {
		return "", nil
	}
	date, err := time.Parse(time.RFC3339Nano, dateString)
	if err != nil {
		return "", err
	}
	return date.Format(time.RFC3339Nano), nil
}
func (a *API) parseDataParams(ctx context.Context, q url.Values) (*store.Params, error) {
	var strPrms = make(map[string]string)
	for _, dateField := range []string{"startDate", "endDate"} {
		dateStr, err := cleanDateString(q.Get(dateField))
		if err != nil {
			return nil, err
		}
		strPrms[dateField] = dateStr
	}
	var boolPrms = make(map[string]bool)
	for _, boolField := range []string{"carelink", "dexcom", "latest", "medtronic"} {
		boolPrms[boolField] = false
		if values, ok := q[boolField]; ok {
			if len(values) < 1 {
				return nil, fmt.Errorf("%s parameter not valid", boolField)
			}
			prmBool, err := strconv.ParseBool(values[len(values)-1])
			if err != nil {
				return nil, fmt.Errorf("%s parameter not valid", boolField)
			}
			boolPrms[boolField] = prmBool
		}
	}
	// get Device model
	var device string
	var deviceErr error
	var UserID = q.Get(":userID")
	if device, deviceErr = a.store.GetDeviceModel(ctx, UserID); deviceErr != nil {
		log.Printf("Error in GetDeviceModel for user %s. Error: %s", UserID, deviceErr)
	}

	LevelFilter := make([]int, 1)
	LevelFilter = append(LevelFilter, 1)
	if device == "DBLHU" {
		LevelFilter = append(LevelFilter, 2)
		LevelFilter = append(LevelFilter, 3)
	}

	p := &store.Params{
		UserID:   q.Get(":userID"),
		DeviceID: q.Get("deviceId"),
		UploadID: q.Get("uploadId"),
		//the query params for type and subtype can contain multiple values seperated
		//by a comma e.g. "type=smbg,cbg" so split them out into an array of values
		Types:         strings.Split(q.Get("type"), ","),
		SubTypes:      strings.Split(q.Get("subType"), ","),
		Date:          store.Date{strPrms["startDate"], strPrms["endDate"]},
		SchemaVersion: &a.schemaVersion,
		Carelink:      boolPrms["carelink"],
		Dexcom:        boolPrms["dexcom"],
		Latest:        boolPrms["latest"],
		Medtronic:     boolPrms["medtronic"],
		LevelFilter:   LevelFilter,
	}
	return p, nil
}

//log error detail and write as application/json
func jsonError(res http.ResponseWriter, err detailedError, startedAt time.Time) {
	logError(&err, startedAt)
	jsonErr, _ := json.Marshal(err)

	res.Header().Add("content-type", "application/json")
	res.WriteHeader(err.Status)
	res.Write(jsonErr)
}

func logError(err *detailedError, startedAt time.Time) {
	err.ID = uuid.New().String()
	log.Println(DataAPIPrefix, fmt.Sprintf("[%s][%s] failed after [%.3f]secs with error [%s][%s] ", err.ID, err.Code, time.Now().Sub(startedAt).Seconds(), err.Message, err.InternalMessage))
}

//set the internal message that we will use for logging
func (d detailedError) setInternalMessage(internal error) detailedError {
	d.InternalMessage = internal.Error()
	return d
}

func (a *API) getTokenData(req *http.Request) *shoreline.TokenData {
	var td *shoreline.TokenData
	if sessionToken := req.Header.Get("x-tidepool-session-token"); sessionToken != "" {
		td = a.shorelineClient.CheckToken(sessionToken)
	} else if restrictedTokens, found := req.URL.Query()["restricted_token"]; found && len(restrictedTokens) == 1 {
		restrictedToken, restrictedTokenErr := a.authClient.GetRestrictedToken(req.Context(), restrictedTokens[0])
		if restrictedTokenErr == nil && restrictedToken != nil && restrictedToken.Authenticates(req) {
			td = &shoreline.TokenData{UserID: restrictedToken.UserID}
		}
	}

	return td
}
func (a *API) userCanViewData(authenticatedUserID string, targetUserIDs []string) bool {
	if len(targetUserIDs) == 1 {
		targetUserID := targetUserIDs[0]
		log.Printf("authenticatedUserID::%v / targetUserID::%v", authenticatedUserID, targetUserID)
		if authenticatedUserID == targetUserID {
			return true
		}
		perms, err := a.perms.UserInGroup(authenticatedUserID, targetUserID)
		if err != nil {
			log.Println(DataAPIPrefix, "Error looking up user in group", err)
			return false
		}
		return !(perms["root"] == nil && perms["view"] == nil)
	}
	userPerms, err := a.perms.GroupsForUser(authenticatedUserID)
	if err != nil {
		log.Println(DataAPIPrefix, "Error looking up users in group", err)
		return false
	}
	authorized := true
	for _, uid := range targetUserIDs {
		if _, exists := userPerms[uid]; !exists {
			authorized = false
			break
		}
	}

	return authorized
}
func (a *API) isAuthorized(req *http.Request, targetUserIDs []string) bool {
	td := a.getTokenData(req)
	if td == nil {
		return false
	}
	if td.IsServer {
		return true
	}
	return a.userCanViewData(td.UserID, targetUserIDs)
}

func newRequestID() string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes) // In case of failure, do not fail request, just use default bytes (zero)
	return hex.EncodeToString(bytes)
}
