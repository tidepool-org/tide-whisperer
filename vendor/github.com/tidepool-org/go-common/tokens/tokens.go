package tokens

import (
	"log"
	"net/http"
	"strings"

	"github.com/tidepool-org/go-common/clients/shoreline"
)

//TODO remove
const TidepoolSessionTokenName = "X-Tidepool-Session-Token"

const TidepoolLegacyServiceSecretHeaderKey = "X-Tidepool-Legacy-Service-Secret"
const AuthorizationHeaderKey = "Authorization"
const TidepoolInternalScope = "tidepool:internal"
const TidepoolPublicScope = "tidepool:public"

func GetHeaderToken(request *http.Request) string {
	if secret := GetServerSecret(request); secret != "" {
		return secret
	}
	if bearer := GetBearerToken(request); bearer != "" {
		return bearer
	}
	return ""
}

func GetBearerToken(request *http.Request) string {
	if request != nil {
		auth := request.Header.Get(AuthorizationHeaderKey)
		if len(auth) > 7 &&
			strings.EqualFold(auth[0:7], "BEARER ") {
			return strings.Split(auth, " ")[1]
		}
	}
	return ""
}

func GetServerSecret(request *http.Request) string {
	if request != nil {
		return request.Header.Get(TidepoolLegacyServiceSecretHeaderKey)
	}
	return ""
}

func IsBearerToken(request *http.Request) bool {
	return GetBearerToken(request) != ""
}

func IsServerSecret(request *http.Request) bool {
	return GetServerSecret(request) != ""
}

func CheckToken(response http.ResponseWriter, request *http.Request, requiredScopes string, client shoreline.Client) *shoreline.TokenData {

	if IsServerSecret(request) {
		if GetServerSecret(request) == client.SecretProvide() {
			return &shoreline.TokenData{
				UserID:   TidepoolLegacyServiceSecretHeaderKey,
				IsServer: true,
			}
		}
		log.Println("validated as a server secret and failed")
		//TODO: at the moment we don't know the real reason for the failure so will just return 401
		response.WriteHeader(http.StatusUnauthorized)
		response.Write([]byte("Status Unauthorized"))
		return nil
	}

	if IsBearerToken(request) {
		tokenData := client.CheckTokenForScopes(requiredScopes, GetBearerToken(request))
		if tokenData != nil {
			return tokenData
		}
		log.Println("validated as a bearer token and failed")
		//TODO: at the moment we don't know the real reason for the failure so will just return 401
		response.WriteHeader(http.StatusUnauthorized)
		response.Write([]byte("Status Unauthorized"))
		return nil
	}
	log.Println("token was neither a bearer token or server secret so forbidden")
	response.WriteHeader(http.StatusForbidden)
	response.Write([]byte("Status Forbidden"))
	return nil
}
