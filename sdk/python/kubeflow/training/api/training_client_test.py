import multiprocessing
from typing import Optional
from unittest.mock import Mock, patch

import pytest
from kubeflow.training import (KubeflowOrgV1PyTorchJob,
                               KubeflowOrgV1PyTorchJobSpec,
                               KubeflowOrgV1ReplicaSpec,
                               KubeflowOrgV1RunPolicy,
                               KubeflowOrgV1SchedulingPolicy, TrainingClient,
                               constants)
from kubernetes.client import (V1Container, V1ObjectMeta, V1PodSpec,
                               V1PodTemplateSpec, V1ResourceRequirements)

LIST_RESPONSE = [{"metadata": {"name": "Dummy V1PodList"}}]
TEST_NAME = "test"


def create_namespaced_custom_object_response(*args, **kwargs):
    if args[2] == "timeout":
        raise multiprocessing.TimeoutError()
    elif args[2] == "runtime":
        raise RuntimeError()


def list_namespaced_pod_response(*args, **kwargs):
    class MockResponse:
        def get(self, timeout):
            # Simulate a response from the Kubernetes API, and pass timeout for verification
            LIST_RESPONSE[0]["timeout"] = timeout
            if args[0] == "timeout":
                raise multiprocessing.TimeoutError()
            if args[0] == "runtime":
                raise Exception()
            return Mock(items=LIST_RESPONSE)

    return MockResponse()


def generate_container() -> V1Container:
    return V1Container(
        name="pytorch",
        image="gcr.io/kubeflow-ci/pytorch-dist-mnist-test:v1.0",
        args=["--backend", "gloo"],
        resources=V1ResourceRequirements(limits={"memory": "1Gi", "cpu": "0.4"}),
    )


def generate_pytorchjob(
    job_namespace: str,
    master: KubeflowOrgV1ReplicaSpec,
    worker: KubeflowOrgV1ReplicaSpec,
    scheduling_policy: Optional[KubeflowOrgV1SchedulingPolicy] = None,
) -> KubeflowOrgV1PyTorchJob:
    return KubeflowOrgV1PyTorchJob(
        api_version=constants.API_VERSION,
        kind=constants.PYTORCHJOB_KIND,
        metadata=V1ObjectMeta(name="pytorchjob-mnist-ci-test", namespace=job_namespace),
        spec=KubeflowOrgV1PyTorchJobSpec(
            run_policy=KubeflowOrgV1RunPolicy(
                clean_pod_policy="None",
                scheduling_policy=scheduling_policy,
            ),
            pytorch_replica_specs={"Master": master, "Worker": worker},
        ),
    )


def create_job():
    job_namespace = TEST_NAME
    container = generate_container()
    master = KubeflowOrgV1ReplicaSpec(
        replicas=1,
        restart_policy="OnFailure",
        template=V1PodTemplateSpec(
            metadata=V1ObjectMeta(
                annotations={constants.ISTIO_SIDECAR_INJECTION: "false"}
            ),
            spec=V1PodSpec(containers=[container]),
        ),
    )

    worker = KubeflowOrgV1ReplicaSpec(
        replicas=1,
        restart_policy="OnFailure",
        template=V1PodTemplateSpec(
            metadata=V1ObjectMeta(
                annotations={constants.ISTIO_SIDECAR_INJECTION: "false"}
            ),
            spec=V1PodSpec(containers=[container]),
        ),
    )
    pytorchjob = generate_pytorchjob(job_namespace, master, worker)
    return pytorchjob


class DummyJobClass:
    def __init__(self, kind) -> None:
        self.kind = kind


