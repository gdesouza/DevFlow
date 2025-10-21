# DevFlow
[![Build Status](https://gdesouza.semaphoreci.com/badges/DevFlow/branches/master.svg?style=shields&key=dbff7292-7a82-4922-b626-72eefeef5b82)](https://gdesouza.semaphoreci.com/projects/DevFlow)

A powerful command-line interface tool for streamlining development workflows with Jira and Bitbucket. Perfect for developers who want to manage tasks and repositories from the terminal.

## ✨ Features

### Task Management (Jira Backend)
- 📋 **List Tasks** - View your assigned Jira issues with filtering and sorting
- 🔍 **Show Details** - Get comprehensive information about specific issues (now includes Assigned Team)
- 💬 **Find Mentions** - Discover issues where you're mentioned in comments
- 💭 **Comment on Issues** - Add comments directly from the CLI
- 🔗 **Add Remote Links** - Attach external docs / references to issues
- ➕ **Create Tasks** - Quickly create new Jira issues (supports epic, story points, sprint, team, labels)
- 👥 **Team Assignment** - Set issue Team via `--team` flag (ID or name)
- 🎯 **Priority & Status** - Visual indicators for issue priority and status
- 📊 **Advanced Filtering** - Filter by status, exclude done tasks, sort by priority
- 🔗 **Direct Links** - Clickable URLs to open issues in your browser

### Bitbucket Integration ✅
- 📝 **List Pull Requests** - View pull requests in your repositories
- ➕ **Create Pull Requests** - Description, reviewers, branch auto-detect, browser open
- 📖 **Show Repository README** - Display README contents for a repository
- 🔐 **API Token Authentication** - Secure authentication with Bitbucket API tokens
- 📊 **Repository Management** - Manage your Bitbucket repositories
- 🔗 **Direct Links** - Clickable URLs to open pull requests in your browser

#### Repository Commands Summary
`devflow repo` provides:
- `list` – Paginated listing (optional interactive UI) with watch toggling
- `search` – Regex-based search by name (optionally description)
- `remotes` – Show HTTPS/SSH clone URLs (single or both)
- `readme` – Fetch and display README contents
- `watch` – Add/remove/toggle/list watched repositories

Watched repositories scope pull request aggregation and targeted operations.

### Configuration Management
- 🔐 **Secure Storage** - API tokens stored securely in your home directory
- ⚙️ **Easy Setup** - Simple configuration commands
- 🔄 **Persistent Config** - Settings persist between CLI sessions

### Developer Experience
- 🚀 **Cross-platform** - Single binary for easy distribution
- 🎨 **Rich Output** - Colorful, visual status and priority indicators
- 📖 **Comprehensive Help** - Detailed help for all commands
- 🧪 **Well Tested** - Unit tests included

## 🚀 Quick Start

### Breaking Changes (v1.0.0)
- Replaced top-level command `jira` with `tasks`
- Replaced top-level command `bitbucket` with separate `repo` and `pullrequest` command groups
- Removed `bitbucket test-auth` (authentication now implicitly verified on other commands)
- Updated pull request subcommands to concise verbs: `list`, `show`, `create`, `mine`

Update any scripts referencing old commands accordingly (e.g. `devflow jira list` -> `devflow tasks list`).

### 1. Installation

```bash
# Clone the repository
git clone <repository-url>
cd devflow

# Install to your Go bin directory (recommended)
make install

# Ensure ~/go/bin is in your PATH (add to ~/.bashrc if needed)
export PATH="$HOME/go/bin:$PATH"
```

**Note:** Version is injected at build time. The source file `cmd/version.go` intentionally sets `var version = "dev"`; release builds override this via `-ldflags "-X 'devflow/cmd.version=vX.Y.Z'"` (see Makefile). `make install` installs the binary to `~/go/bin/devflow` embedding the latest git tag release version. For development builds (`make build`) the CLI will report `dev`. To install a specific tagged release: `make install VERSION=v1.3.3`. Avoid manually editing `version.go` for releases.

### 2. Configuration

Set up your API credentials:

```bash
# Jira configuration
./devflow config set jira.url https://your-domain.atlassian.net
./devflow config set jira.username your-email@example.com
./devflow config set jira.token your-jira-api-token

# Bitbucket configuration (optional)
./devflow config set bitbucket.workspace your-workspace
./devflow config set bitbucket.username your-username
./devflow config set bitbucket.token your-bitbucket-api-token
```

### 3. Get Your API Tokens

#### Jira API Token
1. Go to [Atlassian Account Settings](https://id.atlassian.com/manage-profile/security/api-tokens)
2. Create a new API token
3. Copy the token value
4. Use with: `./devflow config set jira.token YOUR_JIRA_TOKEN`

#### Bitbucket API Token
**⚠️ Important:** Bitbucket now uses API tokens instead of app passwords. These are DIFFERENT from Jira tokens!

1. Go to [Bitbucket Settings > API tokens](https://bitbucket.org/account/settings/api-tokens)
2. Click "Create API token"
3. Give it a name (e.g., "CLI Tool")
4. Select these scopes:
   - ✅ **Pull requests**: Read
   - ✅ **Repositories**: Read
   - ✅ **Webhooks**: Read and write (if needed)
5. Click "Create"
6. **📋 COPY the generated API token immediately** (shown only once!)
7. Use with: `./devflow config set bitbucket.token YOUR_BITBUCKET_API_TOKEN`

**🔍 Token Format Difference:**
- **Jira tokens:** Start with `ATATT3x...` (Atlassian format)
- **Bitbucket personal API tokens (user-level):** Used with Basic auth (email + token)
- **Bitbucket resource access tokens (workspace/project/repo):** Used with Bearer auth (Authorization: Bearer <token>)

**Auth Selection Logic:**
- If you set `bitbucket.username` (your email), DevFlow uses Basic auth.
- If you leave `bitbucket.username` empty, DevFlow assumes a resource access token and uses Bearer.

**❗ Common Mistake:** Don't use a Jira token for Bitbucket - they are completely different!

**📅 Migration Note:** App passwords deprecated Sep 9 2025 and disabled Jun 9 2026. Use personal API tokens (Basic) or resource access tokens (Bearer).

## 📖 Usage Guide

### Git Utilities (Local Repositories)

List and inspect the sync status of all git repositories under a directory (recursively). Shows branch, sync state, dirtiness, ahead/behind counts (approximate), and upstream tracking branch.

Default output now streams each repository line-by-line as soon as it is processed (faster feedback on large trees). Use `--tabular` to wait for all results and render the full table. JSON remains available with `--json`.

```bash
# Basic usage (current directory) - streaming lines: path<TAB>branch<TAB>state
./devflow git list

# Specify a root path to scan
./devflow git list --path ~/code

# Skip fetching remotes (faster, may be stale)
./devflow git list --no-fetch

# Full table (waits for all repos)
./devflow git list --tabular

# JSON output for scripting
./devflow git list --json > repos.json
```

Sample streaming lines:
```
api	main	up-to-date
web	featureX	ahead
infra/terraform/modules/network	main	behind
tools/old-experiment	DETACHED	detached
sandbox/prototype	main	no-upstream
```

Sample table output (`--tabular`):
```
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━┳━━━━━━━━━┳━━━━━━━┳━━━━━━━┳━━━━━━━━━━━━━━┓
┃ Repository                           ┃ Branch   ┃ State      ┃ Dirty ┃ Stashed ┃ Ahead ┃ Behind ┃ Upstream     ┃
┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╋━━━━━━━━━━╋━━━━━━━━━━━━╋━━━━━━━╋━━━━━━━━━╋━━━━━━━╋━━━━━━━╋━━━━━━━━━━━━━━┫
┃ api                                  ┃ main     ┃ up-to-date ┃ clean ┃ no      ┃ 0     ┃ 0      ┃ origin/main  ┃
┃ web                                  ┃ featureX ┃ ahead      ┃ dirty ┃ yes     ┃ 2     ┃ 0      ┃ origin/main  ┃
┃ infra/terraform/modules/network      ┃ main     ┃ behind     ┃ clean ┃ no      ┃ 0     ┃ 3      ┃ origin/main  ┃
┃ tools/old-experiment                 ┃ DETACHED ┃ detached   ┃ clean ┃ no      ┃ 0     ┃ 0      ┃              ┃
┃ sandbox/prototype                    ┃ main     ┃ no-upstream┃ dirty ┃ no      ┃ 0     ┃ 0      ┃              ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┻━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━┻━━━━━━━━━┻━━━━━━━┻━━━━━━━┻━━━━━━━━━━━━━━┛
```

States:
- up-to-date: Local and upstream commit match
- ahead: Local has commits not on upstream
- behind: Upstream has commits not local
- diverged: Both sides have unique commits
- no-upstream: Branch has no configured tracking branch
- detached: HEAD is detached (not on a named branch)

Dirty indicates uncommitted changes in the worktree.

Ahead/Behind Accuracy Note:
The counts use a bounded ancestor traversal (max 2000 commits each side) rather than a true merge-base calculation; for very large, divergent histories counts are approximate but state classification (ahead/behind/diverged) remains reliable.


### Task Commands (formerly Jira)

#### List Your Tasks
```bash
# Basic list of your assigned tasks (includes direct links)
./devflow tasks list
# Output includes 🔗 https://your-domain.atlassian.net/browse/ISSUE-KEY

# Focus on active work (exclude completed tasks)
./devflow tasks list --exclude-done

# Show priority information
./devflow tasks list --exclude-done --priority

# Filter by specific status
./devflow tasks list --filter "In Progress"

# Sort by priority (highest first)
./devflow tasks list --exclude-done --sort priority --priority

# Combine multiple options
./devflow tasks list --exclude-done --sort priority --priority
```

#### Show Issue Details
```bash
# Get comprehensive details about a specific issue
./devflow tasks show ISSUE-123

# Example output includes:
# - Issue summary and key
# - Status with icon
# - Priority with icon
# - Assignee and reporter
# - Assigned Team (if present)
# - Created and updated dates
# - Full description
# - All comments with timestamps
# - Attachments with file sizes
```

#### Find Mentions
```bash
# Find all issues where you're mentioned
./devflow tasks mentioned

# Searches through:
# - Comments (@username mentions)
# - Issue descriptions
# - All text fields
```

#### Create New Tasks
```bash
# Minimum (project key is required)
./devflow tasks create -p ENG "Fix login bug"

# With common metadata
./devflow tasks create -p ENG "Implement search API" \
  --type Story \
  --priority High \
  --assignee alice \
  --labels backend,api,search \
  --epic ENG-42 \
  --story-points 3 \
  --sprint Sprint-24 \
  --team 11843 \
  --description "Adds new search endpoint with filtering"

# Use a file for the description (mutually exclusive with --description)
./devflow tasks create -p ENG "Refactor caching layer" --description-file docs/specs/cache-refactor.md
```

Flags:
- `-p, --project` (required): Jira project key (e.g. ENG, OPS)
- `-t, --type`: Issue type (Task, Story, Bug, etc.; defaults to Task)
- `--priority`: Priority name (Highest, High, Medium, Low, Lowest)
- `--assignee`: Assignee username (may need accountId in some Jira Cloud setups)
- `--labels`: Comma-separated labels (e.g. backend,api,urgent)
- `--epic`: Epic key to link (depends on Jira setup / classic vs next-gen)
- `--story-points`: Numeric estimate (mapped to a common story points custom field)
- `--sprint`: Sprint name or ID (instance-specific custom field)
- `--team`: Team ID or name (maps to custom team field; accepts numeric id or exact team name)
- `-d, --description`: Inline description text
- `--description-file`: Path to a file whose contents become the description

Notes:
- Do not supply both `--description` and `--description-file` (command will error).
- Epic, Story Points, Sprint, and Team rely on specific custom field IDs which may differ in your Jira instance; if they do, creation may fail until field IDs are aligned in code (future enhancement: configurable field IDs).
- The Team field accepts either the numeric ID (preferred) or the exact team name. If Jira rejects the value, the CLI automatically retries without the team field (logged in output).
- Assignee resolution using `name` may not work on all Jira Cloud instances that require `accountId` (planned improvement).

#### Comment on an Issue
```bash
# Add a simple comment
./devflow tasks comment ISSUE-123 "This is now unblocked after deploying service X"

# Use a file for a longer comment (supports markdown-ish formatting rendered as ADF)
./devflow tasks comment ISSUE-123 --file docs/notes/analysis.md
```

#### Add a Remote Link
```bash
# Link an external spec document
./devflow tasks link ISSUE-123 https://docs.internal/specs/auth-redesign.md \
  --title "Auth Redesign Spec" \
  --summary "Initial design draft for new authentication flow"

# Minimal (just URL)
./devflow tasks link ISSUE-123 https://example.com/incident/postmortem-42
```

### Pull Request & Repo Commands

#### Show Repository README (New)
Display the README for a Bitbucket repository (tries common filename variants)
```bash
./devflow repo readme my-repo
./devflow repo readme my-repo --raw   # raw contents only (no header)
```

#### Watched Repositories (New)
Mark repositories as "watched" to scope pull request commands.

Interactive selection (refactored raw key navigation):
```bash
./devflow repo list --interactive
# Keys:
#   Up/Down: move selection
#   Space/Enter: toggle watch for highlighted repo
#   Left/Right: previous/next page
#   w: jump to currently watched repos page
#   g: go to page (then enter number)
#   s: save & exit
#   q: quit without saving
```

Show clone endpoints:
```bash
# Show HTTPS and SSH URLs (labeled)
./devflow repo remotes my-repo

# Only SSH (raw URL output)
./devflow repo remotes --ssh my-repo

# Only HTTPS (raw URL output)
./devflow repo remotes --https my-repo
```

Search repositories:
```bash
# Regex match on repository name (case-insensitive by default)
./devflow repo search 'api-.*'

# Case sensitive
./devflow repo search --case-sensitive 'API-[A-Z]+'

# Include description in the match
./devflow repo search -d 'terraform'
```

Direct watch management commands:
```bash
# Add/remove/toggle/list watched repos
./devflow repo watch add <repo-slug>
./devflow repo watch remove <repo-slug>
./devflow repo watch toggle <repo-slug>
./devflow repo watch list
```

Watched repo slugs are stored (sorted) under `bitbucket.watched_repos` in `~/.devflow/config.json`.

Current PR command behavior (scoped to watched repos):
```bash
# List PRs in a specific watched repo (fails if not watched)
./devflow pullrequest list --repo my-repo
# Aggregate PRs across all watched repos
./devflow pullrequest list

# PRs where you are author (watched repos unless --all-repos)
./devflow pullrequest mine                # aggregate across watched
./devflow pullrequest mine --repo my-repo # single watched repo
./devflow pullrequest mine --all-repos    # workspace-wide (ignores watch list)

# PRs you participate in (author/reviewer) across watched repos
./devflow pullrequest participating       # aggregate across watched
./devflow pullrequest participating --repo my-repo
```


```bash
# List pull requests (includes direct links)
./devflow pullrequest list --repo your-repo-name
# Output includes 🔗 https://bitbucket.org/workspace/repo/pull-requests/ID

# Create a pull request (description, reviewers, auto-detect branches, browser open)
./devflow pullrequest create "Feature implementation" --repo your-repo-name -m "Implements feature X" -R alice -R bob --open

# Show build/status checks for all commits in a pull request
./devflow pullrequest builds your-repo-name 123
# Example output:
# Fetching commits for PR #123 in your-repo-name...
# Found 2 commits. Fetching statuses...
# Commit 1/2: a1b2c3d4e5f6 - Add integration test
# Author: Alice <alice@example.com>  Date: 2025-10-20T14:22:11+00:00
#   🔄 INPROGRESS - build (ci/pipeline)
#     Running CI pipeline
#     🔗 https://bitbucket.org/workspace/your-repo-name/addon/pipelines/home#!/results/123
#     Updated: 3m ago
#   ✅ SUCCESSFUL - lint (lint/check)
#     Updated: 3m ago
#
# Commit 2/2: 0f1e2d3c4b5a - Refactor handlers
# Author: Bob <bob@example.com>  Date: 2025-10-20T15:02:48+00:00
#   ✅ SUCCESSFUL - build (ci/pipeline)
#     🔗 https://bitbucket.org/workspace/your-repo-name/addon/pipelines/home#!/results/124
#     Updated: 2m ago
#
# Status Key: value in parentheses (e.g. ci/pipeline) is the unique upsert key to reuse with `pullrequest set-status`.
# If a status shows only one value (no parentheses) key == name.
# To discover a key outside the CLI, open the Bitbucket Pipelines build/log page; the key often matches the pipeline step or integration identifier.
# Status Icons Legend:
#   ✅ SUCCESSFUL   ❌ FAILED/ERROR   🔄 INPROGRESS/PENDING   🚫 STOPPED/CANCELLED   📝 Other
# (Authentication is validated automatically when running other commands)
```

#### Set or Update a Commit Status (New)
Create or update a build/deployment/check status for a specific commit (upsert by key):
```bash
devflow pullrequest set-status your-repo-name a1b2c3d4e5f6 \
  --state SUCCESSFUL \
  --key ci/pipeline \
  --description "All tests passed"
```
States: SUCCESSFUL, FAILED, INPROGRESS, STOPPED, ERROR, PENDING, CANCELLED
Reusing the same `--key` updates the existing status entry.
If you omit --name or --url they will be reused from an existing status (if present) or default (name=key). --description is optional; omit it to keep existing text.


### Configuration Commands

```bash
# Set configuration values
./devflow config set jira.url https://your-domain.atlassian.net
./devflow config set jira.username your-email@example.com
./devflow config set jira.token your-api-token

# Get configuration values
./devflow config get jira.url
./devflow config get jira.username

# View all configuration
cat ~/.devflow/config.json
```

## 🎨 Visual Indicators

### Status Icons
- 📋 **To Do / Open** - Ready to work
- 🔄 **In Progress / In Review** - Currently working
- ✅ **Done / Closed / Resolved** - Completed
- 📚 **Backlog** - Planned for future
- 🚫 **Blocked / Waiting** - Cannot proceed
- 🔍 **Under investigation** - Research needed
- 📏 **Scoping** - Estimating work
- ❌ **Cancelled** - No longer needed
- 📝 **Default** - Other statuses

### Priority Icons
- 🔴 **Highest** - Critical priority
- 🟠 **High** - Important priority
- 🟡 **Medium** - Normal priority
- 🟢 **Low** - Low priority
- 🔵 **Lowest** - Minimal priority
- ⚪ **Default** - No priority set

## 🔧 Development

### Prerequisites
- Go 1.19 or later
- Git

### Setup Development Environment
```bash
# Install dependencies
go mod tidy

# Run tests
go test ./...

# Run specific tests
go test ./internal/config/

# Build for current platform
go build -o devflow

# Build for multiple platforms
make build-all

# Run linter
golangci-lint run
```

### Project Structure
```
devflow/
├── cmd/                        # CLI commands using Cobra framework
│   ├── root.go                 # Root command
│   ├── jira.go                 # Tasks command group (to be renamed tasks.go)
│   ├── jira_list.go            # List Jira tasks with filtering
│   ├── jira_show.go            # Show detailed issue information
│   ├── jira_mentioned.go       # Find mentions
│   ├── jira_create.go          # Create new tasks
│   ├── bitbucket.go            # Repo & Pull Request command groups
│   ├── bitbucket_list.go       # List pull requests
│   ├── bitbucket_readme.go     # Show repository README
│   ├── bitbucket_create.go     # Create pull requests
│   ├── config.go               # Config command group
│   ├── config_set.go           # Set configuration
│   ├── config_get.go           # Get configuration
│   └── jira_list.go            # List command with advanced features
├── internal/                   # Private application code
│   ├── jira/                   # Jira REST API client
│   │   └── client.go           # Jira API integration
│   ├── bitbucket/              # Bitbucket REST API client
│   │   └── client.go           # Bitbucket API integration
│   └── config/                 # Configuration management
│       ├── config.go           # Config file handling
│       └── config_test.go      # Unit tests
├── pkg/                       # Public packages (future use)
├── scripts/                   # Build and deployment scripts
├── docs/                      # Documentation
├── main.go                    # Application entry point
├── go.mod                     # Go module file
├── go.sum                     # Dependency checksums
├── Makefile                   # Build automation
├── README.md                  # This file
└── .gitignore                 # Git ignore rules
```

## 🐛 Troubleshooting

### Common Issues

#### "API request failed with status: 400"
- Check your Jira URL - it should be `https://your-domain.atlassian.net` (without `/jira`)
- Verify your API token is correct
- Ensure your username is your email address

#### "API request failed with status: 401"
- Your API token may be expired or incorrect
- Check that you're using the correct token type (Jira API token, not Bitbucket)

#### "API request failed with status: 410"
- The API endpoint has changed - the CLI automatically handles this

#### Configuration not found
- Run the configuration commands again
- Check that `~/.devflow/config.json` exists

### Debug Mode
```bash
# Enable verbose output (future feature)
./devflow --verbose tasks list
```

## 🛠️ Development

### Session Summaries

Development session summaries are maintained in [`docs/sessions/`](docs/sessions/) to provide context continuity between development sessions. These summaries document:

- Feature implementation details and architectural decisions
- API integration patterns and authentication flows
- CLI design choices and user experience considerations
- Key technical insights and lessons learned  
- Implementation approach and rationale
- Files modified and their purposes
- Test coverage and validation approach

This helps maintain context for future development work and provides valuable historical information about the project's evolution, especially important for a tool that integrates with multiple external APIs.

### Contributing

When contributing to this project:
1. Follow the existing code structure and patterns
2. Add comprehensive tests for new features
3. Update documentation (README, session summaries)
4. Use the session summary template for significant changes
5. Consider API rate limiting and error handling
6. Test with both Jira and Bitbucket integrations

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-feature`
3. Make your changes and add tests
4. Run tests: `go test ./...`
5. Submit a pull request

### Code Style
- Follow Go conventions
- Add tests for new features
- Update documentation
- Use meaningful commit messages

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) CLI framework
- Uses Go's standard HTTP client for API calls
- Inspired by the need for efficient terminal-based workflow management

---

**Happy coding! 🚀**

For issues, questions, or contributions, please create an issue on GitHub.
