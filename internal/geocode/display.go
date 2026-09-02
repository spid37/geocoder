package geocode

import (
	"strings"
	"unicode"
)

func finalizeResult(res *Result, sa3Name string) {
	if res == nil || res.Address == nil {
		return
	}

	a := res.Address
	a.Suburb = titleCaseWords(a.Suburb)
	a.Street = titleCaseWords(a.Street)
	a.State = strings.ToUpper(strings.TrimSpace(a.State))
	a.Number = strings.TrimSpace(a.Number)
	a.Postcode = strings.TrimSpace(a.Postcode)

	if sa3Name != "" {
		a.Region = sa3Name
	}

	res.MatchedAddress = formatMatchedAddress(a)
	if res.Accuracy == AccuracyStreet {
		res.AddressSlug = buildAddressSlug(a)
	}
	res.SuburbSlug = buildSuburbSlug(a)
	if a.Region != "" && a.State != "" {
		res.RegionSlug = buildRegionSlug(a.Region, a.State)
	}
}

func formatMatchedAddress(a *Address) string {
	var parts []string
	switch {
	case a.Number != "" && a.Street != "":
		parts = append(parts, a.Number+" "+a.Street)
	case a.Street != "":
		parts = append(parts, a.Street)
	}
	if a.Suburb != "" {
		parts = append(parts, a.Suburb)
	}
	if a.State != "" {
		parts = append(parts, a.State)
	}
	if a.Postcode != "" {
		parts = append(parts, a.Postcode)
	}
	return strings.Join(parts, " ")
}

func buildAddressSlug(a *Address) string {
	var parts []string
	if a.Number != "" {
		parts = append(parts, a.Number)
	}
	if a.Street != "" {
		parts = append(parts, a.Street)
	}
	if a.Suburb != "" {
		parts = append(parts, a.Suburb)
	}
	if a.State != "" {
		parts = append(parts, a.State)
	}
	if a.Postcode != "" {
		parts = append(parts, a.Postcode)
	}
	return joinSlugParts(parts...)
}

func buildSuburbSlug(a *Address) string {
	if a.Suburb == "" {
		return ""
	}
	return joinSlugParts(a.Suburb, a.State, a.Postcode)
}

func buildRegionSlug(region, state string) string {
	base := slugifyLower(region)
	state = strings.ToLower(strings.TrimSpace(state))
	if base == "" || state == "" {
		return base
	}
	return base + "-" + state
}

func joinSlugParts(parts ...string) string {
	var out []string
	for _, part := range parts {
		part = slugifyLower(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return strings.Join(out, "-")
}

func slugifyLower(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastHyphen := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if (r == ' ' || r == '-') && b.Len() > 0 && !lastHyphen {
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func titleCaseWords(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	words := strings.Fields(strings.ToLower(s))
	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}
