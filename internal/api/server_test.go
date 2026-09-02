package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spid37/geocoder/internal/api"
	"github.com/spid37/geocoder/internal/testdata"
)

func TestServerAPIKeyAuth(t *testing.T) {
	db := testdata.OpenDB(t)
	srv := api.NewServer(":0", db, t.TempDir(), api.ParseAPIKeys("secret-key"))

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode?suburb=Richmond&state=VIC&postcode=3121", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status: %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/geocode?suburb=Richmond&state=VIC&postcode=3121", nil)
	req.Header.Set("X-API-Key", "secret-key")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("authenticated status: %d body: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health status: %d", rec.Code)
	}
}

func TestServerFixtureGeocodeJSON(t *testing.T) {
	db := testdata.OpenDB(t)
	srv := api.NewServer(":0", db, t.TempDir(), nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode?street=42+Demo+Rd&suburb=Richmond&state=VIC&postcode=3121", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["address_detail_pid"] != "GAVIC999000001" {
		t.Fatalf("pid: %v", body["address_detail_pid"])
	}
	if body["address_slug"] != "42-demo-rd-richmond-vic-3121" {
		t.Fatalf("address_slug: %v", body["address_slug"])
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/geocode?suburb=Richmond&state=VIC&postcode=3121", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("suburb status: %d", rec.Code)
	}
	var suburbBody map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &suburbBody); err != nil {
		t.Fatal(err)
	}
	if _, ok := suburbBody["address_slug"]; ok {
		t.Fatalf("suburb match should not include address_slug: %v", suburbBody["address_slug"])
	}
}

func TestServerSuggestAddresses(t *testing.T) {
	db := testdata.OpenDB(t)
	srv := api.NewServer(":0", db, t.TempDir(), nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/suggest/addresses?q=42+demo+rd+rich&state=VIC", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Suggestions []map[string]any `json:"suggestions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Suggestions) != 1 {
		t.Fatalf("suggestions: %+v", body.Suggestions)
	}
	if body.Suggestions[0]["address_slug"] != "42-demo-rd-richmond-vic-3121" {
		t.Fatalf("slug: %v", body.Suggestions[0]["address_slug"])
	}
}

func TestServerGeocodeNotFound(t *testing.T) {
	db := testdata.OpenDB(t)
	srv := api.NewServer(":0", db, t.TempDir(), nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode?suburb=Nowhere&state=NSW&postcode=2000", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: %d", rec.Code)
	}
}
