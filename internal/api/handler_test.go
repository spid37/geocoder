package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/spid37/geocoder/internal/api"
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
