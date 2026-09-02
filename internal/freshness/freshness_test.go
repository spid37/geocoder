package freshness_test

import (
	"path/filepath"
	"testing"

	"github.com/spid37/geocoder/internal/freshness"
	"github.com/spid37/geocoder/internal/gnaf"
	"github.com/spid37/geocoder/internal/regions"
	"github.com/spid37/geocoder/internal/store"
)

func TestCheckDBMatches(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	gnafManifest := &gnaf.Manifest{
		ReleaseName: "TEST",
		ResourceID:  "gnaf-1",
		ZipPath:     filepath.Join(dir, "gnaf.zip"),
	}
	if err := gnaf.SaveManifest(dir, gnafManifest); err != nil {
		t.Fatal(err)
	}

	regManifest := &regions.Manifest{
		ABSResourceID: "abs-1",
	}
	if err := regions.SaveManifest(dir, regManifest); err != nil {
		t.Fatal(err)
	}

	_ = store.SetMetadata(db, "resource_id", "gnaf-1")
	_ = store.SetMetadata(db, "abs_resource_id", "abs-1")

	report := freshness.CheckDB(dir, db)
	if report.Stale {
		t.Fatalf("expected fresh DB: %+v", report)
	}

	_ = store.SetMetadata(db, "resource_id", "gnaf-old")
	report = freshness.CheckDB(dir, db)
	if !report.Stale || report.DB.GNAFMatchesFiles {
		t.Fatalf("expected stale G-NAF DB: %+v", report)
	}
}
