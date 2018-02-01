{
  global: {
    // User-defined global parameters; accessible to all component and environments, Ex:
    // replicas: 4,
  },
  components: {
    // Component-level parameters, defined initially from 'ks prototype use ...'
    // Each object below should correspond to a component in the components/ directory
    workflows: {
      bucket: "mlkube-testing_temp",
      name: "training-presubmit-1-5-31204805",
      namespace: "kubeflow-test-infra",
      prow_env: "BUILD_NUMBER=2,JOB_NAME=tf-k8s-presubmit,JOB_TYPE=presubmit,PULL_NUMBER=358,REPO_NAME=k8s,REPO_OWNER=tensorflow",
    },
  },
}
