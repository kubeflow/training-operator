// Tests that when cleanPodPolicy is set to "None", none of the pods are deleted
// when the TFJob completes.

local params = std.extVar("__ksonnet/params").components.clean_pod_none_v1alpha2;

local k = import "k.libsonnet";

local parts(namespace, name, image) = {
  job:: {
    apiVersion: "kubeflow.org/v1alpha2",
    kind: "TFJob",
    metadata: {
      name: name,
      namespace: namespace,
    },
    spec: {
      cleanPodPolicy: "None",
      tfReplicaSpecs: {
        Chief: {
          replicas: 1,
          restartPolicy: "Never",
          template: {
            spec: {
              containers: [
                {
                  name: "tensorflow",
                  image: "ubuntu",
                  command: [
                    "echo",
                    "Hello",
                  ],
                },
              ],
            },
          },
        },
        PS: {
          replicas: 2,
          restartPolicy: "Never",
          template: {
            spec: {
              containers: [
                {
                  name: "tensorflow",
                  image: "ubuntu",
                  command: [
                    "tail",
                    "-f",
                    "/dev/null",
                  ],
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
                  image: "ubuntu",
                  command: [
                    "echo",
                    "Hello",
                  ],
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
