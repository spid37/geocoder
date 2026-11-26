package geocode

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/spid37/geocoder/internal/normalize"
)

var (
	ErrRadiusRequired     = errors.New("radius is required")
	ErrInvalidRadius      = errors.New("invalid radius")
	ErrCenterRequired     = errors.New("exactly one of suburb_slug or latitude+longitude is required")
	ErrInvalidCenter      = errors.New("latitude and longitude must be provided together")
	ErrSuburbSlugNotFound = errors.New("suburb_slug not found")
	ErrInvalidCoordinates = errors.New("invalid latitude or longitude")

)

const (
	defaultNearbyLimit = 20
	maxNearbyLimit     = 100
)

type NearbyQuery struct {
	Radius     string
	SuburbSlug string
	Latitude   string
	Longitude  string
	State      string
	Limit      int
}

type SuburbCenter struct {
	Suburb     string  `json:"suburb,omitempty"`
	State      string  `json:"state,omitempty"`
	Postcode   string  `json:"postcode,omitempty"`
	Region     string  `json:"region,omitempty"`
	SuburbSlug string  `json:"suburb_slug,omitempty"`
	RegionSlug string  `json:"region_slug,omitempty"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
}

type NearbySuburb struct {
	Suburb     string  `json:"suburb"`
	State      string  `json:"state"`
	Postcode   string  `json:"postcode"`
	Region     string  `json:"region,omitempty"`
	SuburbSlug string  `json:"suburb_slug"`
	RegionSlug string  `json:"region_slug,omitempty"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	DistanceM  float64 `json:"distance_m"`
}

type SuburbsNearbyResponse struct {
	Center    SuburbCenter   `json:"center"`
	Suburbs   []NearbySuburb `json:"suburbs"`
	Truncated bool           `json:"truncated,omitempty"`
}

type nearbyCentre struct {
	lat, lon       float64
	suburbSlug     string
	excludeSlug    bool
	center         SuburbCenter
}

