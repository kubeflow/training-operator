local env = std.extVar("__ksonnet/environments");
local params = std.extVar("__ksonnet/params").components.master_is_chief_v1alpha2;

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
    "has_chief_or_master.py": importstr "has_chief_or_master.py",
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
          "/var/code/has_chief_or_master.py",
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
    // When a chief is provided TFJob will use it.
    tfReplicaSpecs: {
      CHIEF: {
        replicas: 1,
        restartPolicy: "Never",
        template: podTemplate,
      },
      PS: {
        replicas: 2,
        restartPolicy: "Never",
        template: podTemplate,
      },
      WORKER: {
        replicas: 4,
        restartPolicy: "Never",
        template: podTemplate,
      },
    },
  },
};  // job.

std.prune(k.core.v1.list.new([job, configMap]))
