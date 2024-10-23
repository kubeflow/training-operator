import logging
import os
from urllib.parse import urlparse

import pkg.initiailizer_v2.utils.utils as utils
from pkg.initiailizer_v2.model.huggingface import HuggingFace

logging.basicConfig(
    format="%(asctime)s %(levelname)-8s [%(filename)s:%(lineno)d] %(message)s",
    datefmt="%Y-%m-%dT%H:%M:%SZ",
    level=logging.INFO,
)

if __name__ == "__main__":
    logging.info("Starting pre-trained model initialization")

    try:
        storage_uri = os.environ[utils.STORAGE_URI_ENV]
    except Exception as e:
        logging.error("STORAGE_URI env variable must be set.")
        raise e

    logging.info(f"Storage URI: {storage_uri}")

    storage_uri_parsed = urlparse(storage_uri)

    match storage_uri_parsed.scheme:
        # TODO (andreyvelich): Implement more model providers.
        case utils.HF_SCHEME:
            hf = HuggingFace()
            hf.load_config()
            hf.download_model(storage_uri_parsed)
        case _:
            logging.error(
                f"STORAGE_URI must have the valid model provider. STORAGE_URI: {storage_uri}"
            )
            raise Exception
