# Command reference

Run `devflow <command> --help` for flags and examples specific to a command.

## Global options

All commands support the global output selector where applicable:

```text
--format detailed   Human-readable output (default)
--format json       Normalized JSON
--format raw        Raw API-shaped JSON
--format tabular    Aligned table output
```

The legacy `--json` and `--tabular` flags are deprecated compatibility aliases.

## Top-level commands

| Command | Purpose |
| --- | --- |
| `auth` | Check integration authentication |
| `config` | Read and update configuration |
| `git` | Inspect local Git repositories |
| `jenkins` | Inspect Jenkins builds and logs |
| `repo` | Manage Bitbucket repositories and pipelines |
| `tasks` | Manage Jira tasks and issues |
| `pullrequest` | Manage Bitbucket pull requests |
| `version` | Print the DevFlow version |

## Jira tasks

| Command | Purpose |
| --- | --- |
| `tasks list` | List assigned tasks with filtering and sorting |
| `tasks show <issue-key>` | Show an issue and optional children or pull requests |
| `tasks mentioned` | Find issues where the current user is mentioned |
| `tasks create <title>` | Create a Jira issue |
| `tasks update <issue-key>` | Update Jira issue fields |
| `tasks comment <issue-key>` | Add a comment |
| `tasks link <issue-key> <url>` | Add a remote link |
| `tasks spaces` | List Jira projects |

Useful examples:

```bash
devflow tasks list --exclude-done --sort priority --priority
devflow tasks show ENG-123 --recursive --pull-requests --format json
devflow tasks create --project ENG --type Story "Implement search API"
```

## Bitbucket repositories

| Command | Purpose |
| --- | --- |
| `repo list` | List repositories in the workspace |
| `repo search <regex>` | Search repositories by name or description |
| `repo show <name-or-id>` | Show repository details |
| `repo remotes <repo>` | Show clone URLs |
| `repo readme <repo>` | Fetch the repository README |
| `repo watch ...` | Manage watched repositories |
| `repo pipelines ...` | Inspect Bitbucket Pipelines |

## Pull requests

| Command | Purpose |
| --- | --- |
| `pullrequest list` | List pull requests in watched repositories |
| `pullrequest show <repo> <id>` | Show pull request details |
| `pullrequest create <title>` | Create a pull request |
| `pullrequest mine` | List pull requests authored by the current user |
| `pullrequest participating` | List pull requests where the current user participates |
| `pullrequest comments <repo> <id>` | List comment threads |
| `pullrequest add-comment <repo> <id> <comment>` | Add a comment |
| `pullrequest comment-reply ...` | Reply to a comment thread |
| `pullrequest diff <repo> <id>` | Show the unified diff |
| `pullrequest builds <repo> <id>` | Show commit build statuses |
| `pullrequest set-status ...` | Create or update a commit status |

Watched repositories are stored in `bitbucket.watched_repos` and can be managed with:

```bash
devflow repo watch add my-repository
devflow repo watch remove my-repository
devflow repo watch list
```

## Jenkins

```bash
devflow jenkins builds <job-name>
devflow jenkins logs <job-name> <build-number>
```

## Local Git

```bash
devflow git list [--path <directory>] [--no-fetch]
```

The default mode streams repository status as repositories are processed. Use `--format tabular` to wait for all results and render one aligned table.
