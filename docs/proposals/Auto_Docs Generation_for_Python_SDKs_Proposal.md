# Automated Documentation Generation for Kubeflow Python SDKs Proposal

## Links

- [Motivation](#motivation)
- [Goal](#goal)
- [Implementation](#implementation)
- [Challenges with Other Approaches](#challenges-with-other-approaches)
- [Existing Resources](#existing-resources)

## Motivation

Generate Docs for Katib and Training Operator SDKs #2081: https://github.com/kubeflow/katib/issues/2081

Kubeflow is a comprehensive platform for deploying and managing machine learning workflows on Kubernetes. The Training Operator and Katib SDKs provide Python-based interfaces to interact with various Kubeflow components. However, understanding these SDKs requires navigating through the source code to interpret API docstrings, which can be time-consuming and challenging.

## Goal

Develop an automated system to generate comprehensive and user-friendly documentation for the Kubeflow Python SDKs, specifically the Training Operator and Katib SDKs, by leveraging existing docstrings. The generated documentation will be formatted and integrated into the Kubeflow website, making it easily accessible to users.

## Implementation

We will leverage existing assets from the Kubeflow Training Operator and Katib SDKs to extract docstrings and generate documentation. This involves configuring Sphinx to extract the docstrings, convert them into standard documentation, host it on the Read the Docs platform, and integrate it into the Kubeflow website.

### Challenges with Other Approaches

1. **Hugo Integration**: While Hugo is an excellent tool for static site generation, integrating Sphinx-generated documentation into a Hugo-based site is not straightforward. It would require treating other repositories as submodules and manually managing the integration, which can be cumbersome and error-prone.
2. **Lack of Standardization**: Without a standardized approach, the documentation might lack consistency in formatting and structure, making it harder for users to navigate and understand.
3. **Maintenance Overhead**: Manually integrating and updating documentation with Hugo can lead to a higher maintenance burden. Any changes in the codebase would require corresponding updates in the documentation, which is better handled automatically with Sphinx and Read the Docs.

### Existing Resources

- [Katib Python API Client](https://github.com/kubeflow/katib/blob/master/sdk/python/v1beta1/kubeflow/katib/api/katib_client.py)
  
  Sample code from the Katib client:

    ```python
    def __init__(
        self,
        config_file: str = None,
        context: str = None,
        client_configuration: client.Configuration = None,
        namespace: str = utils.get_default_target_namespace(),
    ):
        """KatibClient constructor.

        Args:
            config_file: Path to the kube-config file. Defaults to ~/.kube/config.
            context: Set the active context. Defaults to current_context from the kube-config.
            client_configuration: Client configuration for cluster authentication.
                You have to provide valid configuration with Bearer token or
                with username and password.
                You can find an example here: https://github.com/kubernetes-client/python/blob/67f9c7a97081b4526470cad53576bc3b71fa6fcc/examples/remote_cluster.py#L31
            namespace: Target Kubernetes namespace. Can be overridden during method invocations.
        """
        self.in_cluster = False
        # If client configuration is not set, use kube-config to access Kubernetes APIs.
        if client_configuration is None:
            # Load kube-config or in-cluster config.
            if config_file or not utils.is_running_in_k8s():
                config.load_kube_config(config_file=config_file, context=context)
            else:
                config.load_incluster_config()
                self.in_cluster = True

        k8s_client = client.ApiClient(client_configuration)
        self.custom_api = client.CustomObjectsApi(k8s_client)
        self.api_client = ApiClient()
        self.namespace = namespace
    ```
After extracting the docstrings from the code, we will automate the process of converting them into comprehensive documentation using Sphinx. This will involve setting up a CI/CD pipeline to continuously update the documentation

- [Training Operator API Client](https://github.com/kubeflow/training-operator/blob/master/sdk/python/kubeflow/training/api/training_client.py)
  
  Sample code from the Training Operator client:

    ```python
    def __init__(
        self,
        config_file: Optional[str] = None,
        context: Optional[str] = None,
        client_configuration: Optional[client.Configuration] = None,
        namespace: str = utils.get_default_target_namespace(),
        job_kind: str = constants.PYTORCHJOB_KIND,
    ):
        """TrainingClient constructor. Configure logging in your application
            as follows to see detailed information from the TrainingClient APIs:
            .. code-block:: python
                import logging
                logging.basicConfig()
                log = logging.getLogger("kubeflow.training.api.training_client")
                log.setLevel(logging.DEBUG)

        Args:
            config_file: Path to the kube-config file. Defaults to ~/.kube/config.
            context: Set the active context. Defaults to current_context from the kube-config.
            client_configuration: Client configuration for cluster authentication.
                You have to provide valid configuration with Bearer token or
                with username and password. You can find an example here:
                https://github.com/kubernetes-client/python/blob/67f9c7a97081b4526470cad53576bc3b71fa6fcc/examples/remote_cluster.py#L31
            namespace: Target Kubernetes namespace. By default it takes namespace
                from `/var/run/secrets/kubernetes.io/serviceaccount/namespace` location
                or set as `default`. Namespace can be overridden during method invocations.
            job_kind: Target Training Job kind (e.g. `TFJob`, `PyTorchJob`, `MPIJob`).
                Job kind can be overridden during method invocations.
                The default Job kind is `PyTorchJob`.

        Raises:
            ValueError: Job kind is invalid.
        """

        # If client configuration is not set, use kube-config to access Kubernetes APIs.
        if client_configuration is None:
            # Load kube-config or in-cluster config.
            if config_file or not utils.is_running_in_k8s():
                config.load_kube_config(config_file=config_file, context=context)
            else:
                config.load_incluster_config()

        k8s_client = client.ApiClient(client_configuration)
        self.custom_api = client.CustomObjectsApi(k8s_client)
        self.core_api = client.CoreV1Api(k8s_client)
        self.api_client = ApiClient()

        self.namespace = namespace
        if job_kind not in constants.JOB_PARAMETERS:
            raise ValueError(
                f"Job kind must be one of these: {list(constants.JOB_PARAMETERS.keys())}"
            )
        self.job_kind = job_kind
    ```
After extracting the docstrings from the code, we will automate the process of converting them into comprehensive documentation using Sphinx. This will involve setting up a CI/CD pipeline to continuously update the documentation

## Other Functionalities

1. **Continuous Integration**: Integrate the documentation generation process with a CI/CD pipeline to ensure documentation is always up-to-date.
2. **Cross-Referencing**: Implement cross-referencing in the documentation to link related sections and enhance usability.
