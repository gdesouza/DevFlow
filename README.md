# Jira-Bitbucket CLI

A powerful command-line interface tool for streamlining development workflows with Jira and Bitbucket. Perfect for developers who want to manage tasks and repositories from the terminal.

## ✨ Features

### Jira Integration
- 📋 **List Tasks** - View your assigned Jira issues with filtering and sorting
- 🔍 **Show Details** - Get comprehensive information about specific issues
- 💬 **Find Mentions** - Discover issues where you're mentioned in comments
- ➕ **Create Tasks** - Quickly create new Jira issues
- 🎯 **Priority & Status** - Visual indicators for issue priority and status
- 📊 **Advanced Filtering** - Filter by status, exclude done tasks, sort by priority
- 🔗 **Direct Links** - Clickable URLs to open issues in your browser

### Bitbucket Integration ✅
- 📝 **List Pull Requests** - View pull requests in your repositories
- ➕ **Create Pull Requests** - Create new pull requests from the command line
- 🔐 **API Token Authentication** - Secure authentication with Bitbucket API tokens
- 📊 **Repository Management** - Manage your Bitbucket repositories
- 🔗 **Direct Links** - Clickable URLs to open pull requests in your browser

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

**Note:** `make install` installs the binary to `~/go/bin/devflow` with the latest release version embedded. For development or custom builds, see the Development section below.

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
- **Bitbucket API tokens:** Generated from API tokens page
- **Both use:** Bearer authentication

**❗ Common Mistake:** Don't use a Jira token for Bitbucket - they are completely different!

**📅 Migration Note:** App passwords were deprecated on September 9, 2025. All existing app passwords will be disabled on June 9, 2026.

## 📖 Usage Guide

### Jira Commands

#### List Your Tasks
```bash
# Basic list of your assigned tasks (includes direct links)
./devflow jira list
# Output includes 🔗 https://your-domain.atlassian.net/browse/ISSUE-KEY

# Focus on active work (exclude completed tasks)
./devflow jira list --exclude-done

# Show priority information
./devflow jira list --exclude-done --priority

# Filter by specific status
./devflow jira list --filter "In Progress"

# Sort by priority (highest first)
./devflow jira list --exclude-done --sort priority --priority

# Combine multiple options
./devflow jira list --exclude-done --sort priority --priority
```

#### Show Issue Details
```bash
# Get comprehensive details about a specific issue
./devflow jira show ISSUE-123

# Example output includes:
# - Issue summary and key
# - Status with icon
# - Priority with icon
# - Assignee and reporter
# - Created and updated dates
# - Full description
# - All comments with timestamps
# - Attachments with file sizes
```

#### Find Mentions
```bash
# Find all issues where you're mentioned
./devflow jira mentioned

# Searches through:
# - Comments (@username mentions)
# - Issue descriptions
# - All text fields
```

#### Create New Tasks
```bash
# Create a new Jira task
./devflow jira create "Fix login bug"

# Note: Requires project key configuration
```

### Bitbucket Commands

```bash
# List pull requests (includes direct links)
./devflow bitbucket list-prs --repo your-repo-name
# Output includes 🔗 https://bitbucket.org/workspace/repo/pull-requests/ID

# Create a pull request
./devflow bitbucket create-pr "Feature implementation" --repo your-repo-name --source feature-branch --dest main

# Test authentication
./devflow bitbucket test-auth
```

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
│   ├── jira.go                 # Jira command group
│   ├── jira_list.go            # List Jira tasks with filtering
│   ├── jira_show.go            # Show detailed issue information
│   ├── jira_mentioned.go       # Find mentions
│   ├── jira_create.go          # Create new tasks
│   ├── bitbucket.go            # Bitbucket command group
│   ├── bitbucket_list.go       # List pull requests
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
./devflow --verbose jira list
```

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