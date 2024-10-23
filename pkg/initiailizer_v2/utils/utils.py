import os
from dataclasses import fields
from typing import Dict

STORAGE_URI_ENV = "STORAGE_URI"
HF_SCHEME = "hf"


# Get DataClass config from the environment variables.
# Env names must be equal to the DataClass parameters.
def get_config_from_env(config) -> Dict[str, str]:
    config_from_env = {}
    for field in fields(config):
        config_from_env[field.name] = os.getenv(field.name.upper())

    return config_from_env
