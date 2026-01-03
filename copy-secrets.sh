#!/bin/bash

# Script to copy GitHub secrets from old repo to new repo
# Note: You'll need to manually enter the secret values when prompted

echo "🔐 Copying GitHub Secrets"
echo "========================="
echo ""
echo "Old repo: terzigolu/josepshbrain-go"
echo "New repo: kutbudev/ramorie-cli"
echo ""
echo "Secrets to copy:"
echo "  1. HOMEBREW_TAP_GITHUB_TOKEN"
echo "  2. NPM_TOKEN"
echo ""
echo "⚠️  Note: You'll need to enter the secret values manually."
echo "    Get them from: https://github.com/terzigolu/josepshbrain-go/settings/secrets/actions"
echo ""
read -p "Press Enter to continue..."

echo ""
echo "📝 Setting HOMEBREW_TAP_GITHUB_TOKEN..."
gh secret set HOMEBREW_TAP_GITHUB_TOKEN --repo kutbudev/ramorie-cli

echo ""
echo "📝 Setting NPM_TOKEN..."
gh secret set NPM_TOKEN --repo kutbudev/ramorie-cli

echo ""
echo "✅ Done! Verifying secrets..."
gh secret list --repo kutbudev/ramorie-cli

echo ""
echo "🎉 Secrets copied successfully!"
