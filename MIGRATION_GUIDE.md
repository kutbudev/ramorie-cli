# Repository Migration Guide
## From `terzigolu/josepshbrain-go` to `kutbudev/ramorie-cli`

**Date:** January 3, 2025
**Migration Type:** GitHub Repository Transfer (Recommended) or Mirror Push
**Target Repository:** https://github.com/kutbudev/ramorie-cli.git

---

## 📋 PRE-MIGRATION CHECKLIST

### ✅ Verify Current State
- [ ] Current repo: `terzigolu/josepshbrain-go` is accessible
- [ ] You have admin access to the source repository
- [ ] Target organization `kutbudev` exists on GitHub
- [ ] You have admin/owner access to `kutbudev` organization
- [ ] All pending PRs are merged or documented
- [ ] All important issues are documented or transferred
- [ ] Latest release is tagged and published

### ✅ Backup Current State
```bash
# Clone with full history
git clone --mirror https://github.com/terzigolu/josepshbrain-go.git josepshbrain-go-backup
cd josepshbrain-go-backup
git bundle create ../josepshbrain-go-full-backup.bundle --all

# Backup releases
# Manually download all release assets from:
# https://github.com/terzigolu/josepshbrain-go/releases
```

---

## 🔄 MIGRATION METHOD 1: GitHub Repository Transfer (RECOMMENDED)

**Advantages:**
- ✅ Preserves full git history
- ✅ Automatic URL redirects (for a period)
- ✅ Maintains stars, watchers, forks
- ✅ Zero downtime
- ✅ Simplest method

**Steps:**

### 1. Transfer Repository via GitHub UI
1. Go to: https://github.com/terzigolu/josepshbrain-go/settings
2. Scroll to "Danger Zone" → "Transfer ownership"
3. Enter new repository name: `ramorie-cli`
4. Enter target owner: `kutbudev`
5. Type repository name to confirm
6. Click "I understand, transfer this repository"

### 2. Verify Transfer
```bash
# Clone from new location
git clone https://github.com/kutbudev/ramorie-cli.git
cd ramorie-cli

# Verify history is intact
git log --oneline | head -20
git tag -l

# Verify all branches transferred
git branch -a
```

### 3. Update Local Clones (For Team Members)
```bash
# If you have existing local clone
cd josepshbrain-go
git remote set-url origin https://github.com/kutbudev/ramorie-cli.git
git remote -v  # Verify
git fetch --all
```

---

## 🔄 MIGRATION METHOD 2: Mirror Push (If Transfer Not Possible)

**Use this if:**
- You don't have transfer permissions
- Target repo already exists
- You need to preserve old repo

**Steps:**

### 1. Create Target Repository
1. Go to: https://github.com/organizations/kutbudev/repositories/new
2. Repository name: `ramorie-cli`
3. Description: "AI-powered task and memory management CLI"
4. Visibility: **Public**
5. **DO NOT** initialize with README, .gitignore, or license
6. Click "Create repository"

### 2. Mirror Push
```bash
# Clone source with full history
git clone --mirror https://github.com/terzigolu/josepshbrain-go.git temp-mirror
cd temp-mirror

# Add new remote
git remote add new-origin https://github.com/kutbudev/ramorie-cli.git

# Push everything (branches, tags, refs)
git push new-origin --mirror

# Verify
git ls-remote new-origin
```

### 3. Verify Migration
```bash
# Clone from new location
cd ..
git clone https://github.com/kutbudev/ramorie-cli.git
cd ramorie-cli

# Verify all tags
git tag -l

# Verify all branches
git branch -a

# Verify commit history
git log --oneline --graph --all | head -50
```

---

## 🔐 SECRETS & TOKENS MIGRATION

### Required GitHub Secrets

Navigate to: https://github.com/kutbudev/ramorie-cli/settings/secrets/actions

Add the following secrets:

