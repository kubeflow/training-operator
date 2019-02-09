local env = std.extVar("__ksonnet/environments");
local params = std.extVar("__ksonnet/params").components.core;

local k = import "k.libsonnet";
local all = import "kubeflow/core/all.libsonnet";

// updatedParams uses the environment namespace if
// the namespace parameter is not explicitly set
local updatedParams = params {
  namespace: if params.namespace == "null" then env.namespace else params.namespace,
};

// Do not prune here since it will remove the status subresource which is an empty dictionary.
k.core.v1.list.new(all.parts(updatedParams).all)
