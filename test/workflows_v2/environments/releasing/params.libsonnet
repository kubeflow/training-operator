local params = import '../../components/params.libsonnet';

params + {
  components+: {
    // Insert component parameter overrides here. Ex:
    // guestbook +: {
    // name: "guestbook-dev",
    // replicas: params.global.replicas,
    // },
    workflows+: {
      bucket: 'kubeflow-releasing-artifacts',
      gcpCredentialsSecretName: 'gcp-credentials',
      name: 'tf-operator-release-d746bde9-kunming',
      namespace: 'kubeflow-releasing',
      project: 'kubeflow-releasing',
      prow_env: 'JOB_NAME=tf-operator-release,JOB_TYPE=tf-operator-release,REPO_NAME=tf-operator,REPO_OWNER=kubeflow,BUILD_NUMBER=204B,PULL_BASE_SHA=d746bde9',
      registry: 'gcr.io/kubeflow-images-public',
      versionTag: 'kubeflow-tf-operator-postsubmit-v2-70cafb1-271-1911',
      zone: 'us-central1-a',
    },
  },
}
