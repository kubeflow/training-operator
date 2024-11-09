import os
import runpy
import shutil
import tempfile

import pytest
from kubeflow.training import MODEL_PATH

import pkg.initializer_v2.utils.utils as utils


class TestModelIntegration:
    """Integration tests for model initialization"""

    @pytest.fixture(autouse=True)
    def setup_teardown(self, monkeypatch):
        """Setup and teardown for each test"""
        # Create temporary directory for model downloads
        current_dir = os.path.dirname(os.path.abspath(__file__))
        self.temp_dir = tempfile.mkdtemp(dir=current_dir)
        os.environ[MODEL_PATH] = self.temp_dir

        # Store original environment
        self.original_env = dict(os.environ)

        # Monkeypatch the constant in the module
        import kubeflow.training as training

        monkeypatch.setattr(training, "MODEL_PATH", self.temp_dir)

        yield

        # Cleanup
        shutil.rmtree(self.temp_dir, ignore_errors=True)
        os.environ.clear()
        os.environ.update(self.original_env)

    def verify_model_files(self, expected_files):
        """Verify downloaded model files"""
        if expected_files:
            actual_files = set(os.listdir(self.temp_dir))
            missing_files = set(expected_files) - actual_files
            assert not missing_files, f"Missing expected files: {missing_files}"

    @pytest.mark.parametrize(
        "test_name, provider, test_case",
        [
            # Public HuggingFace model test
            (
                "HuggingFace - Public model",
                "huggingface",
                {
                    "storage_uri": "hf://hf-internal-testing/tiny-random-bert",
                    "access_token": None,
                    "expected_files": [
                        "config.json",
                        "model.safetensors",
                        "tokenizer.json",
                        "tokenizer_config.json",
                    ],
                    "expected_error": None,
                },
            ),
            (
                "HuggingFace - Invalid model",
                "huggingface",
                {
                    "storage_uri": "hf://invalid/nonexistent-model",
                    "access_token": None,
                    "expected_files": None,
                    "expected_error": Exception,
                },
            ),
            (
                "HuggingFace - Login failure",
                "huggingface",
                {
                    "storage_uri": "hf://hf-internal-testing/tiny-random-bert",
                    "access_token": "invalid token",
                    "expected_files": None,
                    "expected_error": Exception,
                },
            ),
        ],
    )
    def test_model_download(self, test_name, provider, test_case):
        """Test end-to-end model download for different providers"""
        print(f"Running Integration test for {provider}: {test_name}")

        # Setup environment variables based on test case
        os.environ[utils.STORAGE_URI_ENV] = test_case["storage_uri"]
        expected_files = test_case.get("expected_files")

        # Handle token/credentials
        if test_case.get("access_token"):
            os.environ["ACCESS_TOKEN"] = test_case["access_token"]

        # Run the main script
        if test_case["expected_error"]:
            with pytest.raises(test_case["expected_error"]):
                runpy.run_module(
                    "pkg.initializer_v2.model.__main__", run_name="__main__"
                )
        else:
            runpy.run_module("pkg.initializer_v2.model.__main__", run_name="__main__")
            self.verify_model_files(expected_files)

        print("Test execution completed")
