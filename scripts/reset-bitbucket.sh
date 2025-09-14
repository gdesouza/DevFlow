#!/bin/bash

echo "🔄 Resetting Bitbucket Configuration"
echo "==================================="
echo ""

# Clear the old token
echo "🗑️  Clearing old Bitbucket token..."
./devflow config set bitbucket.token "" 2>/dev/null

echo "✅ Old token cleared"
echo ""
echo "📝 Next steps:"
echo "1. Go to: https://bitbucket.org/account/settings/api-tokens"
echo "2. Click 'Create App Password'"
echo "3. Name it: 'CLI Tool'"
echo "4. Select permissions: Pull requests (Read), Repositories (Read)"
echo "5. Click 'Create' and COPY the generated API token"
echo "6. Run: ./devflow config set bitbucket.token YOUR_API_TOKEN"
echo "7. Test: ./devflow bitbucket list-prs --repo autonomy"
echo ""
echo "❗ Remember: API tokens are shown only once!"