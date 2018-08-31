local params = std.extVar("__ksonnet/params").components.estimator_runconfig;

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
      cleanPodPolicy: "All",
      tfReplicaSpecs: {
        Chief: {
          replicas: 1,
          restartPolicy: "Never",
          template: {
            spec: {
              containers: [
                {
                  name: "tensorflow",
                  image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:v20180830-867fcad8",
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
                  image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:v20180830-867fcad8",
                },
              ],
            },
          },
        },
        Worker: {
          replicas: 2,
          restartPolicy: "Never",
          template: {
            spec: {
              containers: [
                {
                  name: "tensorflow",
                  image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:v20180830-867fcad8",
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
