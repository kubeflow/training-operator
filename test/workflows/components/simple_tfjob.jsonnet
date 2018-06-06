local params = std.extVar("__ksonnet/params").components.simple_tfjob;

local k = import "k.libsonnet";

local defaultTestImage = "gcr.io/tf-on-k8s-dogfood/tf_sample:dc944ff";
local parts(namespace, name, image) = {
  local actualImage = if image != "" then
    image
  else defaultTestImage,
  job:: if params.apiVersion == "kubeflow.org/v1alpha1" then {
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
  } else {
    apiVersion: "kubeflow.org/v1alpha2",
    kind: "TFJob",
    metadata: {
      name: name,
      namespace: namespace,
    },
    spec: {
      tfReplicaSpecs: {
        PS: {
          replicas: 2,
          restartPolicy: "Never",
          template: {
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
        Worker: {
          replicas: 4,
          restartPolicy: "Never",
          template: {
            spec: {
              containers: [
                {
                  name: "tensorflow",
                  image: actualImage,
                  command: ["python", "/var/tf_dist_mnist/dist_mnist.py", "--train_steps=100"],
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
