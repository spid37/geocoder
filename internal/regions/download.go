package regions

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Download(dataDir string, force bool) (*Manifest, error) {
	existing, err := LoadManifest(dataDir)
	if err != nil {
		return nil, err
	}

	absRes, err := ResolveABSAllocation()
	if err != nil {
		return nil, fmt.Errorf("resolve ABS allocation: %w", err)
	}

	if !force && manifestCurrent(existing, absRes) {
		fmt.Println("Already have latest region data")
		return existing, nil
	}

	if existing != nil && existing.ABSResourceID != absRes.ID {
		fmt.Printf("New ABS allocation available: %s\n", absRes.Name)
	}

	regionsDir := filepath.Join(dataDir, "regions")
	if err := os.MkdirAll(regionsDir, 0o755); err != nil {
		return nil, err
	}

	fmt.Printf("Downloading %s\n", absRes.Name)
	absZip := filepath.Join(regionsDir, "abs-allocation.zip")
	if err := downloadFile(absRes.URL, absZip); err != nil {
		return nil, err
	}
	absXlsx, err := extractFileMatching(absZip, regionsDir, func(name string) bool {
		up := strings.ToUpper(name)
		return strings.Contains(up, "MB_2021") && strings.HasSuffix(up, ".XLSX")
	})
	if err != nil {
		return nil, err
	}

	m := &Manifest{
		ABSResourceID: absRes.ID,
		ABSPath:       absXlsx,
		DownloadedAt:  time.Now().UTC(),
	}
	if err := SaveManifest(dataDir, m); err != nil {
		return nil, err
	}
	fmt.Printf("Region data ready in %s\n", regionsDir)
	return m, nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: status %d", dest, resp.StatusCode)
	}

	tmp := dest + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	out.Close()
	return os.Rename(tmp, dest)
}

func extractFileMatching(zipPath, destDir string, match func(string) bool) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if !match(f.Name) {
			continue
		}
		base := filepath.Base(f.Name)
		outPath := filepath.Join(destDir, base)
		if err := extractZipEntry(f, outPath); err != nil {
			return "", err
		}
		return outPath, nil
	}
	return "", fmt.Errorf("no matching file in %s", zipPath)
}

func extractZipEntry(f *zip.File, destPath string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		os.Remove(destPath)
		return err
	}
	return out.Close()
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
