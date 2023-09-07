import pytest


def pytest_addoption(parser):
    parser.addoption("--namespace", action="store", default="default")


@pytest.fixture
def job_namespace(request):
    return request.config.getoption("--namespace")
