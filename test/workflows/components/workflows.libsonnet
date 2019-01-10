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
    project:: "kubeflow-ci",
    zone:: "us-east1-d",
    // Default registry to use.
    registry:: "gcr.io/" + $.defaultParams.project,

    // The image tag to use.
    // Defaults to a value based on the name.
    versionTag:: null,

    // The name of the secret containing GCP credentials.
    gcpCredentialsSecretName:: "kubeflow-testing-credentials",
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
      // The directory containing the kubeflow/tf-operator repo
      local srcDir = srcRootDir + "/kubeflow/tf-operator";
      // The image should generally be overwritten in the prow_config.yaml file. This makes it easier
      // to ensure a consistent image is used for all workflows.
      local image = if std.objectHas(params, "testWorkerImage") && std.length(params.testWorkerImage) > 0 then
        params.testWorkerImage
      else
        "gcr.io/kubeflow-ci/test-worker";


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
      local tfJobImage = params.registry + "/tf_operator:" + versionTag;

      // The test server image to use.
      local testServerImage = "gcr.io/kubeflow-images-staging/tf-operator-test-server:v20180613-e06fc0bb-dirty-5ef291";

      // The namespace on the cluster we spin up to deploy into.
      local deployNamespace = "kubeflow";

      // The directory within the kubeflow_testing submodule containing
      // py scripts to use.
      local k8sPy = srcDir;
      local kubeflowPy = srcRootDir + "/kubeflow/testing/py";

      local project = params.project;
      // GKE cluster to use
      // We need to truncate the cluster to no more than 40 characters because
      // cluster names can be a max of 40 characters.
      // We expect the suffix of the cluster name to be unique salt.
      // We prepend a z because cluster name must start with an alphanumeric character
      // and if we cut the prefix we might end up starting with "-" or other invalid
      // character for first character.
      local cluster =
        if std.length(name) > 40 then
          "z" + std.substr(name, std.length(name) - 39, 39)
        else
          name;
      local zone = params.zone;
      {
        // Build an Argo template to execute a particular command.
        // step_name: Name for the template
        // command: List to pass as the container command.
        buildTemplate(step_name, command):: {
          name: step_name,
          container: {
            command: command,
            image: image,
            workingDir: srcDir,
            env: [
              {
                // Add the source directories to the python path.
                name: "PYTHONPATH",
                value: k8sPy + ":" + kubeflowPy,
              },
              {
                // Set the GOPATH
                name: "GOPATH",
                value: goDir,
              },
              {
                name: "GOOGLE_APPLICATION_CREDENTIALS",
                value: "/secret/gcp-credentials/key.json",
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
                // We use a directory in our NFS share to store our kube config.
                // This way we can configure it on a single step and reuse it on subsequent steps.
                name: "KUBECONFIG",
                value: kubeConfig,
              },
            ] + prow_env,
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
                name: "gcp-credentials",
                mountPath: "/secret/gcp-credentials",
              },
            ],
          },
        },  // buildTemplate

        buildTestTemplate(test_name, num_trials=1):: {
          t:: $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate(
            test_name, [
              "python",
              "-m",
              "py." + std.strReplace(test_name, "-", "_"),
              "--cluster=" + cluster,
              "--zone=" + zone,
              "--project=" + project,
              "--app_dir=" + srcDir + "/test/workflows",
              "--params=name=" + test_name + "-" + params.tfJobVersion + ",namespace=default",
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
              name: "gcp-credentials",
              secret: {
                secretName: params.gcpCredentialsSecretName,
              },
            },
            {
              name: dataVolume,
              persistentVolumeClaim: {
                claimName: nfsVolumeClaim,
              },
            },
          ],  // volumes
          // onExit specifies the template that should always run when the workflow completes.
          onExit: "exit-handler",
          templates: [
            {
              name: "e2e",
              dag: {
                // TODO(richardsliu): Consider templatizing the tasks further so that
                // the dependencies are autogenerated.
                tasks: [
                  {
                    name: "checkout",
                    template: "checkout",
                  },

                  {
                    name: "build",
                    template: "build",
                    dependencies: ["checkout"],
                  },
                  {
                    name: "create-pr-symlink",
                    template: "create-pr-symlink",
                    dependencies: ["checkout"],
                  },
                  {
                    name: "py-test",
                    template: "py-test",
                    dependencies: ["checkout"],
                  },
                  {
                    name: "py-lint",
                    template: "py-lint",
                    dependencies: ["checkout"],
                  },

                  {
                    name: "setup-cluster",
                    template: "setup-cluster",
                    dependencies: ["checkout"],
                  },

                  {
                    name: "setup-kubeflow",
                    template: "setup-kubeflow",
                    dependencies: ["setup-cluster", "build"],
                  },
                  {
                    name: "simple-tfjob-tests",
                    template: "simple-tfjob-tests",
                    dependencies: ["setup-kubeflow"],
                  },
                  {
                    name: "shutdown-policy-tests",
                    template: "shutdown-policy-tests",
                    dependencies: ["setup-kubeflow"],
                  },
                  {
                    name: "cleanpod-policy-tests",
                    template: "cleanpod-policy-tests",
                    dependencies: ["setup-kubeflow"],
                  },
                  {
                    name: "distributed-training-tests",
                    template: "distributed-training-tests",
                    dependencies: ["setup-kubeflow"],
                  },
                  {
                    name: "estimator-runconfig-tests",
                    template: "estimator-runconfig-tests",
                    dependencies: ["setup-kubeflow"],
                  },
                  {
                    name: "invalid-tfjob-tests",
                    template: "invalid-tfjob-tests",
                    dependencies: ["setup-kubeflow"],
                  },
                  {
                    name: "replica-restart-policy-tests",
                    template: "replica-restart-policy-tests",
                    dependencies: ["setup-kubeflow"],
                  },
                  {
                    name: "pod-names-validation-tests",
                    template: "pod-names-validation-tests",
                    dependencies: ["setup-kubeflow"],
                  },
                ],  //tasks
              },
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
                  // TODO(jlewi): Switch back to head once kubeflow/testing#271 is submitted.
                  value: "kubeflow/testing@HEAD:271",
                }],
                image: image,
                volumeMounts: [
                  {
                    name: dataVolume,
                    mountPath: mountPath,
                  },
                ],
              },
            },  // checkout
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("build", [
              "python",
              "-m",
              "py.release",
              "build",
              "--src_dir=" + srcDir,
              "--registry=" + params.registry,
              "--project=" + project,
              "--version_tag=" + versionTag,
            ]),  // build
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("py-test", [
              "python",
              "-m",
              "kubeflow.testing.test_py_checks",
              "--artifacts_dir=" + artifactsDir,
              "--src_dir=" + srcDir,
            ]),  // py test
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("py-lint", [
              "python",
              "-m",
              "kubeflow.testing.test_py_lint",
              "--artifacts_dir=" + artifactsDir,
              "--src_dir=" + srcDir,
            ]),  // py lint
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("setup-cluster", [
              "python",
              "-m",
              "py.deploy",
              "setup_cluster",
              "--cluster=" + cluster,
              "--zone=" + zone,
              "--project=" + project,
              "--namespace=" + deployNamespace,
              "--accelerator=nvidia-tesla-k80=1",
              "--junit_path=" + artifactsDir + "/junit_setupcluster.xml",
            ]),  // setup cluster
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("setup-kubeflow", [
              "python",
              "-m",
              "py.deploy",
              "setup_kubeflow",
              "--cluster=" + cluster,
              "--zone=" + zone,
              "--project=" + project,
              "--namespace=" + deployNamespace,
              "--test_app_dir=" + srcDir + "/test/test-app",
              "--image=" + tfJobImage,
              "--tf_job_version=" + params.tfJobVersion,
              "--junit_path=" + artifactsDir + "/junit_setupkubeflow.xml",
            ]),  // setup cluster
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
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("create-pr-symlink", [
              "python",
              "-m",
              "kubeflow.testing.prow_artifacts",
              "--artifacts_dir=" + outputDir,
              "create_pr_symlink",
              "--bucket=" + bucket,
            ]),  // create-pr-symlink
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("teardown-cluster", [
              "python",
              "-m",
              "py.deploy",
              "teardown",
              "--cluster=" + cluster,
              "--zone=" + zone,
              "--project=" + project,
              "--junit_path=" + artifactsDir + "/junit_teardown.xml",
            ]),  // setup cluster
            $.parts(namespace, name, overrides).e2e(prow_env, bucket).buildTemplate("copy-artifacts", [
              "python",
              "-m",
              "kubeflow.testing.prow_artifacts",
              "--artifacts_dir=" + outputDir,
              "copy_artifacts",
              "--bucket=" + bucket,
              // Suffix will be used to give a unique file name to all XML files.
              // This will prevent different versions of the workflow from clobbering each other
              // when uploading the results to gubernator.
              "--suffix=" + params.tfJobVersion,
            ]),  // copy-artifacts
          ],  // templates
        },
      },  // e2e
  },  // parts
}
