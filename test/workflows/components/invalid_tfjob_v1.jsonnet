// This is a test job to ensure we correctly handle the case where the job spec is not
// a valid TFJob and therefore can't be unmarshled to a TFJob struct.
// In this case we want to check that the TFJob status is updated correctly to reflect this.
//
local env = std.extVar("__ksonnet/environments");
local params = std.extVar("__ksonnet/params").components["invalid_tfjob_v1"];

local k = import "k.libsonnet";


local name = params.name;
local namespace = env.namespace;

local podTemplate = {
  spec: {
    containers: [
      {
        name: "tensorflow",
        // image doesn't matter because we won't actually create the pods
        image: "busybox",
      },
    ],
  },
};

local job = {
  apiVersion: "kubeflow.org/v1",
  kind: "TFJob",
  metadata: {
    name: name,
    namespace: namespace,
  },
  spec: {
    // Provide invalid json
    notTheActualField: {
      Ps: {
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

std.prune(k.core.v1.list.new([job]))
