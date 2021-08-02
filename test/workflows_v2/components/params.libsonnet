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
      tfJobVersion: "v1beta1",
    },
    // v1beta2 components
    simple_tfjob_v1beta2: {
      name: "simple-001",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    gpu_tfjob_v1beta2: {
      name: "gpu-simple-001",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    master_is_chief_v1beta2: {
      image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:latest",
      name: "master-is-chief-v1beta2",
    },
    worker0_is_chief_v1beta2: {
      image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:v20180613-e06fc0bb-dirty-5ef291",
      name: "worker0-is-chief-v1beta2",
    },
    clean_pod_all_v1beta2: {
      name: "clean_pod_all",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    clean_pod_running_v1beta2: {
      name: "clean_pod_running",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    clean_pod_none_v1beta2: {
      name: "clean_pod_none",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    estimator_runconfig_v1beta2: {
      name: "estimator_runconfig",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    distributed_training_v1beta2: {
      name: "distributed_training",
      namespace: "kubeflow-test-infra",
      image: "gcr.io/kubeflow-examples/distributed_worker:v20181031-513e107c"
    },
    "invalid_tfjob_v1beta2": {
      name: "invalid-tfjob",
    },
    replica_restart_policy_always_v1beta2: {
       name: "replica-restart-policy-always",
       namespace: "kubeflow-test-infra",
       image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:latest"
    },
    replica_restart_policy_onfailure_v1beta2: {
       name: "replica-restart-policy-onfailure",
       namespace: "kubeflow-test-infra",
       image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:latest"
    },
    replica_restart_policy_never_v1beta2: {
       name: "replica-restart-policy-never",
       namespace: "kubeflow-test-infra",
       image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:latest"
    },
    replica_restart_policy_exitcode_v1beta2: {
       name: "replica-restart-policy-exitcode",
       namespace: "kubeflow-test-infra",
       image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:latest"
    },
    pod_names_validation_v1beta2: {
      name: "pod_names_validation",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    // v1 components
    simple_tfjob_v1: {
      name: "simple-001",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    gpu_tfjob_v1: {
      name: "gpu-simple-001",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    master_is_chief_v1: {
      image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:latest",
      name: "master-is-chief-v1",
    },
    worker0_is_chief_v1: {
      image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:v20180613-e06fc0bb-dirty-5ef291",
      name: "worker0-is-chief-v1",
    },
    clean_pod_all_v1: {
      name: "clean_pod_all",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    clean_pod_running_v1: {
      name: "clean_pod_running",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    clean_pod_none_v1: {
      name: "clean_pod_none",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    estimator_runconfig_v1: {
      name: "estimator_runconfig",
      namespace: "kubeflow-test-infra",
      image: "",
    },
    distributed_training_v1: {
      name: "distributed_training",
      namespace: "kubeflow-test-infra",
      image: "gcr.io/kubeflow-examples/distributed_worker:v20181031-513e107c"
    },
    "invalid_tfjob_v1": {
      name: "invalid-tfjob",
    },
    replica_restart_policy_always_v1: {
       name: "replica-restart-policy-always",
       namespace: "kubeflow-test-infra",
       image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:latest"
    },
    replica_restart_policy_onfailure_v1: {
       name: "replica-restart-policy-onfailure",
       namespace: "kubeflow-test-infra",
       image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:latest"
    },
    replica_restart_policy_never_v1: {
       name: "replica-restart-policy-never",
       namespace: "kubeflow-test-infra",
       image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:latest"
    },
    replica_restart_policy_exitcode_v1: {
       name: "replica-restart-policy-exitcode",
       namespace: "kubeflow-test-infra",
       image: "gcr.io/kubeflow-images-staging/tf-operator-test-server:latest"
    },
    pod_names_validation_v1: {
      name: "pod_names_validation",
      namespace: "kubeflow-test-infra",
      image: "",
    },
  },
}
