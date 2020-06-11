local params = std.extVar("__ksonnet/params").components.distributed_training_v1;

local k = import "k.libsonnet";

local defaultTestImage = "gcr.io/kubeflow-examples/distributed_worker:v20181031-513e107c";
local parts(namespace, name, image) = {
  local actualImage = if image != "" then
    image
  else defaultTestImage,
  job:: {
    apiVersion: "kubeflow.org/v1",
    kind: "TFJob",
    metadata: {
      name: name,
      namespace: namespace,
    },
    spec: {
      tfReplicaSpecs: {
        Worker: {
          replicas: 3,
          restartPolicy: "Never",
          template: {
            metadata: {
              annotations: {
                "sidecar.istio.io/inject": "false",
              },
            },
            spec: {
              containers: [
                {
                  name: "tensorflow",
                  image: actualImage,
                },
              ],
            },
          },
        },
      },
    },
  },
};

std.prune(k.core.v1.list.new([parts(params.namespace, params.name, params.image).job]))
