local params = std.extVar("__ksonnet/params").components.simple_tfjob;

local k = import "k.libsonnet";

local defaultTestImage = "gcr.io/tf-on-k8s-dogfood/tf_sample:dc944ff";
local parts(namespace, name, image) = {
  local actualImage = if image != "" then
    image
  else defaultTestImage,
  job:: {
    apiVersion: params.apiVersion,
    kind: "TFJob",
    metadata: {
      name: name,
      namespace: namespace,
    },
    spec: {
      replicaSpecs: [
        {
          replicas: 1,
          template: {
            spec: {
              containers: [
                {
                  image: actualImage,
                  name: "tensorflow",
                },
              ],
              restartPolicy: "OnFailure",
            },
          },
          tfReplicaType: "MASTER",
        },
        {
          replicas: 1,
          template: {
            spec: {
              containers: [
                {
                  image: actualImage,
                  name: "tensorflow",
                },
              ],
              restartPolicy: "OnFailure",
            },
          },
          tfReplicaType: "WORKER",
        },
        {
          replicas: 2,
          template: {
            spec: {
              containers: [
                {
                  image: actualImage,
                  name: "tensorflow",
                },
              ],
              restartPolicy: "OnFailure",
            },
          },
          tfReplicaType: "PS",
        },
      ],
    },
  },
};

std.prune(k.core.v1.list.new([parts(params.namespace, params.name, params.image).job]))
