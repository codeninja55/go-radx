.PHONY: help fmt lint test test-coverage clean check all check-cgo-deps install-cgo-deps build build-cli

# Default target
.DEFAULT_GOAL := help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOPATH_BIN=$(shell go env GOPATH)/bin
GOLINT=$(GOPATH_BIN)/golangci-lint

# Build parameters
BINARY_NAME=go-dicom
BINARY_PATH=./bin/$(BINARY_NAME)

# Test parameters
TEST_TIMEOUT=10m
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# CGo parameters
export CGO_ENABLED=1

# Platform detection
UNAME_S := $(shell uname -s)
HAS_BREW := $(shell command -v brew 2> /dev/null)
HAS_APT := $(shell command -v apt-get 2> /dev/null)
HAS_YUM := $(shell command -v yum 2> /dev/null)
HAS_DNF := $(shell command -v dnf 2> /dev/null)
HAS_PKG_CONFIG := $(shell command -v pkg-config 2> /dev/null)

# SBOM parameters
TRIVY=$(GOPATH_BIN)/trivy
SBOM_FORMAT=cyclonedx
SBOM_DIR=./sbom
SBOM_FILE=$(SBOM_DIR)/go-dicom.sbom.json
SBOM_VULN_FILE=$(SBOM_DIR)/go-dicom.sbom-with-vulns.json
SBOM_FULL_FILE=$(SBOM_DIR)/go-dicom.sbom-full.json

help: ## Display this help message
	@echo "Go-DICOM Makefile Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

fmt: ## Format Go code with gofmt
	@echo "Running gofmt..."
	@$(GOFMT) -w -s .
	@echo "✓ Code formatted"

fmt-check: ## Check if code is formatted
	@echo "Checking code formatting..."
	@test -z "$$($(GOFMT) -l .)" || (echo "Code is not formatted. Run 'make fmt'" && exit 1)
	@echo "✓ Code formatting check passed"

lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@which $(GOLINT) > /dev/null || (echo "golangci-lint not installed. Run: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin" && exit 1)
	@$(GOLINT) run ./...
	@echo "✓ Linting passed"

.PHONY: help fmt fmt-check lint

test: ## Run tests
	@echo "Running tests..."
	@$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./...
	@echo "✓ Tests passed"

test-short: ## Run short tests only
	@echo "Running short tests..."
	@$(GOTEST) -v -short -timeout $(TEST_TIMEOUT) ./...
	@echo "✓ Short tests passed"

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@$(GOTEST) -v -timeout $(TEST_TIMEOUT) -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@echo "✓ Tests passed with coverage"

coverage-html: test-coverage ## Generate HTML coverage report
	@echo "Generating HTML coverage report..."
	@$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "✓ Coverage report generated: $(COVERAGE_HTML)"

coverage-report: test-coverage ## Display coverage report in terminal
	@echo "Coverage report:"
	@$(GOCMD) tool cover -func=$(COVERAGE_FILE)

.PHONY: test test-short test-coverage coverage-html coverage-report

tidy: ## Tidy Go modules
	@echo "Tidying Go modules..."
	@$(GOMOD) tidy
	@echo "✓ Go modules tidied"

verify: ## Verify Go modules
	@echo "Verifying Go modules..."
	@$(GOMOD) verify
	@echo "✓ Go modules verified"

download: ## Download Go module dependencies
	@echo "Downloading dependencies..."
	@$(GOMOD) download
	@echo "✓ Dependencies downloaded"

check: fmt-check lint check-cgo-deps test ## Run all checks (format, lint, CGo deps, test)
	@echo "✓ All checks passed"

all: clean tidy fmt lint test ## Clean, tidy, format, lint, and test
	@echo "✓ All tasks completed"

clean: ## Remove build artifacts, coverage files, and SBOMs
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@rm -rf ./bin
	@rm -rf $(SBOM_DIR)
	@echo "✓ Cleaned"

.PHONY: tidy verify download check all clean

