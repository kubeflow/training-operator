from dataclasses import dataclass
from typing import Optional


# TODO (andreyvelich): This should be moved under Training V2 SDK.
@dataclass
class HuggingFaceDatasetConfig:
    storage_uri: str
    access_token: Optional[str] = None
