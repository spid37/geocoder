package regions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Manifest struct {
	ABSResourceID string    `json:"abs_resource_id"`
	ABSPath       string    `json:"abs_path,omitempty"`
	DownloadedAt  time.Time `json:"downloaded_at"`
}

func ManifestPath(dataDir string) string {
	return filepath.Join(dataDir, "regions", "manifest.json")
}

func LoadManifest(dataDir string) (*Manifest, error) {
	path := ManifestPath(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read regions manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse regions manifest: %w", err)
	}
	return &m, nil
}

func SaveManifest(dataDir string, m *Manifest) error {
	dir := filepath.Join(dataDir, "regions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ManifestPath(dataDir), data, 0o644)
}
