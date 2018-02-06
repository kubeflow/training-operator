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
      name: "training-presubmit-1-5-01133157",
      namespace: "kubeflow-test-infra",
      prow_env: "JOB_NAME=tf-k8s-presubmit,JOB_TYPE=presubmit,PULL_NUMBER=358,REPO_NAME=k8s,REPO_OWNER=tensorflow,BUILD_NUMBER=0c57",
    },
    e2e_test: {
      namespace: "some-namespace",
      name: "test",
      image: "gcr.io/tf-on-k8s-dogfood/tf_operator:v20180131-cabc1c0-dirty-e3b0c44",
    },
  },
}
