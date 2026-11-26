package geocode_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/spid37/geocoder/internal/geocode"
	"github.com/spid37/geocoder/internal/store"
)

func TestListSuburbsByRegion(t *testing.T) {
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
			('VIC', '3121', 'RICHMOND EAST', 'RICHMOND EAST', -37.82, 145.01, 800, 'Yarra'),
			('VIC', '3123', 'HAWTHORN', 'HAWTHORN', -37.82, 145.02, 600, 'Yarra')`)
	if err != nil {
		t.Fatal(err)
	}

	svc := geocode.NewService(db)
	ctx := context.Background()

	res, err := svc.ListSuburbsByRegion(ctx, "yarra-vic", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.RegionSlug != "yarra-vic" {
		t.Fatalf("region_slug: %q", res.RegionSlug)
	}
	if len(res.Suburbs) != 3 {
		t.Fatalf("suburbs: %+v", res.Suburbs)
	}
	if res.Suburbs[0].Suburb != "Hawthorn" {
		t.Fatalf("sort order: %+v", res.Suburbs)
	}
	if res.Suburbs[0].RegionSlug != "yarra-vic" || res.Suburbs[0].Region != "Yarra" {
		t.Fatalf("first suburb: %+v", res.Suburbs[0])
	}
}

func TestListSuburbsByRegion_unknownSlug(t *testing.T) {
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

	svc := geocode.NewService(db)
	res, err := svc.ListSuburbsByRegion(context.Background(), "no-such-region-vic", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Suburbs) != 0 {
		t.Fatalf("suburbs: %+v", res.Suburbs)
	}
}

func TestListSuburbsByRegion_limit(t *testing.T) {
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
			('VIC', '3121', 'AAA', 'AAA', -37.81, 145.00, 1, 'Yarra'),
			('VIC', '3122', 'BBB', 'BBB', -37.82, 145.01, 1, 'Yarra'),
			('VIC', '3123', 'CCC', 'CCC', -37.83, 145.02, 1, 'Yarra')`)
	if err != nil {
		t.Fatal(err)
	}

	svc := geocode.NewService(db)
	res, err := svc.ListSuburbsByRegion(context.Background(), "yarra-vic", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Suburbs) != 2 {
		t.Fatalf("suburbs: %+v", res.Suburbs)
	}
}

func TestListSuburbsByRegion_stateGuard(t *testing.T) {
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

	svc := geocode.NewService(db)
	res, err := svc.ListSuburbsByRegion(context.Background(), "yarra-vic", "NSW", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Suburbs) != 0 {
		t.Fatalf("suburbs: %+v", res.Suburbs)
	}
}

func TestListSuburbsByRegion_regionsNotLoaded(t *testing.T) {
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

	svc := geocode.NewService(db)
	_, err = svc.ListSuburbsByRegion(context.Background(), "yarra-vic", "", 0)
	if !errors.Is(err, geocode.ErrRegionsNotLoaded) {
		t.Fatalf("err: %v", err)
	}
}

func TestListSuburbsByRegion_complexSA3Slug(t *testing.T) {
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
			'NSW', '2753', 'MOUNT VERNON', 'MOUNT VERNON', -33.87, 150.79, 100, 'Richmond - Windsor')`)
	if err != nil {
		t.Fatal(err)
	}

	svc := geocode.NewService(db)
	res, err := svc.ListSuburbsByRegion(context.Background(), "richmond-windsor-nsw", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Suburbs) != 1 || res.Suburbs[0].Suburb != "Mount Vernon" {
		t.Fatalf("suburbs: %+v", res.Suburbs)
	}
}