func (s *Service) SuburbsNearby(ctx context.Context, q NearbyQuery) (*SuburbsNearbyResponse, error) {
	radiusStr := strings.TrimSpace(q.Radius)
	if radiusStr == "" {
		return nil, ErrRadiusRequired
	}
	radiusM, err := parseRadius(radiusStr)
	if err != nil {
		return nil, ErrInvalidRadius
	}

	stateFilter := normalize.State(q.State)
	limit := clampNearbyLimit(q.Limit)

	centre, err := s.resolveNearbyCentre(q)
	if err != nil {
		return nil, err
	}

	minLat, maxLat, minLon, maxLon := boundingBox(centre.lat, centre.lon, radiusM)

	rows, err := s.db.QueryContext(ctx, `
		SELECT locality_name, state, postcode, latitude, longitude, sa3_name
		FROM locality_centroids
		WHERE latitude BETWEEN ? AND ?
		  AND longitude BETWEEN ? AND ?
		  AND (? = '' OR state = ?)`,
		minLat, maxLat, minLon, maxLon, stateFilter, stateFilter,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []NearbySuburb
	for rows.Next() {
		var locality, st, postcode string
		var lat, lon float64
		var sa3 sql.NullString
		if err := rows.Scan(&locality, &st, &postcode, &lat, &lon, &sa3); err != nil {
			return nil, err
		}
		d := haversineM(centre.lat, centre.lon, lat, lon)
		if d > radiusM {
			continue
		}
		sug := formatSuburbSuggestion(locality, st, postcode, sa3.String)
		if centre.excludeSlug && sug.SuburbSlug == centre.suburbSlug {
			continue
		}
		matches = append(matches, NearbySuburb{
			Suburb:     sug.Suburb,
			State:      sug.State,
			Postcode:   sug.Postcode,
			Region:     sug.Region,
			SuburbSlug: sug.SuburbSlug,
			RegionSlug: sug.RegionSlug,
			Latitude:   lat,
			Longitude:  lon,
			DistanceM:  roundDistanceM(d),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].DistanceM != matches[j].DistanceM {
			return matches[i].DistanceM < matches[j].DistanceM
		}
		return matches[i].SuburbSlug < matches[j].SuburbSlug
	})

	truncated := len(matches) > limit
	if truncated {
		matches = matches[:limit]
	}
	if matches == nil {
		matches = []NearbySuburb{}
	}

	return &SuburbsNearbyResponse{
		Center:    centre.center,
		Suburbs:   matches,
		Truncated: truncated,
	}, nil
}

func (s *Service) resolveNearbyCentre(q NearbyQuery) (nearbyCentre, error) {
	slug := strings.ToLower(strings.TrimSpace(q.SuburbSlug))
	latStr := strings.TrimSpace(q.Latitude)
	lonStr := strings.TrimSpace(q.Longitude)
	hasSlug := slug != ""
	hasLat := latStr != ""
	hasLon := lonStr != ""

	switch {
	case hasSlug && (hasLat || hasLon):
		return nearbyCentre{}, ErrCenterRequired
	case !hasSlug && !hasLat && !hasLon:
		return nearbyCentre{}, ErrCenterRequired
	case hasSlug:
		index, err := s.loadSuburbSlugIndex()
		if err != nil {
			return nearbyCentre{}, err
		}
		c, ok := index[slug]
		if !ok {
			return nearbyCentre{}, ErrSuburbSlugNotFound
		}
		sug := formatSuburbSuggestion(c.locality, c.state, c.postcode, c.sa3Name)
		return nearbyCentre{
			lat:         c.latitude,
			lon:         c.longitude,
			suburbSlug:  c.suburbSlug,
			excludeSlug: true,
			center: SuburbCenter{
				Suburb:     sug.Suburb,
				State:      sug.State,
				Postcode:   sug.Postcode,
				Region:     sug.Region,
				SuburbSlug: sug.SuburbSlug,
				RegionSlug: sug.RegionSlug,
				Latitude:   c.latitude,
				Longitude:  c.longitude,
			},
		}, nil
	case hasLat != hasLon:
		return nearbyCentre{}, ErrInvalidCenter
	default:
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			return nearbyCentre{}, ErrInvalidCoordinates
		}
		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil {
			return nearbyCentre{}, ErrInvalidCoordinates
		}
		if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			return nearbyCentre{}, ErrInvalidCoordinates
		}
		return nearbyCentre{
			lat: lat,
			lon: lon,
			center: SuburbCenter{
				Latitude:  lat,
				Longitude: lon,
			},
		}, nil
	}
}

func (s *Service) loadSuburbSlugIndex() (map[string]suburbCentroid, error) {
	s.suburbBySlugOnce.Do(func() {
		initCtx := context.Background()
		rows, err := s.db.QueryContext(initCtx, `
			SELECT locality_name, state, postcode, latitude, longitude, sa3_name
			FROM locality_centroids`)
		if err != nil {
			s.suburbBySlugErr = err
			return
		}
		defer rows.Close()

		index := make(map[string]suburbCentroid)
		for rows.Next() {
			var locality, st, postcode string
			var lat, lon float64
			var sa3 sql.NullString
			if err := rows.Scan(&locality, &st, &postcode, &lat, &lon, &sa3); err != nil {
				s.suburbBySlugErr = err
				return
			}
			slug := buildSuburbSlug(&Address{
				Suburb:   locality,
				State:    st,
				Postcode: postcode,
			})
			if slug == "" {
				continue
			}
			if _, ok := index[slug]; !ok {
				index[slug] = suburbCentroid{
					locality:   locality,
					state:      st,
					postcode:   postcode,
					latitude:   lat,
					longitude:  lon,
					sa3Name:    sa3.String,
					suburbSlug: slug,
				}
			}
		}
		if err := rows.Err(); err != nil {
			s.suburbBySlugErr = err
			return
		}
		s.suburbBySlug = index
	})
	if s.suburbBySlugErr != nil {
		return nil, s.suburbBySlugErr
	}
	if s.suburbBySlug == nil {
		return map[string]suburbCentroid{}, nil
	}
	return s.suburbBySlug, nil
}

func clampNearbyLimit(limit int) int {
	if limit <= 0 {
		return defaultNearbyLimit
	}
	if limit > maxNearbyLimit {
		return maxNearbyLimit
	}
	return limit
}