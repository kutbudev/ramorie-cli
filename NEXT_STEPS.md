# ✅ Migration Started - Next Steps for Yusuf

**Repository:** https://github.com/kutbudev/ramorie-cli
**Status:** ✅ Code migrated, main branch pushed with full history
**Date:** January 3, 2026

---

## 🎉 COMPLETED

- ✅ All file changes committed and pushed
- ✅ Repository remote updated to `kutbudev/ramorie-cli`
- ✅ Main branch pushed with full git history
- ✅ Migration branch available for reference

**View changes:** https://github.com/kutbudev/ramorie-cli/commit/fa127f7

---

## 🚨 CRITICAL: MANUAL ACTIONS REQUIRED

### 1. Create Homebrew Tap Repository (5 minutes)

**Create:** https://github.com/organizations/kutbudev/repositories/new

- Repository name: `homebrew-tap`
- Description: "Homebrew tap for Kutbu Dev tools"
- Visibility: **Public**
- Initialize with README: ✅ Yes
- Click "Create repository"

**Then add Formula directory:**
```bash
git clone https://github.com/kutbudev/homebrew-tap.git
cd homebrew-tap
mkdir -p Formula
git add Formula/
git commit -m "chore: add Formula directory"
git push
```

---

### 2. Configure GitHub Secrets (10 minutes)

**Navigate to:** https://github.com/kutbudev/ramorie-cli/settings/secrets/actions

Click "New repository secret" for each:

#### Secret 1: HOMEBREW_TAP_GITHUB_TOKEN

1. Go to: https://github.com/settings/tokens/new
2. Token name: `ramorie-homebrew-tap-token`
3. Expiration: **No expiration** (or 1 year)
4. Select scopes:
   - ✅ `repo` (Full control of private repositories)
   - ✅ `workflow` (Update GitHub Action workflows)
5. Click "Generate token"
6. **Copy the token immediately**
7. Add to repo secrets:
   - Name: `HOMEBREW_TAP_GITHUB_TOKEN`
   - Value: [paste token]

#### Secret 2: NPM_TOKEN

1. Login to npm: https://www.npmjs.com/login
2. Go to: https://www.npmjs.com/settings/YOUR_USERNAME/tokens
3. Click "Generate New Token" → "Classic Token"
4. Token type: **Automation**
5. Copy the token
6. Add to repo secrets:
   - Name: `NPM_TOKEN`
   - Value: [paste token]

**Note:** `GITHUB_TOKEN` is auto-provided by GitHub Actions, no action needed.

---

### 3. Verify npm Package Ownership (2 minutes)

Check: https://www.npmjs.com/package/ramorie

**If you own it:** ✅ Ready to proceed

**If you don't own it:**
- Option A: Publish first version manually (see below)
- Option B: Request transfer from current owner

---

### 4. Create First Release (5 minutes)

Once secrets are configured:

```bash
cd /Users/terzigolu/Documents/GitHub/AI_ML/josepshbrain-go

# Ensure you're on main with latest changes
git checkout main
git pull

# Create release tag
git tag -a v1.5.0 -m "Release v1.5.0 - Repository migration to kutbudev/ramorie-cli

- Migrated from terzigolu/josepshbrain-go
- Updated all repository references
- Unified release pipeline
- Full feature parity maintained"

# Push tag (this triggers release workflow)
git push origin v1.5.0
```

**This will trigger:**
- ✅ GoReleaser workflow
- ✅ Binary builds for all platforms
- ✅ GitHub release creation
- ✅ Homebrew formula update
- ✅ npm package publish

**Monitor:** https://github.com/kutbudev/ramorie-cli/actions

---

### 5. Verify Release Workflow (10 minutes)

After pushing the tag, watch the GitHub Actions workflow:

**Check that:**
- [ ] GoReleaser job completes (green checkmark)
- [ ] Binaries are attached to release: https://github.com/kutbudev/ramorie-cli/releases/tag/v1.5.0
- [ ] Homebrew formula updated in `kutbudev/homebrew-tap`
- [ ] npm package published: https://www.npmjs.com/package/ramorie

