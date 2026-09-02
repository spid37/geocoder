package geocode

import (
	"context"
	"database/sql"
	"strings"
	"sync"

	"github.com/spid37/geocoder/internal/normalize"
	"github.com/spid37/geocoder/internal/parse"
)

type Query struct {
	Street   string
	Suburb   string
	State    string
	Postcode string
}

type Result struct {
	Latitude         float64  `json:"latitude"`
	Longitude        float64  `json:"longitude"`
	Accuracy         Accuracy `json:"accuracy"`
	MatchedAddress   string   `json:"matched_address,omitempty"`
	AddressDetailPID string   `json:"address_detail_pid,omitempty"`
	Address          *Address `json:"address,omitempty"`
	AddressSlug      string   `json:"address_slug,omitempty"`
	SuburbSlug       string   `json:"suburb_slug,omitempty"`
	RegionSlug       string   `json:"region_slug,omitempty"`
}

type Service struct {
	db             *sql.DB
	localities     []localityMatch
	localitiesOnce sync.Once
	localitiesErr  error
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Geocode(ctx context.Context, q Query) (*Result, error) {
	q.Suburb = normalize.Locality(q.Suburb)
	q.State = normalize.State(q.State)
	q.Postcode = normalize.Postcode(q.Postcode)

	if q.Street != "" && q.Suburb != "" && q.State != "" && q.Postcode != "" {
		if res, err := s.matchStreet(ctx, q); err != nil {
			return nil, err
		} else if res != nil {
			return res, nil
		}
	}

	if q.Suburb != "" && q.State != "" && q.Postcode != "" {
		if res, err := s.matchLocality(ctx, q); err != nil {
			return nil, err
		} else if res != nil {
			return res, nil
		}
	}

	if q.State != "" && q.Postcode != "" {
		if res, err := s.matchPostcode(ctx, q); err != nil {
			return nil, err
		} else if res != nil {
			return res, nil
		}
	}

	if q.State != "" {
		if res, err := s.matchState(ctx, q); err != nil {
			return nil, err
		} else if res != nil {
			return res, nil
		}
	}

	return nil, nil
}

func (s *Service) matchStreet(ctx context.Context, q Query) (*Result, error) {
	st := parse.StreetAddress(q.Street)
	nameNorm := normalize.StreetName(st.Name)
	typeClause, typeArgs := streetTypeClause(st.Type)

	row := s.db.QueryRowContext(ctx, `
		SELECT address_detail_pid, latitude, longitude,
		       number_first, street_name, street_type, locality_name, state, postcode,
		       sa3_name
		FROM addresses
		WHERE state = ? AND postcode = ? AND locality_name_norm = ?
		  AND street_name_norm = ? AND `+typeClause+`
		  AND number_first = ?
		LIMIT 1`,
		append([]any{q.State, q.Postcode, q.Suburb, nameNorm}, append(typeArgs, st.Number)...)...,
	)

	var pid, number, streetName, streetType, locality, state, postcode string
	var sa3Name sql.NullString
	var lat, lon float64
	err := row.Scan(&pid, &lat, &lon, &number, &streetName, &streetType, &locality, &state, &postcode, &sa3Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	res := &Result{
		Latitude:         lat,
		Longitude:        lon,
		Accuracy:         AccuracyStreet,
		AddressDetailPID: pid,
		Address: &Address{
			Number:   number,
			Street:   formatStreet(streetName, streetType),
			Suburb:   locality,
			State:    state,
			Postcode: postcode,
		},
	}
	finalizeResult(res, sa3Name.String)
	return res, nil
}

func (s *Service) matchLocality(ctx context.Context, q Query) (*Result, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT locality_name, latitude, longitude, sa3_name
		FROM locality_centroids
		WHERE state = ? AND postcode = ? AND locality_name_norm = ?
		LIMIT 1`,
		q.State, q.Postcode, q.Suburb,
	)

	var locality string
	var sa3Name sql.NullString
	var lat, lon float64
	err := row.Scan(&locality, &lat, &lon, &sa3Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	res := &Result{
		Latitude:       lat,
		Longitude:      lon,
		Accuracy:       AccuracySuburb,
		Address: &Address{
			Suburb:   locality,
			State:    q.State,
			Postcode: q.Postcode,
		},
	}
	finalizeResult(res, sa3Name.String)
	return res, nil
}

func (s *Service) matchPostcode(ctx context.Context, q Query) (*Result, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT postcode, latitude, longitude, sa3_name
		FROM postcode_centroids
		WHERE state = ? AND postcode = ?
		LIMIT 1`,
		q.State, q.Postcode,
	)

	var postcode string
	var sa3Name sql.NullString
	var lat, lon float64
	err := row.Scan(&postcode, &lat, &lon, &sa3Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	res := &Result{
		Latitude:   lat,
		Longitude:  lon,
		Accuracy:   AccuracyPostcode,
		Address: &Address{
			State:    q.State,
			Postcode: postcode,
		},
	}
	finalizeResult(res, sa3Name.String)
	return res, nil
}

func (s *Service) matchState(ctx context.Context, q Query) (*Result, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT state, latitude, longitude, sa3_name
		FROM state_centroids
		WHERE state = ?
		LIMIT 1`,
		q.State,
	)

	var state string
	var sa3Name sql.NullString
	var lat, lon float64
	err := row.Scan(&state, &lat, &lon, &sa3Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	res := &Result{
		Latitude:  lat,
		Longitude: lon,
		Accuracy:  AccuracyState,
		Address: &Address{
			State: state,
		},
	}
	finalizeResult(res, sa3Name.String)
	return res, nil
}

func streetTypeClause(streetType string) (string, []any) {
	variants := normalize.StreetTypeMatchValues(streetType)
	if len(variants) == 0 {
		return `1=1`, nil
	}

	placeholders := make([]string, len(variants))
	args := make([]any, len(variants))
	for i, v := range variants {
		placeholders[i] = "?"
		args[i] = v
	}
	return `street_type_norm IN (` + strings.Join(placeholders, ", ") + `)`, args
}
