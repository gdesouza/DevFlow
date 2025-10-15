# 2025-10-15 Repository Search & Remotes Commands

## Summary
Implemented new Bitbucket repository utilities: a regex-based repository search command (`devflow repo search`) and a remotes command (`devflow repo remotes`) that outputs HTTPS and/or SSH clone URLs (raw single-line output when filtered by flag). Updated README with usage examples. These features improve discoverability of repositories and streamline cloning workflows from the CLI.

## Objectives
- Provide fast client-side regex search across workspace repositories.
- Allow users to retrieve clone URLs without visiting the Bitbucket UI.
- Support automation-friendly single URL output for scripting (`--ssh` / `--https`).

## Key Decisions
- Performed search client-side (fetch all repos once) to avoid premature Bitbucket API query syntax complexity.
- Default case-insensitive regex, optional `--case-sensitive` and `--description` flags for broader matching.
- Remotes command performs lightweight slug normalization (lowercase, spaces -> dashes) rather than calling API (keeps zero extra network round-trips).
- Mutually exclusive `--ssh` and `--https` flags enforce predictable single-line output.

## Implementation Summary
Added two new Cobra commands under the existing `repo` group: `search` and `remotes`. The search command loads the workspace repositories via existing Bitbucket client pagination helper, filters with a compiled regex, and reuses `displayReposPage` for consistent formatting and watch markers. The remotes command constructs deterministic HTTPS & SSH clone URLs from workspace + slug and optionally outputs only one. README updated with examples for both features.

## Technical Details
### New Components
- **`cmd/bitbucket_search.go`**: Parses flags, compiles regex (injecting `(?i)` when not case-sensitive), fetches repos, filters, sorts, displays.
- **`cmd/bitbucket_remotes.go`**: Generates clone URLs; flag gating for single-mode output; simple slug normalization.

### Modified Components
- **`cmd/bitbucket.go`**: Registered new subcommands.
- **`README.md`**: Added sections for search and remotes usage.

## Files Modified/Created
- `cmd/bitbucket_search.go` - New repo search command.
- `cmd/bitbucket_remotes.go` - New remotes command (HTTPS/SSH flags).
- `cmd/bitbucket.go` - Added command registrations.
- `README.md` - Documentation updates (search & remotes examples).

## CLI Commands Added
```bash
devflow repo search <regex> [-c|--case-sensitive] [-d|--description]
# Search repositories by regex (name, optional description)

devflow repo remotes <repo-slug> [--ssh | --https]
# Show HTTPS & SSH clone URLs (or a single raw URL when flagged)
```

## Documentation Updates
- Added "Search repositories" and "Show clone endpoints" examples with flag usage.
- Clarified case sensitivity behavior and description matching.
- Remotes examples for raw output suitable for command substitution.

## User Experience Improvements
- Eliminates manual scrolling in Bitbucket UI to locate repositories.
- Single-command retrieval of clone URLs speeds onboarding & scripting.
- Consistent output formatting with existing repo list (watch indicators retained in search results).

## Security Considerations
- No new credential handling; relies on existing Bitbucket token configuration.
- Search operates entirely client-side after authenticated repository fetch.

## Known Issues / TODOs
- Potential performance impact for very large workspaces (future enhancement: server-side query parameter usage or incremental streaming).
- Slug derivation in `remotes` is heuristic; could add an optional validation step that confirms existence via API.
- No direct JSON output mode for search yet (future `--json` flag could aid scripting).

## Next Steps
- Add optional `--json` output to `repo search` for automation.
- Introduce server-side filtering using Bitbucket query language when beneficial.
- Provide a `--validate` flag for `repo remotes` to confirm repository existence.

## Lessons Learned
- Reusing existing display helper minimized duplicated formatting logic.
- Client-side regex offers flexibility beyond typical Bitbucket search UI constraints with minimal code.

## Related Commits
- `c1118c9`: feat(repo): add repository search and remotes commands with ssh/https flag outputs

---

## Notes
Consider caching repository list for a short TTL to speed successive search invocations within the same session (while respecting potential staleness trade-offs).
