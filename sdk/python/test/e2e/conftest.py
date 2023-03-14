import os
import pytest

from kubeflow.training.utils.utils import is_running_in_k8s
from kubeflow.training import TrainingClient


def pytest_addoption(parser):
    parser.addoption("--namespace", action="store", default="default")


@pytest.fixture
def job_namespace(request):
    return request.config.getoption("--namespace")
