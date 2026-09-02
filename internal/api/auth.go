package api

import (
	"net/http"
	"strings"
)

type APIKeyAuth struct {
	keys map[string]struct{}
}

func ParseAPIKeys(commaSeparated string) *APIKeyAuth {
	if strings.TrimSpace(commaSeparated) == "" {
		return nil
	}

	keys := make(map[string]struct{})
	for _, part := range strings.Split(commaSeparated, ",") {
		key := strings.TrimSpace(part)
		if key != "" {
			keys[key] = struct{}{}
		}
	}
	if len(keys) == 0 {
		return nil
	}
	return &APIKeyAuth{keys: keys}
}

func (a *APIKeyAuth) required() bool {
	return a != nil && len(a.keys) > 0
}

func (a *APIKeyAuth) valid(r *http.Request) bool {
	if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" {
		_, ok := a.keys[key]
		return ok
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		key := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		_, ok := a.keys[key]
		return ok
	}
	return false
}

func (a *APIKeyAuth) wrap(fn http.HandlerFunc) http.HandlerFunc {
	if a == nil || !a.required() {
		return fn
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.valid(r) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or missing API key"})
			return
		}
		fn(w, r)
	}
}
