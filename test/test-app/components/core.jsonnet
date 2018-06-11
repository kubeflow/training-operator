local env = std.extVar("__ksonnet/environments");
local params = std.extVar("__ksonnet/params").components.core;

local k = import "k.libsonnet";
local all = import "kubeflow/core/all.libsonnet";

// updatedParams uses the environment namespace if
// the namespace parameter is not explicitly set
local updatedParams = params {
  namespace: if params.namespace == "null" then env.namespace else params.namespace,
};

std.prune(k.core.v1.list.new(all.parts(updatedParams).all))
