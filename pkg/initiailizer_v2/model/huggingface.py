import logging
from urllib.parse import ParseResult

import huggingface_hub

import pkg.initiailizer_v2.utils.utils as utils

# TODO (andreyvelich): This should be moved to SDK V2 constants.
import sdk.python.kubeflow.storage_initializer.constants as constants
from pkg.initiailizer_v2.model.config import HuggingFaceModelInputConfig

logging.basicConfig(
    format="%(asctime)s %(levelname)-8s [%(filename)s:%(lineno)d] %(message)s",
    datefmt="%Y-%m-%dT%H:%M:%SZ",
    level=logging.INFO,
)


class HuggingFace(utils.ModelProvider):

    def load_config(self):
        config_dict = utils.get_config_from_env(HuggingFaceModelInputConfig)
        logging.info(f"Config for HuggingFace model initiailizer: {config_dict}")
        self.config = HuggingFaceModelInputConfig(**config_dict)

    def download_model(self, storage_uri_parsed: ParseResult):
        model_uri = storage_uri_parsed.netloc + storage_uri_parsed.path
        logging.info(f"Downloading model: {model_uri}")
        logging.info("-" * 40)

        if self.config.access_token:
            huggingface_hub.login(self.config.access_token)

        # TODO (andreyvelich): We should verify these patterns for different models.
        huggingface_hub.snapshot_download(
            repo_id=model_uri,
            local_dir=constants.VOLUME_PATH_MODEL,
            allow_patterns=["*.json", "*.safetensors", "*.model"],
            ignore_patterns=["*.msgpack", "*.h5", "*.bin"],
        )

        logging.info("Model has been downloaded")
