local env = std.extVar("__ksonnet/environments");
local params = std.extVar("__ksonnet/params").components.worker0_is_chief_v1alpha2;

local k = import "k.libsonnet";

local actualImage = "python:2.7";
local name = params.name;
local namespace = env.namespace;
local configMap = {
  apiVersion: "v1",
  kind: "ConfigMap",
  metadata: {
    name: name,
    namespace: namespace,
  },
  data: {
    "worker0_is_chief.py": importstr "worker0_is_chief.py",
  },
};

local podTemplate = {
  spec: {
    containers: [
      {
        name: "tensorflow",
        image: actualImage,
        command: [
          "python",
          "/var/code/worker0_is_chief.py",
        ],
        volumeMounts: [
          {
            mountPath: "/var/code",
            name: "code",
          },
        ],
      },
    ],
    volumes: [
      {
        configMap: {
          name: name,
        },
        name: "code",
      },
    ],
  },
};

local job = {
  apiVersion: "kubeflow.org/v1alpha2",
  kind: "TFJob",
  metadata: {
    name: name,
    namespace: namespace,
  },
  spec: {
    // When no chief is provided TFJob will use worker 0 as the chief.
    tfReplicaSpecs: {
      PS: {
        replicas: 2,
        restartPolicy: "Never",
        template: podTemplate,
      },
      Worker: {
        replicas: 4,
        restartPolicy: "Never",
        template: podTemplate,
      },
    },
  },
};  // job.

std.prune(k.core.v1.list.new([job, configMap]))
