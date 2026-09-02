package geocode

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/spid37/geocoder/internal/normalize"
	"github.com/spid37/geocoder/internal/parse"
)

const (
	defaultSuggestLimit = 10
	maxSuggestLimit     = 25
	minSuggestQueryLen  = 2
)

type SuburbSuggestion struct {
	Suburb     string `json:"suburb"`
	State      string `json:"state"`
	Postcode   string `json:"postcode"`
	Region     string `json:"region,omitempty"`
	SuburbSlug string `json:"suburb_slug"`
	RegionSlug string `json:"region_slug,omitempty"`
}

type RegionSuggestion struct {
	Region     string `json:"region"`
	State      string `json:"state"`
	RegionSlug string `json:"region_slug"`
}

type AddressSuggestion struct {
	Number         string `json:"number,omitempty"`
	Street         string `json:"street,omitempty"`
	Suburb         string `json:"suburb"`
	State          string `json:"state"`
	Postcode       string `json:"postcode"`
	Region         string `json:"region,omitempty"`
	MatchedAddress string `json:"matched_address"`
	AddressSlug    string `json:"address_slug"`
	SuburbSlug     string `json:"suburb_slug"`
	RegionSlug     string `json:"region_slug,omitempty"`
}

type AddressSuggestQuery struct {
	Q        string
	Suburb   string
	State    string
	Postcode string
	Limit    int
}

type addressSuggestScope struct {
	street     parse.Street
	localities []localityMatch
	state      string
	postcode   string
}

type localityMatch struct {
	nameNorm string
	state    string
	postcode string
}

const maxLocalityMatches = 25

func (s *Service) SuggestAddresses(ctx context.Context, q AddressSuggestQuery) ([]AddressSuggestion, error) {
	if len(strings.TrimSpace(q.Q)) < minSuggestQueryLen {
		return nil, fmt.Errorf("query must be at least %d characters", minSuggestQueryLen)
	}

	scope, err := s.resolveAddressSuggestScope(ctx, q)
	if err != nil {
		return nil, err
	}

	nameNorm := normalize.StreetName(scope.street.Name)
	if nameNorm == "" {
		return nil, fmt.Errorf("query must include a street name")
	}

	limit := clampLimit(q.Limit)
	out := make([]AddressSuggestion, 0, limit)

	if len(scope.localities) == 0 {
		if scope.state == "" {
			return out, nil
		}
		batch, err := s.queryAddressSuggestionsBroad(ctx, scope.street, nameNorm, scope.state, scope.postcode, limit)
		if err != nil {
			return nil, err
		}
		return batch, nil
	}

	if scope.postcode != "" {
		filtered := scope.localities[:0]
		for _, loc := range scope.localities {
			if loc.postcode == scope.postcode {
				filtered = append(filtered, loc)
			}
		}
		scope.localities = filtered
		if len(scope.localities) == 0 {
			return out, nil
		}
	}

	batch, err := s.queryAddressSuggestionsBatch(ctx, scope.street, nameNorm, scope.localities, limit)
	if err != nil {
		return nil, err
	}
	return batch, nil
}

func (s *Service) queryAddressSuggestionsBatch(ctx context.Context, st parse.Street, nameNorm string, localities []localityMatch, limit int) ([]AddressSuggestion, error) {
	out := make([]AddressSuggestion, 0, limit)
	for _, loc := range localities {
		remaining := limit - len(out)
		if remaining <= 0 {
			break
		}
		batch, err := s.queryAddressSuggestions(ctx, st, nameNorm, loc, remaining)
		if err != nil {
			return nil, err
		}
		out = append(out, batch...)
		if st.Number != "" && len(batch) > 0 {
			break
		}
	}
	return out, nil
}

