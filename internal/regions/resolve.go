package regions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const ckanBaseURL = "https://data.gov.au/data/api/3/action"

type Resource struct {
	ID           string
	Name         string
	URL          string
	LastModified time.Time
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

func resolveResource(datasetID string, match func(ckanResource) bool) (*Resource, error) {
	url := fmt.Sprintf("%s/package_show?id=%s", ckanBaseURL, datasetID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pkg ckanPackageResponse
	if err := json.NewDecoder(resp.Body).Decode(&pkg); err != nil {
		return nil, err
	}
	if !pkg.Success {
		return nil, fmt.Errorf("ckan package_show failed for %s", datasetID)
	}

	var candidates []Resource
	for _, r := range pkg.Result.Resources {
		if !strings.EqualFold(r.State, "active") || r.URL == "" {
			continue
		}
		if !match(r) {
			continue
		}
		lastMod, _ := time.Parse(time.RFC3339, r.LastModified)
		candidates = append(candidates, Resource{
			ID:           r.ID,
			Name:         r.Name,
			URL:          r.URL,
			LastModified: lastMod,
		})
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no matching resource in dataset %s", datasetID)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].LastModified.After(candidates[j].LastModified)
	})
	return &candidates[0], nil
}

func ResolveABSAllocation() (*Resource, error) {
	return resolveResource(ABSAllocationDatasetID, func(r ckanResource) bool {
		return strings.EqualFold(r.Format, "ZIP") &&
			strings.Contains(strings.ToUpper(r.Name), "ALLOCATION")
	})
}
