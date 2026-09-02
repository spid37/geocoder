package store

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spid37/geocoder/internal/gnaf"
)

func LoadFromZip(db *sql.DB, zipPath string, manifest *gnaf.Manifest) error {
	start := time.Now()

	z, err := gnaf.OpenZipArchive(zipPath)
	if err != nil {
		return err
	}
	defer z.Close()

	if err := clearTables(db); err != nil {
		return err
	}
	if err := createStagingTables(db); err != nil {
		return err
	}

	fmt.Println("Importing authority codes...")
	if err := importStreetTypes(db, z); err != nil {
		return err
	}

	for _, state := range z.States() {
		fmt.Printf("Importing %s...\n", state)
		if err := importStateRows(db, z, state); err != nil {
			return err
		}
	}

	fmt.Println("Building address table from G-NAF joins...")
	count, err := materializeAddresses(db)
	if err != nil {
		return err
	}
	fmt.Printf("Materialized %d addresses\n", count)

	if err := linkMeshBlocksDuringLoad(db, z); err != nil {
		return err
	}

	if err := dropStagingTables(db); err != nil {
		return err
	}

	fmt.Println("Building centroids...")
	if err := BuildCentroids(db); err != nil {
		return err
	}

	fmt.Println("Creating indexes...")
	if err := createIndexes(db); err != nil {
		return err
	}

	if manifest != nil {
		_ = SetMetadata(db, "release_name", manifest.ReleaseName)
		_ = SetMetadata(db, "resource_id", manifest.ResourceID)
		_ = SetMetadata(db, "datum", manifest.Datum)
		_ = SetMetadata(db, "loaded_at", time.Now().UTC().Format(time.RFC3339))
	}

	fmt.Printf("Load complete in %s\n", time.Since(start).Round(time.Second))
	return nil
}

