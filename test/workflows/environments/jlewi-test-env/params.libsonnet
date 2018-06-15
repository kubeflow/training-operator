local params = std.extVar('__ksonnet/params');
local globals = import 'globals.libsonnet';
local envParams = params + {
  components+: {
    master_is_chief_v1alpha2+: {
      name: 'jlewi-master-is-chief',
      namespace: 'kubeflow',
      image: 'gcr.io/kubeflow-images-staging/tf-operator-test-server:v20180613-e06fc0bb-dirty-5ef291',
    },
  },
};

{
  components: {
    [x]: envParams.components[x] + globals
    for x in std.objectFields(envParams.components)
  },
}