{
  // TODO(https://github.com/ksonnet/ksonnet/issues/222): Taking namespace as an argument is a work around for the fact that ksonnet
  // doesn't support automatically piping in the namespace from the environment to prototypes.

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
  parts(namespace, name, overrides={}):: {
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
      local image = "gcr.io/kubeflow-ci/test-worker";
      // The name of the NFS volume claim to use for test files.
      // local nfsVolumeClaim = "kubeflow-testing";
      local nfsVolumeClaim = "nfs-external";
      // The name to use for the volume to use to contain test data.
      local dataVolume = "kubeflow-test-volume";
      local versionTag = if params.versionTag != null then
        params.versionTag
      else name;
      local tfJobImage = params.registry + "/tf_operator:" + versionTag;

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
      local chart = srcDir + "/bin/tf-job-operator-chart-0.2.1-" + versionTag + ".tgz";
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
              steps: [
                [{
                  name: "checkout",
                  template: "checkout",
                }],
                [
                  {
                    name: "build",
                    template: "build",
                  },
                  {
                    name: "create-pr-symlink",
                    template: "create-pr-symlink",
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
                [  // Setup cluster needs to run after build because we depend on the chart
                  // created by the build statement.
                  {
                    name: "setup-cluster",
                    template: "setup-cluster",
                  },
                ],
                [
                  {
                    name: "run-tests",
                    template: "run-tests",
                  },
                  {
                    name: "run-gpu-tests",
                    template: "run-gpu-tests",
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
                image: image,
                volumeMounts: [
                  {
                    name: dataVolume,
                    mountPath: mountPath,
                  },
                ],
              },
            },  // checkout
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("build", [
              "python",
              "-m",
              "py.release",
              "build",
              "--src_dir=" + srcDir,
              "--registry=" + params.registry,
              "--project=" + project,
              "--version_tag=" + versionTag,
            ]),  // build
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("py-test", [
              "python",
              "-m",
              "kubeflow.testing.test_py_checks",
              "--artifacts_dir=" + artifactsDir,
              "--src_dir=" + srcDir,
            ]),  // py test
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("py-lint", [
              "python",
              "-m",
              "kubeflow.testing.test_py_lint",
              "--artifacts_dir=" + artifactsDir,
              "--src_dir=" + srcDir,
            ]),  // py lint
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("setup-cluster", [
              "python",
              "-m",
              "py.deploy",
              "setup",
              "--cluster=" + cluster,
              "--zone=" + zone,
              "--project=" + project,
              "--namespace=" + deployNamespace,
              "--image=" + tfJobImage,
              "--accelerator=nvidia-tesla-k80=1",
              "--test_app_dir=" + srcDir + "/test/test-app",
              "--junit_path=" + artifactsDir + "/junit_setupcluster.xml",
            ]),  // setup cluster
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("run-tests", [
              "python",
              "-m",
              "py.test_runner",
              "test",
              "--cluster=" + cluster,
              "--zone=" + zone,
              "--project=" + project,
              "--app_dir=" + srcDir + "/test/workflows",
              "--component=simple_tfjob",
              "--params=name=simple-tfjob,namespace=default",
              "--junit_path=" + artifactsDir + "/junit_e2e.xml",
            ]),  // run tests
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("run-gpu-tests", [
              "python",
              "-m",
              "py.test_runner",
              "test",
              "--cluster=" + cluster,
              "--zone=" + zone,
              "--project=" + project,
              "--app_dir=" + srcDir + "/test/workflows",
              "--component=gpu_tfjob",
              "--params=name=gpu-tfjob,namespace=default",
              "--junit_path=" + artifactsDir + "/junit_gpu-tests.xml",
            ]),  // run gpu_tests
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("create-pr-symlink", [
              "python",
              "-m",
              "kubeflow.testing.prow_artifacts",
              "--artifacts_dir=" + outputDir,
              "create_pr_symlink",
              "--bucket=" + bucket,
            ]),  // create-pr-symlink
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("teardown-cluster", [
              "python",
              "-m",
              "py.deploy",
              "teardown",
              "--cluster=" + cluster,
              "--zone=" + zone,
              "--project=" + project,
              "--junit_path=" + artifactsDir + "/junit_teardown.xml",
            ]),  // setup cluster
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("copy-artifacts", [
              "python",
              "-m",
              "kubeflow.testing.prow_artifacts",
              "--artifacts_dir=" + outputDir,
              "copy_artifacts",
              "--bucket=" + bucket,
            ]),  // copy-artifacts
          ],  // templates
        },
      },  // e2e
  },  // parts
}
