import pytest
from kubeflow.training import HuggingFaceDatasetConfig, HuggingFaceModelInputConfig


@pytest.mark.parametrize(
    "storage_uri, access_token, expected_storage_uri, expected_access_token",
    [
        ("hf://dataset/path", None, "hf://dataset/path", None),
        ("hf://dataset/path", "dummy_token", "hf://dataset/path", "dummy_token"),
    ],
)
def test_huggingface_dataset_config_creation(
    storage_uri, access_token, expected_storage_uri, expected_access_token
):
    """Test HuggingFaceDatasetConfig creation with different parameters"""
    config = HuggingFaceDatasetConfig(
        storage_uri=storage_uri, access_token=access_token
    )
    assert config.storage_uri == expected_storage_uri
    assert config.access_token == expected_access_token


@pytest.mark.parametrize(
    "storage_uri, access_token, expected_storage_uri, expected_access_token",
    [
        ("hf://model/path", None, "hf://model/path", None),
        ("hf://model/path", "dummy_token", "hf://model/path", "dummy_token"),
    ],
)
def test_huggingface_model_config_creation(
    storage_uri, access_token, expected_storage_uri, expected_access_token
):
    """Test HuggingFaceModelInputConfig creation with different parameters"""
    config = HuggingFaceModelInputConfig(
        storage_uri=storage_uri, access_token=access_token
    )
    assert config.storage_uri == expected_storage_uri
    assert config.access_token == expected_access_token
