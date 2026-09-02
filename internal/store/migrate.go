package store

import (
	"database/sql"
	"fmt"
)

func migrate(db *sql.DB) error {
	schema, err := schemaSQL.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return migrateRegionColumns(db)
}

func migrateRegionColumns(db *sql.DB) error {
	type alter struct {
		table  string
		column string
		typ    string
	}
	alters := []alter{
		{"addresses", "mb_code_2021", "TEXT"},
		{"addresses", "sa3_code", "TEXT"},
		{"addresses", "sa3_name", "TEXT"},
		{"locality_centroids", "sa3_code", "TEXT"},
		{"locality_centroids", "sa3_name", "TEXT"},
		{"postcode_centroids", "sa3_code", "TEXT"},
		{"postcode_centroids", "sa3_name", "TEXT"},
		{"state_centroids", "sa3_code", "TEXT"},
		{"state_centroids", "sa3_name", "TEXT"},
	}

	for _, a := range alters {
		if err := addColumnIfMissing(db, a.table, a.column, a.typ); err != nil {
			return err
		}
	}
	return nil
}

func addColumnIfMissing(db *sql.DB, table, column, typ string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, colType string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, typ))
	return err
}
