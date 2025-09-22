package api

import (
	"encoding/json"
	"log"
	"net/http"
)

//go:generate go run -modfile=../tools/go.mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=config.yaml api.yaml

func Float(v float64) *float64 {
	return &v
}

func String(v string) *string {
	return &v
}

func Boolean(v bool) *bool {
	return &v
}

func FromString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func FromInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Println("Recovered panic: ", err)

				jsonBody, _ := json.Marshal(map[string]string{
					"error": "There was an internal server error",
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(jsonBody)
			}

		}()

		next.ServeHTTP(w, r)
	})
}
