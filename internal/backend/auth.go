package backend

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
)

func APIKeyMiddleware(expectedKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := r.Header.Get("X-API-Key")

			if len(got) == 0 || subtle.ConstantTimeCompare([]byte(got), []byte(expectedKey)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "missing or invalid api key",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
