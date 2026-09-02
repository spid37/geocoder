package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spid37/geocoder/internal/freshness"
	"github.com/spid37/geocoder/internal/geocode"
	"github.com/spid37/geocoder/internal/gnaf"
	"github.com/spid37/geocoder/internal/regions"
	"github.com/spid37/geocoder/internal/store"
	"github.com/spid37/geocoder/internal/version"
)

type Server struct {
	httpServer *http.Server
	dataDir    string
}

func NewServer(addr string, db *sql.DB, dataDir string, apiKeys *APIKeyAuth) *Server {
	svc := geocode.NewService(db)
	h := &Handler{Geocoder: svc}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/geocode", apiKeys.wrap(h.Geocode))
	mux.HandleFunc("GET /v1/suggest/suburbs", apiKeys.wrap(h.SuggestSuburbs))
	mux.HandleFunc("GET /v1/suggest/regions", apiKeys.wrap(h.SuggestRegions))
	mux.HandleFunc("GET /v1/suggest/addresses", apiKeys.wrap(h.SuggestAddresses))
	mux.HandleFunc("GET /version", handleVersion)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"version": version.String(),
		})
	})
	mux.HandleFunc("GET /health/data", func(w http.ResponseWriter, r *http.Request) {
		body := buildDataHealth(db, dataDir)
		body["version"] = version.String()
		writeJSON(w, http.StatusOK, body)
	})

	handler := versionMiddleware(accessLog(os.Stdout)(mux))

	return &Server{
		dataDir: dataDir,
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
}

func buildDataHealth(db *sql.DB, dataDir string) map[string]any {
	release, _ := store.GetMetadata(db, "release_name")
	resourceID, _ := store.GetMetadata(db, "resource_id")
	datum, _ := store.GetMetadata(db, "datum")
	loadedAt, _ := store.GetMetadata(db, "loaded_at")
	regionsLoadedAt, _ := store.GetMetadata(db, "regions_loaded_at")
	absResourceID, _ := store.GetMetadata(db, "abs_resource_id")

	gnafManifest, _ := gnaf.LoadManifest(dataDir)
	regManifest, _ := regions.LoadManifest(dataDir)
	dbFreshness := freshness.CheckDB(dataDir, db)

	return map[string]any{
		"status":    "ok",
		"gnaf":      buildGNAFHealth(release, resourceID, datum, loadedAt, gnafManifest, dbFreshness),
		"regions":   buildRegionsHealth(regionsLoadedAt, absResourceID, regManifest, dbFreshness),
		"freshness": map[string]any{"db_stale": dbFreshness.Stale, "details": dbFreshness.Details},
	}
}

func buildGNAFHealth(release, resourceID, datum, loadedAt string, manifest *gnaf.Manifest, f freshness.Report) map[string]any {
	block := map[string]any{
		"release_name": release,
		"resource_id":  resourceID,
		"datum":        datum,
		"loaded_at":    loadedAt,
	}
	if manifest != nil {
		block["files_resource_id"] = manifest.ResourceID
		block["db_matches_files"] = f.DB.GNAFMatchesFiles
	}
	return block
}

func buildRegionsHealth(regionsLoadedAt, absResourceID string, manifest *regions.Manifest, f freshness.Report) map[string]any {
	block := map[string]any{
		"regions_loaded_at":     regionsLoadedAt,
		"abs_resource_id":       absResourceID,
		"files_abs_resource_id": "",
	}
	if manifest != nil {
		block["files_abs_resource_id"] = manifest.ABSResourceID
		block["db_matches_files"] = f.DB.RegionsMatchesFiles
	}
	return block
}

func (s *Server) ListenAndServe() error {
	fmt.Printf("Listening on %s\n", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Handler() http.Handler {
	return s.httpServer.Handler
}
