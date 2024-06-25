# Releasing the Training Operator

## Prerequisite

- [Write](https://docs.github.com/en/organizations/managing-access-to-your-organizations-repositories/repository-permission-levels-for-an-organization#permission-levels-for-repositories-owned-by-an-organization)
  permission for the Training Operator repository.

- Maintainer access to the [Training SDK](https://pypi.org/project/kubeflow-training/).

- Create a [GitHub Token](https://docs.github.com/en/github/authenticating-to-github/keeping-your-account-and-data-secure/creating-a-personal-access-token).

- Install `PyGithub` to generate the [Changelog](./../../CHANGELOG.md):

  ```
  pip install PyGithub==1.55
  ```

- Install `twine` to publish the SDK package:

  ```
  pip install twine==3.4.1
  ```

  - Create a [PyPI Token](https://pypi.org/help/#apitoken) to publish Training SDK.

  - Add the following config to your `~/.pypirc` file:

    ```
    [pypi]
       username = __token__
       password = <PYPI_TOKEN>
    ```

## Versioning policy

Training Operator version format follows [Semantic Versioning](https://semver.org/).
Training Operator versions are in the format of `vX.Y.Z`, where `X` is the major version, `Y` is
the minor version, and `Z` is the patch version.
The patch version contains only bug fixes.

Additionally, Training Operator does pre-releases in this format: `vX.Y.Z-rc.N` where `N` is a number
of the `Nth` release candidate (RC) before an upcoming public release named `vX.Y.Z`.

## Release branches and tags

Training Operator releases are tagged with tags like `vX.Y.Z`, for example `v0.11.0`.

Release branches are in the format of `vX.Y-branch`, where `X.Y` stands for
the minor release.

`vX.Y.Z` releases are released from the `vX.Y-branch` branch. For example,
`v1.8.0` release should be on `v1.8-branch` branch.

If you want to push changes to the `vX.Y-branch` release branch, you have to
cherry pick your changes from the `master` branch and submit a PR.

## Create a new Training Operator release

### Create release branch

1. Depends on what version you want to release,

   - Major or Minor version - Use the GitHub UI to create a release branch from `master` and name
     the release branch `vX.Y-branch`
   - Patch version - You don't need to create a new release branch.

1. Fetch the upstream changes into your local directory:

   ```
   git fetch upstream
   ```

1. Checkout into the release branch:

   ```
   git checkout vX.Y-branch
   ```

### Release Training SDK

1. Update the [VERSION in `gen-sdk.sh` script](../../hack/python-sdk/gen-sdk.sh#L27) with this
   semantic: `X.Y.ZRC.N` for the RC or `X.Y.Z` for the public release.

   For example:

   ```
   VERSION=1.8.0rc1
   ```

1. Generate Training SDK:

   ```
   make generate
   ```

1. Submit PR

#### Release Training Operator Image

1. Make sure the last commit you want to release past `kubeflow-training-operator-postsubmit` testing.

1. Check out that commit (in this example, we'll use `6214e560`).

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