func createStagingTables(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS gnaf_street_type_aut (code TEXT PRIMARY KEY, name TEXT)`,
		`CREATE TABLE IF NOT EXISTS gnaf_state (state_pid TEXT PRIMARY KEY, state_abbreviation TEXT)`,
		`CREATE TABLE IF NOT EXISTS gnaf_address_detail (
			address_detail_pid TEXT PRIMARY KEY,
			street_locality_pid TEXT,
			locality_pid TEXT,
			number_first TEXT,
			postcode TEXT,
			confidence TEXT,
			date_retired TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS gnaf_street_locality (
			street_locality_pid TEXT PRIMARY KEY,
			street_name TEXT,
			street_type_code TEXT,
			date_retired TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS gnaf_locality (
			locality_pid TEXT PRIMARY KEY,
			locality_name TEXT,
			state_pid TEXT,
			date_retired TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS gnaf_address_default_geocode (
			address_detail_pid TEXT PRIMARY KEY,
			longitude REAL,
			latitude REAL,
			date_retired TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS gnaf_address_mesh_block_2021 (
			address_detail_pid TEXT PRIMARY KEY,
			mb_2021_pid TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS gnaf_mb_2021 (
			mb_2021_pid TEXT PRIMARY KEY,
			mb_2021_code TEXT NOT NULL
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}

	clears := []string{
		`DELETE FROM gnaf_street_type_aut`,
		`DELETE FROM gnaf_state`,
		`DELETE FROM gnaf_address_detail`,
		`DELETE FROM gnaf_street_locality`,
		`DELETE FROM gnaf_locality`,
		`DELETE FROM gnaf_address_default_geocode`,
		`DELETE FROM gnaf_address_mesh_block_2021`,
		`DELETE FROM gnaf_mb_2021`,
	}
	for _, s := range clears {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

func dropStagingTables(db *sql.DB) error {
	stmts := []string{
		`DROP TABLE IF EXISTS gnaf_street_type_aut`,
		`DROP TABLE IF EXISTS gnaf_state`,
		`DROP TABLE IF EXISTS gnaf_address_detail`,
		`DROP TABLE IF EXISTS gnaf_street_locality`,
		`DROP TABLE IF EXISTS gnaf_locality`,
		`DROP TABLE IF EXISTS gnaf_address_default_geocode`,
		`DROP TABLE IF EXISTS gnaf_address_mesh_block_2021`,
		`DROP TABLE IF EXISTS gnaf_mb_2021`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

func importStreetTypes(db *sql.DB, z *gnaf.ZipArchive) error {
	rc, err := z.OpenAuthorityPSV("Authority_Code_STREET_TYPE_AUT")
	if err != nil {
		return err
	}
	defer rc.Close()

	return streamInsert(db, rc, `
		INSERT OR IGNORE INTO gnaf_street_type_aut (code, name) VALUES (?, ?)`,
		func(fields []string) ([]any, bool) {
			if len(fields) < 2 {
				return nil, false
			}
			return []any{fields[0], fields[1]}, true
		},
	)
}

func importStateRows(db *sql.DB, z *gnaf.ZipArchive, state string) error {
	if err := importStandardTable(db, z, state, "STATE", `
		INSERT OR IGNORE INTO gnaf_state (state_pid, state_abbreviation) VALUES (?, ?)`,
		func(fields []string) ([]any, bool) {
			if len(fields) < 5 {
				return nil, false
			}
			return []any{fields[0], fields[4]}, true
		},
	); err != nil {
		return err
	}

	if err := importStandardTable(db, z, state, "ADDRESS_DETAIL", `
		INSERT OR REPLACE INTO gnaf_address_detail (
			address_detail_pid, street_locality_pid, locality_pid, number_first, postcode, confidence, date_retired
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		func(fields []string) ([]any, bool) {
			if len(fields) < 30 {
				return nil, false
			}
			return []any{fields[0], fields[22], fields[24], fields[17], fields[26], fields[29], fields[3]}, true
		},
	); err != nil {
		return err
	}

	if err := importStandardTable(db, z, state, "STREET_LOCALITY", `
		INSERT OR REPLACE INTO gnaf_street_locality (
			street_locality_pid, street_name, street_type_code, date_retired
		) VALUES (?, ?, ?, ?)`,
		func(fields []string) ([]any, bool) {
			if len(fields) < 6 {
				return nil, false
			}
			return []any{fields[0], fields[4], fields[5], fields[2]}, true
		},
	); err != nil {
		return err
	}

	if err := importStandardTable(db, z, state, "LOCALITY", `
		INSERT OR REPLACE INTO gnaf_locality (locality_pid, locality_name, state_pid, date_retired) VALUES (?, ?, ?, ?)`,
		func(fields []string) ([]any, bool) {
			if len(fields) < 7 {
				return nil, false
			}
			return []any{fields[0], fields[3], fields[6], fields[2]}, true
		},
	); err != nil {
		return err
	}

	return importStandardTable(db, z, state, "ADDRESS_DEFAULT_GEOCODE", `
		INSERT OR REPLACE INTO gnaf_address_default_geocode (address_detail_pid, longitude, latitude, date_retired) VALUES (?, ?, ?, ?)`,
		func(fields []string) ([]any, bool) {
			if len(fields) < 7 {
				return nil, false
			}
			return []any{fields[3], fields[5], fields[6], fields[2]}, true
		},
	)
}

func importMB2021Rows(db *sql.DB, z *gnaf.ZipArchive, state string) error {
	return importStandardTable(db, z, state, "MB_2021", `
		INSERT OR REPLACE INTO gnaf_mb_2021 (mb_2021_pid, mb_2021_code) VALUES (?, ?)`,
		func(fields []string) ([]any, bool) {
			if len(fields) < 4 {
				return nil, false
			}
			if fields[0] == "" || fields[3] == "" || fields[2] != "" {
				return nil, false
			}
			return []any{fields[0], fields[3]}, true
		},
	)
}

func importMeshBlockRows(db *sql.DB, z *gnaf.ZipArchive, state string) error {
	return importStandardTable(db, z, state, "ADDRESS_MESH_BLOCK_2021", `
		INSERT OR REPLACE INTO gnaf_address_mesh_block_2021 (address_detail_pid, mb_2021_pid) VALUES (?, ?)`,
		func(fields []string) ([]any, bool) {
			if len(fields) < 6 {
				return nil, false
			}
			if fields[2] != "" {
				return nil, false
			}
			if fields[3] == "" || fields[5] == "" {
				return nil, false
			}
			return []any{fields[3], fields[5]}, true
		},
	)
}

func linkMeshBlocksDuringLoad(db *sql.DB, z *gnaf.ZipArchive) error {
	fmt.Println("Importing mesh block 2021 lookup...")
	for _, state := range z.States() {
		if err := importMB2021Rows(db, z, state); err != nil {
			return err
		}
	}
	fmt.Println("Importing mesh block 2021 links...")
	for _, state := range z.States() {
		if err := importMeshBlockRows(db, z, state); err != nil {
			return err
		}
	}
	res, err := db.Exec(`
		UPDATE addresses SET mb_code_2021 = (
			SELECT b.mb_2021_code FROM gnaf_address_mesh_block_2021 m
			INNER JOIN gnaf_mb_2021 b ON b.mb_2021_pid = m.mb_2021_pid
			WHERE m.address_detail_pid = addresses.address_detail_pid
		)`)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	fmt.Printf("Linked mesh blocks to %d addresses\n", n)
	return nil
}

func importStandardTable(db *sql.DB, z *gnaf.ZipArchive, state, table, insertSQL string, mapRow func([]string) ([]any, bool)) error {
	rc, err := z.OpenStandardPSV(state, table)
	if err != nil {
		return err
	}
	defer rc.Close()
	return streamInsert(db, rc, insertSQL, mapRow)
}

func streamInsert(db *sql.DB, r io.Reader, insertSQL string, mapRow func([]string) ([]any, bool)) error {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	if !scanner.Scan() {
		return fmt.Errorf("read header: %w", scanner.Err())
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		tx.Rollback()
		return err
	}

	count := 0
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "|")
		args, ok := mapRow(fields)
		if !ok {
			continue
		}
		if _, err := stmt.Exec(args...); err != nil {
			stmt.Close()
			tx.Rollback()
			return fmt.Errorf("insert row: %w", err)
		}
		count++
		if count%batchSize == 0 {
			stmt.Close()
			if err := tx.Commit(); err != nil {
				return err
			}
			tx, err = db.Begin()
			if err != nil {
				return err
			}
			stmt, err = tx.Prepare(insertSQL)
			if err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		stmt.Close()
		tx.Rollback()
		return err
	}

	stmt.Close()
	return tx.Commit()
}

func materializeAddresses(db *sql.DB) (int64, error) {
	res, err := db.Exec(`
		INSERT INTO addresses (
			address_detail_pid, address_label, number_first, street_name, street_type,
			street_name_norm, street_type_norm, locality_name, locality_name_norm,
			state, postcode, latitude, longitude
		)
		SELECT
			ad.address_detail_pid,
			TRIM(
				COALESCE(ad.number_first, '') || ' ' ||
				COALESCE(sl.street_name, '') || ' ' ||
				COALESCE(sta.name, '') || ' ' ||
				COALESCE(l.locality_name, '') || ' ' ||
				COALESCE(s.state_abbreviation, '') || ' ' ||
				COALESCE(ad.postcode, '')
			),
			ad.number_first,
			sl.street_name,
			COALESCE(sta.name, ''),
			UPPER(TRIM(COALESCE(sl.street_name, ''))),
			UPPER(TRIM(COALESCE(sta.name, ''))),
			l.locality_name,
			UPPER(TRIM(COALESCE(l.locality_name, ''))),
			s.state_abbreviation,
			ad.postcode,
			adg.latitude,
			adg.longitude
		FROM gnaf_address_detail ad
		INNER JOIN gnaf_street_locality sl ON ad.street_locality_pid = sl.street_locality_pid
		INNER JOIN gnaf_locality l ON ad.locality_pid = l.locality_pid
		INNER JOIN gnaf_state s ON l.state_pid = s.state_pid
		INNER JOIN gnaf_address_default_geocode adg ON ad.address_detail_pid = adg.address_detail_pid
		LEFT JOIN gnaf_street_type_aut sta ON sl.street_type_code = sta.code
		WHERE (ad.date_retired IS NULL OR ad.date_retired = '')
		  AND (sl.date_retired IS NULL OR sl.date_retired = '')
		  AND (l.date_retired IS NULL OR l.date_retired = '')
		  AND (adg.date_retired IS NULL OR adg.date_retired = '')
		  AND CAST(COALESCE(ad.confidence, '0') AS INTEGER) > -1
	`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