sbom: ## Generate CycloneDX SBOM (packages only)
	@echo "Generating CycloneDX v1.6 SBOM..."
	@mkdir -p $(SBOM_DIR)
	@which $(TRIVY) > /dev/null || (echo "Trivy not installed. Run 'make install-sbom-tools'" && exit 1)
	@$(TRIVY) fs --format $(SBOM_FORMAT) --output $(SBOM_FILE) .
	@echo "✓ SBOM generated: $(SBOM_FILE)"

sbom-vuln: ## Generate CycloneDX SBOM with vulnerabilities
	@echo "Generating CycloneDX v1.6 SBOM with vulnerabilities..."
	@mkdir -p $(SBOM_DIR)
	@which $(TRIVY) > /dev/null || (echo "Trivy not installed. Run 'make install-sbom-tools'" && exit 1)
	@$(TRIVY) fs --format $(SBOM_FORMAT) --scanners vuln --output $(SBOM_VULN_FILE) .
	@echo "✓ SBOM with vulnerabilities generated: $(SBOM_VULN_FILE)"

sbom-full: ## Generate CycloneDX SBOM with vulnerabilities and licenses
	@echo "Generating comprehensive CycloneDX v1.6 SBOM..."
	@mkdir -p $(SBOM_DIR)
	@which $(TRIVY) > /dev/null || (echo "Trivy not installed. Run 'make install-sbom-tools'" && exit 1)
	@$(TRIVY) fs --format $(SBOM_FORMAT) --scanners vuln,license --list-all-pkgs --output $(SBOM_FULL_FILE) .
	@echo "✓ Comprehensive SBOM generated: $(SBOM_FULL_FILE)"
	@echo "  Includes: packages, vulnerabilities, and licenses"

sbom-validate: ## Validate generated SBOM files
	@echo "Validating SBOM files..."
	@if [ -f $(SBOM_FILE) ]; then \
		echo "Checking $(SBOM_FILE)..."; \
		$(TRIVY) sbom $(SBOM_FILE) --quiet && echo "✓ $(SBOM_FILE) is valid"; \
	fi
	@if [ -f $(SBOM_VULN_FILE) ]; then \
		echo "Checking $(SBOM_VULN_FILE)..."; \
		$(TRIVY) sbom $(SBOM_VULN_FILE) --quiet && echo "✓ $(SBOM_VULN_FILE) is valid"; \
	fi
	@if [ -f $(SBOM_FULL_FILE) ]; then \
		echo "Checking $(SBOM_FULL_FILE)..."; \
		$(TRIVY) sbom $(SBOM_FULL_FILE) --quiet && echo "✓ $(SBOM_FULL_FILE) is valid"; \
	fi
	@echo "✓ SBOM validation complete"

sbom-scan: sbom-full ## Scan the generated SBOM for vulnerabilities
	@echo "Scanning SBOM for vulnerabilities..."
	@$(TRIVY) sbom --scanners vuln $(SBOM_FULL_FILE)
	@echo "✓ SBOM vulnerability scan complete"

.PHONY: sbom sbom-vuln sbom-full sbom-validate sbom-scan

check-cgo-deps: ## Check CGo dependencies (C compiler, libraries)
	@echo "Checking CGo dependencies..."
	@# Check CGO_ENABLED
	@if [ "$$(go env CGO_ENABLED)" != "1" ]; then \
		echo "✗ CGO_ENABLED is not set to 1"; \
		echo "  Run: export CGO_ENABLED=1"; \
		exit 1; \
	fi
	@echo "✓ CGO_ENABLED=1"
	@# Check C compiler
	@if ! command -v gcc > /dev/null && ! command -v clang > /dev/null; then \
		echo "✗ No C compiler found (gcc or clang required)"; \
		echo "  macOS: xcode-select --install"; \
		echo "  Linux: apt-get install build-essential (or equivalent)"; \
		exit 1; \
	fi
	@echo "✓ C compiler available"
	@# Check pkg-config
	@if ! command -v pkg-config > /dev/null; then \
		echo "✗ pkg-config not found"; \
		echo "  macOS: brew install pkg-config"; \
		echo "  Linux: apt-get install pkg-config"; \
		exit 1; \
	fi
	@echo "✓ pkg-config available"
	@# Check libjpeg
	@if ! pkg-config --exists libjpeg 2>/dev/null; then \
		echo "✗ libjpeg-turbo not found"; \
		echo "  macOS: brew install jpeg-turbo"; \
		echo "  Ubuntu/Debian: apt-get install libturbojpeg0-dev"; \
		echo "  Fedora/RHEL: yum install turbojpeg-devel"; \
		exit 1; \
	fi
	@echo "✓ libjpeg-turbo installed (version: $$(pkg-config --modversion libjpeg))"
	@# Check OpenJPEG
	@if ! pkg-config --exists libopenjp2 2>/dev/null; then \
		echo "✗ OpenJPEG not found"; \
		echo "  macOS: brew install openjpeg"; \
		echo "  Ubuntu/Debian: apt-get install libopenjp2-7-dev"; \
		echo "  Fedora/RHEL: yum install openjpeg2-devel"; \
		exit 1; \
	fi
	@echo "✓ OpenJPEG installed (version: $$(pkg-config --modversion libopenjp2))"
	@echo "✓ All CGo dependencies satisfied"

