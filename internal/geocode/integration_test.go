//go:build integration

package geocode_test

import (
	"context"
	"os"
	"testing"

	"github.com/spid37/geocoder/internal/geocode"
	"github.com/spid37/geocoder/internal/testdata"
)

func TestIntegrationMelbourne(t *testing.T) {
	dbPath := os.Getenv("GEOCODER_TEST_DB")
	if dbPath == "" {
		t.Skip("set GEOCODER_TEST_DB to run integration tests against a loaded database")
	}

	svc := geocode.NewService(testdata.OpenPath(t, dbPath))
	res, err := svc.Geocode(context.Background(), geocode.Query{
		Street: "1 Collins St", Suburb: "Melbourne", State: "VIC", Postcode: "3000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("expected result")
	}
	if res.AddressDetailPID != "GAVIC424463642" {
		t.Fatalf("pid: %s", res.AddressDetailPID)
	}
	if !floatClose(res.Latitude, -37.81363721) || !floatClose(res.Longitude, 144.97361666) {
		t.Fatalf("coords: %f, %f", res.Latitude, res.Longitude)
	}
	if res.Address.Region != "Melbourne City" {
		t.Fatalf("region: %q", res.Address.Region)
	}
}
