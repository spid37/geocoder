package freshness

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/spid37/geocoder/internal/gnaf"
	"github.com/spid37/geocoder/internal/regions"
	"github.com/spid37/geocoder/internal/store"
)

type DBStatus struct {
	GNAFMatchesFiles    bool `json:"gnaf_db_matches_files"`
	RegionsMatchesFiles bool `json:"regions_db_matches_files"`
}

type FilesStatus struct {
	GNAFUpToDate    bool `json:"gnaf_up_to_date"`
	RegionsUpToDate bool `json:"regions_up_to_date"`
}

type Report struct {
	DB      DBStatus    `json:"db"`
	Files   FilesStatus `json:"files"`
	Stale   bool        `json:"stale"`
	Details []string    `json:"details,omitempty"`
}

func CheckDB(dataDir string, db *sql.DB) Report {
	var r Report

	gnafManifest, _ := gnaf.LoadManifest(dataDir)
	regManifest, _ := regions.LoadManifest(dataDir)

	dbResourceID, _ := store.GetMetadata(db, "resource_id")
	dbABSID, _ := store.GetMetadata(db, "abs_resource_id")

	if gnafManifest != nil && gnafManifest.ResourceID != "" {
		if dbResourceID == "" {
			r.DB.GNAFMatchesFiles = false
			r.Stale = true
			r.Details = append(r.Details, "G-NAF not loaded into database — run data load")
		} else {
			r.DB.GNAFMatchesFiles = gnafManifest.ResourceID == dbResourceID
			if !r.DB.GNAFMatchesFiles {
				r.Stale = true
				r.Details = append(r.Details,
					fmt.Sprintf("database G-NAF resource (%s) differs from downloaded files (%s) — run data load",
						dbResourceID, gnafManifest.ResourceID))
			}
		}
	}

	if regManifest != nil {
		if dbABSID == "" {
			r.DB.RegionsMatchesFiles = false
			r.Stale = true
			r.Details = append(r.Details, "regions not loaded into database — run regions load")
		} else {
			r.DB.RegionsMatchesFiles = regManifest.ABSResourceID == dbABSID
			if !r.DB.RegionsMatchesFiles {
				r.Stale = true
				r.Details = append(r.Details,
					"database region resources differ from downloaded files — run regions load")
			}
		}
	}

	return r
}

func Check(dataDir string, db *sql.DB) (Report, error) {
	r := CheckDB(dataDir, db)

	gnafStatus, err := gnaf.CheckUpdate(dataDir)
	if err != nil {
		return r, fmt.Errorf("check G-NAF update: %w", err)
	}
	if gnafStatus.Local == nil {
		r.Files.GNAFUpToDate = false
		r.Stale = true
		r.Details = append(r.Details, "no local G-NAF download — run data download")
	} else {
		r.Files.GNAFUpToDate = gnafStatus.UpToDate
		if !gnafStatus.UpToDate {
			r.Stale = true
			r.Details = append(r.Details,
				fmt.Sprintf("G-NAF update available: latest is %s (%s), local is %s (%s)",
					gnafStatus.Latest.Name, gnafStatus.Latest.ResourceID,
					gnafStatus.Local.ReleaseName, gnafStatus.Local.ResourceID))
		}
	}

	regStatus, err := regions.CheckUpdate(dataDir)
	if err != nil {
		return r, fmt.Errorf("check regions update: %w", err)
	}
	if regStatus.Local == nil {
		r.Files.RegionsUpToDate = false
		r.Stale = true
		r.Details = append(r.Details, "no local region data — run regions download")
	} else {
		r.Files.RegionsUpToDate = regStatus.UpToDate
		if !regStatus.ABS.UpToDate {
			r.Stale = true
			r.Details = append(r.Details,
				fmt.Sprintf("ABS allocation update available: latest is %s (%s), local is %s",
					regStatus.ABS.Name, regStatus.ABS.LatestResourceID, regStatus.ABS.LocalResourceID))
		}
	}

	return r, nil
}

func FormatWarnings(report Report) string {
	if len(report.Details) == 0 {
		return ""
	}
	return "Data may be stale:\n  - " + strings.Join(report.Details, "\n  - ")
}
