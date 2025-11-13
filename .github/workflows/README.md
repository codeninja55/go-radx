# GitHub Actions Workflows

This directory contains the GitHub Actions workflows for the go-radx project.

## Release Process

The release process is handled by two separate workflows:

### 1. `release.yml` - Automatic Tagging

**Trigger:** Push to `main` branch (excluding documentation and GitHub workflow changes)

This workflow automatically creates semantic version tags based on commit messages:
- `feat:` commits trigger minor version bump
- `fix:` commits trigger patch version bump
- `BREAKING CHANGE` or `!` triggers major version bump

**⚠️ IMPORTANT: Personal Access Token Required**

For the tag to automatically trigger the GoReleaser workflow, you MUST configure a Personal Access Token:

1. Create a PAT at https://github.com/settings/tokens with `repo` scope
2. Add it as a secret named `GH_PAT` in Settings → Secrets and variables → Actions
3. The workflow will use this token to create tags that can trigger other workflows

Without the PAT:
- Tags will still be created
- GoReleaser will NOT run automatically
- You'll need to manually trigger the GoReleaser workflow

### 2. `goreleaser.yml` - Build and Release

**Triggers:**
- Automatic: When tags matching `v*.*.*` are pushed (requires PAT in release.yml)
- Manual: Workflow dispatch with tag input

This workflow:
- Builds cross-platform binaries with CGO support for JPEG/JPEG2000 decompression
- Creates GitHub releases with artifacts
- Generates changelogs

**Manual Release (if automatic trigger fails):**
1. Go to [Actions → GoReleaser workflow](../../actions/workflows/goreleaser.yml)
2. Click "Run workflow"
3. Enter the tag to release (e.g., `v0.4.2`)
4. Click "Run workflow"

## Setup Instructions

### Quick Setup (Automatic Releases)

1. **Create a Personal Access Token:**
   - Go to https://github.com/settings/tokens/new
   - Name: `go-radx-release` (or any name you prefer)
   - Expiration: Choose based on your preference
   - Scopes: Select `repo` (full control of private repositories)
   - Click "Generate token"
   - Copy the token (you won't see it again!)

2. **Add PAT to Repository Secrets:**
   - Go to repository Settings → Secrets and variables → Actions
   - Click "New repository secret"
   - Name: `GH_PAT`
   - Value: Paste your Personal Access Token
   - Click "Add secret"

3. **Verify Setup:**
   - Next merge to main will create a tag
   - Check Actions tab to see if GoReleaser triggers automatically
   - If successful, releases will be fully automated

### Alternative: Manual Release Process

If you prefer not to use a PAT:

1. Tags will still be created automatically on merge to main
2. After a tag is created, manually trigger the GoReleaser workflow
3. This gives you more control over when releases are published

## Workflow Files

| Workflow | Purpose | Trigger |
|----------|---------|---------|
| `release.yml` | Create semantic version tags | Push to main |
| `goreleaser.yml` | Build binaries and create releases | Tag push or manual |
| `test.yml` | Run tests | PR and push |
| `lint.yml` | Code quality checks | PR and push |
| `security.yml` | Security scanning | PR and push |
| `docs.yml` | Deploy documentation | Push to main |
| `benchmark.yml` | Performance testing | Manual |

## Troubleshooting

### GoReleaser Not Triggering Automatically

**Symptom:** Tag is created but GoReleaser doesn't run

**Solution:**
1. Verify `GH_PAT` secret is configured correctly
2. Check that the PAT has `repo` scope
3. Ensure the PAT hasn't expired
4. As a fallback, manually trigger the GoReleaser workflow

### Build Failures

**Common issues:**
- **CGO errors**: The workflow uses `goreleaser-cross` with all necessary dependencies
- **Version conflicts**: Check if tag already exists with `git tag`
- **Configuration issues**: Validate `.goreleaser.yml` syntax

### Tag Creation Issues

**Symptom:** No tag created after merge to main

**Possible causes:**
1. Commit touched only ignored paths (docs, .github, etc.)
2. No conventional commit format in merge commit
3. Workflow syntax error

**Debug steps:**
1. Check Actions tab for workflow runs
2. Review workflow logs for errors
3. Verify commit message format

## Commit Message Format

For automatic version bumping, use conventional commits:

```
feat: add new DICOM export feature      # Minor version bump
fix: resolve memory leak in parser      # Patch version bump
feat!: redesign API endpoints           # Major version bump
chore: update dependencies              # Patch version bump
docs: improve README                    # Patch version bump
```

## Security Notes

- The `GH_PAT` should only have necessary permissions (`repo` scope)
- Rotate the PAT periodically for security
- Never commit the PAT to the repository
- Consider using fine-grained PATs for more restricted access