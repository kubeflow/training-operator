local params = std.extVar('__ksonnet/params');
local globals = import 'globals.libsonnet';
local envParams = params + {
  components+: {
    worker0_is_chief_v1alpha1+: {
      name: 'jlewi-worker0-is-chief',
      namespace: 'kubeflow',
    },
  },
};

{
  components: {
    [x]: envParams.components[x] + globals
    for x in std.objectFields(envParams.components)
  },
}