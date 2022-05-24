package data

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/gorilla/mux"
	"github.com/mdblp/go-common/clients/auth"
	tideV2Client "github.com/mdblp/tide-whisperer-v2/v2/client/tidewhisperer"
	"github.com/tidepool-org/go-common/clients/opa"
	"github.com/tidepool-org/go-common/clients/status"
	"github.com/tidepool-org/tide-whisperer/store"
)

type (
	// API struct for tide-whisperer
	API struct {
		store         store.Storage
		authClient    auth.ClientInterface
		perms         opa.Client
		schemaVersion store.SchemaVersion
		logger        *log.Logger
		tideV2Client  tideV2Client.ClientInterface
	}

	varsHandler func(http.ResponseWriter, *http.Request, map[string]string)

	// so we can wrap and marshal the detailed error
	detailedError struct {
		Status          int    `json:"status"`  // Http status code
		ID              string `json:"id"`      // provided to user so that we can better track down issues
		Code            string `json:"code"`    // Code which may be used to translate the message to the final user
		Message         string `json:"message"` // Understandable message sent to the client
		InternalMessage string `json:"-"`       // used only for logging so we don't want to serialize it out
	}

	//generic type as device data can be comprised of many things
	deviceData map[string]interface{}
)

const (
	// DataAPIPrefix logging prefix
	DataAPIPrefix             = "api/data "
	medtronicLoopBoundaryDate = "2017-09-01"
	slowQueryDuration         = 0.1 // seconds
)

var (
	errorStatusCheck       = detailedError{Status: http.StatusInternalServerError, Code: "data_status_check", Message: "checking of the status endpoint showed an error"}
	errorNoViewPermission  = detailedError{Status: http.StatusForbidden, Code: "data_cant_view", Message: "user is not authorized to view data"}
	errorNoPermissions     = detailedError{Status: http.StatusInternalServerError, Code: "data_perms_error", Message: "error finding permissions for user"}
	errorRunningQuery      = detailedError{Status: http.StatusInternalServerError, Code: "data_store_error", Message: "internal server error"}
	errorLoadingEvents     = detailedError{Status: http.StatusInternalServerError, Code: "json_marshal_error", Message: "internal server error"}
	errorTideV2Http        = detailedError{Status: http.StatusInternalServerError, Code: "tidev2_error", Message: "internal server error"}
	errorInvalidParameters = detailedError{Status: http.StatusBadRequest, Code: "invalid_parameters", Message: "one or more parameters are invalid"}
	errorNotfound          = detailedError{Status: http.StatusNotFound, Code: "data_not_found", Message: "no data for specified user"}
)

func InitAPI(storage store.Storage, auth auth.ClientInterface, permsClient opa.Client, schemaV store.SchemaVersion, logger *log.Logger, V2Client tideV2Client.ClientInterface) *API {
	return &API{
		store:         storage,
		authClient:    auth,
		perms:         permsClient,
		schemaVersion: schemaV,
		logger:        logger,
		tideV2Client:  V2Client,
	}
}

// SetHandlers set the API routes
func (a *API) SetHandlers(prefix string, rtr *mux.Router) {
	rtr.HandleFunc("/swagger", a.get501).Methods("GET")

	a.setHandlesV1(prefix+"/v1", rtr)
	rtr.HandleFunc("/v2", a.get501).Methods("GET")

	// v0 routes:
	rtr.HandleFunc("/status", a.getStatus).Methods("GET")
}

func (h varsHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	h(res, req, vars)
}

func (a *API) get501(res http.ResponseWriter, req *http.Request) {
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
func (a *API) getStatus(res http.ResponseWriter, req *http.Request) {
	start := time.Now()
	var s status.ApiStatus
	if err := a.store.Ping(); err != nil {
		errorLog := errorStatusCheck.setInternalMessage(err)
		a.logError(&errorLog, start)
		s = status.NewApiStatus(errorLog.Status, err.Error())
	} else {
		s = status.NewApiStatus(http.StatusOK, "OK")
	}
	if jsonDetails, err := json.Marshal(s); err != nil {
		a.jsonError(res, errorLoadingEvents.setInternalMessage(err), start)
	} else {
		res.Header().Add("content-type", "application/json")
		res.WriteHeader(s.Status.Code)
		res.Write(jsonDetails)
	}
	return
}

// log error detail and write as application/json
func (a *API) jsonError(res http.ResponseWriter, err detailedError, startedAt time.Time) {
	a.logError(&err, startedAt)
	jsonErr, _ := json.Marshal(err)

	res.Header().Add("content-type", "application/json")
	res.WriteHeader(err.Status)
	res.Write(jsonErr)
}

func (a *API) logError(err *detailedError, startedAt time.Time) {
	err.ID = uuid.New().String()
	a.logger.Println(DataAPIPrefix, fmt.Sprintf("[%s][%s] failed after [%.3f]secs with error [%s][%s] ", err.ID, err.Code, time.Now().Sub(startedAt).Seconds(), err.Message, err.InternalMessage))
}

// set the internal message that we will use for logging
func (d detailedError) setInternalMessage(internal error) detailedError {
	d.InternalMessage = internal.Error()
	return d
}

func (a *API) isAuthorized(req *http.Request, targetUserIDs []string) bool {
	td := a.authClient.Authenticate(req)
	if td == nil {
		a.logger.Printf("%s - %s %s HTTP/%d.%d - Missing header token", req.RemoteAddr, req.Method, req.URL.String(), req.ProtoMajor, req.ProtoMinor)
		return false
	}
	if td.IsServer {
		return true
	}
	if len(targetUserIDs) == 1 {
		targetUserID := targetUserIDs[0]
		if td.UserId == targetUserID {
			return true
		}
	}

	auth, err := a.perms.GetOpaAuth(req)
	if err != nil {
		log.Println(DataAPIPrefix, fmt.Sprintf("Opa authorization error [%v] ", err))
		return false
	}
	return auth.Result.Authorized
}
