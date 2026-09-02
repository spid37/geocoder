package version

import (
	"os"
	"testing"
)

func TestStringLdflags(t *testing.T) {
	orig := ldflagsVersion
	ldflagsVersion = "2026.09.02+1"
	t.Cleanup(func() { ldflagsVersion = orig })

	t.Setenv(envVar, "")
	if got := String(); got != "2026.09.02+1" {
		t.Fatalf("String() = %q, want ldflags version", got)
	}
}

func TestStringEnvFallback(t *testing.T) {
	orig := ldflagsVersion
	ldflagsVersion = ""
	t.Cleanup(func() { ldflagsVersion = orig })

	t.Setenv(envVar, "2026.01.01+99")
	if got := String(); got != "2026.01.01+99" {
		t.Fatalf("String() = %q, want env fallback", got)
	}
}

func TestStringLdflagsOverridesEnv(t *testing.T) {
	orig := ldflagsVersion
	ldflagsVersion = "2026.12.31+42"
	t.Cleanup(func() { ldflagsVersion = orig })

	t.Setenv(envVar, "2026.01.01+1")
	if got := String(); got != "2026.12.31+42" {
		t.Fatalf("String() = %q, want ldflags override", got)
	}
}

func TestStringEmptyWhenUnset(t *testing.T) {
	if os.Getenv(envVar) != "" {
		t.Skip("VERSION is set in the environment")
	}
	orig := ldflagsVersion
	ldflagsVersion = ""
	t.Cleanup(func() { ldflagsVersion = orig })

	if got := String(); got != "" {
		t.Fatalf("String() = %q, want empty", got)
	}
}
