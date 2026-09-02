package geocode

import "testing"

func TestMatchLocalityTokens(t *testing.T) {
	tests := []struct {
		tokens   []string
		locality string
		want     bool
	}{
		{[]string{"MOUNT"}, "MOUNT WAVERLEY", true},
		{[]string{"MT"}, "MOUNT WAVERLEY", true},
		{[]string{"MT", "WAVERLEY"}, "MOUNT WAVERLEY", true},
		{[]string{"WAVERLEY"}, "MOUNT WAVERLEY", true},
		{[]string{"MT"}, "MITCHAM", false},
		{[]string{"MORNING"}, "MORNINGTON", true},
		{[]string{"ST"}, "ST KILDA", true},
		{[]string{"SAINT"}, "ST KILDA", true},
		{[]string{"NTH"}, "NORTH MELBOURNE", true},
		{[]string{"STH"}, "SOUTH YARRA", true},
		{[]string{"UPR"}, "UPPER FERNTREE GULLY", true},
		{[]string{"LWR"}, "LOWER PLENTY", true},
		{[]string{"PT"}, "POINT COOK", true},
	}

	for _, tt := range tests {
		got := matchLocalityTokens(tt.tokens, tt.locality)
		if got != tt.want {
			t.Errorf("matchLocalityTokens(%v, %q) = %v, want %v", tt.tokens, tt.locality, got, tt.want)
		}
	}
}
