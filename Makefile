.DEFAULT_GOAL := help

.PHONY: build build-all clean test lint release check-clean install help

# Build for current platform
build: ## Build for current platform
	mkdir -p bin
	go build -ldflags "-X 'devflow/cmd.version=dev'" -o bin/devflow

# Build for multiple platforms
build-all: ## Build for multiple platforms
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/devflow-linux-amd64
	GOOS=darwin GOARCH=amd64 go build -o bin/devflow-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -o bin/devflow-darwin-arm64
	GOOS=windows GOARCH=amd64 go build -o bin/devflow-windows-amd64.exe

# Clean build artifacts
clean: ## Clean build artifacts
	rm -rf bin/

# Run tests
test: ## Run tests
	go test ./...

# Run linter
lint: ## Run linter
	golangci-lint run

# Install dependencies
deps: ## Install dependencies
	go mod tidy
	go mod download

# Development setup
dev-setup: deps ## Development setup
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Check for unstaged/uncommitted files
check-clean: ## Check for unstaged/uncommitted files
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Warning: There are unstaged or uncommitted files. Please commit or stash them."; \
		exit 1; \
	fi

# Create a new release tag
release: check-clean ## Create a new release tag (requires BUMP_TYPE=major|minor|patch)
	@new_version=$$(./scripts/bump-version.sh $(BUMP_TYPE)); \
	go build -ldflags "-X 'devflow/cmd.version=v$$new_version'" -o bin/devflow; \
	git tag "v$$new_version"; \
	echo "Created tag v$$new_version and built binary with embedded version v$$new_version"

# Install the program to the Go bin directory
install: ## Install the program to the Go bin directory
	@latest_version=$$(./scripts/git-latest-release.sh); \
	if [ "$$latest_version" = "No releases found" ]; then \
		version="0.0.0"; \
	else \
		version=$${latest_version#v}; \
	fi; \
	go install -ldflags "-X 'devflow/cmd.version=v$$version'" .

# Show help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'