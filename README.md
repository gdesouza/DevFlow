# DevFlow

[![Build Status](https://gdesouza.semaphoreci.com/badges/DevFlow/branches/main.svg?style=shields&key=dbff7292-7a82-4922-b626-72eefeef5b82)](https://gdesouza.semaphoreci.com/projects/DevFlow)
[![codecov](https://codecov.io/gh/gdesouza/DevFlow/branch/main/graph/badge.svg?token=Y6UTBMGXV9)](https://codecov.io/gh/gdesouza/DevFlow)

DevFlow is a command-line tool for working with Jira, Bitbucket, Jenkins, and local Git repositories from one interface.

It is designed for developers who want to inspect tasks, manage pull requests, check build status, and automate common workflow operations without switching between several web applications.

## Features

- Manage Jira tasks, comments, links, assignments, and status updates.
- List, inspect, create, and update Bitbucket pull requests.
- Search and inspect Bitbucket repositories.
- Review Jenkins builds and console logs.
- Inspect local Git repository status across a directory tree.
- Produce detailed, tabular, normalized JSON, or raw API-shaped output.

## Installation

### Debian or Ubuntu

```bash
curl -s https://packagecloud.io/install/repositories/gdesouza/devflow/script.deb.sh | sudo bash
sudo apt-get install devflow
```

### Prebuilt binaries

Download a binary for your platform from the [DevFlow releases page](https://github.com/gdesouza/DevFlow/releases).

### From source

```bash
git clone https://github.com/gdesouza/DevFlow.git
cd DevFlow
make install
```

### Using `go install`

Install the latest version directly from GitHub:

```bash
go install github.com/gdesouza/DevFlow@latest
```

To install the current checkout instead:

```bash
go install .
```

If the `devflow` command is not found afterward, add Go's binary directory to your `PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

Verify the installation:

```bash
devflow version
devflow --help
```

## Quick start

Configure the integrations you use:

```bash
devflow config set jira.url https://your-domain.atlassian.net
devflow config set jira.username you@example.com
devflow config set jira.token "$JIRA_TOKEN"

devflow config set bitbucket.workspace your-workspace
devflow config set bitbucket.username you@example.com
devflow config set bitbucket.token "$BITBUCKET_TOKEN"
```

List your assigned Jira tasks:

```bash
devflow tasks list
```

Inspect a task and its child items:

```bash
devflow tasks show ENG-123 --children
```

List pull requests for a repository:

```bash
devflow pullrequest list --repo my-repository
```

Inspect recent Jenkins builds:

```bash
devflow jenkins builds my-pipeline
```

## Configuration

DevFlow stores configuration under `~/.devflow/`.

Use the CLI to manage configuration values:

```bash
devflow config set <section>.<key> <value>
devflow config get <section>.<key>
```

Verify Bitbucket authentication with:

```bash
devflow auth status
```

See [Configuration](docs/configuration.md) for supported credentials, authentication modes, and integration-specific requirements.

## Common commands

### Jira

```bash
devflow tasks list
devflow tasks list --filter "In Progress"
devflow tasks show ENG-123
devflow tasks show ENG-123 --children --pull-requests
devflow tasks create --project ENG "Investigate API timeout"
devflow tasks comment ENG-123 "Investigation is complete"
```

### Bitbucket repositories

```bash
devflow repo list
devflow repo search 'api-.*'
devflow repo show my-repository
devflow repo remotes my-repository
devflow repo readme my-repository
```

### Pull requests

```bash
devflow pullrequest list --repo my-repository
devflow pullrequest show my-repository 123
devflow pullrequest create "Improve caching" --repo my-repository
devflow pullrequest comments my-repository 123
devflow pullrequest diff my-repository 123
```

### Jenkins

```bash
devflow jenkins builds my-pipeline
devflow jenkins logs my-pipeline 42
```

### Local Git repositories

```bash
devflow git list
devflow git list --path ~/src --no-fetch
```

See the [command reference](docs/commands.md) for the complete command and flag reference.

## Output formats

Commands use the global `--format` option where applicable:

```bash
devflow tasks show ENG-123 --format detailed
devflow tasks show ENG-123 --format json
devflow tasks show ENG-123 --format raw
devflow tasks show ENG-123 --format tabular
```

- `detailed`: human-readable output; the default.
- `json`: normalized output containing relevant fields.
- `raw`: raw JSON shaped closely like the API response.
- `tabular`: aligned output intended for terminal use.

Normalized JSON is intended for scripts:

```bash
devflow tasks show ENG-123 --format json |
  jq '{key, summary, status, pull_requests}'
```

The legacy `--json` and `--tabular` flags remain available for compatibility but are deprecated.

## Development

Requirements:

- Go 1.25 or newer
- `golangci-lint`

Common commands:

```bash
make build
make test
make lint
```

Run the application locally:

```bash
go run . --help
```

Development session notes are kept in [docs/sessions](docs/sessions/).

## License

DevFlow is released under the MIT License. See [LICENSE](LICENSE).
