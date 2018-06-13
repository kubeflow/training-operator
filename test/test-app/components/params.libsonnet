{
  global: {
    // User-defined global parameters; accessible to all component and environments, Ex:
    // replicas: 4,
  },
  components: {
    // Component-level parameters, defined initially from 'ks prototype use ...'
    // Each object below should correspond to a component in the components/ directory
    core: {
      cloud: "null",
      disks: "null",
      jupyterHubAuthenticator: "null",
      jupyterHubImage: "gcr.io/kubeflow/jupyterhub-k8s:v20180531-3bb991b1",
      jupyterHubServiceType: "ClusterIP",
      jupyterNotebookPVCMount: "null",
      jupyterNotebookRegistry: "gcr.io",
      jupyterNotebookRepoName: "kubeflow-images-public",
      name: "core",
      namespace: "null",
      reportUsage: "false",
      tfAmbassadorImage: "quay.io/datawire/ambassador:0.30.1",
      tfAmbassadorServiceType: "ClusterIP",
      tfDefaultImage: "null",
      tfJobImage: "gcr.io/kubeflow-images-public/tf_operator:v20180522-77375baf",
      tfJobUiServiceType: "ClusterIP",
      tfJobVersion: "v1alpha1",
      tfStatsdImage: "quay.io/datawire/statsd:0.30.1",
      usageId: "unknown_cluster",
    },
  },
}
