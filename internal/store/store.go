package store

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL embed.FS

func Open(path string, readOnly bool) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}

	dsn := path
	if readOnly {
		dsn = path + "?mode=ro"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if !readOnly {
		if err := migrate(db); err != nil {
			db.Close()
			return nil, err
		}
	}

	db.SetMaxOpenConns(1)
	return db, nil
}

func SetMetadata(db *sql.DB, key, value string) error {
	_, err := db.Exec(`
		INSERT INTO metadata (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}

func GetMetadata(db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}
