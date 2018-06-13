local env = std.extVar("__ksonnet/environments");
local params = std.extVar("__ksonnet/params").components.master_is_chief_v1alpha2;

local k = import "k.libsonnet";

local name = params.name;
local namespace = env.namespace;

local podTemplate = {
  spec: {
    containers: [
      {
        name: "tensorflow",
        image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:latest",
      },
    ],
  },
};

local job = {
  apiVersion: "kubeflow.org/v1alpha2",
  kind: "TFJob",
  metadata: {
    name: name,
    namespace: namespace,
  },
  spec: {
    // When a chief is provided TFJob will use it.
    tfReplicaSpecs: {
      CHIEF: {
        replicas: 1,
        restartPolicy: "Never",
        template: podTemplate,
      },
      PS: {
        replicas: 2,
        restartPolicy: "Never",
        template: podTemplate,
      },
      WORKER: {
        replicas: 4,
        restartPolicy: "Never",
        template: podTemplate,
      },
    },
  },
};  // job.

std.prune(k.core.v1.list.new([job]))
