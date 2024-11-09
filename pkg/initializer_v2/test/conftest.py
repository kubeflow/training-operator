import os
import sys

import pytest

# Add project root to path if needed
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "../../..")))


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


@pytest.fixture
def huggingface_model_instance():
    """Fixture for HuggingFace Model instance"""
    from pkg.initializer_v2.model.huggingface import HuggingFace

    return HuggingFace()


@pytest.fixture
def huggingface_dataset_instance():
    """Fixture for HuggingFace Dataset instance"""
    from pkg.initializer_v2.dataset.huggingface import HuggingFace

    return HuggingFace()


@pytest.fixture
def real_hf_token():
    """Fixture to provide real HuggingFace token for E2E tests"""
    token = os.getenv("HUGGINGFACE_TOKEN")
    # if not token:
    #     pytest.skip("HUGGINGFACE_TOKEN environment variable not set")
    return token
