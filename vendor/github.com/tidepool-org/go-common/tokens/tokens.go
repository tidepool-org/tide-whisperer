package tokens

import (
	"log"
	"net/http"

	"github.com/tidepool-org/go-common/clients/shoreline"
)

//TODO: retire token
const TidepoolSessionTokenName = "X-Tidepool-Session-Token"

const TidepoolLegacyServiceSecretHeaderKey = "X-Tidepool-Legacy-Service-Secret"
const AuthorizationHeaderKey = "Authorization"
const TidepoolInternalScope = "tidepool:internal"
const TidepoolPublicScope = "tidepool:public"

func GetServerSecret(request *http.Request) string {
	if request == nil {
		log.Fatal("No request was given")
	}
	return request.Header.Get(TidepoolLegacyServiceSecretHeaderKey)
}

func GetBearerToken(request *http.Request) string {
	if request == nil {
		log.Fatal("No request was given")
	}
	return request.Header.Get(AuthorizationHeaderKey)
}

func CheckToken(response http.ResponseWriter, request *http.Request, requiredScopes string, client shoreline.Client) *shoreline.TokenData {

	if serverSecret := GetServerSecret(request); serverSecret != "" {
		if serverSecret == client.GetSecret() {
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

	if bearerToken := GetBearerToken(request); bearerToken != "" {
		tokenData := client.CheckTokenForScopes(requiredScopes, bearerToken)
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
