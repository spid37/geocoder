package gnaf

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

var gnafStates = []string{"ACT", "NSW", "NT", "OT", "QLD", "SA", "TAS", "VIC", "WA"}

type ZipArchive struct {
	reader   *zip.ReadCloser
	standard string // prefix ending with Standard/
}

func OpenZipArchive(zipPath string) (*ZipArchive, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	var standardPrefix string
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "Standard/ACT_ADDRESS_DETAIL_psv.psv") {
			standardPrefix = strings.TrimSuffix(f.Name, "ACT_ADDRESS_DETAIL_psv.psv")
			break
		}
	}
	if standardPrefix == "" {
		r.Close()
		return nil, fmt.Errorf("G-NAF Standard tables not found in %s", zipPath)
	}

	return &ZipArchive{reader: r, standard: standardPrefix}, nil
}

func (z *ZipArchive) Close() error {
	return z.reader.Close()
}

func (z *ZipArchive) States() []string {
	return gnafStates
}

func (z *ZipArchive) OpenStandardPSV(state, table string) (io.ReadCloser, error) {
	name := fmt.Sprintf("%s%s_%s_psv.psv", z.standard, state, table)
	return z.openEntry(name)
}

func (z *ZipArchive) OpenAuthorityPSV(table string) (io.ReadCloser, error) {
	authPrefix := strings.Replace(z.standard, "Standard/", "Authority Code/", 1)
	name := authPrefix + table + "_psv.psv"
	return z.openEntry(name)
}

func (z *ZipArchive) openEntry(name string) (io.ReadCloser, error) {
	for _, f := range z.reader.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			return rc, nil
		}
	}
	return nil, fmt.Errorf("zip entry not found: %s", filepath.Base(name))
}

func ValidateGNAFZip(zipPath string) error {
	z, err := OpenZipArchive(zipPath)
	if err != nil {
		return err
	}
	z.Close()
	return nil
}
