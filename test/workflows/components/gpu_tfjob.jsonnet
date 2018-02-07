local params = std.extVar("__ksonnet/params").components.gpu_tfjob;

local k = import 'k.libsonnet';

local defaultTestImage = "gcr.io/tf-on-k8s-dogfood/tf_sample_gpu:dc944ff";
local parts(namespace, name, image) = {
  local actualImage = if image != "" then
    image
  else defaultTestImage,
  job:: {
    apiVersion: "kubeflow.org/v1alpha1",
    kind: "TFJob",
    metadata: {
      name: name,
      namespace: namespace,
    },
    spec: {
      replicaSpecs: [
        {
          template: {
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
          tfReplicaType: "MASTER",
        },
      ],
    },
  },  // job
};

std.prune(k.core.v1.list.new([parts(params.namespace, params.name, params.image).job]))
