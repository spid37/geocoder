package version

import (
	"os"
	"strings"
)

// Set at link time via -ldflags from the root VERSION file (see Makefile).
var ldflagsVersion string

const envVar = "VERSION"

// String returns the application version.
// Priority: -ldflags (make build), VERSION env var.
func String() string {
	if v := strings.TrimSpace(ldflagsVersion); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv(envVar))
}
