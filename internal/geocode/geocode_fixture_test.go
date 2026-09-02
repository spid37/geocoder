package geocode_test

import (
	"context"
	"testing"

	"github.com/spid37/geocoder/internal/geocode"
	"github.com/spid37/geocoder/internal/testdata"
)

func TestGeocodeFixtureRichmond(t *testing.T) {
	svc := geocode.NewService(testdata.OpenDB(t))

	res, err := svc.Geocode(context.Background(), geocode.Query{
		Street: "42 Demo Rd", Suburb: "Richmond", State: "VIC", Postcode: "3121",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("expected result")
	}

	if res.Accuracy != geocode.AccuracyStreet {
		t.Fatalf("accuracy: %s", res.Accuracy)
	}
	if res.AddressDetailPID != "GAVIC999000001" {
		t.Fatalf("pid: %s", res.AddressDetailPID)
	}
	if !floatClose(res.Latitude, -37.8182) || !floatClose(res.Longitude, 145.0012) {
		t.Fatalf("coords: %f, %f", res.Latitude, res.Longitude)
	}
	if res.MatchedAddress != "42 Demo Rd Richmond VIC 3121" {
		t.Fatalf("matched: %q", res.MatchedAddress)
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
}

func TestGeocodeFixtureStreetTypeAlias(t *testing.T) {
	svc := geocode.NewService(testdata.OpenDB(t))

	res, err := svc.Geocode(context.Background(), geocode.Query{
		Street: "42 Demo Road", Suburb: "Richmond", State: "VIC", Postcode: "3121",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.AddressDetailPID != "GAVIC999000001" {
		t.Fatalf("road alias: %+v", res)
	}
}

func TestGeocodeFixtureMelbourne(t *testing.T) {
	svc := geocode.NewService(testdata.OpenDB(t))

	res, err := svc.Geocode(context.Background(), geocode.Query{
		Street: "1 Collins St", Suburb: "Melbourne", State: "VIC", Postcode: "3000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.AddressDetailPID != "GAVIC424463642" {
		t.Fatalf("melbourne: %+v", res)
	}
	if res.Address.Region != "Melbourne City" {
		t.Fatalf("region: %q", res.Address.Region)
	}
}

func TestGeocodeFixtureSuburbFallback(t *testing.T) {
	svc := geocode.NewService(testdata.OpenDB(t))

	res, err := svc.Geocode(context.Background(), geocode.Query{
		Suburb: "Richmond", State: "VIC", Postcode: "3121",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.Accuracy != geocode.AccuracySuburb {
		t.Fatalf("suburb: %+v", res)
	}
	if res.AddressSlug != "" {
		t.Fatalf("address slug should be empty: %q", res.AddressSlug)
	}
	if !floatClose(res.Latitude, -37.8182) {
		t.Fatalf("lat: %f", res.Latitude)
	}
}

func TestGeocodeFixtureNotFound(t *testing.T) {
	svc := geocode.NewService(testdata.OpenDB(t))

	res, err := svc.Geocode(context.Background(), geocode.Query{
		Suburb: "Nowhere", State: "NSW", Postcode: "2000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res != nil {
		t.Fatalf("expected nil, got %+v", res)
	}
}

func TestSuggestAddressesFixture(t *testing.T) {
	svc := geocode.NewService(testdata.OpenDB(t))
	ctx := context.Background()
	scope := geocode.AddressSuggestQuery{Suburb: "Richmond", State: "VIC", Postcode: "3121"}

	suggestions, err := svc.SuggestAddresses(ctx, geocode.AddressSuggestQuery{Q: "42 demo", Suburb: scope.Suburb, State: scope.State, Postcode: scope.Postcode})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("42 demo: %+v", suggestions)
	}
	if suggestions[0].AddressSlug != "42-demo-rd-richmond-vic-3121" {
		t.Fatalf("slug: %q", suggestions[0].AddressSlug)
	}

	suggestions, err = svc.SuggestAddresses(ctx, geocode.AddressSuggestQuery{Q: "42 demo rd rich", State: "VIC"})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("42 demo rd rich: %+v", suggestions)
	}
	if suggestions[0].AddressSlug != "42-demo-rd-richmond-vic-3121" {
		t.Fatalf("free-text slug: %q", suggestions[0].AddressSlug)
	}

	suggestions, err = svc.SuggestAddresses(ctx, geocode.AddressSuggestQuery{Q: "42 demo rd richmond", State: "VIC"})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("42 demo rd richmond: %+v", suggestions)
	}
	if suggestions[0].AddressSlug != "42-demo-rd-richmond-vic-3121" {
		t.Fatalf("full suburb slug: %q", suggestions[0].AddressSlug)
	}

	suggestions, err = svc.SuggestAddresses(ctx, geocode.AddressSuggestQuery{Q: "demo", Suburb: scope.Suburb, State: scope.State, Postcode: scope.Postcode})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 3 {
		t.Fatalf("demo: %+v", suggestions)
	}

	suggestions, err = svc.SuggestAddresses(ctx, geocode.AddressSuggestQuery{Q: "demo road", Suburb: scope.Suburb, State: scope.State, Postcode: scope.Postcode})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 3 {
		t.Fatalf("demo road: %+v", suggestions)
	}

	suggestions, err = svc.SuggestAddresses(ctx, geocode.AddressSuggestQuery{Q: "demo", Suburb: scope.Suburb, State: scope.State})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 3 {
		t.Fatalf("demo with suburb+state: %+v", suggestions)
	}
}

func TestSuggestFixtureOrdering(t *testing.T) {
	svc := geocode.NewService(testdata.OpenDB(t))

	suggestions, err := svc.SuggestSuburbs(context.Background(), "mount", "VIC", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) < 3 {
		t.Fatalf("suggestions: %+v", suggestions)
	}
	if suggestions[0].Suburb != "Mount Waverley" {
		t.Fatalf("first: %+v", suggestions[0])
	}
	if suggestions[1].Suburb != "Mount Martha" {
		t.Fatalf("second: %+v", suggestions[1])
	}
	if suggestions[2].Suburb != "Mount Evelyn" {
		t.Fatalf("third: %+v", suggestions[2])
	}
}
