// Tests that when cleanPodPolicy is set to "Running", only the Running pods are deleted
// when the TFJob completes. The completed pods will not be deleted.
local params = std.extVar("__ksonnet/params").components.clean_pod_running_v1;

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
        cleanPodPolicy: "Running",
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