| Secret Name | Description | How to Get | Required For |
|-------------|-------------|------------|--------------|
| `GITHUB_TOKEN` | **Auto-provided by GitHub Actions** | No action needed | Creating releases |
| `HOMEBREW_TAP_GITHUB_TOKEN` | Personal Access Token for Homebrew tap | See below | Publishing to Homebrew |
| `NPM_TOKEN` | npm automation token | See below | Publishing to npm |

### Creating HOMEBREW_TAP_GITHUB_TOKEN

1. Go to: https://github.com/settings/tokens/new
2. Token name: `ramorie-homebrew-tap-token`
3. Expiration: **No expiration** (or 1 year, then rotate)
4. Scopes:
   - ✅ `repo` (Full control of private repositories)
   - ✅ `workflow` (Update GitHub Action workflows)
5. Click "Generate token"
6. **Copy the token immediately** (you won't see it again)
7. Add to GitHub secrets as `HOMEBREW_TAP_GITHUB_TOKEN`

**Prerequisites:**
- Create Homebrew tap repository: `https://github.com/kutbudev/homebrew-tap`
- Initialize with README.md
- Create `Formula/` directory

### Creating NPM_TOKEN

1. Login to npm: https://www.npmjs.com/login
2. Go to: https://www.npmjs.com/settings/YOUR_USERNAME/tokens
3. Click "Generate New Token" → "Classic Token"
4. Token type: **Automation**
5. Copy the token
6. Add to GitHub secrets as `NPM_TOKEN`

**Prerequisites:**
- Verify npm package ownership: https://www.npmjs.com/package/ramorie
- If you don't own it, publish first version manually or request transfer

---

## 📦 HOMEBREW TAP SETUP

### Create Homebrew Tap Repository

```bash
# Create new repo on GitHub: kutbudev/homebrew-tap
# Then clone it
git clone https://github.com/kutbudev/homebrew-tap.git
cd homebrew-tap

# Create directory structure
mkdir -p Formula

# Create README
cat > README.md << 'EOF'
# Kutbu Dev Homebrew Tap

Official Homebrew tap for Kutbu Dev tools.

## Installation

```bash
brew tap kutbudev/tap
brew install ramorie
```

## Available Formulae

- **ramorie** - AI-powered task and memory management CLI
EOF

git add .
git commit -m "Initial tap setup"
git push origin main
```

---

## 📝 FILE CHANGES SUMMARY

All file changes have been applied to your local repository. Review the changes:

```bash
cd /Users/terzigolu/Documents/GitHub/AI_ML/josepshbrain-go

# Review all changes
git status
git diff

# Files modified:
# - go.mod (module path)
# - .goreleaser.yaml (release config)
# - npm/package.json (repo URL)
# - npm/postinstall.js (download URLs)
# - npm/bin/ramorie (error messages)
# - npm/README.md (install instructions)
# - install.sh (API and download URLs)
# - README.md (all references, badges, links)
```

---

## 🚀 POST-MIGRATION STEPS

### 1. Commit and Push Changes

```bash
cd /Users/terzigolu/Documents/GitHub/AI_ML/josepshbrain-go

# Create a new branch for migration changes
git checkout -b migration/update-repo-references

# Stage all changes
git add go.mod .goreleaser.yaml npm/ install.sh README.md MIGRATION_GUIDE.md

# Commit
git commit -m "chore: update repository references to kutbudev/ramorie-cli

- Update Go module path
- Update GoReleaser config for new repo and Homebrew tap
- Update npm package repository URL
- Update install script download URLs
- Update README with new repository links
- Add migration guide"

# Push to NEW repository
git remote set-url origin https://github.com/kutbudev/ramorie-cli.git
git push -u origin migration/update-repo-references

# Create PR and merge to main
# Or push directly to main if you prefer:
# git checkout main
# git merge migration/update-repo-references
# git push origin main
```

### 2. Create First Release in New Repo

```bash
# After merging changes to main
git checkout main
git pull

# Tag a new release (increment from current version)
git tag -a v1.5.0 -m "Release v1.5.0 - Repository migration to kutbudev/ramorie-cli"
git push origin v1.5.0
```

This will trigger:
- ✅ GoReleaser workflow
- ✅ Binary builds for all platforms
- ✅ GitHub release creation
- ✅ Homebrew formula update
- ✅ npm package publish

### 3. Verify Release Workflow

Monitor: https://github.com/kutbudev/ramorie-cli/actions

Check that:
- [ ] GoReleaser job completes successfully
- [ ] Binaries are attached to release
- [ ] Homebrew formula is updated in `kutbudev/homebrew-tap`
- [ ] npm package is published to npmjs.org

### 4. Test All Install Methods

```bash
# Test Homebrew (may need to wait for tap update)
brew tap kutbudev/tap
brew install ramorie
ramorie --version

# Test npm
npm install -g ramorie
ramorie --version

# Test curl script
curl -sSL https://raw.githubusercontent.com/kutbudev/ramorie-cli/main/install.sh | bash
ramorie --version

# Test Go install
go install github.com/kutbudev/ramorie-cli/cmd/jbraincli@latest
ramorie --version
```

### 5. Update Old Repository (If Not Transferred)

If you used mirror push instead of transfer, update the old repo:

```bash
# Clone old repo
git clone https://github.com/terzigolu/josepshbrain-go.git old-repo
cd old-repo

# Create deprecation notice
cat > README.md << 'EOF'
# ⚠️ REPOSITORY MOVED

This repository has been moved to:

## 🔗 New Location
**https://github.com/kutbudev/ramorie-cli**

Please update your bookmarks, clones, and dependencies.

### Update Your Local Clone
```bash
git remote set-url origin https://github.com/kutbudev/ramorie-cli.git
```

### Install the CLI
```bash
npm install -g ramorie
# or
brew tap kutbudev/tap
brew install ramorie
```

---

**All future development, releases, and issues should use the new repository.**
EOF

git add README.md
git commit -m "docs: repository moved to kutbudev/ramorie-cli"
git push origin main

# Archive the repository
# Go to: https://github.com/terzigolu/josepshbrain-go/settings
# Scroll to "Danger Zone" → "Archive this repository"
```

---

## ✅ VERIFICATION CHECKLIST

### Repository
- [ ] New repo `kutbudev/ramorie-cli` is accessible
- [ ] All branches transferred
- [ ] All tags transferred
- [ ] Commit history intact
- [ ] GitHub Actions workflows present

### Secrets
- [ ] `HOMEBREW_TAP_GITHUB_TOKEN` added to new repo secrets
- [ ] `NPM_TOKEN` added to new repo secrets
- [ ] Homebrew tap repository `kutbudev/homebrew-tap` created

### Release Pipeline
- [ ] First release tagged in new repo
- [ ] GitHub Actions workflow triggered
- [ ] GoReleaser job succeeded
- [ ] Binaries uploaded to GitHub release
- [ ] Homebrew formula updated
- [ ] npm package published

### Install Methods
- [ ] `npm install -g ramorie` works
- [ ] `brew install kutbudev/tap/ramorie` works
- [ ] `curl install.sh | bash` works
- [ ] `go install github.com/kutbudev/ramorie-cli/cmd/jbraincli@latest` works
- [ ] Direct binary download from releases works

### Documentation
- [ ] README.md updated with new URLs
- [ ] Badges show correct repo
- [ ] All links point to new repo
- [ ] Old repo archived (if applicable)

### npm Package
- [ ] Package published to npmjs.org
- [ ] Package version matches git tag
- [ ] `ramorie` command works after npm install
- [ ] Binaries download correctly from new repo

### Homebrew
- [ ] Formula exists in `kutbudev/homebrew-tap`
- [ ] Formula points to correct release URLs
- [ ] `brew install` works
- [ ] `brew test ramorie` passes

---

## 🆘 TROUBLESHOOTING

### Issue: npm install fails to download binary

**Cause:** Release assets not yet available or wrong version in package.json

**Fix:**
```bash
# Ensure release exists
curl -I https://github.com/kutbudev/ramorie-cli/releases/download/v1.5.0/ramorie_1.5.0_darwin_arm64.tar.gz

# Update npm package version
cd npm
npm version 1.5.0 --no-git-tag-version
git add package.json
git commit -m "chore: bump npm version to 1.5.0"
git push
```

### Issue: Homebrew formula not updating

**Cause:** GoReleaser Homebrew push failed or token invalid

**Fix:**
1. Check GitHub Actions logs for errors
2. Verify `HOMEBREW_TAP_GITHUB_TOKEN` has correct permissions
3. Manually update formula if needed:
```bash
cd homebrew-tap
# Edit Formula/ramorie.rb with new version/URLs
git add Formula/ramorie.rb
git commit -m "chore: update ramorie to v1.5.0"
git push
```

### Issue: Go module import errors

**Cause:** Old import paths in code

**Fix:**
```bash
# Find all old imports
grep -r "github.com/terzigolu/josepshbrain-go" .

# Update imports (if any exist in .go files)
find . -name "*.go" -type f -exec sed -i '' 's|github.com/terzigolu/josepshbrain-go|github.com/kutbudev/ramorie-cli|g' {} +

# Update go.mod
go mod tidy
```

### Issue: GitHub Actions workflow fails

**Cause:** Missing secrets or permissions

**Fix:**
1. Verify all secrets are set: https://github.com/kutbudev/ramorie-cli/settings/secrets/actions
2. Check workflow permissions: Settings → Actions → General → Workflow permissions → "Read and write permissions"
3. Re-run failed workflow

---

## 📞 WHAT YUSUF MUST DO

### Critical Manual Actions

1. **GitHub Repository Transfer or Creation**
   - Transfer `terzigolu/josepshbrain-go` to `kutbudev/ramorie-cli` (recommended)
   - OR create new repo `kutbudev/ramorie-cli` and mirror push

2. **Create Homebrew Tap Repository**
   - Create: https://github.com/kutbudev/homebrew-tap
   - Initialize with README and Formula/ directory

3. **Generate and Add GitHub Secrets**
   - Create Personal Access Token for Homebrew tap
   - Add `HOMEBREW_TAP_GITHUB_TOKEN` to repo secrets
   - Add `NPM_TOKEN` to repo secrets (get from npmjs.com)

4. **Verify npm Package Ownership**
   - Ensure you own `ramorie` package on npmjs.org
   - If not, publish first version manually or request transfer

5. **Commit and Push Changes**
   - Review all file changes in this repo
   - Commit to new repository
   - Push to `kutbudev/ramorie-cli`

6. **Create First Release**
   - Tag version (e.g., v1.5.0)
   - Push tag to trigger release workflow
   - Monitor GitHub Actions

7. **Test All Install Methods**
   - npm install
   - brew install
   - curl script
   - go install

8. **Archive Old Repository** (if not transferred)
   - Add deprecation notice to README
   - Archive repository on GitHub

---

## 📅 TIMELINE ESTIMATE

- **Preparation:** 30 minutes (review, backup)
- **Migration:** 15 minutes (transfer or mirror)
- **Secrets Setup:** 20 minutes (tokens, permissions)
- **File Changes:** Already done ✅
- **First Release:** 10 minutes (tag, push, monitor)
- **Verification:** 30 minutes (test all install methods)
- **Cleanup:** 15 minutes (archive old repo, update docs)

**Total:** ~2 hours

---

## 🎯 SUCCESS CRITERIA

Migration is complete when:
- ✅ New repo `kutbudev/ramorie-cli` is live with full history
- ✅ GitHub Actions release workflow succeeds
- ✅ All 5 install methods work (npm, brew, curl, go, direct download)
- ✅ npm package downloads binaries from new repo
- ✅ Homebrew formula points to new repo
- ✅ Documentation updated everywhere
- ✅ Old repo archived with redirect notice

---

**Last Updated:** January 3, 2025
**Prepared By:** Cascade AI
**For:** Yusuf (kutbudev)
