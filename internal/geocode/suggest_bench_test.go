//go:build integration

package geocode_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/spid37/geocoder/internal/geocode"
	"github.com/spid37/geocoder/internal/testdata"
)

func TestIntegrationSuggestAddressesLatency(t *testing.T) {
	dbPath := os.Getenv("GEOCODER_TEST_DB")
	if dbPath == "" {
		t.Skip("set GEOCODER_TEST_DB to run integration tests against a loaded database")
	}

	svc := geocode.NewService(testdata.OpenPath(t, dbPath))
	ctx := context.Background()
	// Warm locality cache so latency checks reflect steady-state behaviour.
	if _, err := svc.SuggestAddresses(ctx, geocode.AddressSuggestQuery{Q: "rich", State: "VIC"}); err != nil {
		t.Fatal(err)
	}

	queries := []geocode.AddressSuggestQuery{
		{Q: "1 collins st melbourne vic 3000", State: "VIC"},
		{Q: "1 collins st melb", State: "VIC", Postcode: "3000"},
		{Q: "collins st melbourne", State: "VIC", Postcode: "3000"},
	}

	for _, q := range queries {
		q := q
		t.Run(q.Q, func(t *testing.T) {
			start := time.Now()
			res, err := svc.SuggestAddresses(ctx, q)
			elapsed := time.Since(start)
			if err != nil {
				t.Fatal(err)
			}
			if len(res) == 0 {
				t.Fatalf("no results")
			}
			t.Logf("%d results in %v", len(res), elapsed.Round(time.Millisecond))
			if elapsed > 500*time.Millisecond {
				t.Fatalf("too slow: %v", elapsed)
			}
		})
	}
}
