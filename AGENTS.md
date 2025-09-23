# Agent Guidelines for devflow

## Build/Test/Lint Commands
- Build: `make build` (current platform) or `go build -o bin/devflow`
- Test all: `make test` or `go test ./...`
- Test single package: `go test ./internal/config`
- Lint: `make lint` or `golangci-lint run`
- Clean: `make clean`

## Code Style Guidelines
- Go version: 1.25+ (see go.mod)
- Package structure: `cmd/` for CLI commands, `internal/` for private packages
- Imports: Standard library first, then external packages, then internal packages
- Error handling: Wrap errors with context using `fmt.Errorf("message: %w", err)`
- Structs: Use JSON tags for serialization (`json:"field_name"`)
- Variables: camelCase for local vars, PascalCase for exported types/functions
- Config files: Store in `~/.devflow/` directory with 0600 permissions
- Testing: Use table-driven tests, `t.TempDir()` for temporary files
- CLI: Use cobra.Command with `Use`, `Short`, and `Long` descriptions
- Types: Prefer explicit types over `interface{}`, use struct embedding when appropriate

## Dependencies
- CLI framework: github.com/spf13/cobra
- Linting: golangci-lint (install via `make dev-setup`)