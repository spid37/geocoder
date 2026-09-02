package normalize

import "testing"

func TestExpandStreetType(t *testing.T) {
	if got := ExpandStreetType("st"); got != "STREET" {
		t.Fatalf("ExpandStreetType(st) = %q", got)
	}
	if got := ExpandStreetType("RD"); got != "ROAD" {
		t.Fatalf("ExpandStreetType(RD) = %q", got)
	}
}

func TestText(t *testing.T) {
	if got := Text("  melbourne  "); got != "MELBOURNE" {
		t.Fatalf("Text() = %q", got)
	}
}

func TestStreetTypeMatchValues(t *testing.T) {
	variants := StreetTypeMatchValues("St")
	got := map[string]bool{}
	for _, v := range variants {
		got[v] = true
	}
	if !got["ST"] || !got["STREET"] {
		t.Fatalf("StreetTypeMatchValues(St) = %v, want ST and STREET", variants)
	}
}
