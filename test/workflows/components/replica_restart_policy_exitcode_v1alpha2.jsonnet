local params = std.extVar("__ksonnet/params").components.replica_restart_policy_exitcode_v1alpha2;

local k = import "k.libsonnet";

local defaultTestImage = "gcr.io/kubeflow-images-staging/tf-operator-test-server:latest";

local parts(namespace, name, image) = {
  local actualImage = if image != "" then
    image
  else defaultTestImage,
  job:: {
    apiVersion: "kubeflow.org/v1alpha2",
    kind: "TFJob",
    metadata: {
      name: name,
      namespace: namespace,
    },
    spec: {
      tfReplicaSpecs: {
        PS: {
          replicas: 1,
          restartPolicy: "ExitCode",
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
          replicas: 2,
          restartPolicy: "ExitCode",
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
      },
    },
  },
};

std.prune(k.core.v1.list.new([parts(params.namespace, params.name, params.image).job]))
