package parse

import (
	"regexp"
	"strings"

	"github.com/spid37/geocoder/internal/normalize"
)

var leadingNumber = regexp.MustCompile(`^(\d+[A-Z]?)\s+(.*)$`)

type Street struct {
	Number     string
	Name       string
	Type       string
	Raw        string
}

func StreetAddress(raw string) Street {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Street{Raw: raw}
	}

	s := Street{Raw: raw}
	m := leadingNumber.FindStringSubmatch(normalize.Text(raw))
	if m == nil {
		parts := normalize.Tokenize(raw)
		if len(parts) >= 2 {
			s.Type = normalize.ExpandStreetType(parts[len(parts)-1])
			s.Name = strings.Join(parts[:len(parts)-1], " ")
		} else if len(parts) == 1 {
			s.Name = parts[0]
		}
		return s
	}

	s.Number = m[1]
	remainder := strings.TrimSpace(m[2])
	parts := strings.Fields(remainder)
	if len(parts) == 0 {
		return s
	}

	s.Type = normalize.ExpandStreetType(parts[len(parts)-1])
	if len(parts) > 1 {
		s.Name = strings.Join(parts[:len(parts)-1], " ")
	} else {
		s.Name = parts[0]
		if normalize.IsKnownStreetType(parts[0]) {
			s.Name = ""
		} else {
			s.Type = ""
		}
	}
	return s
}
