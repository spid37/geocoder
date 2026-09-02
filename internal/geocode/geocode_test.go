package geocode_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/spid37/geocoder/internal/geocode"
	"github.com/spid37/geocoder/internal/store"
)

func TestGeocodeFallback(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := store.Open(dbPath, false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT INTO addresses (
			address_detail_pid, address_label, number_first, street_name, street_type,
			street_name_norm, street_type_norm, locality_name, locality_name_norm,
			state, postcode, latitude, longitude
		) VALUES (
			'PID1', '1 COLLINS STREET MELBOURNE VIC 3000', '1', 'COLLINS', 'ST',
			'COLLINS', 'ST', 'MELBOURNE', 'MELBOURNE', 'VIC', '3000', -37.8136, 144.9631
		)`)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.BuildCentroids(db); err != nil {
		t.Fatal(err)
	}

	svc := geocode.NewService(db)
	ctx := context.Background()

	res, err := svc.Geocode(ctx, geocode.Query{
		Street: "1 Collins St", Suburb: "Melbourne", State: "VIC", Postcode: "3000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.Accuracy != geocode.AccuracyStreet {
		t.Fatalf("street match: %+v", res)
	}
	if res.Address == nil || res.Address.Number != "1" {
		t.Fatalf("address: %+v", res.Address)
	}

	res, err = svc.Geocode(ctx, geocode.Query{Suburb: "Melbourne", State: "VIC", Postcode: "3000"})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.Accuracy != geocode.AccuracySuburb {
		t.Fatalf("suburb match: %+v", res)
	}

	res, err = svc.Geocode(ctx, geocode.Query{State: "VIC", Postcode: "3000"})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.Accuracy != geocode.AccuracyPostcode {
		t.Fatalf("postcode match: %+v", res)
	}

	res, err = svc.Geocode(ctx, geocode.Query{State: "VIC"})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.Accuracy != geocode.AccuracyState {
		t.Fatalf("state match: %+v", res)
	}
}

func TestGeocodeRegionMountEliza(t *testing.T) {
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
			'PID2', '42 DEMO RD RICHMOND VIC 3121', '42', 'DEMO', 'RD',
			'DEMO', 'RD', 'RICHMOND', 'RICHMOND', 'VIC', '3121', -37.8182, 145.0012,
			'21305', 'Yarra'
		)`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		INSERT INTO locality_centroids (
			state, postcode, locality_name, locality_name_norm,
			latitude, longitude, address_count,
			sa3_code, sa3_name
		) VALUES (
			'VIC', '3121', 'RICHMOND', 'RICHMOND', -37.8182, 145.0012, 1,
			'21305', 'Yarra'
		)`)
	if err != nil {
		t.Fatal(err)
	}

	svc := geocode.NewService(db)
	res, err := svc.Geocode(context.Background(), geocode.Query{
		Street: "42 Demo Rd", Suburb: "Richmond", State: "VIC", Postcode: "3121",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.Address == nil {
		t.Fatalf("street match: %+v", res)
	}
	if res.Address.Suburb != "Richmond" {
		t.Fatalf("suburb display: %q", res.Address.Suburb)
	}
	if res.Address.Region != "Yarra" {
		t.Fatalf("region: %q", res.Address.Region)
	}
	if res.AddressSlug != "42-demo-rd-richmond-vic-3121" {
		t.Fatalf("address slug: %q", res.AddressSlug)
	}
	if res.SuburbSlug != "richmond-vic-3121" {
		t.Fatalf("suburb slug: %q", res.SuburbSlug)
	}
	if res.RegionSlug != "yarra-vic" {
		t.Fatalf("region slug: %q", res.RegionSlug)
	}

	res, err = svc.Geocode(context.Background(), geocode.Query{
		Suburb: "Richmond", State: "VIC", Postcode: "3121",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.Accuracy != geocode.AccuracySuburb || res.Address.Region != "Yarra" {
		t.Fatalf("suburb region: %+v", res)
	}
	if res.AddressSlug != "" {
		t.Fatalf("address slug should be empty for suburb match: %q", res.AddressSlug)
	}
	if res.SuburbSlug != "richmond-vic-3121" {
		t.Fatalf("suburb slug: %q", res.SuburbSlug)
	}
}
