import multiprocessing
import pytest
from unittest.mock import patch, Mock

from typing import Optional
from kubeflow.training import TrainingClient
from kubeflow.training import KubeflowOrgV1ReplicaSpec
from kubeflow.training import KubeflowOrgV1PyTorchJob
from kubeflow.training import KubeflowOrgV1PyTorchJobSpec
from kubeflow.training import KubeflowOrgV1RunPolicy
from kubeflow.training import KubeflowOrgV1SchedulingPolicy
from kubeflow.training import constants

from kubernetes.client import V1PodTemplateSpec
from kubernetes.client import V1ObjectMeta
from kubernetes.client import V1PodSpec
from kubernetes.client import V1Container
from kubernetes.client import V1ResourceRequirements


def create_namespaced_custom_object_response(*args, **kwargs):
    if args[2] == "timeout":
        raise multiprocessing.TimeoutError()
    elif args[2] == "runtime":
        raise RuntimeError()


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
    job_namespace = "test"
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


test_data = [
    (
        "invalid extra parameter",
        {"job": create_job(), "namespace": "test", "base_image": "test_image"},
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
        {"job": create_job(), "namespace": "test"},
        "success",
    ),
    (
        "valid flow to create job from func",
        {
            "name": "test-job",
            "namespace": "test",
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
            "namespace": "test",
            "base_image": "docker.io/test-training",
            "num_workers": 2,
        },
        "success",
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
    ), patch("kubernetes.client.CoreV1Api", return_value=Mock()), patch(
        "kubernetes.config.load_kube_config", return_value=Mock()
    ):
        client = TrainingClient(job_kind=constants.PYTORCHJOB_KIND)
        yield client


@pytest.mark.parametrize("test_name,kwargs,expected_output", test_data)
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
