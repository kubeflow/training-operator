# Releasing the Training Operator

## Prerequisite

1. Permissions
   - You need write permissions on the repository to create a release tag/branch.
1. Prepare your Github Token

1. Install Github python dependencies to generate changlog
   ```
   pip install PyGithub==1.55
   ```

### Release Process

1. Make sure the last commit you want to release past `kubeflow-training-operator-postsubmit` testing.

1. Check out that commit (in this example, we'll use `6214e560`).

1. Depends on what version you want to release,

   - Major or Minor version - Use the GitHub UI to cut a release branch and name the release branch `v{MAJOR}.${MINOR}-branch`
   - Patch version - You don't need to cut release branch.

1. Create a new PR against the release branch to change container image in manifest to point to that commit hash.

   ```
   images:
   - name: kubeflow/training-operator
     newName: kubeflow/training-operator
     newTag: ${commit_hash}
   ```

   > note: post submit job will always build a new image using the `PULL_BASE_HASH` as image tag.

1. Create a tag and push tag to upstream.

   ```
   git tag v1.2.0
   git push upstream v1.2.0
   ```

1. Update the Changelog by running:

   ```
   python docs/release/changelog.py --token=<github-token> --range=<previous-release>..<current-release>
   ```

   If you are creating the **first minor pre-release** or the **minor** release (`X.Y`), your
   `previous-release` is equal to the latest release on the `vX.Y-branch` branch.
   For example: `--range=v1.7.1..v1.8.0`

   Otherwise, your `previous-release` is equal to the latest release on the `vX.Y-branch` branch.
   For example: `--range=v1.7.0..v1.8.0-rc.0`

   Group PRs in the Changelog into Features, Bug fixes, Documentation, etc.
   Check this example: [v1.7.0-rc.0](https://github.com/kubeflow/training-operator/blob/master/CHANGELOG.md#v170-rc0-2023-07-07)

   Finally, submit a PR with the updated Changelog.