**If workflow fails:**
- Check the Actions logs for errors
- Verify secrets are set correctly
- Common issues:
  - Missing `HOMEBREW_TAP_GITHUB_TOKEN` → Add the secret
  - Missing `NPM_TOKEN` → Add the secret
  - npm package ownership → Publish manually first time

---

### 6. Test All Install Methods (15 minutes)

Once release succeeds, test each install method:

#### Test 1: npm
```bash
npm install -g ramorie
ramorie --version
ramorie --help
```

#### Test 2: Homebrew
```bash
brew tap kutbudev/tap
brew install ramorie
ramorie --version
```

#### Test 3: curl script
```bash
curl -sSL https://raw.githubusercontent.com/kutbudev/ramorie-cli/main/install.sh | bash
ramorie --version
```

#### Test 4: Go install
```bash
go install github.com/kutbudev/ramorie-cli/cmd/jbraincli@latest
ramorie --version
```

#### Test 5: Direct download
1. Visit: https://github.com/kutbudev/ramorie-cli/releases/latest
2. Download binary for your platform
3. Extract and run: `./ramorie --version`

---

## 📋 CHECKLIST

Copy this to track your progress:

```
Migration Execution:
- [x] Code pushed to kutbudev/ramorie-cli
- [x] Main branch updated with full history
- [ ] Homebrew tap repository created
- [ ] HOMEBREW_TAP_GITHUB_TOKEN secret added
- [ ] NPM_TOKEN secret added
- [ ] npm package ownership verified
- [ ] First release tag created (v1.5.0)
- [ ] GitHub Actions workflow succeeded
- [ ] Binaries uploaded to release
- [ ] Homebrew formula updated
- [ ] npm package published

Testing:
- [ ] npm install works
- [ ] brew install works
- [ ] curl install works
- [ ] go install works
- [ ] Direct download works
```

---

## ⏱️ TIME ESTIMATE

- Homebrew tap: **5 min**
- GitHub secrets: **10 min**
- npm ownership check: **2 min**
- Create release: **5 min**
- Verify workflow: **10 min**
- Test installs: **15 min**

**Total: ~45 minutes**

---

## 🆘 TROUBLESHOOTING

### Issue: GitHub Actions workflow fails

**Check:**
1. Secrets are set: https://github.com/kutbudev/ramorie-cli/settings/secrets/actions
2. Homebrew tap exists: https://github.com/kutbudev/homebrew-tap
3. Workflow permissions: Settings → Actions → General → "Read and write permissions"

**View logs:** https://github.com/kutbudev/ramorie-cli/actions

### Issue: npm publish fails

**Possible causes:**
- Package name already taken by someone else
- Missing NPM_TOKEN
- Token doesn't have publish permissions

**Fix:**
- Verify ownership: https://www.npmjs.com/package/ramorie
- Regenerate token with "Automation" type
- Or publish manually first time:
  ```bash
  cd npm
  npm publish --access public
  ```

### Issue: Homebrew formula not updating

**Check:**
- Token has `repo` and `workflow` scopes
- Homebrew tap repository exists
- GoReleaser logs show Homebrew push attempt

**Manual fix:**
```bash
cd homebrew-tap
# Edit Formula/ramorie.rb manually if needed
git add Formula/ramorie.rb
git commit -m "chore: update ramorie formula"
git push
```

---

## 📚 REFERENCE DOCS

- **Full Migration Guide:** `MIGRATION_GUIDE.md`
- **Quick Summary:** `MIGRATION_SUMMARY.md`
- **This File:** `NEXT_STEPS.md`

---

## 🎯 SUCCESS CRITERIA

Migration is complete when:
- ✅ GitHub Actions release workflow succeeds
- ✅ All 5 install methods work
- ✅ npm downloads binaries from new repo
- ✅ Homebrew formula points to new repo
- ✅ Documentation is accurate

---

**Current Status:** 🟡 Awaiting manual actions (secrets + Homebrew tap)
**Next Action:** Create Homebrew tap repository
**Then:** Add GitHub secrets
**Finally:** Create release tag v1.5.0
