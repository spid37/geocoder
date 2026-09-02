package geocode

import "testing"

func TestDisplayAndSlugs(t *testing.T) {
	res := &Result{
		Accuracy: AccuracyStreet,
		Address: &Address{
			Number:   "42",
			Street:   "DEMO RD",
			Suburb:   "RICHMOND",
			State:    "vic",
			Postcode: "3121",
		},
	}
	finalizeResult(res, "Yarra")

	if res.Address.Suburb != "Richmond" {
		t.Fatalf("suburb: %q", res.Address.Suburb)
	}
	if res.Address.Street != "Demo Rd" {
		t.Fatalf("street: %q", res.Address.Street)
	}
	if res.Address.State != "VIC" {
		t.Fatalf("state: %q", res.Address.State)
	}
	if res.Address.Region != "Yarra" {
		t.Fatalf("region: %q", res.Address.Region)
	}
	if res.MatchedAddress != "42 Demo Rd Richmond VIC 3121" {
		t.Fatalf("matched: %q", res.MatchedAddress)
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
