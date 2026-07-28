# Configuration

DevFlow stores configuration in `~/.devflow/config.json`.

Configuration values can be managed with the CLI:

```bash
devflow config set <section>.<key> <value>
devflow config get <section>.<key>
```

## Jira

```bash
devflow config set jira.url https://your-domain.atlassian.net
devflow config set jira.username you@example.com
devflow config set jira.token "$JIRA_TOKEN"
```

The Jira username is normally an email address. Create API tokens from [Atlassian account security](https://id.atlassian.com/manage-profile/security/api-tokens).

## Bitbucket

```bash
devflow config set bitbucket.workspace your-workspace
devflow config set bitbucket.username you@example.com
devflow config set bitbucket.token "$BITBUCKET_TOKEN"
```

When `bitbucket.username` is set, DevFlow uses Basic authentication with the username and token. If it is empty, DevFlow uses the token as a Bearer token.

Create tokens from [Bitbucket API token settings](https://bitbucket.org/account/settings/api-tokens). Use a token with only the scopes required by the commands you intend to run.

Verify the configured credentials with:

```bash
devflow auth status
```

## Jenkins

```bash
devflow config set jenkins.url https://jenkins.example.com
devflow config set jenkins.username you
devflow config set jenkins.token "$JENKINS_TOKEN"
```

## Security

Do not commit tokens or place them directly in shell history when avoidable. Prefer environment variables when setting credentials. The configuration directory is created with restricted permissions by DevFlow.
