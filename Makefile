.PHONY: build build-all clean test lint

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