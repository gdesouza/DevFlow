#!/bin/bash

echo "🧪 Testing Bitbucket Integration"
echo "==============================="

# Check if token is set
TOKEN_OUTPUT=$(./devflow config get bitbucket.token 2>/dev/null)
if echo "$TOKEN_OUTPUT" | grep -q "No value set"; then
    echo "❌ Bitbucket API token not set"
    echo "Run: ./jira-bitbucket-cli config set bitbucket.token YOUR_API_TOKEN"
    echo "Get token from: https://bitbucket.org/account/settings/api-tokens"
    exit 1
fi

# Extract just the token value (remove "bitbucket.token = " prefix)
TOKEN=$(echo "$TOKEN_OUTPUT" | sed 's/bitbucket\.token = //')

if [ -z "$TOKEN" ] || [ "$TOKEN" = "PLACEHOLDER_APP_PASSWORD" ]; then
    echo "❌ Bitbucket API token not set"
    echo "Run: ./jira-bitbucket-cli config set bitbucket.token YOUR_API_TOKEN"
    echo "Get token from: https://bitbucket.org/account/settings/api-tokens"
    exit 1
fi

# Note: Both Jira and Bitbucket API tokens can start with ATATT3x
# since they're both Atlassian products. Let's test the actual API call instead.

echo "✅ Bitbucket API token is configured"

# Test list-prs command
echo ""
echo "📋 Testing: List Pull Requests"
echo "./devflow bitbucket list-prs --repo autonomy"
echo "Output:"
./devflow bitbucket list-prs --repo autonomy 2>&1

echo ""
echo "🎯 If you see pull requests above, Bitbucket integration is working!"
echo "If you get a 401 error, you need a new API token:"
echo "1. Go to: https://bitbucket.org/account/settings/api-tokens"
echo "2. Create a new API token with appropriate scopes"
echo "3. Run: ./devflow config set bitbucket.token YOUR_NEW_API_TOKEN"