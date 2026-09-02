# Application version (YYYY.MM.DD+BUILD). Override: make build VERSION=2026.09.02+2
VERSION_FILE ?= VERSION
VERSION ?= $(shell if [ -f "$(VERSION_FILE)" ]; then grep -v '^[[:space:]]*\#' "$(VERSION_FILE)" | grep -v '^[[:space:]]*$$' | head -n1 | tr -d '[:space:]'; else echo dev; fi)

BINARY ?= geocoder
CMD ?= ./cmd/geocoder
SET_VERSION ?= ./scripts/set-version.sh

LDFLAGS := -s -w -X github.com/spid37/geocoder/internal/version.ldflagsVersion=$(VERSION)
TEST_LDFLAGS := -X github.com/spid37/geocoder/internal/version.ldflagsVersion=$(VERSION)

.PHONY: build test clean version version-bump

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o "$(BINARY)" "$(CMD)"

test:
	go test -ldflags="$(TEST_LDFLAGS)" ./...

clean:
	rm -f "$(BINARY)"

version: ## Print VERSION from file
	@echo "$(VERSION)"

version-bump: ## Validate and normalize VERSION
	$(SET_VERSION) --bump-build
