{
  global: {},
  // TODO(jlewi): Having the component name not match the TFJob name is confusing.
  // Job names can't have hyphens in the name. Moving forward we should use hyphens
  // not underscores in component names.
  components: {
    // Component-level parameters, defined initially from 'ks prototype use ...'
    // Each object below should correspond to a component in the components/ directory
    workflows: {
      bucket: "kubeflow-ci_temp",
      name: "some-very-very-very-very-very-long-name-jlewi-tf-k8s-presubmit-test-374-6e32",
      namespace: "kubeflow-test-infra",
      prow_env: "JOB_NAME=tf-k8s-presubmit-test,JOB_TYPE=presubmit,PULL_NUMBER=374,REPO_NAME=k8s,REPO_OWNER=tensorflow,BUILD_NUMBER=6e32",
      versionTag: "",
      tfJobVersion: "v1alpha2",
    },
    master_is_chief_v1alpha2: {
      image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:latest",
      name: "master-is-chief-v1alpha2",
    },
    worker0_is_chief_v1alpha2: {
      image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:v20180613-e06fc0bb-dirty-5ef291",
      name: "worker0-is-chief-v1alpha2",
    },
    simple_tfjob_v1alpha2: {
      name: "simple-001",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    gpu_tfjob_v1alpha2: {
      name: "gpu-simple-001",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    clean_pod_all: {
      name: "clean_pod_all",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    clean_pod_running: {
      name: "clean_pod_running",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    clean_pod_none: {
      name: "clean_pod_none",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    estimator_runconfig: {
      name: "estimator_runconfig",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    distributed_training_v1alpha2: {
      name: "distributed_training",
      namespace: "kubeflow-test-infra",
      image: "gcr.io/kubeflow-examples/distributed_worker:v20181031-513e107c"
    },
    "invalid-tfjob": {
      name: "invalid-tfjob",
    },
  },
}
