package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/spid37/geocoder/internal/config"
	"github.com/spid37/geocoder/internal/geocode"
)

type Handler struct {
	Geocoder *geocode.Service
}

func (h *Handler) Geocode(w http.ResponseWriter, r *http.Request) {
	q := geocode.Query{
		Street:   r.URL.Query().Get("street"),
		Suburb:   r.URL.Query().Get("suburb"),
		State:    r.URL.Query().Get("state"),
		Postcode: r.URL.Query().Get("postcode"),
	}

	if q.Street == "" && q.Suburb == "" && q.State == "" && q.Postcode == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "at least one of street, suburb, state, postcode is required"})
		return
	}

	result, err := h.Geocoder.Geocode(r.Context(), q)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "geocoding failed"})
		return
	}
	if result == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no match found"})
		return
	}

	w.Header().Set("X-Attribution", config.Attribution)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) SuggestSuburbs(w http.ResponseWriter, r *http.Request) {
	h.handleSuggest(w, r, func(ctx context.Context, q, state string, limit int) (any, error) {
		return h.Geocoder.SuggestSuburbs(ctx, q, state, limit)
	})
}

func (h *Handler) SuggestRegions(w http.ResponseWriter, r *http.Request) {
	h.handleSuggest(w, r, func(ctx context.Context, q, state string, limit int) (any, error) {
		return h.Geocoder.SuggestRegions(ctx, q, state, limit)
	})
}

func (h *Handler) SuburbsNearby(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"))
	result, err := h.Geocoder.SuburbsNearby(r.Context(), geocode.NearbyQuery{
		Radius:     r.URL.Query().Get("radius"),
		SuburbSlug: r.URL.Query().Get("suburb_slug"),
		Latitude:   r.URL.Query().Get("latitude"),
		Longitude:  r.URL.Query().Get("longitude"),
		State:      r.URL.Query().Get("state"),
		Limit:      limit,
	})
	if err != nil {
		switch {
		case errors.Is(err, geocode.ErrSuburbSlugNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		case errors.Is(err, geocode.ErrRadiusRequired),
			errors.Is(err, geocode.ErrInvalidRadius),
			errors.Is(err, geocode.ErrCenterRequired),
			errors.Is(err, geocode.ErrInvalidCenter),
			errors.Is(err, geocode.ErrInvalidCoordinates):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "suburbs nearby failed"})
		}
		return
	}

	w.Header().Set("X-Attribution", config.Attribution)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) ListSuburbs(w http.ResponseWriter, r *http.Request) {
	regionSlug := r.URL.Query().Get("region_slug")
	if strings.TrimSpace(regionSlug) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "region_slug is required"})
		return
	}

	state := r.URL.Query().Get("state")
	limit := parseLimit(r.URL.Query().Get("limit"))

	result, err := h.Geocoder.ListSuburbsByRegion(r.Context(), regionSlug, state, limit)
	if err != nil {
		if errors.Is(err, geocode.ErrRegionsNotLoaded) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
			return
		}
		if errors.Is(err, geocode.ErrRegionSlugRequired) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list suburbs failed"})
		return
	}

	w.Header().Set("X-Attribution", config.Attribution)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) SuggestAddresses(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	suburb := r.URL.Query().Get("suburb")
	state := r.URL.Query().Get("state")
	postcode := r.URL.Query().Get("postcode")
	limit := parseLimit(r.URL.Query().Get("limit"))

	suggestions, err := h.Geocoder.SuggestAddresses(r.Context(), geocode.AddressSuggestQuery{
		Q: q, Suburb: suburb, State: state, Postcode: postcode, Limit: limit,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("X-Attribution", config.Attribution)
	writeJSON(w, http.StatusOK, map[string]any{"suggestions": suggestions})
}

func (h *Handler) handleSuggest(w http.ResponseWriter, r *http.Request, suggest func(context.Context, string, string, int) (any, error)) {
	q := r.URL.Query().Get("q")
	state := r.URL.Query().Get("state")
	limit := parseLimit(r.URL.Query().Get("limit"))

	suggestions, err := suggest(r.Context(), q, state, limit)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("X-Attribution", config.Attribution)
	writeJSON(w, http.StatusOK, map[string]any{"suggestions": suggestions})
}

func parseLimit(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
