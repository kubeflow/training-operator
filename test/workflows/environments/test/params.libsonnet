local params = std.extVar('__ksonnet/params');
local globals = import 'globals.libsonnet';
local envParams = params + {
  components+: {
    workflows+: {
      namespace: 'kubeflow-test-infra',
      name: 'jlewi-tf-operator-release-v1alpha2-632-7e55',
      prow_env: 'JOB_NAME=tf-operator-release-v1alpha2,JOB_TYPE=presubmit,REPO_NAME=tf-operator,REPO_OWNER=kubeflow,BUILD_NUMBER=7e55,PULL_NUMBER=632',
      versionTag: "",
      tfJobVersion: 'v1alpha2',
    },
  },
};

{
  components: {
    [x]: envParams.components[x] + globals
    for x in std.objectFields(envParams.components)
  },
}