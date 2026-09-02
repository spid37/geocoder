package gnaf

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/spid37/geocoder/internal/config"
)

type Release struct {
	Name         string
	ResourceID   string
	URL          string
	LastModified time.Time
	Datum        string
}

type ckanPackageResponse struct {
	Success bool `json:"success"`
	Result  struct {
		Resources []ckanResource `json:"resources"`
	} `json:"result"`
}

type ckanResource struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Format       string `json:"format"`
	URL          string `json:"url"`
	State        string `json:"state"`
	LastModified string `json:"last_modified"`
}

func ResolveLatestRelease(client *http.Client) (*Release, error) {
	if client == nil {
		client = http.DefaultClient
	}

	url := fmt.Sprintf("%s/package_show?id=%s", config.CKANBaseURL, config.GNAFDatasetID)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ckan request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ckan request: status %d", resp.StatusCode)
	}

	var pkg ckanPackageResponse
	if err := json.NewDecoder(resp.Body).Decode(&pkg); err != nil {
		return nil, fmt.Errorf("decode ckan response: %w", err)
	}
	if !pkg.Success {
		return nil, fmt.Errorf("ckan package_show failed")
	}

	var candidates []Release
	for _, r := range pkg.Result.Resources {
		if !strings.EqualFold(r.State, "active") {
			continue
		}
		if !strings.EqualFold(r.Format, "ZIP") {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(r.Description), "GDA2020") {
			continue
		}
		if r.URL == "" {
			continue
		}

		lastMod, _ := time.Parse(time.RFC3339, r.LastModified)
		candidates = append(candidates, Release{
			Name:         r.Name,
			ResourceID:   r.ID,
			URL:          r.URL,
			LastModified: lastMod,
			Datum:        "GDA2020",
		})
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no GDA2020 ZIP resource found in G-NAF dataset")
	}

	sort.Slice(candidates, func(i, j int) bool {
		if !candidates[i].LastModified.Equal(candidates[j].LastModified) {
			return candidates[i].LastModified.After(candidates[j].LastModified)
		}
		return candidates[i].Name > candidates[j].Name
	})

	return &candidates[0], nil
}
