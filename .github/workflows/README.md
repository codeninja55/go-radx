# GitHub Actions Workflows

This directory contains the GitHub Actions workflows for the go-radx project.

## Release Process

The release process is handled by two main workflows:

### 1. `release.yml` - Automated Tagging and Release

**Trigger:** Push to `main` branch (excluding documentation and GitHub workflow changes)

This workflow automatically:
- Creates a semantic version tag based on commit messages
- Runs GoReleaser to build cross-platform binaries with CGO support
- Creates a GitHub release with artifacts

**Configuration Options:**

#### Option A: Unified Workflow (Current Default)
The workflow tags and releases in a single job. This is the simplest approach and works out of the box.

#### Option B: Using Personal Access Token (For Separate Workflows)
If you want the tag creation to trigger a separate GoReleaser workflow:

1. Create a Personal Access Token (PAT) with `contents:write` and `actions:write` permissions
2. Add it as a repository secret named `GH_PAT`
3. The workflow will use this token instead of `GITHUB_TOKEN` to create tags that can trigger other workflows

### 2. `goreleaser.yml` - Manual Release Trigger

**Triggers:**
- Push of tags matching `v*.*.*` pattern (automatic if using PAT)
- Manual workflow dispatch with tag input

This workflow can be used:
- Automatically when tags are pushed (if using PAT in release.yml)
- Manually to re-run releases for existing tags
- As a backup if the automatic release fails

**Manual Trigger:**
1. Go to Actions → GoReleaser workflow
2. Click "Run workflow"
3. Enter the tag to release (e.g., `v0.4.1`)
4. Click "Run workflow"

## Other Workflows

- `test.yml` - Runs tests on pull requests and pushes
- `lint.yml` - Runs linting checks
- `security.yml` - Security vulnerability scanning
- `docs.yml` - Documentation deployment to GitHub Pages
- `docs-check.yml` - Documentation build validation
- `benchmark.yml` - Performance benchmarking
- `claude.yml` - Claude Code integration

## Troubleshooting

### Release Not Triggering Automatically

If the release doesn't trigger automatically after merge to main:

1. Check if a tag was created: `git fetch --tags && git tag`
2. If tag exists but release didn't run, manually trigger the GoReleaser workflow
3. Consider setting up the `GH_PAT` secret for automatic workflow chaining

### GoReleaser Failures

Common issues and solutions:

1. **CGO Dependencies Missing**: The workflow uses `goreleaser-cross` which includes necessary cross-compilation tools
2. **Version Already Exists**: Delete the existing release and tag, then re-run
3. **Build Failures**: Check the GoReleaser configuration in `.goreleaser.yml`

## Requirements

- Go 1.25.4+
- Docker (for cross-compilation via goreleaser-cross)
- CGO dependencies (libjpeg-turbo, OpenJPEG) - handled by goreleaser-cross