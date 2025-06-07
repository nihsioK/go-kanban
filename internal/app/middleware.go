package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xeipuuv/gojsonschema"
)

type Claims struct {
	Username string `json:"username"`
	ID       string `json:"id"`
	jwt.RegisteredClaims
}

func (a *App) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

func (a *App) JWTAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			RespondWithError(w, http.StatusUnauthorized, "No authorization token provided")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			return a.JWTKey, nil
		})

		if err != nil || !token.Valid {
			RespondWithError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), "claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *App) Validate(schemaKey string, next http.Handler) http.Handler {
	schema, ok := a.Schemas[schemaKey]
	if !ok {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			RespondWithError(w, http.StatusInternalServerError, "Schema not found")
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil || json.Unmarshal(bodyBytes, &body) != nil {
			RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		schemaLoader := gojsonschema.NewStringLoader(schema)
		documentLoader := gojsonschema.NewGoLoader(body)

		result, err := gojsonschema.Validate(schemaLoader, documentLoader)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Schema validation error")
			return
		}
		if !result.Valid() {
			var errs []string
			for _, e := range result.Errors() {
				errs = append(errs, e.String())
			}
			RespondWithError(w, http.StatusBadRequest, strings.Join(errs, ", "))
			return
		}

		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		next.ServeHTTP(w, r)
	})
}
