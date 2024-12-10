import runpy
from unittest.mock import MagicMock, patch

import pytest


@pytest.mark.parametrize(
    "test_name, test_case",
    [
        (
            "Successful download with HuggingFace provider",
            {
                "storage_uri": "hf://dataset/path",
                "access_token": "test_token",
                "mock_config_error": False,
                "expected_error": None,
            },
        ),
        (
            "Missing storage URI environment variable",
            {
                "storage_uri": None,
                "access_token": None,
                "mock_config_error": False,
                "expected_error": Exception,
            },
        ),
        (
            "Invalid storage URI scheme",
            {
                "storage_uri": "invalid://dataset/path",
                "access_token": None,
                "mock_config_error": False,
                "expected_error": Exception,
            },
        ),
        (
            "Config loading failure",
            {
                "storage_uri": "hf://dataset/path",
                "access_token": None,
                "mock_config_error": True,
                "expected_error": Exception,
            },
        ),
    ],
)
def test_dataset_main(test_name, test_case, mock_env_vars):
    """Test main script with different scenarios"""
    print(f"Running test: {test_name}")

    # Setup mock environment variables
    env_vars = {
        "STORAGE_URI": test_case["storage_uri"],
        "ACCESS_TOKEN": test_case["access_token"],
    }
    mock_env_vars(**env_vars)

    # Setup mock HuggingFace instance
    mock_hf_instance = MagicMock()
    if test_case["mock_config_error"]:
        mock_hf_instance.load_config.side_effect = Exception

    with patch(
        "pkg.initializer_v2.dataset.huggingface.HuggingFace",
        return_value=mock_hf_instance,
    ) as mock_hf:

        # Execute test
        if test_case["expected_error"]:
            with pytest.raises(test_case["expected_error"]):
                runpy.run_module(
                    "pkg.initializer_v2.dataset.__main__", run_name="__main__"
                )
        else:
            runpy.run_module("pkg.initializer_v2.dataset.__main__", run_name="__main__")

            # Verify HuggingFace instance methods were called
            mock_hf_instance.load_config.assert_called_once()
            mock_hf_instance.download_dataset.assert_called_once()

        # Verify HuggingFace class instantiation
        if test_case["storage_uri"] and test_case["storage_uri"].startswith("hf://"):
            mock_hf.assert_called_once()

    print("Test execution completed")
