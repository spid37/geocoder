package geocode

import (
	"math"
	"testing"
)

func TestParseRadius(t *testing.T) {
	tests := []struct {
		in    string
		wantM float64
		ok    bool
	}{
		{"10km", 10_000, true},
		{"500m", 500, true},
		{"5mi", 5 * 1609.344, true},
		{" 2.5 KM ", 2500, true},
		{"", 0, false},
		{"0m", 0, false},
		{"10yd", 0, false},
		{"600km", 0, false},
	}
	for _, tt := range tests {
		got, err := parseRadius(tt.in)
		if tt.ok {
			if err != nil {
				t.Fatalf("%q: %v", tt.in, err)
			}
			if math.Abs(got-tt.wantM) > 0.01 {
				t.Fatalf("%q: got %v want %v", tt.in, got, tt.wantM)
			}
		} else if err == nil {
			t.Fatalf("%q: expected error", tt.in)
		}
	}
}

func TestBoundingBox(t *testing.T) {
	minLat, maxLat, minLon, maxLon := boundingBox(-37.8182, 145.0012, 10_000)
	if minLat >= maxLat || minLon >= maxLon {
		t.Fatalf("invalid box: %v %v %v %v", minLat, maxLat, minLon, maxLon)
	}
	if minLat > -37.8182 || maxLat < -37.8182 {
		t.Fatal("latitude range should contain center")
	}
}

func TestHaversineM_RichmondHawthorn(t *testing.T) {
	// fixture centroids: Richmond, Hawthorn (VIC)
	const richLat, richLon = -37.8182, 145.0012
	const hawLat, hawLon = -37.82, 145.02
	d := haversineM(richLat, richLon, hawLat, hawLon)
	if d < 1600 || d > 1750 {
		t.Fatalf("distance %v m seems wrong", d)
	}
}

func TestBoundingBox_southernLatitudeWiderLon(t *testing.T) {
	_, _, minLon1, maxLon1 := boundingBox(-37.8182, 145.0, 500_000)
	_, _, minLon2, maxLon2 := boundingBox(-43.0, 145.0, 500_000)
	span1 := maxLon1 - minLon1
	span2 := maxLon2 - minLon2
	if span2 <= span1 {
		t.Fatalf("southern box should span more longitude: %v vs %v", span2, span1)
	}
}
