from dataclasses import dataclass
from typing import Optional


# TODO (andreyvelich): This should be moved under Training V2 SDK.
@dataclass
class HuggingFaceDatasetConfig:
    access_token: Optional[str] = None
