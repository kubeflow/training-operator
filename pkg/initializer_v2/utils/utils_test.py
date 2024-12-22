import os

import pytest
from kubeflow.training import HuggingFaceDatasetConfig, HuggingFaceModelInputConfig

import pkg.initializer_v2.utils.utils as utils


@pytest.fixture
def mock_env_vars():
    """Fixture to set and clean up environment variables"""
    original_env = dict(os.environ)

    def _set_env_vars(**kwargs):
        for key, value in kwargs.items():
            if value is None:
                os.environ.pop(key, None)
            else:
                os.environ[key] = str(value)
        return os.environ

    yield _set_env_vars

    # Cleanup
    os.environ.clear()
    os.environ.update(original_env)


@pytest.mark.parametrize(
    "config_class,env_vars,expected",
    [
        (
            HuggingFaceModelInputConfig,
            {"STORAGE_URI": "hf://test", "ACCESS_TOKEN": "token"},
            {"storage_uri": "hf://test", "access_token": "token"},
        ),
        (
            HuggingFaceModelInputConfig,
            {"STORAGE_URI": "hf://test"},
            {"storage_uri": "hf://test", "access_token": None},
        ),
        (
            HuggingFaceDatasetConfig,
            {"STORAGE_URI": "hf://test", "ACCESS_TOKEN": "token"},
            {"storage_uri": "hf://test", "access_token": "token"},
        ),
        (
            HuggingFaceDatasetConfig,
            {"STORAGE_URI": "hf://test"},
            {"storage_uri": "hf://test", "access_token": None},
        ),
    ],
)
def test_get_config_from_env(mock_env_vars, config_class, env_vars, expected):
    mock_env_vars(**env_vars)
    result = utils.get_config_from_env(config_class)
    assert result == expected
