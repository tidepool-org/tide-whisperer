package auth

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"

	auth0 "github.com/auth0-community/go-auth0"
	jose "gopkg.in/square/go-jose.v2"
	jwt "gopkg.in/square/go-jose.v2/jwt"
)

// Response for simple message structure
type Response struct {
	Message string `json:"message"`
}

func loadPublicKey(data []byte) (interface{}, error) {
	input := data

	block, _ := pem.Decode(data)
	if block != nil {
		input = block.Bytes
	}

	// Try to load SubjectPublicKeyInfo
	pub, err0 := x509.ParsePKIXPublicKey(input)
	if err0 == nil {
		return pub, nil
	}

	cert, err1 := x509.ParseCertificate(input)
	if err1 == nil {
		return cert.PublicKey, nil
	}

	return nil, fmt.Errorf("square/go-jose: parse error, got '%s' and '%s'", err0, err1)
}

// CheckJwt for auth
func CheckJwt(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		cert := `-----BEGIN CERTIFICATE-----
MIIC/zCCAeegAwIBAgIJToXrM8+O9YLWMA0GCSqGSIb3DQEBCwUAMB0xGzAZBgNV
BAMTEnRpZGVwb29sLmF1dGgwLmNvbTAeFw0xNzA1MTUyMDE4MDFaFw0zMTAxMjIy
MDE4MDFaMB0xGzAZBgNVBAMTEnRpZGVwb29sLmF1dGgwLmNvbTCCASIwDQYJKoZI
hvcNAQEBBQADggEPADCCAQoCggEBAJg+lsoTMYS6JNsR9wpQWl0AXeK6HMb6qWVx
Wl5KcEH/3zJVBo6pprBLXaztTCcEUlc3RQ7m0vNcbVO5LX4eYcrfKTHtxJBj3T6W
JAhOUNaBpjn0rU1x1aZBUoY3PRvKCI+1dYFi8UzRf9MovdcP6zC64ZIa+Hsfw+qo
RKWcSwZGRPuWQyvHb0OeehrcBeFrHYwqW0YY6Abh4cZlUrre6usqs3lcfFvnvKqj
oo3+J8fLTkoVWfyrqqhtpQIrEn3jewNBKxx1ej5j2pJG1b7YYcBKp5OTzEd/pieG
GyVXxWV4+y2O7ZEBn/T5cwi3OO6V5VxO58Rd076PTHR7uJkWxlkCAwEAAaNCMEAw
DwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUm7gCoVLZ3idP5ygn1pH1G2N7gdMw
DgYDVR0PAQH/BAQDAgKEMA0GCSqGSIb3DQEBCwUAA4IBAQCMeR+5NtI1sKq9HOLg
jzfmtP/TBeco93oeSVZ82RJOURDOkolmJp8xkMu/aY56dOTdEAUEPqqXyLTE38iy
i++7XzWyyISVIRwqwdkHIp7BiUD89WM5ZdVdynfJo/dvZDcDD43ybJpWRu+qPzmZ
hbWLdP0mvN3yKrwJF9zOsiFsBPDURFgi4jaLjVhXuq8DfIsDMdB/WBpmvm5LpgpN
oGEQQeAP42HBJJzePjC0zijyk3F3f+eM9EHS12O8hfw2o/nAO+IK8/jHXBNGjo9S
38eB4lkn3McpRoZtqv56+VUCfrNi8A1h+ReNfQLSuorbcK21vqPgDiQT9PiuWyCm
tWud
-----END CERTIFICATE-----
`

		// secret, _ := loadPublicKey([]byte(cert))
		// secretProvider := auth0.NewKeyProvider(secret)

		// configuration := auth0.NewConfiguration(
		// 	secretProvider,
		// 	//[]string{"http://localhost:8009/data", "https://tidepool.auth0.com/userinfo"},
		// 	[]string{"https://dev-api.tidepool.org/data", "https://tidepool.auth0.com/userinfo"},
		// 	"https://tidepool.auth0.com/",
		// 	jose.RS256,
		// )

		//
		configuration := auth0.NewConfiguration(
			auth0.NewJWKClient(auth0.JWKClientOptions{URI: "https://tidepool.auth0.com/.well-known/jwks.json"}),
			[]string{"https://dev-api.tidepool.org/data", "https://tidepool.auth0.com/userinfo"},
			"https://tidepool.auth0.com/",
			jose.RS256,
		)
		//
		validator := auth0.NewValidator(configuration)

		token, err := validator.ValidateRequest(r)

		if err != nil {
			fmt.Println("Token is not valid or missing token")
			fmt.Println("error was: ", err.Error())
			response := Response{
				Message: "Missing or invalid token.",
			}

			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(response)

		} else {
			// Ensure the token has the correct scope
			result := checkScope(r, validator, token)
			if result == true {
				// If the token is valid and we have the right scope, we'll pass through the middleware
				h.ServeHTTP(w, r)
			} else {
				response := Response{
					Message: "You do not have the read:data scope.",
				}
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(response)

			}
		}
	})
}

func checkScope(r *http.Request, validator *auth0.JWTValidator, token *jwt.JSONWebToken) bool {
	claims := map[string]interface{}{}
	err := validator.Claims(r, token, &claims)

	if err != nil {
		fmt.Println(err)
		return false
	}
	if strings.Contains(claims["scope"].(string), "read:data") {
		return true
	}
	return false
}
