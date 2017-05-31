package main

import (
	"encoding/json"
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

func checkJwt(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		configuration := auth0.NewConfiguration(
			auth0.NewJWKClient(auth0.JWKClientOptions{URI: "https://tidepool/.well-known/jwks.json"}),
			[]string{"http://localhost:8009/data"},
			"https://tidepool.auth0.com/",
			jose.RS256,
		)
		validator := auth0.NewValidator(configuration)

		token, err := validator.ValidateRequest(r)

		fmt.Println("## token", token)

		if err != nil {
			fmt.Println("Token is not valid or missing token")

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
	fmt.Println("## claims", claims)
	if strings.Contains(claims["scope"].(string), "read:data") {
		return true
	}
	return false
}