func (s *Service) queryAddressSuggestions(ctx context.Context, st parse.Street, nameNorm string, loc localityMatch, limit int) ([]AddressSuggestion, error) {
	typeClause, typeArgs := streetTypeClause(st.Type)

	query := `
		SELECT number_first, street_name, street_type, locality_name, state, postcode, sa3_name
		FROM addresses
		WHERE state = ? AND postcode = ? AND locality_name_norm = ?
		  AND street_name_norm LIKE ?
		  AND ` + typeClause
	args := []any{loc.state, loc.postcode, loc.nameNorm, nameNorm + "%"}
	args = append(args, typeArgs...)
	if st.Number != "" {
		query += ` AND number_first = ?`
		args = append(args, st.Number)
	}
	query += ` ORDER BY CAST(number_first AS INTEGER), number_first, street_name_norm LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AddressSuggestion, 0, limit)
	for rows.Next() {
		var number, streetName, streetType, locality, st, pc string
		var sa3Name sql.NullString
		if err := rows.Scan(&number, &streetName, &streetType, &locality, &st, &pc, &sa3Name); err != nil {
			return nil, err
		}
		out = append(out, formatAddressSuggestion(number, streetName, streetType, locality, st, pc, sa3Name.String))
	}
	return out, rows.Err()
}

func (s *Service) queryAddressSuggestionsBroad(ctx context.Context, st parse.Street, nameNorm, state, postcode string, limit int) ([]AddressSuggestion, error) {
	typeClause, typeArgs := streetTypeClause(st.Type)

	query := `
		SELECT number_first, street_name, street_type, locality_name, state, postcode, sa3_name
		FROM addresses
		WHERE state = ? AND street_name_norm LIKE ?
		  AND ` + typeClause
	args := []any{state, nameNorm + "%"}
	args = append(args, typeArgs...)
	if postcode != "" {
		query += ` AND postcode = ?`
		args = append(args, postcode)
	}
	if st.Number != "" {
		query += ` AND number_first = ?`
		args = append(args, st.Number)
	}
	query += ` ORDER BY locality_name_norm, CAST(number_first AS INTEGER), number_first LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AddressSuggestion, 0, limit)
	for rows.Next() {
		var number, streetName, streetType, locality, stName, pc string
		var sa3Name sql.NullString
		if err := rows.Scan(&number, &streetName, &streetType, &locality, &stName, &pc, &sa3Name); err != nil {
			return nil, err
		}
		out = append(out, formatAddressSuggestion(number, streetName, streetType, locality, stName, pc, sa3Name.String))
	}
	return out, rows.Err()
}

func (s *Service) resolveAddressSuggestScope(ctx context.Context, q AddressSuggestQuery) (addressSuggestScope, error) {
	q.Suburb = normalize.Locality(q.Suburb)
	q.State = normalize.State(q.State)
	q.Postcode = normalize.Postcode(q.Postcode)

	if q.Suburb != "" && q.State != "" && q.Postcode != "" {
		return addressSuggestScope{
			street: parse.StreetAddress(q.Q),
			localities: []localityMatch{{
				nameNorm: q.Suburb,
				state:    q.State,
				postcode: q.Postcode,
			}},
			state:    q.State,
			postcode: q.Postcode,
		}, nil
	}

	tokens, state, postcode := tokenizeAddressQuery(q.Q)
	if q.State != "" {
		state = q.State
	}
	if q.Postcode != "" {
		postcode = q.Postcode
	}

	streetText := q.Q
	var localities []localityMatch
	if q.Suburb != "" {
		var err error
		localities, err = s.findMatchingLocalities(ctx, normalize.Tokenize(q.Suburb), state)
		if err != nil {
			return addressSuggestScope{}, err
		}
	} else if streetTokens, matches, ok := s.longestLocalitySuffix(ctx, tokens, state); ok {
		localities = matches
		streetText = strings.Join(streetTokens, " ")
	}

	return addressSuggestScope{
		street:     parse.StreetAddress(streetText),
		localities: localities,
		state:      state,
		postcode:   postcode,
	}, nil
}

func (s *Service) longestLocalitySuffix(ctx context.Context, tokens []string, state string) (streetTokens []string, localities []localityMatch, ok bool) {
	if len(tokens) < 2 {
		return tokens, nil, false
	}

	maxSuffix := len(tokens) - 1
	if maxSuffix > 4 {
		maxSuffix = 4
	}
	for suffixLen := maxSuffix; suffixLen >= 1; suffixLen-- {
		suffix := tokens[len(tokens)-suffixLen:]
		matches, err := s.findMatchingLocalities(ctx, suffix, state)
		if err != nil || len(matches) == 0 {
			continue
		}
		return tokens[:len(tokens)-suffixLen], matches, true
	}
	return tokens, nil, false
}

