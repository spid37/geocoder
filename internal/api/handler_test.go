package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spid37/geocoder/internal/api"
	"github.com/spid37/geocoder/internal/config"
	"github.com/spid37/geocoder/internal/geocode"
	"github.com/spid37/geocoder/internal/store"
)

func TestGeocodeResponseShape(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT INTO addresses (
			address_detail_pid, address_label, number_first, street_name, street_type,
			street_name_norm, street_type_norm, locality_name, locality_name_norm,
			state, postcode, latitude, longitude,
			sa3_code, sa3_name
		) VALUES (
			'PID1', '42 DEMO RD RICHMOND VIC 3121', '42', 'DEMO', 'RD',
			'DEMO', 'RD', 'RICHMOND', 'RICHMOND', 'VIC', '3121', -37.8182, 145.0012,
			'21305', 'Yarra'
		)`)
	if err != nil {
		t.Fatal(err)
	}

	h := &api.Handler{Geocoder: geocode.NewService(db)}
	req := httptest.NewRequest(http.MethodGet, "/v1/geocode?street=42+Demo+Rd&suburb=Richmond&state=VIC&postcode=3121", nil)
	rec := httptest.NewRecorder()
	h.Geocode(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"latitude", "longitude", "accuracy", "address", "address_slug", "suburb_slug", "region_slug"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("missing key %q in %v", key, body)
		}
	}
	if _, ok := body["regions"]; ok {
		t.Fatalf("regions should not be present: %v", body["regions"])
	}

	address, ok := body["address"].(map[string]any)
	if !ok {
		t.Fatalf("address: %v", body["address"])
	}
	if address["suburb"] != "Richmond" {
		t.Fatalf("suburb: %v", address["suburb"])
	}
	if address["region"] != "Yarra" {
		t.Fatalf("region: %v", address["region"])
	}
	if body["address_slug"] != "42-demo-rd-richmond-vic-3121" {
		t.Fatalf("address_slug: %v", body["address_slug"])
	}
	if body["suburb_slug"] != "richmond-vic-3121" {
		t.Fatalf("suburb_slug: %v", body["suburb_slug"])
	}
	if body["region_slug"] != "yarra-vic" {
		t.Fatalf("region_slug: %v", body["region_slug"])
	}
}

func TestSuggestSuburbs(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT INTO locality_centroids (
			state, postcode, locality_name, locality_name_norm,
			latitude, longitude, address_count, sa3_name
		) VALUES (
			'VIC', '3121', 'RICHMOND', 'RICHMOND', -37.8182, 145.0012, 500, 'Yarra'
		)`)
	if err != nil {
		t.Fatal(err)
	}

	h := &api.Handler{Geocoder: geocode.NewService(db)}
	req := httptest.NewRequest(http.MethodGet, "/v1/suggest/suburbs?q=rich&state=VIC", nil)
	rec := httptest.NewRecorder()
	h.SuggestSuburbs(rec, req)

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
	if body.Suggestions[0]["suburb"] != "Richmond" {
		t.Fatalf("suburb: %v", body.Suggestions[0]["suburb"])
	}
}

