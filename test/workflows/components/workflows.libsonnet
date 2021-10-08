{
  // TODO(https://github.com/ksonnet/ksonnet/issues/222): Taking namespace as an argument is a work around for the fact that ksonnet
  // doesn't support automatically piping in the namespace from the environment to prototypes.

  // TODO(jlewi): We should refactor the test_runner step so that we don't have to get K8s credentials
  // on each individual step. Instead we should do what we do in our kubeflow/kubeflow tests
  // and have a separate step that modifies .kubeconfig and then on subsequent steps
  // just set the environment variable KUBE_CONFIG.

  // convert a list of two items into a map representing an environment variable
  // TODO(jlewi): Should we move this into kubeflow/core/util.libsonnet
  listToMap:: function(v)
    {
      name: v[0],
      value: v[1],
    },

  // Function to turn comma separated list of prow environment variables into a dictionary.
  parseEnv:: function(v)
    local pieces = std.split(v, ",");
    if v != "" && std.length(pieces) > 0 then
      std.map(
        function(i) $.listToMap(std.split(i, "=")),
        std.split(v, ",")
      )
    else [],

  // default parameters.
  defaultParams:: {
    // Default registry to use.
    registry:: "809251082950.dkr.ecr.us-west-2.amazonaws.com/training-operator",

    // The image tag to use.
    // Defaults to a value based on the name.
    versionTag:: null,
  },

  // overrides is a dictionary of parameters to provide in addition to defaults.
  parts(namespace, name, overrides):: {
    // Workflow to run the e2e test.
    e2e(prow_env, bucket):
      local params = $.defaultParams + overrides;
      // mountPath is the directory where the volume to store the test data
      // should be mounted.
      local mountPath = "/mnt/" + "test-data-volume";
      // testDir is the root directory for all data for a particular test run.
      local testDir = mountPath + "/" + name;
      // outputDir is the directory to sync to GCS to contain the output for this job.
      local outputDir = testDir + "/output";
      local artifactsDir = outputDir + "/artifacts";
      local goDir = testDir + "/go";
      // Source directory where all repos should be checked out
      local srcRootDir = testDir + "/src";
      // The directory containing the kubeflow/training-operator repo
      local srcDir = srcRootDir + "/kubeflow/training-operator";
      local testWorkerImage = "public.ecr.aws/j1r0q0g6/kubeflow-testing:latest";

      // value of KUBECONFIG environment variable. This should be  a full path.
      local kubeConfig = testDir + "/.kube/kubeconfig";

      // The name of the NFS volume claim to use for test files.
      // local nfsVolumeClaim = "kubeflow-testing";
      local nfsVolumeClaim = "nfs-external";
      // The name to use for the volume to use to contain test data.
      local dataVolume = "kubeflow-test-volume";
      local versionTag = if std.objectHas(params, "versionTag") && params.versionTag != "null" && std.length(params.versionTag) > 0 then
        params.versionTag
      else name;
      local tfJobImage = params.registry + "/training-operator:" + versionTag;

      // The test server image to use.
      // local testServerImage = "gcr.io/kubeflow-images-staging/training-operator-test-server:v20180613-e06fc0bb-dirty-5ef291";

      // The namespace on the cluster we spin up to deploy into.
      local deployNamespace = "kubeflow";

      // The directory within the kubeflow_testing submodule containing
      // py scripts to use.
      local k8sPy = srcDir;
      local kubeflowPyTesting = srcRootDir + "/kubeflow/testing/py";
      local kubeflowPyTFJob = srcRootDir + "/kubeflow/training-operator/py";
      local TFJobSDK = srcRootDir + "/kubeflow/training-operator/sdk/python";

      local project = params.project;
      
      // EKS cluster name, better to be meaningful to trace back to prow job
      // Maximum length of cluster name is 100. We set to 80 as maximum here and truncate
      local cluster =
        if std.length(name) > 80 then
          std.substr(name, std.length(name) - 79, 79)
        else
          name;
      local registry = params.registry;
      {
        // Build an Argo template to execute a particular command.
        // step_name: Name for the template
        // command: List to pass as the container command.
        buildTemplate(step_name, image, command, env_vars=[], volume_mounts=[]):: {
          name: step_name,
          activeDeadlineSeconds: 7200,
          container: {
            command: command,
            image: image,
            workingDir: srcDir,
            env: [
              {
                // Add the source directories to the python path.
                name: "PYTHONPATH",
                value: k8sPy + ":" + kubeflowPyTFJob + ":" +  kubeflowPyTesting + ":" + TFJobSDK,
              },
              {
                // Set the GOPATH
                name: "GOPATH",
                value: goDir,
              },
              {
                name: "CLUSTER_NAME",
                value: cluster,
              },
              {
                name: "ECR_REGISTRY",
                value: registry,
              },
              {
                name: "DEPLOY_NAMESPACE",
                value: deployNamespace,
              },
              {
                name: "GIT_TOKEN",
                valueFrom: {
                  secretKeyRef: {
                    name: "github-token",
                    key: "github_token",
                  },
                },
              },
              {
                name: "AWS_REGION",
                value: "us-west-2",
              },
              {
                // We use a directory in our NFS share to store our kube config.
                // This way we can configure it on a single step and reuse it on subsequent steps.
                name: "KUBECONFIG",
                value: kubeConfig,
              },
            ] + prow_env + env_vars,
            volumeMounts: [
              {
                name: dataVolume,
                mountPath: mountPath,
              },
              {
                name: "github-token",
                mountPath: "/secret/github-token",
              },
              {
                name: "aws-secret",
                mountPath: "/root/.aws/",
              },
            ] + volume_mounts,
          },
        },  // buildTemplate

        buildTestTemplate(test_name, num_trials=1):: {
          t:: $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate(
            test_name, testWorkerImage, [
              "python",
              "-m",
              "kubeflow.tf_operator." + std.strReplace(test_name, "-", "_"),
              "--app_dir=" + srcDir + "/test/workflows",
              "--params=name=" + test_name + "-" + params.tfJobVersion + ",namespace=kubeflow",
              "--tfjob_version=" + params.tfJobVersion,
              "--num_trials=" + num_trials,
              "--artifacts_path=" + artifactsDir,
            ]),
        }.t,  // buildTestTemplate

        apiVersion: "argoproj.io/v1alpha1",
        kind: "Workflow",
        metadata: {
          name: name,
          namespace: namespace,
        },
        // TODO(jlewi): Use OnExit to run cleanup steps.
        spec: {
          entrypoint: "e2e",
          volumes: [
            {
              name: "github-token",
              secret: {
                secretName: "github-token",
              },
            },
            {
              name: dataVolume,
              persistentVolumeClaim: {
                claimName: nfsVolumeClaim,
              },
            },
            {
              name: "docker-config",
              configMap: {
                name: "docker-config",
              },
            },
            {
              name: "aws-secret",
              secret: {
                secretName: "aws-secret",
              },
            },
          ],  // volumes
          // onExit specifies the template that should always run when the workflow completes.
          onExit: "exit-handler",
          templates: [
            {
              name: "e2e",
              steps: [
                [
                  {
                    name: "checkout",
                    template: "checkout",
                  },
                ],
                [
                  {
                    name: "build",
                    template: "build",
                  },
                  {
                    name: "copy-to-gopath",
                    template: "copy-to-gopath",
                  },
                  {
                    name: "py-test",
                    template: "py-test",
                  },
                  {
                    name: "py-lint",
                    template: "py-lint",
                  },
                ],
                [
                  {
                    name: "setup-cluster",
                    template: "setup-cluster",
                  },
                ],
                [
                  {
                    name: "setup-training-operator",
                    template: "setup-training-operator",
                  },
                ],
                [
                  {
                    name: "simple-tfjob-tests",
                    template: "simple-tfjob-tests",
                  },
                  {
                    name: "shutdown-policy-tests",
                    template: "shutdown-policy-tests",
                  },
                  {
                    name: "cleanpod-policy-tests",
                    template: "cleanpod-policy-tests",
                  },
                  {
                    name: "distributed-training-tests",
                    template: "distributed-training-tests",
                  },
                  {
                    name: "estimator-runconfig-tests",
                    template: "estimator-runconfig-tests",
                  },
                  {
                    name: "invalid-tfjob-tests",
                    template: "invalid-tfjob-tests",
                  },
                  {
                    name: "replica-restart-policy-tests",
                    template: "replica-restart-policy-tests",
                  },
                  {
                    name: "pod-names-validation-tests",
                    template: "pod-names-validation-tests",
                  },
                  {
                    name: "tfjob-sdk-tests",
                    template: "tfjob-sdk-tests",
                  },
                ],
              ],
            },
            {
              name: "exit-handler",
              steps: [
                [{
                  name: "teardown-cluster",
                  template: "teardown-cluster",
                }],
                [{
                  name: "copy-artifacts",
                  template: "copy-artifacts",
                }],
              ],
            },
            {
              name: "checkout",
              container: {
                command: [
                  "/usr/local/bin/checkout.sh",
                  srcRootDir,
                ],
                env: prow_env + [{
                  name: "EXTRA_REPOS",
                  value: "kubeflow/testing@HEAD",
                }],
                image: testWorkerImage,
                volumeMounts: [
                  {
                    name: dataVolume,
                    mountPath: mountPath,
                  },
                ],
              },
            },  // checkout
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("build", "gcr.io/kaniko-project/executor:v1.5.1", [
              #"scripts/build.sh",
              "/kaniko/executor",
              "--dockerfile=" + srcDir + "/build/images/training-operator/Dockerfile",
              "--context=dir://" + srcDir,
              "--destination=" + "$(ECR_REGISTRY):$(PULL_BASE_SHA)",
            ],
            # need to add volume mounts and extra env.
            volume_mounts=[
              {
                name: "docker-config",
                mountPath: "/kaniko/.docker/",
              },
            ]
            ),  // build
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("copy-to-gopath", testWorkerImage, [
              "scripts/copy-to-gopath.sh",
            ]),  // copy-to-gopath
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("py-test", testWorkerImage, [
              "python",
              "-m",
              "kubeflow.testing.test_py_checks",
              "--artifacts_dir=" + artifactsDir,
              "--src_dir=" + srcDir,
            ]),  // py test
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("py-lint", testWorkerImage, [
              "python",
              "-m",
              "kubeflow.testing.test_py_lint",
              "--artifacts_dir=" + artifactsDir,
              "--src_dir=" + srcDir,
            ]),  // py lint
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("setup-cluster", testWorkerImage, [
              "/usr/local/bin/create-eks-cluster.sh",
            ]),  // setup cluster
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("setup-training-operator", testWorkerImage, [
              "scripts/setup-training-operator.sh",
            ]),  // setup training-operator
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTestTemplate(
              "simple-tfjob-tests", 2),
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTestTemplate(
              "shutdown-policy-tests"),
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTestTemplate(
              "cleanpod-policy-tests"),
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTestTemplate(
              "estimator-runconfig-tests"),
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTestTemplate(
              "distributed-training-tests"),
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTestTemplate(
              "invalid-tfjob-tests"),
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTestTemplate(
              "replica-restart-policy-tests"),
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTestTemplate(
              "pod-names-validation-tests"),
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("create-pr-symlink", testWorkerImage, [
              "python",
              "-m",
              "kubeflow.testing.prow_artifacts",
              "--artifacts_dir=" + outputDir,
              "create_pr_symlink",
              "--bucket=" + bucket,
            ]),  // create-pr-symlink
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("teardown-cluster", testWorkerImage, [
              "/usr/local/bin/delete-eks-cluster.sh",
            ]),  // teardown cluster
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("copy-artifacts", testWorkerImage, [
              "python",
              "-m",
              "kubeflow.testing.cloudprovider.aws.prow_artifacts",
              "--artifacts_dir=" + outputDir,
              "copy_artifacts_to_s3",
              "--bucket=" + bucket,
            ]),  // copy-artifacts
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("tfjob-sdk-tests", testWorkerImage, [
              "/bin/sh",
              "-xc",
              "python3.8 -m pip install -r sdk/python/requirements.txt; pytest sdk/python/test --log-cli-level=info --log-cli-format='%(levelname)s|%(asctime)s|%(pathname)s|%(lineno)d| %(message)s' --junitxml=" + artifactsDir + "/junit_sdk-test.xml"
            ]), 
          ],  // templates
        },
      },  // e2e
  },  // parts
}