func (s *Service) findMatchingLocalities(ctx context.Context, tokens []string, state string) ([]localityMatch, error) {
	if len(tokens) == 0 {
		return nil, nil
	}
	for _, t := range tokens {
		if len(t) < minSuggestQueryLen {
			return nil, nil
		}
	}

	localities, err := s.loadLocalities(ctx)
	if err != nil {
		return nil, err
	}

	var out []localityMatch
	for _, loc := range localities {
		if state != "" && loc.state != state {
			continue
		}
		if !matchLocalityTokens(tokens, loc.nameNorm) {
			continue
		}
		out = append(out, loc)
		if len(out) >= maxLocalityMatches {
			break
		}
	}
	return out, nil
}

func (s *Service) loadLocalities(ctx context.Context) ([]localityMatch, error) {
	s.localitiesOnce.Do(func() {
		rows, err := s.db.QueryContext(ctx, `
			SELECT locality_name_norm, state, postcode
			FROM locality_centroids
			ORDER BY address_count DESC, locality_name_norm, postcode`)
		if err != nil {
			s.localitiesErr = err
			return
		}
		defer rows.Close()

		for rows.Next() {
			var loc localityMatch
			if err := rows.Scan(&loc.nameNorm, &loc.state, &loc.postcode); err != nil {
				s.localitiesErr = err
				return
			}
			s.localities = append(s.localities, loc)
		}
		s.localitiesErr = rows.Err()
	})
	return s.localities, s.localitiesErr
}

// matchLocalityTokens reports whether query tokens match a locality name,
// allowing partial word prefixes and common abbreviations (e.g. "MT" → "MOUNT").
func matchLocalityTokens(queryTokens []string, localityNorm string) bool {
	localityWords := strings.Fields(localityNorm)
	if len(queryTokens) == 0 || len(localityWords) == 0 {
		return false
	}

	if len(queryTokens) == 1 {
		qt := expandLocalityWord(queryTokens[0])
		if strings.HasPrefix(localityNorm, qt) {
			return true
		}
		for _, word := range localityWords {
			if strings.HasPrefix(word, qt) {
				return true
			}
		}
		return false
	}

	if len(queryTokens) > len(localityWords) {
		return false
	}
	for i, qt := range queryTokens {
		if !strings.HasPrefix(localityWords[i], expandLocalityWord(qt)) {
			return false
		}
	}
	return true
}

var localityWordAbbrev = map[string]string{
	// Australia Post AS4590 placename abbreviations; expanded to G-NAF canonical forms.
	"MT":    "MOUNT",
	"SAINT": "ST", // G-NAF stores ST KILDA, not SAINT KILDA
	"PT":    "POINT",
	"PNT":   "POINT",
	"NTH":   "NORTH",
	"STH":   "SOUTH",
	"UPR":   "UPPER",
	"LWR":   "LOWER",
	"FT":    "FORT",
}

func expandLocalityWord(word string) string {
	word = normalize.Text(word)
	if expanded, ok := localityWordAbbrev[word]; ok {
		return expanded
	}
	return word
}

func tokenizeAddressQuery(q string) (tokens []string, state, postcode string) {
	tokens = normalize.Tokenize(q)
	if len(tokens) == 0 {
		return nil, "", ""
	}
	if isPostcodeToken(tokens[len(tokens)-1]) {
		postcode = tokens[len(tokens)-1]
		tokens = tokens[:len(tokens)-1]
	}
	if len(tokens) > 0 && isStateToken(tokens[len(tokens)-1]) {
		state = tokens[len(tokens)-1]
		tokens = tokens[:len(tokens)-1]
	}
	return tokens, state, postcode
}

var auStates = map[string]struct{}{
	"ACT": {}, "NSW": {}, "NT": {}, "QLD": {}, "SA": {}, "TAS": {}, "VIC": {}, "WA": {},
}

func isStateToken(s string) bool {
	_, ok := auStates[normalize.State(s)]
	return ok
}

