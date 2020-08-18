local params = std.extVar("__ksonnet/params").components.gpu_tfjob_v1;

local k = import "k.libsonnet";

local defaultTestImage = "gcr.io/kubeflow-examples/tf_smoke:v20180723-65c28134";
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
        Chief: {
          replicas: 1,
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
                  image: actualImage,
                  name: "tensorflow",
                  resources: {
                    limits: {
                      "nvidia.com/gpu": 1,
                    },
                  },
                },
              ],
              restartPolicy: "OnFailure",
            },
          },
        },
      },
    },
  },  // job
};

std.prune(k.core.v1.list.new([parts(params.namespace, params.name, params.image).job]))
