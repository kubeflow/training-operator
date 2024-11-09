from pkg.initializer_v2.model.config import HuggingFaceModelInputConfig


def test_huggingface_model_config_creation():
    """Test HuggingFaceModelInputConfig creation with different parameters"""
    # Test with required parameters only
    config = HuggingFaceModelInputConfig(storage_uri="hf://model/path")
    assert config.storage_uri == "hf://model/path"
    assert config.access_token is None

    # Test with all parameters
    config = HuggingFaceModelInputConfig(
        storage_uri="hf://model/path", access_token="dummy_token"
    )
    assert config.storage_uri == "hf://model/path"
    assert config.access_token == "dummy_token"
