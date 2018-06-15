{
  global: {},
  components: {
    // Component-level parameters, defined initially from 'ks prototype use ...'
    // Each object below should correspond to a component in the components/ directory
    workflows: {
      bucket: 'kubeflow-ci_temp',
      name: 'some-very-very-very-very-very-long-name-jlewi-tf-k8s-presubmit-test-374-6e32',
      namespace: 'kubeflow-test-infra',
      prow_env: 'JOB_NAME=tf-k8s-presubmit-test,JOB_TYPE=presubmit,PULL_NUMBER=374,REPO_NAME=k8s,REPO_OWNER=tensorflow,BUILD_NUMBER=6e32',
      versionTag: '',
      tfJobVersion: 'v1alpha1',
    },
    master_is_chief_v1alpha1: {
      name: 'master-is-chief-v1alpha1',
    },
    master_is_chief_v1alpha2: {
      image: 'gcr.io/kubeflow-images-staging/tf-operator-test-server:latest',
      name: 'master-is-chief-v1alpha2',
    },
    worker0_is_chief_v1alpha1: {
      image: 'gcr.io/kubeflow-images-staging/tf-operator-test-server:v20180613-e06fc0bb-dirty-5ef291',
      name: 'worker0-is-chief-v1alpha1',
    },
    worker0_is_chief_v1alpha2: {
      name: 'worker0-is-chief-v1alpha2',
    },
    simple_tfjob: {
      name: 'simple-001',
      namespace: 'kubeflow-test-infra',
      image: '',
      apiVersion: 'kubeflow.org/v1alpha1',
    },
    gpu_tfjob: {
      name: 'gpu-simple-001',
      namespace: 'kubeflow-test-infra',
      image: '',
    },
  },
}