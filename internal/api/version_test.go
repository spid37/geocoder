package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spid37/geocoder/internal/api"
	"github.com/spid37/geocoder/internal/testdata"
	"github.com/spid37/geocoder/internal/version"
)

func TestServerVersionEndpoint(t *testing.T) {
	db := testdata.OpenDB(t)
	srv := api.NewServer(":0", db, t.TempDir(), nil)

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	if got := rec.Header().Get("X-API-Version"); got != version.String() {
		t.Fatalf("X-API-Version: %q, want %q", got, version.String())
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["version"] != version.String() {
		t.Fatalf("version: %q", body["version"])
	}
	if _, ok := body["status"]; ok {
		t.Fatalf("unexpected status field: %v", body)
	}
}

func TestServerHealthVersion(t *testing.T) {
	db := testdata.OpenDB(t)
	srv := api.NewServer(":0", db, t.TempDir(), nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	if got := rec.Header().Get("X-API-Version"); got != version.String() {
		t.Fatalf("X-API-Version: %q, want %q", got, version.String())
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["version"] != version.String() {
		t.Fatalf("version: %q", body["version"])
	}
}
