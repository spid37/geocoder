package normalize

import (
	"strings"
)

var streetTypeAbbrev = map[string]string{
	"ST":     "STREET",
	"STR":    "STREET",
	"RD":     "ROAD",
	"AVE":    "AVENUE",
	"AV":     "AVENUE",
	"DR":     "DRIVE",
	"CT":     "COURT",
	"CRT":    "COURT",
	"PL":     "PLACE",
	"LN":     "LANE",
	"LANE":   "LANE",
	"TCE":    "TERRACE",
	"TER":    "TERRACE",
	"CRES":   "CRESCENT",
	"CR":     "CRESCENT",
	"PDE":    "PARADE",
	"PAR":    "PARADE",
	"HWY":    "HIGHWAY",
	"HWAY":   "HIGHWAY",
	"CL":     "CLOSE",
	"GR":     "GROVE",
	"GRV":    "GROVE",
	"BLVD":   "BOULEVARD",
	"BVD":    "BOULEVARD",
	"ESP":    "ESPLANADE",
	"SQ":     "SQUARE",
	"WK":     "WALK",
	"WAY":    "WAY",
	"CCT":    "CIRCUIT",
	"CH":     "CHASE",
	"MEWS":   "MEWS",
	"ROW":    "ROW",
	"TRACK":  "TRACK",
	"TRK":    "TRACK",
}

func Text(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	return strings.ToUpper(s)
}

func Locality(s string) string {
	return Text(s)
}

func State(s string) string {
	return Text(s)
}

func Postcode(s string) string {
	return strings.TrimSpace(s)
}

func ExpandStreetType(streetType string) string {
	t := Text(streetType)
	if expanded, ok := streetTypeAbbrev[t]; ok {
		return expanded
	}
	return t
}

// IsKnownStreetType reports whether s is a recognized street type token (abbreviation or full name).
func IsKnownStreetType(s string) bool {
	t := Text(s)
	if _, ok := streetTypeAbbrev[t]; ok {
		return true
	}
	if _, ok := streetTypeToGNAF[t]; ok {
		return true
	}
	for _, expanded := range streetTypeAbbrev {
		if expanded == t {
			return true
		}
	}
	return false
}


// streetTypeToGNAF maps expanded/common names to G-NAF authority type names.
var streetTypeToGNAF = map[string]string{
	"STREET":    "ST",
	"ROAD":      "RD",
	"AVENUE":    "AV",
	"DRIVE":     "DR",
	"COURT":     "CT",
	"CRESCENT":  "CR",
	"PLACE":     "PL",
	"PARADE":    "PDE",
	"HIGHWAY":   "HWY",
	"CLOSE":     "CL",
	"GROVE":     "GR",
	"BOULEVARD": "BVD",
	"TERRACE":   "TCE",
	"CIRCUIT":   "CCT",
	"ESPLANADE": "ESP",
	"SQUARE":    "SQ",
	"WALK":      "WALK",
	"LANE":      "LANE",
	"WAY":       "WAY",
}

// StreetTypeNorm stores G-NAF authority abbreviations (ST, RD) in the database.
func StreetTypeNorm(streetType string) string {
	t := Text(streetType)
	if gnaf, ok := streetTypeToGNAF[t]; ok {
		return gnaf
	}
	return t
}

// StreetTypeMatchValues returns normalized street type tokens that should match G-NAF records.
func StreetTypeMatchValues(streetType string) []string {
	t := Text(streetType)
	if t == "" {
		return nil
	}

	seen := make(map[string]struct{}, 4)
	add := func(v string) {
		if v != "" {
			seen[v] = struct{}{}
		}
	}

	add(t)
	if expanded, ok := streetTypeAbbrev[t]; ok {
		add(expanded)
	}
	if gnaf, ok := streetTypeToGNAF[t]; ok {
		add(gnaf)
	}
	if expanded := ExpandStreetType(t); expanded != t {
		add(expanded)
		if gnaf, ok := streetTypeToGNAF[expanded]; ok {
			add(gnaf)
		}
	}

	out := make([]string, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	return out
}

func StreetName(s string) string {
	return Text(s)
}

func Tokenize(s string) []string {
	return strings.Fields(Text(s))
}
