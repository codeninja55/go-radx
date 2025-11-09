.PHONY: help setup benchmark benchmark-full benchmark-nocgo visualize clean check-deps

# Colors for output
CYAN := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RESET := \033[0m

# Configuration
MAX_FILES ?= 20
TESTDATA ?= ../testdata
GO_BINARY := pixel_benchmark
UV := uv run

##@ General

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\n$(CYAN)Usage:$(RESET)\n  make $(GREEN)<target>$(RESET)\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  $(GREEN)%-18s$(RESET) %s\n", $$1, $$2 } /^##@/ { printf "\n$(CYAN)%s$(RESET)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

setup: ## Install dependencies and build binaries (run this first!)
	@chmod +x setup.sh
	@./setup.sh

##@ Benchmarking

benchmark: check-deps ## Run benchmarks (default: 20 files, CGo enabled)
	@echo "$(CYAN)Building Go binary with CGo...$(RESET)"
	@CGO_ENABLED=1 go build -o $(GO_BINARY) ./pixel_benchmark.go || \
		(echo "$(YELLOW)⚠️  CGo build failed, trying without CGo...$(RESET)" && \
		 CGO_ENABLED=0 go build -o $(GO_BINARY) ./pixel_benchmark.go)
	@echo ""
	@echo "$(CYAN)Running benchmarks ($(MAX_FILES) files)...$(RESET)"
	@$(UV) python pixel_benchmark.py \
		--testdata $(TESTDATA) \
		--go-binary ./$(GO_BINARY) \
		--max-files $(MAX_FILES) \
		--output benchmark_results.json
	@echo ""
	@echo "$(GREEN)✅ Benchmarks complete! Results saved to benchmark_results.json$(RESET)"
	@echo "$(CYAN)Run 'make visualize' to generate charts$(RESET)"

benchmark-full: MAX_FILES=0 ## Run benchmarks on ALL files (warning: slow!)
benchmark-full: benchmark

benchmark-nocgo: ## Run benchmarks without CGo (native formats only)
	@echo "$(CYAN)Building Go binary without CGo...$(RESET)"
	@CGO_ENABLED=0 go build -o $(GO_BINARY)_nocgo ./pixel_benchmark.go
	@echo ""
	@echo "$(YELLOW)⚠️  Note: Compressed formats (JPEG, JPEG 2000) will not work$(RESET)"
	@echo ""
	@echo "$(CYAN)Running benchmarks ($(MAX_FILES) files)...$(RESET)"
	@$(UV) python pixel_benchmark.py \
		--testdata $(TESTDATA) \
		--go-binary ./$(GO_BINARY)_nocgo \
		--max-files $(MAX_FILES) \
		--output benchmark_results_nocgo.json
	@echo ""
	@echo "$(GREEN)✅ Benchmarks complete!$(RESET)"

##@ Visualization

visualize: check-deps ## Generate visualization charts from benchmark results
	@if [ -f benchmark_results.json ]; then \
		echo "$(CYAN)Generating charts from benchmark_results.json...$(RESET)"; \
		$(UV) python visualize_benchmarks.py benchmark_results.json; \
		echo ""; \
		echo "$(GREEN)✅ Charts saved:$(RESET)"; \
		echo "  - benchmark_times.png"; \
		echo "  - benchmark_speedup.png"; \
		echo "  - benchmark_correctness.png"; \
	else \
		echo "$(YELLOW)No benchmark results found.$(RESET)"; \
		echo "Run '$(GREEN)make benchmark$(RESET)' first."; \
		exit 1; \
	fi

##@ Maintenance

clean: ## Remove all generated files
	@echo "$(CYAN)Cleaning up...$(RESET)"
	@rm -f $(GO_BINARY) $(GO_BINARY)_nocgo
	@rm -f benchmark_results*.json
	@rm -f *.png
	@echo "$(GREEN)✅ Clean complete$(RESET)"

check-deps: ## Verify all dependencies are installed
	@echo "$(CYAN)Checking dependencies...$(RESET)"
	@command -v uv >/dev/null 2>&1 || \
		(echo "$(YELLOW)⚠️  uv not found. Run 'make setup' to install$(RESET)" && exit 1)
	@command -v go >/dev/null 2>&1 || \
		(echo "$(YELLOW)⚠️  Go not found. Please install Go 1.23+$(RESET)" && exit 1)
	@$(UV) python -c "import pydicom; import numpy; import matplotlib" 2>/dev/null || \
		(echo "$(YELLOW)⚠️  Python dependencies missing. Run 'make setup'$(RESET)" && exit 1)
	@echo "$(GREEN)✅ All dependencies installed$(RESET)"

##@ Quick Examples

quick: MAX_FILES=10 ## Quick benchmark (10 files)
quick: benchmark

test: MAX_FILES=5 ## Test benchmark (5 files)
test: benchmark

# Default target
.DEFAULT_GOAL := help
