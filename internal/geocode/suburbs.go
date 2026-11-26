package geocode

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/spid37/geocoder/internal/normalize"
)

var (
	ErrRegionsNotLoaded   = errors.New("region data not loaded; run: geocoder regions load")
	ErrRegionSlugRequired = errors.New("region_slug is required")
)

const (
	defaultSuburbListLimit = 100
	maxSuburbListLimit     = 500
)

type SuburbsByRegionResponse struct {
	RegionSlug string             `json:"region_slug"`
	Suburbs    []SuburbSuggestion `json:"suburbs"`
	Truncated  bool               `json:"truncated,omitempty"`
}

func (s *Service) ListSuburbsByRegion(ctx context.Context, regionSlug, stateFilter string, limit int) (*SuburbsByRegionResponse, error) {
	regionSlug = strings.ToLower(strings.TrimSpace(regionSlug))
	if regionSlug == "" {
		return nil, ErrRegionSlugRequired
	}

	stateFilter = normalize.State(stateFilter)
	limit = clampSuburbListLimit(limit)

	catalog, err := s.loadRegionCatalog()
	if err != nil {
		return nil, err
	}

	ref, found := catalog.bySlug[regionSlug]
	if !found {
		return &SuburbsByRegionResponse{RegionSlug: regionSlug, Suburbs: []SuburbSuggestion{}}, nil
	}

	if stateFilter != "" && stateFilter != normalize.State(ref.state) {
		return &SuburbsByRegionResponse{RegionSlug: regionSlug, Suburbs: []SuburbSuggestion{}}, nil
	}

	fetchLimit := limit + 1
	rows, err := s.db.QueryContext(ctx, `
		SELECT locality_name, state, postcode, sa3_name
		FROM locality_centroids
		WHERE sa3_name = ? AND state = ?
		ORDER BY locality_name_norm, postcode
		LIMIT ?`,
		ref.sa3Name, ref.state, fetchLimit,
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

	truncated := len(out) > limit
	if truncated {
		out = out[:limit]
	}

	return &SuburbsByRegionResponse{
		RegionSlug: regionSlug,
		Suburbs:    out,
		Truncated:  truncated,
	}, nil
}

func (s *Service) loadRegionCatalog() (*regionCatalog, error) {
	s.regionCatalogOnce.Do(func() {
		initCtx := context.Background()

		var exists int
		err := s.db.QueryRowContext(initCtx, `SELECT 1 FROM locality_centroids LIMIT 1`).Scan(&exists)
		centroidsExist := err == nil
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			s.regionCatalogErr = err
			return
		}

		if !centroidsExist {
			s.regionCatalog = &regionCatalog{bySlug: make(map[string]regionRef)}
			return
		}

		catalog := &regionCatalog{bySlug: make(map[string]regionRef)}

		rows, err := s.db.QueryContext(initCtx, `
			SELECT sa3_name, state
			FROM locality_centroids
			WHERE sa3_name IS NOT NULL AND sa3_name != ''
			GROUP BY sa3_name, state
			ORDER BY sa3_name, state`)
		if err != nil {
			s.regionCatalogErr = err
			return
		}
		defer rows.Close()

		sa3Assigned := false
		for rows.Next() {
			var sa3, st string
			if err := rows.Scan(&sa3, &st); err != nil {
				s.regionCatalogErr = err
				return
			}
			slug := buildRegionSlug(sa3, st)
			if slug == "" {
				continue
			}
			if _, ok := catalog.bySlug[slug]; !ok {
				catalog.bySlug[slug] = regionRef{
					sa3Name: sa3,
					state:   strings.TrimSpace(st),
				}
			}
			sa3Assigned = true
		}
		if err := rows.Err(); err != nil {
			s.regionCatalogErr = err
			return
		}

		if !sa3Assigned {
			s.regionCatalogErr = ErrRegionsNotLoaded
			return
		}

		s.regionCatalog = catalog
	})
	if s.regionCatalogErr != nil {
		return nil, s.regionCatalogErr
	}
	return s.regionCatalog, nil
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
