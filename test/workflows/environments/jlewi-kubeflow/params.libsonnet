local params = std.extVar('__ksonnet/params');
local globals = import 'globals.libsonnet';
local envParams = params + {
  components+: {
    set+: {
      image: 'gcr.io/tf-on-k8s-dogfood/tf_sample:dc944ff',
    },
    simple_tfjob+: {
      image: 'gcr.io/tf-on-k8s-dogfood/tf_sample:dc944ff',
      apiVersion: 'kubeflow.org/v1alpha2',
    },
  },
};

{
  components: {
    [x]: envParams.components[x] + globals
    for x in std.objectFields(envParams.components)
  },
}