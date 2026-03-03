# Release Process

This document describes how to create and publish a new release of DevFlow.

## Prerequisites

1. Ensure you have commit access to the repository
2. Ensure all changes are committed and pushed to the main branch
3. Update `CHANGELOG.md` with release notes (if applicable)

## Creating a Release

DevFlow uses semantic versioning (`v<major>.<minor>.<patch>`). To create a new release:

### 1. Tag the Release

```bash
# For a patch release (bug fixes)
git tag v1.0.1

# For a minor release (new features, backwards compatible)
git tag v1.1.0

# For a major release (breaking changes)
git tag v2.0.0

# Push the tag to GitHub
git push origin <tag-name>
```

### 2. Automated Release Process

Once you push a version tag (format: `v*.*.*`), the GitHub Actions workflow will automatically:

1. **Build the binary** with the version embedded
2. **Create a Debian package** (`devflow_<version>_amd64.deb`)
3. **Upload to PackageCloud** for easy installation via apt
4. **Create a GitHub Release** with:
   - Debian package attachment
   - Binaries for multiple platforms (Linux, macOS, Windows)
   - SHA256 checksums
   - Installation instructions

### 3. PackageCloud Setup

For the PackageCloud upload to work, you need to configure the `PACKAGECLOUD_TOKEN` secret in your GitHub repository:

1. Go to https://packagecloud.io and create an account
2. Create a repository named `devflow`
3. Get your API token from https://packagecloud.io/api_token
4. Add it as a secret in GitHub:
   - Go to your repository → Settings → Secrets and variables → Actions
   - Click "New repository secret"
   - Name: `PACKAGECLOUD_TOKEN`
   - Value: Your PackageCloud API token

## Installation Methods

After a release is published, users can install DevFlow in several ways:

### From PackageCloud (Recommended for Debian/Ubuntu)

```bash
# Add the repository (one-time setup)
curl -s https://packagecloud.io/install/repositories/gdesouza/devflow/script.deb.sh | sudo bash

# Install DevFlow
sudo apt-get install devflow

# Update to latest version
sudo apt-get update && sudo apt-get upgrade devflow
```

### From Debian Package

```bash
# Download the .deb file from GitHub Releases
wget https://github.com/gdesouza/DevFlow/releases/download/v1.0.0/devflow_1.0.0_amd64.deb

# Install the package
sudo dpkg -i devflow_1.0.0_amd64.deb
```

### From Source with Go

```bash
# Install specific version
go install github.com/gdesouza/DevFlow@v1.0.0

# Install latest version
go install github.com/gdesouza/DevFlow@latest
```

### From Pre-built Binaries

Download the appropriate binary for your platform from the [GitHub Releases page](https://github.com/gdesouza/DevFlow/releases):

- `devflow-linux-amd64` - Linux (Intel/AMD 64-bit)
- `devflow-linux-arm64` - Linux (ARM 64-bit)
- `devflow-darwin-amd64` - macOS (Intel)
- `devflow-darwin-arm64` - macOS (Apple Silicon)
- `devflow-windows-amd64.exe` - Windows

```bash
# Example for Linux
wget https://github.com/gdesouza/DevFlow/releases/download/v1.0.0/devflow-linux-amd64
chmod +x devflow-linux-amd64
sudo mv devflow-linux-amd64 /usr/local/bin/devflow
```

## Supported Distributions

The Debian package is automatically uploaded to PackageCloud for the following distributions:

- **Ubuntu**: 20.04 (Focal), 22.04 (Jammy), 24.04 (Noble)
- **Debian**: 11 (Bullseye), 12 (Bookworm)

## Troubleshooting

### Release workflow fails

1. Check the GitHub Actions logs for detailed error messages
2. Ensure the `PACKAGECLOUD_TOKEN` secret is set correctly
3. Verify the tag format matches `v*.*.*` (e.g., `v1.0.0`, not `1.0.0`)

### PackageCloud upload fails

1. Verify your PackageCloud token is valid
2. Ensure the repository exists at https://packagecloud.io/gdesouza/devflow
3. Check if the distribution names match your PackageCloud repository configuration

### Version mismatch

The version displayed by `devflow version` should match the git tag. If it shows "dev":
- Ensure you're building from a tagged commit
- The `-ldflags` flag must be set correctly during build

## Manual Release (Fallback)

If the automated workflow fails, you can create a release manually:

```bash
# Build the binary
make build

# Create Debian package manually
mkdir -p pkg/DEBIAN pkg/usr/bin
cp bin/devflow pkg/usr/bin/
# ... (create control file)
dpkg-deb --build pkg devflow_1.0.0_amd64.deb

# Upload to PackageCloud
gem install package_cloud
package_cloud push gdesouza/devflow/ubuntu/jammy devflow_1.0.0_amd64.deb

# Create GitHub release manually via web interface
```

## Version Management

Version information is embedded at build time using Go's `-ldflags`:

```bash
# Development build (shows "dev")
go build -o devflow

# Release build (shows actual version)
go build -ldflags "-X 'devflow/cmd.version=v1.0.0'" -o devflow
```

The Makefile includes targets for version management:
- `make release BUMP_TYPE=patch|minor|major` - Create and tag a new release
- `make install VERSION=v1.0.0` - Install a specific version
