import os
import runpy
import shutil
import tempfile

import pytest

import pkg.initializer_v2.utils.utils as utils
from sdk.python.kubeflow.storage_initializer.constants import VOLUME_PATH_DATASET


class TestDatasetE2E:
    """E2E tests for dataset initialization"""

    @pytest.fixture(autouse=True)
    def setup_teardown(self, monkeypatch):
        """Setup and teardown for each test"""
        # Create temporary directory for dataset downloads
        current_dir = os.path.dirname(os.path.abspath(__file__))
        self.temp_dir = tempfile.mkdtemp(dir=current_dir)
        os.environ[VOLUME_PATH_DATASET] = self.temp_dir

        # Store original environment
        self.original_env = dict(os.environ)

        # Monkeypatch the constant in the module
        import sdk.python.kubeflow.storage_initializer.constants as constants

        monkeypatch.setattr(constants, "VOLUME_PATH_DATASET", self.temp_dir)

        yield

        # Cleanup
        shutil.rmtree(self.temp_dir, ignore_errors=True)
        os.environ.clear()
        os.environ.update(self.original_env)

    def verify_dataset_files(self, expected_files):
        """Verify downloaded dataset files"""
        if expected_files:
            actual_files = set(os.listdir(self.temp_dir))
            missing_files = set(expected_files) - actual_files
            assert not missing_files, f"Missing expected files: {missing_files}"

    @pytest.mark.parametrize(
        "test_name, provider, test_case",
        [
            # Public HuggingFace dataset test
            (
                "HuggingFace - Public dataset",
                "huggingface",
                {
                    "storage_uri": "hf://karpathy/tiny_shakespeare",
                    "access_token": None,
                    "expected_files": ["tiny_shakespeare.py"],
                    "expected_error": None,
                },
            ),
            # Private HuggingFace dataset test
            # (
            #     "HuggingFace - Private dataset",
            #     "huggingface",
            #     {
            #         "storage_uri": "hf://username/private-dataset",
            #         "use_real_token": True,
            #         "expected_files": ["config.json", "dataset.safetensors"],
            #         "expected_error": None
            #     }
            # ),
            # Invalid HuggingFace dataset test
            (
                "HuggingFace - Invalid dataset",
                "huggingface",
                {
                    "storage_uri": "hf://invalid/nonexistent-dataset",
                    "access_token": None,
                    "expected_files": None,
                    "expected_error": Exception,
                },
            ),
        ],
    )
    def test_dataset_download(self, test_name, provider, test_case, real_hf_token):
        """Test end-to-end dataset download for different providers"""
        print(f"Running E2E test for {provider}: {test_name}")

        # Setup environment variables based on test case
        os.environ[utils.STORAGE_URI_ENV] = test_case["storage_uri"]
        expected_files = test_case.get("expected_files")

        # Handle token/credentials
        if test_case.get("use_real_token"):
            os.environ["ACCESS_TOKEN"] = real_hf_token
        elif test_case.get("access_token"):
            os.environ["ACCESS_TOKEN"] = test_case["access_token"]

        # Run the main script
        if test_case["expected_error"]:
            with pytest.raises(test_case["expected_error"]):
                runpy.run_module(
                    "pkg.initializer_v2.dataset.__main__", run_name="__main__"
                )
        else:
            runpy.run_module("pkg.initializer_v2.dataset.__main__", run_name="__main__")
            self.verify_dataset_files(expected_files)

        print("Test execution completed")
