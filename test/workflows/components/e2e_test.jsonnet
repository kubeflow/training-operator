local params = std.extVar("__ksonnet/params").components.e2e_test;

local k = import 'k.libsonnet';

local defaultTestImage = "gcr.io/tf-on-k8s-dogfood/tf_sample:dc944ff";
local parts = {
  job(namespace, name, image):: {
  	local actualImage= if image != "" then
  		image
  		else defaultTestImage,
    apiVersion: "batch/v1",
    kind: "Job",
    metadata: {
      name: name,
      namespace: namespace,
    },
    spec: {
      template: {
        spec: {
          containers: [{
            name: "runner",
            image: image,
            command: [
              "/opt/mlkube/test/e2e",
              "--image=" + actualImage,
              "--name=" + name,
              "--namespace=" + namespace,
              "-alsologtostderr",
              "-v=1",
            ],
          }],
          restartPolicy: "OnFailure",
          serviceAccountName: "tf-job-operator",
        },
      },
    },
  },
};

std.prune(k.core.v1.list.new([parts.job(params.namespace, params.name, params.image)]))
