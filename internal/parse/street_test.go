package parse

import "testing"

func TestStreetAddress(t *testing.T) {
	tests := []struct {
		input  string
		number string
		name   string
		typ    string
	}{
		{"123 George St", "123", "GEORGE", "STREET"},
		{"42 Smith Road", "42", "SMITH", "ROAD"},
		{"1A Main Avenue", "1A", "MAIN", "AVENUE"},
		{"Lot 5 Example Avenue", "", "LOT 5 EXAMPLE", "AVENUE"},
		{"George Street", "", "GEORGE", "STREET"},
		{"34 Demo Rd", "34", "DEMO", "ROAD"},
		{"34 demo", "34", "DEMO", ""},
		{"34 ST", "34", "", "STREET"},
	}

	for _, tt := range tests {
		got := StreetAddress(tt.input)
		if got.Number != tt.number || got.Name != tt.name || got.Type != tt.typ {
			t.Errorf("StreetAddress(%q) = %+v, want number=%q name=%q type=%q", tt.input, got, tt.number, tt.name, tt.typ)
		}
	}
}
