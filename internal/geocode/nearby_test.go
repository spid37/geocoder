package geocode_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/spid37/geocoder/internal/geocode"
	"github.com/spid37/geocoder/internal/store"
)

func insertNearbyFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO locality_centroids (
			state, postcode, locality_name, locality_name_norm,
			latitude, longitude, address_count, sa3_name
		) VALUES
			('VIC', '3121', 'RICHMOND', 'RICHMOND', -37.8182, 145.0012, 500, 'Yarra'),
			('VIC', '3123', 'HAWTHORN', 'HAWTHORN', -37.82, 145.02, 400, 'Yarra'),
			('VIC', '3000', 'MELBOURNE', 'MELBOURNE', -37.813, 144.963, 1000, 'Melbourne City')`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSuburbsNearby_suburbSlugCentre(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	insertNearbyFixtures(t, db)

	svc := geocode.NewService(db)
	res, err := svc.SuburbsNearby(context.Background(), geocode.NearbyQuery{
		Radius:     "15km",
		SuburbSlug: "richmond-vic-3121",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Center.SuburbSlug != "richmond-vic-3121" {
		t.Fatalf("center: %+v", res.Center)
	}
	for _, s := range res.Suburbs {
		if s.SuburbSlug == "richmond-vic-3121" {
			t.Fatal("centre suburb should be excluded")
		}
	}
	for i := 1; i < len(res.Suburbs); i++ {
		if res.Suburbs[i-1].DistanceM > res.Suburbs[i].DistanceM {
			t.Fatalf("sort: %+v", res.Suburbs)
		}
	}
	if len(res.Suburbs) < 1 {
		t.Fatalf("expected nearby suburbs: %+v", res.Suburbs)
	}
}

func TestSuburbsNearby_latLngCentre(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	insertNearbyFixtures(t, db)

	svc := geocode.NewService(db)
	res, err := svc.SuburbsNearby(context.Background(), geocode.NearbyQuery{
		Radius:    "10km",
		Latitude:  "-37.8182",
		Longitude: "145.0012",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Center.Suburb != "" {
		t.Fatalf("lat/lng center should not set suburb name: %+v", res.Center)
	}
	if len(res.Suburbs) == 0 {
		t.Fatal("expected suburbs")
	}
}

func TestSuburbsNearby_unknownSlug(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	insertNearbyFixtures(t, db)

	svc := geocode.NewService(db)
	_, err = svc.SuburbsNearby(context.Background(), geocode.NearbyQuery{
		Radius:     "5km",
		SuburbSlug: "no-such-vic-3000",
	})
	if !errors.Is(err, geocode.ErrSuburbSlugNotFound) {
		t.Fatalf("err: %v", err)
	}
}

func TestSuburbsNearby_validation(t *testing.T) {
	svc := geocode.NewService(nil)
	_, err := svc.SuburbsNearby(context.Background(), geocode.NearbyQuery{})
	if !errors.Is(err, geocode.ErrRadiusRequired) {
		t.Fatalf("err: %v", err)
	}
	_, err = svc.SuburbsNearby(context.Background(), geocode.NearbyQuery{Radius: "5km"})
	if !errors.Is(err, geocode.ErrCenterRequired) {
		t.Fatalf("err: %v", err)
	}
	_, err = svc.SuburbsNearby(context.Background(), geocode.NearbyQuery{
		Radius:     "5km",
		SuburbSlug: "x",
		Latitude:   "1",
	})
	if !errors.Is(err, geocode.ErrCenterRequired) {
		t.Fatalf("err: %v", err)
	}
}

func TestSuburbsNearby_limitAndTruncated(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	insertNearbyFixtures(t, db)

	svc := geocode.NewService(db)
	res, err := svc.SuburbsNearby(context.Background(), geocode.NearbyQuery{
		Radius:    "50km",
		Latitude:  "-37.8182",
		Longitude: "145.0012",
		Limit:     1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Suburbs) != 1 || !res.Truncated {
		t.Fatalf("res: %+v", res)
	}
}

func TestSuburbsNearby_emptyRadius(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	insertNearbyFixtures(t, db)

	svc := geocode.NewService(db)
	res, err := svc.SuburbsNearby(context.Background(), geocode.NearbyQuery{
		Radius:    "1m",
		Latitude:  "-10",
		Longitude: "10",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Suburbs) != 0 {
		t.Fatalf("suburbs: %+v", res.Suburbs)
	}
}

func TestSuburbsNearby_stateFilter(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	insertNearbyFixtures(t, db)

	svc := geocode.NewService(db)
	res, err := svc.SuburbsNearby(context.Background(), geocode.NearbyQuery{
		Radius:    "100km",
		Latitude:  "-37.8182",
		Longitude: "145.0012",
		State:     "NSW",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Suburbs) != 0 {
		t.Fatalf("suburbs: %+v", res.Suburbs)
	}
}