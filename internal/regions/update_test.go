package regions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifestCurrent(t *testing.T) {
	dir := t.TempDir()
	absPath := filepath.Join(dir, "abs.xlsx")
	if err := os.WriteFile(absPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	existing := &Manifest{
		ABSResourceID: "abs-1",
		ABSPath:       absPath,
	}
	absRes := &Resource{ID: "abs-1"}

	if !manifestCurrent(existing, absRes) {
		t.Fatal("expected current manifest")
	}

	absRes.ID = "abs-2"
	if manifestCurrent(existing, absRes) {
		t.Fatal("expected stale ABS resource")
	}
}
