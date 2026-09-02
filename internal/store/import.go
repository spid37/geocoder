package store

import (
	"database/sql"
	"fmt"

	"github.com/spid37/geocoder/internal/gnaf"
)

const batchSize = 5000

func Load(db *sql.DB, manifest *gnaf.Manifest) error {
	if manifest == nil {
		return fmt.Errorf("manifest is required")
	}
	if manifest.ZipPath == "" {
		return fmt.Errorf("manifest has no zip_path — run geocoder data download first")
	}
	return LoadFromZip(db, manifest.ZipPath, manifest)
}

func clearTables(db *sql.DB) error {
	_, err := db.Exec(`
		DELETE FROM addresses;
		DELETE FROM locality_centroids;
		DELETE FROM postcode_centroids;
		DELETE FROM state_centroids;
	`)
	return err
}

func createIndexes(db *sql.DB) error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_addresses_street ON addresses(state, postcode, locality_name_norm, street_name_norm, number_first)`,
		`CREATE INDEX IF NOT EXISTS idx_addresses_locality ON addresses(state, postcode, locality_name_norm)`,
		`CREATE INDEX IF NOT EXISTS idx_addresses_mb_code ON addresses(mb_code_2021)`,
	}
	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}
	return nil
}
