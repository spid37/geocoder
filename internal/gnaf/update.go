package gnaf

import "net/http"

type UpdateStatus struct {
	Local    *Manifest
	Latest   *Release
	UpToDate bool
}

func CheckUpdate(dataDir string) (*UpdateStatus, error) {
	return CheckUpdateWithClient(dataDir, nil)
}

func CheckUpdateWithClient(dataDir string, client *http.Client) (*UpdateStatus, error) {
	local, err := LoadManifest(dataDir)
	if err != nil {
		return nil, err
	}

	latest, err := ResolveLatestRelease(client)
	if err != nil {
		return nil, err
	}

	upToDate := local != nil &&
		local.ResourceID == latest.ResourceID &&
		local.ZipPath != "" &&
		ValidateGNAFZip(local.ZipPath) == nil

	return &UpdateStatus{
		Local:    local,
		Latest:   latest,
		UpToDate: upToDate,
	}, nil
}
