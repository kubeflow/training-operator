from pkg.initializer_v2.dataset.config import HuggingFaceDatasetConfig


def test_huggingface_dataset_config_creation():
    """Test HuggingFaceModelInputConfig creation with different parameters"""
    # Test with required parameters only
    config = HuggingFaceDatasetConfig(storage_uri="hf://dataset/path")
    assert config.storage_uri == "hf://dataset/path"
    assert config.access_token is None

    # Test with all parameters
    config = HuggingFaceDatasetConfig(
        storage_uri="hf://dataset/path", access_token="dummy_token"
    )
    assert config.storage_uri == "hf://dataset/path"
    assert config.access_token == "dummy_token"