func TestListSuburbs_whitespaceRegionSlug(t *testing.T) {
	h := &api.Handler{Geocoder: geocode.NewService(nil)}
	req := httptest.NewRequest(http.MethodGet, "/v1/suburbs?region_slug=%20%20", nil)
	rec := httptest.NewRecorder()
	h.ListSuburbs(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListSuburbs_missingRegionSlug(t *testing.T) {
	h := &api.Handler{Geocoder: geocode.NewService(nil)}
	req := httptest.NewRequest(http.MethodGet, "/v1/suburbs", nil)
	rec := httptest.NewRecorder()
	h.ListSuburbs(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListSuburbs_regionsNotLoaded(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT INTO locality_centroids (
			state, postcode, locality_name, locality_name_norm,
			latitude, longitude, address_count
		) VALUES (
			'VIC', '3121', 'RICHMOND', 'RICHMOND', -37.8182, 145.0012, 500)`)
	if err != nil {
		t.Fatal(err)
	}

	h := &api.Handler{Geocoder: geocode.NewService(db)}
	req := httptest.NewRequest(http.MethodGet, "/v1/suburbs?region_slug=yarra-vic", nil)
	rec := httptest.NewRecorder()
	h.ListSuburbs(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "regions load") {
		t.Fatalf("body: %s", rec.Body.String())
	}
}

func TestListSuburbs_success(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT INTO locality_centroids (
			state, postcode, locality_name, locality_name_norm,
			latitude, longitude, address_count, sa3_name
		) VALUES (
			'VIC', '3121', 'RICHMOND', 'RICHMOND', -37.8182, 145.0012, 500, 'Yarra')`)
	if err != nil {
		t.Fatal(err)
	}

	h := &api.Handler{Geocoder: geocode.NewService(db)}
	req := httptest.NewRequest(http.MethodGet, "/v1/suburbs?region_slug=yarra-vic", nil)
	rec := httptest.NewRecorder()
	h.ListSuburbs(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Attribution"); got != config.Attribution {
		t.Fatalf("X-Attribution: %q", got)
	}

	var body struct {
		RegionSlug string `json:"region_slug"`
		Suburbs    []struct {
			Suburb     string `json:"suburb"`
			SuburbSlug string `json:"suburb_slug"`
			RegionSlug string `json:"region_slug"`
		} `json:"suburbs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.RegionSlug != "yarra-vic" || len(body.Suburbs) != 1 {
		t.Fatalf("body: %+v", body)
	}
	if body.Suburbs[0].SuburbSlug != "richmond-vic-3121" {
		t.Fatalf("suburb_slug: %q", body.Suburbs[0].SuburbSlug)
	}
}

func TestListSuburbs_unknownSlug(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT INTO locality_centroids (
			state, postcode, locality_name, locality_name_norm,
			latitude, longitude, address_count, sa3_name
		) VALUES (
			'VIC', '3121', 'RICHMOND', 'RICHMOND', -37.8182, 145.0012, 500, 'Yarra')`)
	if err != nil {
		t.Fatal(err)
	}

	h := &api.Handler{Geocoder: geocode.NewService(db)}
	req := httptest.NewRequest(http.MethodGet, "/v1/suburbs?region_slug=no-such-vic", nil)
	rec := httptest.NewRecorder()
	h.ListSuburbs(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if string(body["suburbs"]) != "[]" {
		t.Fatalf("suburbs JSON: %s", body["suburbs"])
	}
}

func TestListSuburbs_stateMismatchEmpty(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT INTO locality_centroids (
			state, postcode, locality_name, locality_name_norm,
			latitude, longitude, address_count, sa3_name
		) VALUES (
			'VIC', '3121', 'RICHMOND', 'RICHMOND', -37.8182, 145.0012, 500, 'Yarra')`)
	if err != nil {
		t.Fatal(err)
	}

	h := &api.Handler{Geocoder: geocode.NewService(db)}
	req := httptest.NewRequest(http.MethodGet, "/v1/suburbs?region_slug=yarra-vic&state=NSW", nil)
	rec := httptest.NewRecorder()
	h.ListSuburbs(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if string(body["suburbs"]) != "[]" {
		t.Fatalf("suburbs: %s", body["suburbs"])
	}
}

func TestListSuburbs_invalidLimitUsesDefault(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT INTO locality_centroids (
			state, postcode, locality_name, locality_name_norm,
			latitude, longitude, address_count, sa3_name
		) VALUES (
			'VIC', '3121', 'RICHMOND', 'RICHMOND', -37.8182, 145.0012, 500, 'Yarra')`)
	if err != nil {
		t.Fatal(err)
	}

	h := &api.Handler{Geocoder: geocode.NewService(db)}
	req := httptest.NewRequest(http.MethodGet, "/v1/suburbs?region_slug=yarra-vic&limit=abc", nil)
	rec := httptest.NewRecorder()
	h.ListSuburbs(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Suburbs []map[string]any `json:"suburbs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Suburbs) != 1 {
		t.Fatalf("suburbs: %+v", body.Suburbs)
	}
}

func TestSuburbsNearby_missingRadius(t *testing.T) {
	h := &api.Handler{Geocoder: geocode.NewService(nil)}
	req := httptest.NewRequest(http.MethodGet, "/v1/suburbs/nearby?suburb_slug=richmond-vic-3121", nil)
	rec := httptest.NewRecorder()
	h.SuburbsNearby(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSuburbsNearby_notFoundSlug(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
		INSERT INTO locality_centroids (
			state, postcode, locality_name, locality_name_norm,
			latitude, longitude, address_count, sa3_name
		) VALUES (
			'VIC', '3121', 'RICHMOND', 'RICHMOND', -37.8182, 145.0012, 500, 'Yarra')`)
	if err != nil {
		t.Fatal(err)
	}
	h := &api.Handler{Geocoder: geocode.NewService(db)}
	req := httptest.NewRequest(http.MethodGet, "/v1/suburbs/nearby?radius=5km&suburb_slug=no-such-vic-3000", nil)
	rec := httptest.NewRecorder()
	h.SuburbsNearby(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSuburbsNearby_success(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
		INSERT INTO locality_centroids (
			state, postcode, locality_name, locality_name_norm,
			latitude, longitude, address_count, sa3_name
		) VALUES
			('VIC', '3121', 'RICHMOND', 'RICHMOND', -37.8182, 145.0012, 500, 'Yarra'),
			('VIC', '3123', 'HAWTHORN', 'HAWTHORN', -37.82, 145.02, 400, 'Yarra')`)
	if err != nil {
		t.Fatal(err)
	}
	h := &api.Handler{Geocoder: geocode.NewService(db)}
	req := httptest.NewRequest(http.MethodGet, "/v1/suburbs/nearby?radius=10km&suburb_slug=richmond-vic-3121", nil)
	rec := httptest.NewRecorder()
	h.SuburbsNearby(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Attribution"); got != config.Attribution {
		t.Fatalf("X-Attribution: %q", got)
	}
	var body struct {
		Center  map[string]any   `json:"center"`
		Suburbs []map[string]any `json:"suburbs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Center["suburb_slug"] != "richmond-vic-3121" {
		t.Fatalf("center: %+v", body.Center)
	}
	if len(body.Suburbs) < 1 {
		t.Fatalf("suburbs: %+v", body.Suburbs)
	}
	if _, ok := body.Suburbs[0]["distance_m"]; !ok {
		t.Fatalf("distance_m missing: %+v", body.Suburbs[0])
	}
}

func TestSuburbsNearby_latLngCentre(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
		INSERT INTO locality_centroids (
			state, postcode, locality_name, locality_name_norm,
			latitude, longitude, address_count, sa3_name
		) VALUES (
			'VIC', '3121', 'RICHMOND', 'RICHMOND', -37.8182, 145.0012, 500, 'Yarra')`)
	if err != nil {
		t.Fatal(err)
	}
	h := &api.Handler{Geocoder: geocode.NewService(db)}
	req := httptest.NewRequest(http.MethodGet, "/v1/suburbs/nearby?radius=5km&latitude=-37.8182&longitude=145.0012", nil)
	rec := httptest.NewRecorder()
	h.SuburbsNearby(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSuburbsNearby_invalidRadius(t *testing.T) {
	h := &api.Handler{Geocoder: geocode.NewService(nil)}
	req := httptest.NewRequest(http.MethodGet, "/v1/suburbs/nearby?radius=10yd&latitude=0&longitude=0", nil)
	rec := httptest.NewRecorder()
	h.SuburbsNearby(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSuburbsNearby_invalidCoordinatesHTTP(t *testing.T) {
	h := &api.Handler{Geocoder: geocode.NewService(nil)}
	req := httptest.NewRequest(http.MethodGet, "/v1/suburbs/nearby?radius=5km&latitude=NaN&longitude=145", nil)
	rec := httptest.NewRecorder()
	h.SuburbsNearby(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSuburbsNearby_emptySuburbsJSON(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
		INSERT INTO locality_centroids (
			state, postcode, locality_name, locality_name_norm,
			latitude, longitude, address_count, sa3_name
		) VALUES (
			'VIC', '3121', 'RICHMOND', 'RICHMOND', -37.8182, 145.0012, 500, 'Yarra')`)
	if err != nil {
		t.Fatal(err)
	}
	h := &api.Handler{Geocoder: geocode.NewService(db)}
	req := httptest.NewRequest(http.MethodGet, "/v1/suburbs/nearby?radius=1m&latitude=-10&longitude=10", nil)
	rec := httptest.NewRecorder()
	h.SuburbsNearby(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if string(body["suburbs"]) != "[]" {
		t.Fatalf("suburbs: %s", body["suburbs"])
	}
}
