local params = import "../../components/params.libsonnet";
params + {
  components +: {
    // Insert component parameter overrides here. Ex:
    // guestbook +: {
    //   name: "guestbook-dev",
    //   replicas: params.global.replicas,
    // },
    workflows +: {
      gcpCredentialsSecretName: "gcp-credentials",
      name: "jlewi-tf-operator-release-403-6ed0",
      namespace: "kubeflow-releasing",
      project: "kubeflow-releasing",
      prow_env: "JOB_NAME=tf-operator-release,JOB_TYPE=presubmit,PULL_NUMBER=403,REPO_NAME=tf-operator,REPO_OWNER=kubeflow,BUILD_NUMBER=6ed0",
      registry: "gcr.io/kubeflow-images-staging",
      versionTag: "v20180224-403",
      zone: "us-central1-a",
    },
  },
}
