package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/spid37/geocoder/internal/gnaf"
	"github.com/spid37/geocoder/internal/regions"
)

type LoadRegionsOptions struct {
	GNAFZip string
}

func LoadRegions(db *sql.DB, manifest *regions.Manifest, opts LoadRegionsOptions) error {
	start := time.Now()
	if err := migrateRegionColumns(db); err != nil {
		return err
	}

	fmt.Println("Loading ABS mesh block allocation...")
	if err := regions.LoadMeshBlocks(db, manifest.ABSPath); err != nil {
		return err
	}

	linked, err := meshBlocksLinked(db)
	if err != nil {
		return err
	}
	if !linked {
		return fmt.Errorf("addresses are not linked to mesh blocks — run 'geocoder data load' first")
	}

	if opts.GNAFZip != "" {
		needsResolve, err := meshBlockCodesNeedResolve(db)
		if err != nil {
			return err
		}
		if needsResolve {
			fmt.Println("Resolving mesh block codes from G-NAF...")
			if err := resolveMeshBlockCodes(db, opts.GNAFZip); err != nil {
				return err
			}
		}
	}

	fmt.Println("Assigning SA3 from mesh blocks...")
	if err := assignSA3Regions(db); err != nil {
		return err
	}
	if err := ensureMeshBlockIndex(db); err != nil {
		return err
	}

	fmt.Println("Rebuilding centroids with region data...")
	if err := RebuildCentroids(db); err != nil {
		return err
	}

	var withSA3 int
	_ = db.QueryRow(`SELECT COUNT(*) FROM addresses WHERE sa3_name IS NOT NULL AND sa3_name != ''`).Scan(&withSA3)
	fmt.Printf("Addresses with SA3: %d\n", withSA3)

	_ = SetMetadata(db, "regions_loaded_at", time.Now().UTC().Format(time.RFC3339))
	_ = SetMetadata(db, "abs_resource_id", manifest.ABSResourceID)
	fmt.Printf("Region load complete in %s\n", time.Since(start).Round(time.Second))
	return nil
}

func resolveMeshBlockCodes(db *sql.DB, zipPath string) error {
	z, err := gnaf.OpenZipArchive(zipPath)
	if err != nil {
		return err
	}
	defer z.Close()

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS gnaf_mb_2021 (
			mb_2021_pid TEXT PRIMARY KEY,
			mb_2021_code TEXT NOT NULL
		)`); err != nil {
		return err
	}
	if _, err := db.Exec(`DELETE FROM gnaf_mb_2021`); err != nil {
		return err
	}

	for _, state := range z.States() {
		if err := importMB2021Rows(db, z, state); err != nil {
			return err
		}
	}

	res, err := db.Exec(`
		UPDATE addresses SET mb_code_2021 = (
			SELECT b.mb_2021_code FROM gnaf_mb_2021 b
			WHERE b.mb_2021_pid = addresses.mb_code_2021
		)
		WHERE mb_code_2021 LIKE 'MB%'`)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	fmt.Printf("  Resolved mesh block codes for %d addresses\n", n)

	_, err = db.Exec(`DROP TABLE IF EXISTS gnaf_mb_2021`)
	return err
}

func meshBlockCodesNeedResolve(db *sql.DB) (bool, error) {
	var one int
	err := db.QueryRow(`SELECT 1 FROM addresses WHERE mb_code_2021 LIKE 'MB%' LIMIT 1`).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func assignSA3Regions(db *sql.DB) error {
	_, err := db.Exec(`
		UPDATE addresses SET
			sa3_code = (SELECT sa3_code FROM mesh_blocks mb WHERE mb.mb_code = addresses.mb_code_2021),
			sa3_name = (SELECT sa3_name FROM mesh_blocks mb WHERE mb.mb_code = addresses.mb_code_2021)
		WHERE mb_code_2021 IS NOT NULL AND mb_code_2021 != ''`)
	return err
}

func meshBlocksLinked(db *sql.DB) (bool, error) {
	var one int
	err := db.QueryRow(`SELECT 1 FROM addresses WHERE mb_code_2021 IS NOT NULL AND mb_code_2021 != '' LIMIT 1`).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func ensureMeshBlockIndex(db *sql.DB) error {
	_, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_addresses_mb_code ON addresses(mb_code_2021)`)
	return err
}

func RebuildCentroids(db *sql.DB) error {
	if _, err := db.Exec(`
		DELETE FROM locality_centroids;
		DELETE FROM postcode_centroids;
		DELETE FROM state_centroids;
	`); err != nil {
		return err
	}

	queries := []string{
		`INSERT INTO locality_centroids (
			state, postcode, locality_name, locality_name_norm,
			latitude, longitude, address_count, sa3_code, sa3_name
		)
		SELECT state, postcode, MAX(locality_name), locality_name_norm,
		       AVG(latitude), AVG(longitude), COUNT(*),
		       MAX(sa3_code), MAX(sa3_name)
		FROM addresses
		GROUP BY state, postcode, locality_name_norm`,

		`INSERT INTO postcode_centroids (
			state, postcode, latitude, longitude, address_count, sa3_code, sa3_name
		)
		SELECT state, postcode, AVG(latitude), AVG(longitude), COUNT(*),
		       MAX(sa3_code), MAX(sa3_name)
		FROM addresses
		GROUP BY state, postcode`,

		`INSERT INTO state_centroids (
			state, latitude, longitude, address_count, sa3_code, sa3_name
		)
		SELECT state, AVG(latitude), AVG(longitude), COUNT(*),
		       MAX(sa3_code), MAX(sa3_name)
		FROM addresses
		GROUP BY state`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
