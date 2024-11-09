from unittest.mock import MagicMock, patch

import pytest
from kubeflow.training import DATASET_PATH

import pkg.initializer_v2.utils.utils as utils


@pytest.fixture
def huggingface_dataset_instance():
    """Fixture for HuggingFace Dataset instance"""
    from pkg.initializer_v2.dataset.huggingface import HuggingFace

    return HuggingFace()


# Test cases for config loading
@pytest.mark.parametrize(
    "test_name, test_config, expected",
    [
        (
            "Full config with token",
            {"storage_uri": "hf://dataset/path", "access_token": "test_token"},
            {"storage_uri": "hf://dataset/path", "access_token": "test_token"},
        ),
        (
            "Minimal config without token",
            {"storage_uri": "hf://dataset/path"},
            {"storage_uri": "hf://dataset/path", "access_token": None},
        ),
    ],
)
def test_load_config(test_name, test_config, expected, huggingface_dataset_instance):
    """Test config loading with different configurations"""
    print(f"Running test: {test_name}")

    with patch.object(utils, "get_config_from_env", return_value=test_config):
        huggingface_dataset_instance.load_config()
        assert (
            huggingface_dataset_instance.config.storage_uri == expected["storage_uri"]
        )
        assert (
            huggingface_dataset_instance.config.access_token == expected["access_token"]
        )

    print("Test execution completed")


@pytest.mark.parametrize(
    "test_name, test_case",
    [
        (
            "Successful download with token",
            {
                "config": {
                    "storage_uri": "hf://username/dataset-name",
                    "access_token": "test_token",
                },
                "should_login": True,
                "expected_repo_id": "username/dataset-name",
                "mock_login_side_effect": None,
                "mock_download_side_effect": None,
                "expected_error": None,
            },
        ),
        (
            "Successful download without token",
            {
                "config": {"storage_uri": "hf://org/dataset-v1", "access_token": None},
                "should_login": False,
                "expected_repo_id": "org/dataset-v1",
                "mock_login_side_effect": None,
                "mock_download_side_effect": None,
                "expected_error": None,
            },
        ),
        (
            "Login failure",
            {
                "config": {
                    "storage_uri": "hf://username/dataset-name",
                    "access_token": "test_token",
                },
                "should_login": True,
                "expected_repo_id": "username/dataset-name",
                "mock_login_side_effect": Exception,
                "mock_download_side_effect": None,
                "expected_error": Exception,
            },
        ),
        (
            "Download failure",
            {
                "config": {
                    "storage_uri": "hf://invalid/repo/name",
                    "access_token": None,
                },
                "should_login": False,
                "expected_repo_id": "invalid/repo/name",
                "mock_login_side_effect": None,
                "mock_download_side_effect": Exception,
                "expected_error": Exception,
            },
        ),
    ],
)
def test_download_dataset(test_name, test_case, huggingface_dataset_instance):
    """Test dataset download with different configurations"""

    print(f"Running test: {test_name}")

    huggingface_dataset_instance.config = MagicMock(**test_case["config"])

    with patch("huggingface_hub.login") as mock_login, patch(
        "huggingface_hub.snapshot_download"
    ) as mock_download:

        # Configure mock behavior
        if test_case["mock_login_side_effect"]:
            mock_login.side_effect = test_case["mock_login_side_effect"]
        if test_case["mock_download_side_effect"]:
            mock_download.side_effect = test_case["mock_download_side_effect"]

        # Execute test
        if test_case["expected_error"]:
            with pytest.raises(test_case["expected_error"]):
                huggingface_dataset_instance.download_dataset()
        else:
            huggingface_dataset_instance.download_dataset()

            # Verify login behavior
            if test_case["should_login"]:
                mock_login.assert_called_once_with(test_case["config"]["access_token"])
            else:
                mock_login.assert_not_called()

            # Verify download parameters
            mock_download.assert_called_once_with(
                repo_id=test_case["expected_repo_id"],
                local_dir=DATASET_PATH,
                repo_type="dataset",
            )
    print("Test execution completed")
