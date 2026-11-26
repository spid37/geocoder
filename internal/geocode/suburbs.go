package geocode

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/spid37/geocoder/internal/normalize"
)

var ErrRegionsNotLoaded = errors.New("region data not loaded; run: geocoder regions load")

const (
	defaultSuburbListLimit = 100
	maxSuburbListLimit     = 100
)

type SuburbsByRegionResponse struct {
	RegionSlug string             `json:"region_slug"`
	Suburbs    []SuburbSuggestion `json:"suburbs"`
}

func (s *Service) ListSuburbsByRegion(ctx context.Context, regionSlug, stateFilter string, limit int) (*SuburbsByRegionResponse, error) {
	regionSlug = strings.TrimSpace(regionSlug)
	if regionSlug == "" {
		return nil, fmt.Errorf("region_slug is required")
	}

	stateFilter = normalize.State(stateFilter)
	limit = clampSuburbListLimit(limit)

	loaded, err := s.regionsDataLoaded(ctx)
	if err != nil {
		return nil, err
	}
	if !loaded {
		return nil, ErrRegionsNotLoaded
	}

	sa3Name, state, found, err := s.resolveSA3FromRegionSlug(ctx, regionSlug)
	if err != nil {
		return nil, err
	}
	if !found {
		return &SuburbsByRegionResponse{RegionSlug: regionSlug, Suburbs: []SuburbSuggestion{}}, nil
	}

	if stateFilter != "" && stateFilter != state {
		return &SuburbsByRegionResponse{RegionSlug: regionSlug, Suburbs: []SuburbSuggestion{}}, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT locality_name, state, postcode, sa3_name
		FROM locality_centroids
		WHERE sa3_name = ? AND state = ?
		ORDER BY locality_name_norm, postcode
		LIMIT ?`,
		sa3Name, state, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SuburbSuggestion
	for rows.Next() {
		var locality, st, postcode string
		var sa3 sql.NullString
		if err := rows.Scan(&locality, &st, &postcode, &sa3); err != nil {
			return nil, err
		}
		out = append(out, formatSuburbSuggestion(locality, st, postcode, sa3.String))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []SuburbSuggestion{}
	}

	return &SuburbsByRegionResponse{RegionSlug: regionSlug, Suburbs: out}, nil
}

func (s *Service) regionsDataLoaded(ctx context.Context) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `
		SELECT 1 FROM locality_centroids
		WHERE sa3_name IS NOT NULL AND sa3_name != ''
		LIMIT 1`).Scan(&one)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}

	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM locality_centroids`).Scan(&count); err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	return true, nil
}

func (s *Service) resolveSA3FromRegionSlug(ctx context.Context, regionSlug string) (sa3Name, state string, found bool, err error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT sa3_name, state
		FROM locality_centroids
		WHERE sa3_name IS NOT NULL AND sa3_name != ''`)
	if err != nil {
		return "", "", false, err
	}
	defer rows.Close()

	for rows.Next() {
		var sa3, st string
		if err := rows.Scan(&sa3, &st); err != nil {
			return "", "", false, err
		}
		if buildRegionSlug(sa3, st) == regionSlug {
			return sa3, strings.ToUpper(strings.TrimSpace(st)), true, nil
		}
	}
	return "", "", false, rows.Err()
}

func clampSuburbListLimit(limit int) int {
	if limit <= 0 {
		return defaultSuburbListLimit
	}
	if limit > maxSuburbListLimit {
		return maxSuburbListLimit
	}
	return limit
}