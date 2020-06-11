{
  all(params):: [

                  $.parts(params.namespace).serviceAccount,
                  $.parts(params.namespace).operatorRole,
                  $.parts(params.namespace).operatorRoleBinding,
                  $.parts(params.namespace).crd,
                  $.parts(params.namespace).tfJobDeploy(params.tfJobImage),
                ],

  parts(namespace):: {
    crd: {
      apiVersion: "apiextensions.k8s.io/v1beta1",
      kind: "CustomResourceDefinition",
      metadata: {
        name: "tfjobs.kubeflow.org",
      },
      spec: {
        group: "kubeflow.org",
        names: {
          kind: "TFJob",
          singular: "tfjob",
          plural: "tfjobs",
        },
        subresources: {
	  status: {},
        },
        validation: {
          openAPIV3Schema: {
            properties: {
              spec: {
                properties: {
                  tfReplicaSpecs: {
                    properties: {
                      // The validation works when the configuration contains
                      // `Worker`, `PS` or `Chief`. Otherise it will not be validated.
                      Worker: {
                        properties: {
                          // We do not validate pod template because of
                          // https://github.com/kubernetes/kubernetes/issues/54579
                          replicas: {
                            type: "integer",
                            minimum: 1,
                          },
                        },
                      },
                      PS: {
                        properties: {
                          replicas: {
                            type: "integer",
                            minimum: 1,
                          },
                        },
                      },
                      Chief: {
                        properties: {
                          replicas: {
                            type: "integer",
                            minimum: 1,
                            maximum: 1,
                          },
                        },
                      },
                    },
                  },
                },
              },
            },
          },
        },
        versions: [
          {
            name: "v1beta2",
            served: true,
            storage: true,
          },
          {
            name: "v1",
            served: true,
            storage: false,
          },
        ],
      },
    }, // crd

    tfJobDeploy(image): {
      apiVersion: "extensions/v1beta1",
      kind: "Deployment",
      metadata: {
        name: "tf-job-operator",
        namespace: namespace,
      },
      spec: {
        replicas: 1,
        template: {
          metadata: {
            labels: {
              name: "tf-job-operator",
            },
          },
          spec: {
            containers: [
              {
                command: [
                  "/opt/tf-operator.v1",
                ],
                env: [
                  {
                    name: "MY_POD_NAMESPACE",
                    valueFrom: {
                      fieldRef: {
                        fieldPath: "metadata.namespace",
                      },
                    },
                  },
                  {
                    name: "MY_POD_NAME",
                    valueFrom: {
                      fieldRef: {
                        fieldPath: "metadata.name",
                      },
                    },
                  },
                ],
                image: image,
                name: "tf-job-operator",
              },
            ],
            serviceAccountName: "tf-job-operator",
          },
        },
      },
    },  // tfJobDeploy

    serviceAccount: {
      apiVersion: "v1",
      kind: "ServiceAccount",
      metadata: {
        labels: {
          app: "tf-job-operator",
        },
        name: "tf-job-operator",
        namespace: namespace,
      },
    },

    operatorRole: {
      apiVersion: "rbac.authorization.k8s.io/v1beta1",
      kind: "ClusterRole",
      metadata: {
        labels: {
          app: "tf-job-operator",
        },
        name: "tf-job-operator",
      },
      rules: [
        {
          apiGroups: [
            "tensorflow.org",
            "kubeflow.org",
          ],
          resources: [
            "tfjobs",
            "tfjobs/status",
          ],
          verbs: [
            "*",
          ],
        },
        {
          apiGroups: [
            "",
          ],
          resources: [
            "pods",
            "services",
            "endpoints",
            "events",
          ],
          verbs: [
            "*",
          ],
        },
        {
          apiGroups: [
            "apps",
            "extensions",
          ],
          resources: [
            "deployments",
          ],
          verbs: [
            "*",
          ],
        },
      ],
    },  // operator-role

    operatorRoleBinding:: {
      apiVersion: "rbac.authorization.k8s.io/v1beta1",
      kind: "ClusterRoleBinding",
      metadata: {
        labels: {
          app: "tf-job-operator",
        },
        name: "tf-job-operator",
      },
      roleRef: {
        apiGroup: "rbac.authorization.k8s.io",
        kind: "ClusterRole",
        name: "tf-job-operator",
      },
      subjects: [
        {
          kind: "ServiceAccount",
          name: "tf-job-operator",
          namespace: namespace,
        },
      ],
    },  // operator-role binding
  },
}
