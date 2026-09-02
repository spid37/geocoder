package testdata

import (
	"database/sql"
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/spid37/geocoder/internal/store"
)

//go:embed fixtures.sql
var fixturesSQL string

func moduleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir
		}
		dir = parent
	}
}

func resolveDBPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(moduleRoot(), path)
}

func OpenDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec(fixturesSQL); err != nil {
		t.Fatal(err)
	}
	return db
}

func OpenPath(t *testing.T, path string) *sql.DB {
	t.Helper()

	path = resolveDBPath(path)
	if _, err := os.Stat(path); err != nil {
		t.Skipf("database not found at %s: %v", path, err)
	}

	db, err := store.Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
