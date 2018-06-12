local params = std.extVar('__ksonnet/params');
local globals = import 'globals.libsonnet';
local envParams = params + {
  components+: {
    workflows+: {
      namespace: 'kubeflow-releasing',
      name: 'jlewi-tf-operator-release-+v1alpha2-632-074c',
      prow_env: 'JOB_NAME=tf-operator-release-+v1alpha2,JOB_TYPE=presubmit,REPO_NAME=tf-operator,REPO_OWNER=kubeflow,BUILD_NUMBER=074c,PULL_NUMBER=632',
      versionTag: 'v20180611-pr632--074c',
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