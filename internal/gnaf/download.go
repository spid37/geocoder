package gnaf

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func DownloadRelease(dataDir string, release *Release, force bool) (*Manifest, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	existing, err := LoadManifest(dataDir)
	if err != nil {
		return nil, err
	}
	if !force && existing != nil && existing.ResourceID == release.ResourceID && existing.ZipPath != "" {
		if err := ValidateGNAFZip(existing.ZipPath); err == nil {
			fmt.Printf("Already have latest release: %s\n", existing.ReleaseName)
			return existing, nil
		}
	}

	if existing != nil && existing.ResourceID != release.ResourceID {
		fmt.Printf("New release available: %s (was %s)\n", release.Name, existing.ReleaseName)
	}

	safeName := sanitizeFilename(release.Name)
	zipPath := filepath.Join(dataDir, fmt.Sprintf("gnaf-%s.zip", safeName))

	if !force {
		if _, err := os.Stat(zipPath); err == nil {
			if err := ValidateGNAFZip(zipPath); err == nil {
				m := &Manifest{
					ReleaseName:  release.Name,
					ResourceID:   release.ResourceID,
					DownloadedAt: time.Now().UTC(),
					Datum:        release.Datum,
					ZipPath:      zipPath,
				}
				if err := SaveManifest(dataDir, m); err != nil {
					return nil, err
				}
				fmt.Printf("Using existing archive: %s\n", zipPath)
				return m, nil
			}
		}
	}

	fmt.Printf("Downloading %s\n", release.Name)
	fmt.Printf("URL: %s\n", release.URL)
	if err := downloadFile(release.URL, zipPath); err != nil {
		return nil, err
	}

	fmt.Println("Validating G-NAF archive...")
	if err := ValidateGNAFZip(zipPath); err != nil {
		return nil, err
	}

	m := &Manifest{
		ReleaseName:  release.Name,
		ResourceID:   release.ResourceID,
		DownloadedAt: time.Now().UTC(),
		Datum:        release.Datum,
		ZipPath:      zipPath,
	}
	if err := SaveManifest(dataDir, m); err != nil {
		return nil, err
	}

	fmt.Printf("Download complete: %s\n", zipPath)
	return m, nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: status %d", resp.StatusCode)
	}

	tmp := dest + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	written, err := io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(tmp)
		return fmt.Errorf("write file: %w", err)
	}

	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename file: %w", err)
	}

	fmt.Printf("Downloaded %d bytes to %s\n", written, dest)
	return nil
}

func sanitizeFilename(name string) string {
	name = strings.ToLower(name)
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "")
	return replacer.Replace(name)
}
