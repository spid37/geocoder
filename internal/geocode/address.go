package geocode

import "strings"

type Address struct {
	Number   string `json:"number,omitempty"`
	Street   string `json:"street,omitempty"`
	Suburb   string `json:"suburb,omitempty"`
	State    string `json:"state,omitempty"`
	Postcode string `json:"postcode,omitempty"`
	Region   string `json:"region,omitempty"`
}

func formatStreet(name, typ string) string {
	name = strings.TrimSpace(name)
	typ = strings.TrimSpace(typ)
	switch {
	case name != "" && typ != "":
		return name + " " + typ
	case name != "":
		return name
	default:
		return typ
	}
}
