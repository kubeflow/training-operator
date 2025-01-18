# Releasing the Training Operator

## Overview
The release process is now automated using GitHub Actions in [.github/workflows/release.yaml](.github/workflows/release.yaml). The workflow is triggered when the VERSION file is updated in any of the v*-branch branches.

## Prerequisites

- Write permission for the Training Operator repository
- PyPI token stored as `PYPI_TOKEN` in GitHub repository secrets
- GitHub token with appropriate permissions

## Release Process

1. Create a release branch (if needed)
   - For major/minor releases: Create a new branch `vX.Y-branch` from master
   - For patch releases: Use existing branch

2. Update version files:
   - Update VERSION file with new version (e.g., `v1.8.0` or `v1.8.0-rc.1`)
   - Update version in:
     - [hack/python-sdk/gen-sdk.sh](hack/python-sdk/gen-sdk.sh)
     - [hack/python-sdk/swagger_config.json](hack/python-sdk/swagger_config.json) 
     - [sdk/python/setup.py](sdk/python/setup.py)

3. Push changes and create PR:
   ```bash
   git checkout -b update-version-X.Y.Z
   git add VERSION
   git commit -m "Update version to vX.Y.Z"
   git push origin update-version-X.Y.Z

Once merged, the release workflow will automatically:

- Validate version consistency
- Build and publish SDK to PyPI
- Build and push container images
- Create Git tag
- Generate changelog
- Create draft GitHub release
- Review and publish the draft release on GitHub

## Version Format

- **Release versions:** `vX.Y.Z` (e.g., `v1.8.0`)
- **Release candidates:** `vX.Y.Z-rc.N` (e.g., `v1.8.0-rc.1`)

## Release Branches

- **Format:** `vX.Y-branch`
- **Example:** `v1.8-branch` for `1.8.x` releases
- Cherry-pick changes from master for patch releases
