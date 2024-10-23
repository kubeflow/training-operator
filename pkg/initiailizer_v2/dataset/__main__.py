import logging
import os
from urllib.parse import urlparse

import pkg.initiailizer_v2.utils.utils as utils
from pkg.initiailizer_v2.dataset.huggingface import HuggingFace

logging.basicConfig(
    format="%(asctime)s %(levelname)-8s [%(filename)s:%(lineno)d] %(message)s",
    datefmt="%Y-%m-%dT%H:%M:%SZ",
    level=logging.INFO,
)

if __name__ == "__main__":
    logging.info("Starting dataset initialization")

    try:
        storage_uri = os.environ[utils.STORAGE_URI_ENV]
    except Exception as e:
        logging.error("STORAGE_URI env variable must be set.")
        raise e

    logging.info(f"Storage URI: {storage_uri}")

    storage_uri_parsed = urlparse(storage_uri)

    match storage_uri_parsed.scheme:
        # TODO (andreyvelich): Implement more dataset providers.
        case utils.HF_SCHEME:
            hf = HuggingFace()
            hf.load_config()
            hf.download_dataset(storage_uri_parsed)
        case _:
            logging.error("STORAGE_URI must have the valid dataset provider")
            raise Exception
