package gnaf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Manifest struct {
	ReleaseName  string    `json:"release_name"`
	ResourceID   string    `json:"resource_id"`
	DownloadedAt time.Time `json:"downloaded_at"`
	Datum        string    `json:"datum"`
	ZipPath      string    `json:"zip_path,omitempty"`
}

func ManifestPath(dataDir string) string {
	return filepath.Join(dataDir, "manifest.json")
}

func LoadManifest(dataDir string) (*Manifest, error) {
	path := ManifestPath(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

func SaveManifest(dataDir string, m *Manifest) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	path := ManifestPath(dataDir)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}
