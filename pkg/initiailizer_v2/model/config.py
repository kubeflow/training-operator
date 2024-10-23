from dataclasses import dataclass
from typing import Optional


# TODO (andreyvelich): This should be moved under Training V2 SDK.
@dataclass
class HuggingFaceModelInputConfig:
    invalid: str
    access_token: Optional[str] = None
