# Session Template

Use this template to create new session summaries. Copy this file and rename it using the format `YYYY-MM-DD-feature-name.md`.

---

# Session: [Feature/Task Name]
**Date**: YYYY-MM-DD
**Duration**: ~X hours
**Participants**: [List of participants]

## Objectives
- [Primary goal 1]
- [Primary goal 2]
- [Additional goals as needed]

## Key Decisions
- [Important architectural or design decision with rationale]
- [Technology/approach choices and why they were selected]
- [Trade-offs considered and decisions made]

## Implementation Summary
[High-level overview of what was implemented, focusing on the main components and their relationships]

## Technical Details

### New Components
- **[Component Name]**: [Purpose and key functionality]
- **[Another Component]**: [Purpose and key functionality]

### Modified Components  
- **[Component Name]**: [What was changed and why]
- **[Another Component]**: [What was changed and why]

## Files Modified/Created
- `cmd/[command].go` - [Brief description of CLI command changes]
- `internal/[package]/[file].go` - [Brief description of internal logic changes]
- `internal/config/[file].go` - [Configuration-related changes]
- `[file]_test.go` - [Test additions/modifications]

## CLI Commands Added/Modified
### New Commands
```bash
devflow [new-command] [flags]  # Description of what it does
```

### Modified Commands
```bash
devflow [existing-command] [new-flags]  # Description of changes
```

## Configuration Changes
```yaml
# Example of new configuration sections
new_integration:
  enabled: true
  api_endpoint: "https://api.example.com"
  timeout: 30s
  
# Modified existing configuration
existing_config:
  new_option: value
```

## API Integration Details
### Jira APIs Used
- **Endpoint**: `/rest/api/2/search` - [Purpose and usage]
- **Authentication**: [Method and considerations]

### Bitbucket APIs Used  
- **Endpoint**: `/2.0/repositories/{workspace}/{repo_slug}/pullrequests` - [Purpose and usage]
- **Authentication**: [Method and considerations]

## Tests Added
- [Test suite name]: [What functionality is tested]
- [Specific test cases]: [Edge cases or important scenarios covered]
- [Integration tests]: [API integration or end-to-end scenarios]
- [Coverage improvements]: [Any notable coverage increases]

## Documentation Updates
- [README sections added/modified]
- [Command help text updates]
- [Configuration documentation changes]

## User Experience Improvements
- [Output formatting enhancements]
- [New visual indicators or colors]
- [Error message improvements]
- [Performance optimizations]

## Security Considerations
- [Token storage and encryption]
- [API credential management]
- [Permission and access control]
- [Security audit findings]

## Lessons Learned
- [Technical insights gained during implementation]
- [What worked well in the approach taken]
- [What could be improved or done differently]
- [API behavior discoveries or quirks]
- [Performance considerations discovered]

## Known Issues/TODOs
- [Any outstanding issues that need addressing]
- [Future improvements identified during implementation]  
- [Technical debt or refactoring opportunities]
- [Rate limiting or API quota considerations]

## Next Steps
- [Immediate follow-up actions needed]
- [Future enhancements to consider]
- [Related features that could build on this work]
- [Integration opportunities with other tools]

## Related Commits
- `[commit-hash]`: [Brief description of commit]
- `[commit-hash]`: [Brief description of commit]

---

## Notes
[Any additional context, references, API documentation links, or important details that don't fit in the above sections]