// Tests that when cleanPodPolicy is set to "All", all of the pods are deleted
// when the TFJob completes.
local params = std.extVar("__ksonnet/params").components.clean_pod_all_v1;

local k = import "k.libsonnet";

local parts(namespace, name, image) = {
  job:: {
    apiVersion: "kubeflow.org/v1",
    kind: "TFJob",
    metadata: {
      name: name,
      namespace: namespace,
    },
    spec: {
      runPolicy: {
        cleanPodPolicy: "All",
      },
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
            metadata: {
              annotations: {
                "sidecar.istio.io/inject": "false",
              },
            },
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
            metadata: {
              annotations: {
                "sidecar.istio.io/inject": "false",
              },
            },
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
