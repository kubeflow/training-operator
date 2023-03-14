import os
import pytest

from kubeflow.training.utils.utils import is_running_in_k8s
from kubeflow.training import TrainingClient


def pytest_addoption(parser):
    parser.addoption("--namespace", action="store", default="default")


@pytest.fixture
def job_namespace(request):
    return request.config.getoption("--namespace")

@pytest.fixture
def training_client():
    if is_running_in_k8s():
        return TrainingClient()
    else:
        return TrainingClient(config_file=os.getenv("KUBECONFIG", "~/.kube/config"))