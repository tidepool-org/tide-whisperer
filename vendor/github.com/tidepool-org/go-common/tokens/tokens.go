package tokens

import (
	"log"
	"net/http"
	"strings"

	"github.com/tidepool-org/go-common/clients/shoreline"
)

const TidepoolSessionTokenName = "x-tidepool-session-token"
const TidepoolInternalScope = "tidepool:internal"
const TidepoolPublicScope = "tidepool:public"

func GetHeaderToken(request *http.Request) string {
	if bearer := GetBearerToken(request); bearer != "" {
		return bearer
	}
	return GetSessionToken(request)
}

func GetBearerToken(request *http.Request) string {
	if request != nil {
		auth := request.Header.Get("Authorization")
		if len(auth) > 7 &&
			strings.EqualFold(auth[0:7], "BEARER ") {
			return strings.Split(auth, " ")[1]
		}
	}
	return ""
}

func GetSessionToken(request *http.Request) string {
	if request != nil {
		return request.Header.Get(TidepoolSessionTokenName)
	}
	return ""
}

func IsBearerToken(request *http.Request) bool {
	return GetBearerToken(request) != ""
}

func IsSessionToken(request *http.Request) bool {
	return GetSessionToken(request) != ""
}

func CheckToken(response http.ResponseWriter, request *http.Request, requiredScopes string, client shoreline.Client) *shoreline.TokenData {
	if IsBearerToken(request) || IsSessionToken(request) {
		if IsBearerToken(request) {
			tokenData := client.CheckTokenForScopes(requiredScopes, GetBearerToken(request))
			if tokenData != nil {
				return tokenData
			}
			log.Println("validated as a bearer token and failed")
		} else if IsSessionToken(request) {
			tokenData := client.CheckToken(GetSessionToken(request))
			if tokenData != nil {
				return tokenData
			}
			log.Println("validated as a session token and failed")
		}
		//TODO: at the moment we don't know the real reason for the failure so will just return 401
		response.WriteHeader(http.StatusUnauthorized)
		response.Write([]byte("Status Unauthorized"))
		return nil
	}
	log.Println("token was neither a bearer or session token so forbidden")
	response.WriteHeader(http.StatusForbidden)
	response.Write([]byte("Status Forbidden"))
	return nil
}