install-cgo-deps: ## Install CGo dependencies (macOS/Linux only)
	@echo "Installing CGo dependencies..."
ifeq ($(UNAME_S),Darwin)
	@# macOS with Homebrew
	@if [ -n "$(HAS_BREW)" ]; then \
		echo "Installing via Homebrew..."; \
		brew install jpeg-turbo openjpeg pkg-config; \
	else \
		echo "✗ Homebrew not found. Install from https://brew.sh"; \
		exit 1; \
	fi
else ifeq ($(UNAME_S),Linux)
	@# Linux - try apt, then yum/dnf
	@if [ -n "$(HAS_APT)" ]; then \
		echo "Installing via apt-get..."; \
		sudo apt-get update; \
		sudo apt-get install -y libturbojpeg0-dev libopenjp2-7-dev build-essential pkg-config; \
	elif [ -n "$(HAS_DNF)" ]; then \
		echo "Installing via dnf..."; \
		sudo dnf groupinstall -y "Development Tools"; \
		sudo dnf install -y turbojpeg-devel openjpeg2-devel pkg-config; \
	elif [ -n "$(HAS_YUM)" ]; then \
		echo "Installing via yum..."; \
		sudo yum groupinstall -y "Development Tools"; \
		sudo yum install -y turbojpeg-devel openjpeg2-devel pkgconfig; \
	else \
		echo "✗ Unsupported package manager. Please install manually:"; \
		echo "  - libjpeg-turbo"; \
		echo "  - OpenJPEG 2.5+"; \
		exit 1; \
	fi
else
	@echo "✗ Unsupported platform: $(UNAME_S)"
	@echo "  Only macOS and Linux are supported"
	@exit 1
endif
	@echo "✓ CGo dependencies installed"
	@echo "Run 'make check-cgo-deps' to verify installation"

build: ## Build the project with CGo enabled
	@echo "Building with CGo..."
	@CGO_ENABLED=1 $(GOBUILD) -v ./...
	@echo "✓ Build complete"

build-cli: ## Build the dicomutil CLI tool
	@echo "Building dicomutil CLI..."
	@CGO_ENABLED=1 $(GOBUILD) -v -o $(BINARY_PATH) ./cmd/dicomutil
	@echo "✓ CLI built: $(BINARY_PATH)"

install-tools: install-lint-tools install-sbom-tools ## Install all development tools

install-lint-tools: ## Install linting tools (golangci-lint)
	@echo "Installing golangci-lint..."
	@$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "✓ golangci-lint installed at $(GOPATH_BIN)/golangci-lint"
	@$(GOPATH_BIN)/golangci-lint version

install-sbom-tools: ## Install SBOM tools (Trivy)
	@echo "Installing Trivy..."
	@which brew > /dev/null && brew install trivy || \
		(echo "Installing Trivy via install script..." && \
		curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b $(GOPATH_BIN))
	@echo "✓ Trivy installed at $(GOPATH_BIN)/trivy"
	@$(TRIVY) version

.PHONY: install-tools install-lint-tools install-sbom-tools
