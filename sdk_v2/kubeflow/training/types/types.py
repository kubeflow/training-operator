from dataclasses import dataclass
from typing import Optional


# Representation for the Training Runtime.
@dataclass
class Runtime:
    name: str
    phase: str


# Representation for the TrainJob.
# TODO (andreyvelich): Discuss what fields we want user to see.
@dataclass
class TrainJob:
    name: str
    runtime_ref: str
    creation_timestamp: str
    status: Optional[str] = None


# Representation for the Pod.
@dataclass
class Pod:
    name: str
    type: str
    status: Optional[str] = None


@dataclass
class HuggingFaceDatasetConfig:
    storage_uri: str
    access_token: Optional[str] = None


@dataclass
class HuggingFaceModelInputConfig:
    storage_uri: str
    access_token: Optional[str] = None
