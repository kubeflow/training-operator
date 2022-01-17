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
      name: 'training-operator-release-d746bde9-kunming',
      namespace: 'kubeflow-releasing',
      project: 'kubeflow-releasing',
      prow_env: 'JOB_NAME=training-operator-release,JOB_TYPE=training-operator-release,REPO_NAME=training-operator,REPO_OWNER=kubeflow,BUILD_NUMBER=204B,PULL_BASE_SHA=d746bde9',
      registry: 'gcr.io/kubeflow-images-public',
      versionTag: 'kubeflow-training-operator-postsubmit-v2-70cafb1-271-1911',
      zone: 'us-central1-a',
    },
  },
}
