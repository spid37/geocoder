package regions

import "fmt"

type DatasetUpdate struct {
	Name             string
	LocalResourceID  string
	LatestResourceID string
	UpToDate         bool
}

type UpdateStatus struct {
	Local    *Manifest
	ABS      DatasetUpdate
	UpToDate bool
}

func CheckUpdate(dataDir string) (*UpdateStatus, error) {
	local, err := LoadManifest(dataDir)
	if err != nil {
		return nil, err
	}

	absRes, err := ResolveABSAllocation()
	if err != nil {
		return nil, fmt.Errorf("resolve ABS allocation: %w", err)
	}

	status := &UpdateStatus{
		Local: local,
		ABS: DatasetUpdate{
			Name:             absRes.Name,
			LatestResourceID: absRes.ID,
			UpToDate:         local != nil && local.ABSResourceID == absRes.ID && fileExists(local.ABSPath),
		},
	}
	if local != nil {
		status.ABS.LocalResourceID = local.ABSResourceID
	}
	status.UpToDate = status.ABS.UpToDate
	return status, nil
}

func manifestCurrent(existing *Manifest, absRes *Resource) bool {
	return existing != nil &&
		existing.ABSResourceID == absRes.ID &&
		fileExists(existing.ABSPath)
}
