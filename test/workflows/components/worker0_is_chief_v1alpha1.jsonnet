local env = std.extVar("__ksonnet/environments");
local params = std.extVar("__ksonnet/params").components.worker0_is_chief_v1alpha1;

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
        template: podTemplate,
        tfReplicaType: "MASTER",
      },
      {
        replicas: 2,
        template: podTemplate,
        tfReplicaType: "PS",
      },
      {
        replicas: 4,
        template: podTemplate,
        tfReplicaType: "WORKER",
      },
  ],
    terminationPolicy: {
      chief: {
        replicaName: "WORKER",
        replicaIndex: 0,
      },
    },
},
};  // job.

std.prune(k.core.v1.list.new([job, configMap]))
