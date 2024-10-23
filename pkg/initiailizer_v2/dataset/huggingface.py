import logging
from urllib.parse import ParseResult

import huggingface_hub

import pkg.initiailizer_v2.utils.utils as utils

# TODO (andreyvelich): This should be moved to SDK V2 constants.
import sdk.python.kubeflow.storage_initializer.constants as constants
from pkg.initiailizer_v2.dataset.config import HuggingFaceDatasetConfig

logging.basicConfig(
    format="%(asctime)s %(levelname)-8s [%(filename)s:%(lineno)d] %(message)s",
    datefmt="%Y-%m-%dT%H:%M:%SZ",
    level=logging.INFO,
)


class HuggingFace(utils.DatasetProvider):

    def load_config(self):
        config_dict = utils.get_config_from_env(HuggingFaceDatasetConfig)
        logging.info(f"Config for HuggingFace dataset initiailizer: {config_dict}")
        self.config = HuggingFaceDatasetConfig(**config_dict)

    def download_dataset(self, storage_uri_parsed: ParseResult):
        dataset_uri = storage_uri_parsed.netloc + storage_uri_parsed.path
        logging.info(f"Downloading dataset: {dataset_uri}")
        logging.info("-" * 40)

        if self.config.access_token:
            huggingface_hub.login(self.config.access_token)

        huggingface_hub.snapshot_download(
            repo_id=dataset_uri,
            repo_type="dataset",
            local_dir=constants.VOLUME_PATH_DATASET,
        )

        logging.info("Dataset has been downloaded")
