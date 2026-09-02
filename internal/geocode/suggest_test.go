package geocode_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/spid37/geocoder/internal/geocode"
	"github.com/spid37/geocoder/internal/store"
)

func TestSuggestSuburbsAndRegions(t *testing.T) {
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
			('NSW', '2753', 'MOUNT VERNON', 'MOUNT VERNON', -33.87, 150.79, 100, 'Richmond - Windsor')`)
	if err != nil {
		t.Fatal(err)
	}

	svc := geocode.NewService(db)
	ctx := context.Background()

	suburbs, err := svc.SuggestSuburbs(ctx, "rich", "VIC", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(suburbs) != 2 || suburbs[0].Suburb != "Richmond East" {
		t.Fatalf("rich VIC: %+v", suburbs)
	}
	if suburbs[0].SuburbSlug != "richmond-east-vic-3121" {
		t.Fatalf("slug: %q", suburbs[0].SuburbSlug)
	}

	suburbs, err = svc.SuggestSuburbs(ctx, "east", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(suburbs) != 1 {
		t.Fatalf("east: %+v", suburbs)
	}

	regions, err := svc.SuggestRegions(ctx, "yarra", "VIC", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(regions) != 1 || regions[0].Region != "Yarra" {
		t.Fatalf("regions: %+v", regions)
	}
	if regions[0].RegionSlug != "yarra-vic" {
		t.Fatalf("region slug: %q", regions[0].RegionSlug)
	}
}
