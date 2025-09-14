.PHONY: build build-all clean test lint release check-clean

# Build for current platform
build:
	go build -o bin/devflow

# Build for multiple platforms
build-all:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/devflow-linux-amd64
	GOOS=darwin GOARCH=amd64 go build -o bin/devflow-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -o bin/devflow-darwin-arm64
	GOOS=windows GOARCH=amd64 go build -o bin/devflow-windows-amd64.exe

# Clean build artifacts
clean:
	rm -rf bin/

# Run tests
test:
	go test ./...

# Run linter
lint:
	golangci-lint run

# Install dependencies
deps:
	go mod tidy
	go mod download

# Development setup
dev-setup: deps
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Check for unstaged/uncommitted files
check-clean:
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Warning: There are unstaged or uncommitted files. Please commit or stash them."; \
		exit 1; \
	fi

# Create a new release tag
release: check-clean
	@new_version=$$(./scripts/bump-version.sh $(BUMP_TYPE)); \
	git tag "v$$new_version"; \
	echo "Created tag v$$new_version"