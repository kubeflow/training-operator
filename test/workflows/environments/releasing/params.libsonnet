local params = import "../../components/params.libsonnet";
params + {
  components +: {
    // Insert component parameter overrides here. Ex:
    // guestbook +: {
    //   name: "guestbook-dev",
    //   replicas: params.global.replicas,
    // },
    workflows +: {
      bucket: "kubeflow-releasing-artifacts",
      gcpCredentialsSecretName: "gcp-credentials",
      name: "jlewi-tf-operator-release-a7511ff-78e4",
      namespace: "kubeflow-releasing",
      project: "kubeflow-releasing",
      prow_env: "JOB_NAME=tf-operator-release,JOB_TYPE=tf-operator-release,REPO_NAME=tf-operator,REPO_OWNER=kubeflow,BUILD_NUMBER=78e4,PULL_BASE_SHA=a7511ff",
      registry: "gcr.io/kubeflow-images-public",
      versionTag: "v20180329-a7511ff",
      zone: "us-central1-a",
    },
  },
}
