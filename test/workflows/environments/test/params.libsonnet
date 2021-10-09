local params = std.extVar('__ksonnet/params');
local globals = import 'globals.libsonnet';
local envParams = params + {
  components+: {
    workflows+: {
      namespace: 'kubeflow-test-infra',
      name: 'training-operator-release-d746bde9-kunming',
      prow_env: 'JOB_NAME=training-operator-release,JOB_TYPE=training-operator-release,REPO_NAME=training-operator,REPO_OWNER=kubeflow,BUILD_NUMBER=01A3,PULL_BASE_SHA=d746bde9',
      versionTag: 'v20190702-d746bde9',
      registry: 'gcr.io/kubeflow-images-public',
      bucket: 'kubeflow-releasing-artifacts',
    },
  },
};

{
  components: {
    [x]: envParams.components[x] + globals
    for x in std.objectFields(envParams.components)
  },
}