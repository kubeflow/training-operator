local env = std.extVar("__ksonnet/environments");
local params = std.extVar("__ksonnet/params").components.worker0_is_chief_v1alpha1;

local k = import "k.libsonnet";

local actualImage = "gcr.io/kubeflow-images-staging/tf-operator-test-server:latest";
local name = params.name;
local namespace = env.namespace;

local podTemplate = {
  spec: {
    containers: [
      {
        name: "tensorflow",
        image: actualImage,
      },
    ],
    volumes: [
      {
        configMap: {
          name: name,
        },
        name: "code",
      },
    ],
  },
};

local job = {
  apiVersion: "kubeflow.org/v1alpha1",
  kind: "TFJob",
  metadata: {
    name: name,
    namespace: namespace,
  },
  spec: {
    replicaSpecs: [
      {
        replicas: 1,
        template: podTemplate,
        tfReplicaType: "MASTER",
      },
      {
        replicas: 2,
        template: podTemplate,
        tfReplicaType: "PS",
      },
      {
        replicas: 4,
        template: podTemplate,
        tfReplicaType: "WORKER",
      },
  ],
    terminationPolicy: {
      chief: {
        replicaName: "WORKER",
        replicaIndex: 0,
      },
    },
},
};  // job.

std.prune(k.core.v1.list.new([job]))
