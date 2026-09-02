package api

import (
	"net/http"

	"github.com/spid37/geocoder/internal/version"
)

func versionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := version.String(); v != "" {
			w.Header().Set("X-API-Version", v)
		}
		next.ServeHTTP(w, r)
	})
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"version": version.String()})
}
