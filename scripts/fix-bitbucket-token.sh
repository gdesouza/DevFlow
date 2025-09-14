#!/bin/bash

echo "🔧 Fix Bitbucket Token - Step by Step"
echo "===================================="
echo ""

echo "❌ PROBLEM: You have a Jira token, but need a Bitbucket API token"
echo "📝 SOLUTION: Get a proper Bitbucket API token"
echo ""

echo "Step 1: Clear the wrong token"
echo "-----------------------------"
echo "✅ Already done - Jira token cleared"
echo ""

echo "Step 2: Get the correct Bitbucket API token"
echo "-------------------------------------------"
echo "1. Open: https://bitbucket.org/account/settings/api-tokens"
echo "2. Click: 'Create API token'"
echo "3. Name: 'CLI Tool' or 'jira-bitbucket-cli'"
echo "4. Scopes needed:"
echo "   ✅ Pull requests: Read"
echo "   ✅ Repositories: Read"
echo "5. Click: 'Create'"
echo "6. 📋 COPY the generated API token immediately!"
echo ""

echo "Step 3: Set the correct token"
echo "-----------------------------"
echo "Run this command with your new token:"
echo "./devflow config set bitbucket.token YOUR_NEW_API_TOKEN"
echo ""

echo "Step 4: Test it works"
echo "--------------------"
echo "Run: ./devflow bitbucket list-prs --repo autonomy"
echo ""

echo "❗ IMPORTANT:"
echo "- API tokens are shown ONLY ONCE"
echo "- Save it immediately after creation"
echo "- Use the generated password, not a Jira token"
echo ""

echo "🔄 Current status: Ready for correct Bitbucket API token"