test_data_create_job = [
    (
        "invalid extra parameter",
        {"job": create_job(), "namespace": TEST_NAME, "base_image": "test_image"},
        ValueError,
    ),
    ("invalid job kind", {"job_kind": "invalid_job_kind"}, ValueError),
    (
        "job name missing ",
        {"train_func": lambda: "test train function"},
        ValueError,
    ),
    ("job name missing", {"base_image": "test_image"}, ValueError),
    (
        "uncallable train function",
        {"name": "test job", "train_func": "uncallable train function"},
        ValueError,
    ),
    (
        "invalid TFJob replica",
        {
            "name": "test job",
            "train_func": lambda: "test train function",
            "job_kind": constants.TFJOB_KIND,
        },
        ValueError,
    ),
    (
        "invalid PyTorchJob replica",
        {
            "name": "test job",
            "train_func": lambda: "test train function",
            "job_kind": constants.PYTORCHJOB_KIND,
        },
        ValueError,
    ),
    (
        "paddle job can't be created using function",
        {
            "name": "test job",
            "train_func": lambda: "test train function",
            "job_kind": constants.PADDLEJOB_KIND,
        },
        ValueError,
    ),
    (
        "invalid job object",
        {"job": DummyJobClass(constants.TFJOB_KIND)},
        ValueError,
    ),
    (
        "create_namespaced_custom_object timeout error",
        {"job": create_job(), "namespace": "timeout"},
        TimeoutError,
    ),
    (
        "create_namespaced_custom_object runtime error",
        {"job": create_job(), "namespace": "runtime"},
        RuntimeError,
    ),
    (
        "valid flow",
        {"job": create_job(), "namespace": TEST_NAME},
        "success",
    ),
    (
        "valid flow to create job from func",
        {
            "name": "test-job",
            "namespace": TEST_NAME,
            "train_func": lambda: print("Test Training Function"),
            "base_image": "docker.io/test-training",
            "num_workers": 3,
            "packages_to_install": ["boto3==1.34.14"],
            "pip_index_url": "https://pypi.custom.com/simple",
        },
        "success",
    ),
    (
        "valid flow to create job using image",
        {
            "name": "test-job",
            "namespace": TEST_NAME,
            "base_image": "docker.io/test-training",
            "num_workers": 2,
        },
        "success",
    ),
]

test_data_get_job_pods = [
    (
        "valid flow with default namespace and default timeout",
        {
            "name": TEST_NAME,
        },
        f"{constants.JOB_NAME_LABEL}={TEST_NAME}",
        LIST_RESPONSE,
    ),
    (
        "invalid replica_type",
        {"name": TEST_NAME, "replica_type": "invalid_replica_type"},
        "Label not relevant",
        ValueError,
    ),
    (
        "invalid replica_type (uppercase)",
        {"name": TEST_NAME, "replica_type": constants.REPLICA_TYPE_WORKER},
        "Label not relevant",
        ValueError,
    ),
    (
        "valid flow with specific timeout, replica_index, replica_type and master role",
        {
            "name": TEST_NAME,
            "namespace": "test_namespace",
            "timeout": 60,
            "is_master": True,
            "replica_type": constants.REPLICA_TYPE_MASTER.lower(),
            "replica_index": 0,
        },
        f"{constants.JOB_NAME_LABEL}={TEST_NAME},{constants.JOB_ROLE_LABEL}={constants.JOB_ROLE_MASTER}"
        f",{constants.REPLICA_TYPE_LABEL}={constants.REPLICA_TYPE_MASTER.lower()},{constants.REPLICA_INDEX_LABEL}=0",
        LIST_RESPONSE,
    ),
    (
        "invalid flow with TimeoutError",
        {
            "name": TEST_NAME,
            "namespace": "timeout",
        },
        "Label not relevant",
        TimeoutError,
    ),
    (
        "invalid flow with RuntimeError",
        {
            "name": TEST_NAME,
            "namespace": "runtime",
        },
        "Label not relevant",
        RuntimeError,
    ),
]


@pytest.fixture
def training_client():
    with patch(
        "kubernetes.client.CustomObjectsApi",
        return_value=Mock(
            create_namespaced_custom_object=Mock(
                side_effect=create_namespaced_custom_object_response
            )
        ),
    ), patch(
        "kubernetes.client.CoreV1Api",
        return_value=Mock(
            list_namespaced_pod=Mock(side_effect=list_namespaced_pod_response)
        ),
    ), patch(
        "kubernetes.config.load_kube_config", return_value=Mock()
    ):
        client = TrainingClient(job_kind=constants.PYTORCHJOB_KIND)
        yield client


@pytest.mark.parametrize("test_name,kwargs,expected_output", test_data_create_job)
def test_create_job(training_client, test_name, kwargs, expected_output):
    """
    test create_job function of training client
    """
    print("Executing test:", test_name)
    try:
        training_client.create_job(**kwargs)
        assert expected_output == "success"
    except Exception as e:
        assert type(e) is expected_output
    print("test execution complete")


@pytest.mark.parametrize(
    "test_name,kwargs,expected_label_selector,expected_output",
    test_data_get_job_pods,
)
def test_get_job_pods(
    training_client, test_name, kwargs, expected_label_selector, expected_output
):
    """
    test get_job_pods function of training client
    """
    print("Executing test:", test_name)
    try:
        out = training_client.get_job_pods(**kwargs)
        # Verify that list_namespaced_pod called with specified arguments
        training_client.core_api.list_namespaced_pod.assert_called_with(
            kwargs.get("namespace", constants.DEFAULT_NAMESPACE),
            label_selector=expected_label_selector,
            async_req=True,
        )
        assert out[0].pop("timeout") == kwargs.get("timeout", constants.DEFAULT_TIMEOUT)
        assert out == expected_output
    except Exception as e:
        assert type(e) is expected_output
    print("test execution complete")
