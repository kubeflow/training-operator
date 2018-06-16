local env = std.extVar("__ksonnet/environments");
local params = std.extVar("__ksonnet/params").components.worker0_is_chief_v1alpha2;

local k = import "k.libsonnet";

local actualImage = if std.objectHas(params, "image") && std.length(params.image) > 0 then
  params.image
else
  "gcr.io/kubeflow-images-staging/tf-operator-test-server:latest";

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
    // When no chief is provided TFJob will use worker 0 as the chief.
    tfReplicaSpecs: {
      Ps: {
        replicas: 2,
        restartPolicy: "Never",
        template: podTemplate,
      },
      Worker: {
        replicas: 4,
        restartPolicy: "Never",
        template: podTemplate,
      },
    },
  },
};  // job.

std.prune(k.core.v1.list.new([job]))
