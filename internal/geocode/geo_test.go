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
	if d < 1000 || d > 25_000 {
		t.Fatalf("distance %v m seems wrong", d)
	}
}