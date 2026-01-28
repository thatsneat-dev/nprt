.PHONY: build test format-check format vulncheck clean version-bump release-prep

BINARY_NAME := nprt
BUILD_DIR := bin
VERSION_FILE := VERSION
VERSION := $(shell cat $(VERSION_FILE) 2>/dev/null || echo "0.0.0")

# Detect if this is a CI build (set CI=true in CI environment)
ifdef CI
	BUILD_VERSION := $(VERSION)
else
	BUILD_VERSION := $(VERSION)-dev
endif

build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-X main.version=$(BUILD_VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/nprt

test:
	go test -v ./...

format-check:
	@echo "Checking formatting with gofumpt..."
	@if command -v gofumpt >/dev/null 2>&1; then \
		DIFF=$$(gofumpt -d .); \
		if [ -n "$$DIFF" ]; then \
			echo "$$DIFF"; \
			echo ""; \
			echo "The above files are not properly formatted. Run 'make format' to fix."; \
			exit 1; \
		fi; \
		echo "All files are properly formatted."; \
	else \
		echo "gofumpt not installed. Install with: go install mvdan.cc/gofumpt@latest"; \
		exit 1; \
	fi

format:
	@echo "Formatting with gofumpt..."
	@if command -v gofumpt >/dev/null 2>&1; then \
		gofumpt -w .; \
	else \
		echo "gofumpt not installed. Install with: go install mvdan.cc/gofumpt@latest"; \
		exit 1; \
	fi

vuln-check:
	@echo "Checking for vulnerabilities with govulncheck..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not installed. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
		exit 1; \
	fi

clean:
	rm -rf $(BUILD_DIR)

# Version bumping targets
# Usage: make version-bump TYPE=patch|minor|major
version-bump:
	@if [ -z "$(TYPE)" ]; then \
		echo "Usage: make version-bump TYPE=<patch|minor|major>"; \
		echo "Current version: $(VERSION)"; \
		exit 1; \
	fi
	@CURRENT=$(VERSION); \
	MAJOR=$$(echo $$CURRENT | cut -d. -f1); \
	MINOR=$$(echo $$CURRENT | cut -d. -f2); \
	PATCH=$$(echo $$CURRENT | cut -d. -f3); \
	case "$(TYPE)" in \
		major) \
			MAJOR=$$((MAJOR + 1)); \
			MINOR=0; \
			PATCH=0; \
			;; \
		minor) \
			MINOR=$$((MINOR + 1)); \
			PATCH=0; \
			;; \
		patch) \
			PATCH=$$((PATCH + 1)); \
			;; \
		*) \
			echo "Invalid TYPE: $(TYPE). Use patch, minor, or major."; \
			exit 1; \
			;; \
	esac; \
	NEW_VERSION="$$MAJOR.$$MINOR.$$PATCH"; \
	echo "$$NEW_VERSION" > $(VERSION_FILE); \
	echo "Version bumped: $(VERSION) -> $$NEW_VERSION"

# Release preparation - run all checks and sync before tagging
# Usage: make release-prep TYPE=patch|minor|major
release-prep:
	@if [ -z "$(TYPE)" ]; then \
		echo "Usage: make release-prep TYPE=<patch|minor|major>"; \
		echo "Current version: $(VERSION)"; \
		exit 1; \
	fi
	@echo "=== Release Preparation ==="
	@echo ""
	@echo "Step 1/5: Formatting code..."
	@$(MAKE) format
	@echo ""
	@echo "Step 2/5: Running go mod tidy..."
	go mod tidy
	@echo ""
	@echo "Step 3/5: Syncing vendor directory..."
	go mod vendor
	@echo ""
	@echo "Step 4/5: Bumping version..."
	@$(MAKE) version-bump TYPE=$(TYPE)
	@echo ""
	@echo "Step 5/5: Updating flake.lock..."
	nix flake update
	@echo ""
	@echo "=== Release Preparation Complete ==="
	@echo "Next steps:"
	@echo "  1. Review changes: git diff"
	@echo "  2. Commit: git commit -am 'release: v$$(cat $(VERSION_FILE))'"
	@echo "  3. Tag: git tag v$$(cat $(VERSION_FILE))"
	@echo "  4. Push: git push && git push --tags"
