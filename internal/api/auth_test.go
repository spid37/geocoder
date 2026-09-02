package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIKeyAuthOpen(t *testing.T) {
	auth := ParseAPIKeys("")
	if auth != nil {
		t.Fatal("expected nil auth for empty keys")
	}

	called := false
	handler := ParseAPIKeys("").wrap(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	handler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !called {
		t.Fatal("expected handler to run without API key")
	}
}

func TestAPIKeyAuthRequired(t *testing.T) {
	auth := ParseAPIKeys("secret, other-key")
	handler := auth.wrap(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "secret")
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("X-API-Key status: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer other-key")
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Bearer status: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "wrong")
	handler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong key status: %d", rec.Code)
	}
}

func TestParseAPIKeysIgnoresEmptyParts(t *testing.T) {
	auth := ParseAPIKeys("a,, b, ")
	if auth == nil || !auth.valid(reqWithKey("a")) || !auth.valid(reqWithKey("b")) {
		t.Fatal("expected two valid keys")
	}
}

func reqWithKey(key string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-API-Key", key)
	return r
}
