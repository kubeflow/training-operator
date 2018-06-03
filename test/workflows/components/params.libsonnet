{
  global: {
    // User-defined global parameters; accessible to all component and environments, Ex:
    // replicas: 4,
  },
  components: {
    // Component-level parameters, defined initially from 'ks prototype use ...'
    // Each object below should correspond to a component in the components/ directory
    workflows: {
      bucket: "kubeflow-ci_temp",
      name: "some-very-very-very-very-very-long-name-jlewi-tf-k8s-presubmit-test-374-6e32",
      namespace: "kubeflow-test-infra",
      prow_env: "JOB_NAME=tf-k8s-presubmit-test,JOB_TYPE=presubmit,PULL_NUMBER=374,REPO_NAME=k8s,REPO_OWNER=tensorflow,BUILD_NUMBER=6e32",
      versionTag: null,
    },
    simple_tfjob: {
      name: "simple-001",
      namespace: "kubeflow-test-infra",
      image: "",
      apiVersion: "kubeflow.org/v1alpha1",
    },
    gpu_tfjob: {
      name: "gpu-simple-001",
      namespace: "kubeflow-test-infra",
      image: "",
    },
  },
}