func isPostcodeToken(s string) bool {
	if len(s) != 4 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (s *Service) SuggestSuburbs(ctx context.Context, query, state string, limit int) ([]SuburbSuggestion, error) {
	query = normalize.Locality(query)
	state = normalize.State(state)
	if len(query) < minSuggestQueryLen {
		return nil, fmt.Errorf("query must be at least %d characters", minSuggestQueryLen)
	}
	limit = clampLimit(limit)

	prefix := query + "%"
	word := "% " + query + "%"

	rows, err := s.db.QueryContext(ctx, `
		SELECT locality_name, state, postcode, sa3_name
		FROM locality_centroids
		WHERE (locality_name_norm LIKE ? OR locality_name_norm LIKE ?)
		  AND (? = '' OR state = ?)
		ORDER BY address_count DESC, locality_name_norm, postcode
		LIMIT ?`,
		prefix, word, state, state, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SuburbSuggestion
	for rows.Next() {
		var locality, st, postcode string
		var sa3Name sql.NullString
		if err := rows.Scan(&locality, &st, &postcode, &sa3Name); err != nil {
			return nil, err
		}
		out = append(out, formatSuburbSuggestion(locality, st, postcode, sa3Name.String))
	}
	return out, rows.Err()
}

func (s *Service) SuggestRegions(ctx context.Context, query, state string, limit int) ([]RegionSuggestion, error) {
	query = strings.TrimSpace(query)
	state = normalize.State(state)
	if len(query) < minSuggestQueryLen {
		return nil, fmt.Errorf("query must be at least %d characters", minSuggestQueryLen)
	}
	limit = clampLimit(limit)

	prefix := strings.ToUpper(query) + "%"
	word := "% " + strings.ToUpper(query) + "%"

	rows, err := s.db.QueryContext(ctx, `
		SELECT sa3_name, state
		FROM locality_centroids
		WHERE sa3_name IS NOT NULL AND sa3_name != ''
		  AND (UPPER(sa3_name) LIKE ? OR UPPER(sa3_name) LIKE ?)
		  AND (? = '' OR state = ?)
		GROUP BY sa3_name, state
		ORDER BY SUM(address_count) DESC, sa3_name
		LIMIT ?`,
		prefix, word, state, state, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RegionSuggestion
	for rows.Next() {
		var region, st string
		if err := rows.Scan(&region, &st); err != nil {
			return nil, err
		}
		out = append(out, formatRegionSuggestion(region, st))
	}
	return out, rows.Err()
}

func formatSuburbSuggestion(locality, state, postcode, sa3Name string) SuburbSuggestion {
	addr := &Address{
		Suburb:   titleCaseWords(locality),
		State:    strings.ToUpper(strings.TrimSpace(state)),
		Postcode: strings.TrimSpace(postcode),
		Region:   sa3Name,
	}
	sug := SuburbSuggestion{
		Suburb:     addr.Suburb,
		State:      addr.State,
		Postcode:   addr.Postcode,
		SuburbSlug: buildSuburbSlug(addr),
	}
	if sa3Name != "" {
		sug.Region = sa3Name
		sug.RegionSlug = buildRegionSlug(sa3Name, addr.State)
	}
	return sug
}

func formatRegionSuggestion(region, state string) RegionSuggestion {
	st := strings.ToUpper(strings.TrimSpace(state))
	return RegionSuggestion{
		Region:     region,
		State:      st,
		RegionSlug: buildRegionSlug(region, st),
	}
}

func formatAddressSuggestion(number, streetName, streetType, locality, state, postcode, sa3Name string) AddressSuggestion {
	addr := &Address{
		Number:   strings.TrimSpace(number),
		Street:   formatStreet(streetName, streetType),
		Suburb:   titleCaseWords(locality),
		State:    strings.ToUpper(strings.TrimSpace(state)),
		Postcode: strings.TrimSpace(postcode),
		Region:   sa3Name,
	}
	addr.Street = titleCaseWords(addr.Street)

	sug := AddressSuggestion{
		Number:         addr.Number,
		Street:         addr.Street,
		Suburb:         addr.Suburb,
		State:          addr.State,
		Postcode:       addr.Postcode,
		MatchedAddress: formatMatchedAddress(addr),
		AddressSlug:    buildAddressSlug(addr),
		SuburbSlug:     buildSuburbSlug(addr),
	}
	if sa3Name != "" {
		sug.Region = sa3Name
		sug.RegionSlug = buildRegionSlug(sa3Name, addr.State)
	}
	return sug
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return defaultSuggestLimit
	}
	if limit > maxSuggestLimit {
		return maxSuggestLimit
	}
	return limit
}
