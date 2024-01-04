from kubeflow.storage_init_container.abstract_model_provider import modelProvider
from kubeflow.storage_init_container.abstract_dataset_provider import datasetProvider
from dataclasses import dataclass, field
from typing import Literal
from urllib.parse import urlparse
import json, os
from typing import Dict, Any, Union
from datasets import load_dataset
from peft import LoraConfig
import transformers
from transformers import TrainingArguments
import enum
import huggingface_hub


class TRANSFORMER_TYPES(str, enum.Enum):
    """Types of Transformers."""

    AutoModelForSequenceClassification = "AutoModelForSequenceClassification"
    AutoModelForTokenClassification = "AutoModelForTokenClassification"
    AutoModelForQuestionAnswering = "AutoModelForQuestionAnswering"
    AutoModelForCausalLM = "AutoModelForCausalLM"
    AutoModelForMaskedLM = "AutoModelForMaskedLM"
    AutoModelForImageClassification = "AutoModelForImageClassification"


INIT_CONTAINER_MOUNT_PATH = "/workspace"


@dataclass
class HuggingFaceModelParams:
    model_uri: str
    transformer_type: TRANSFORMER_TYPES
    access_token: str = None
    download_dir: str = field(default=os.path.join(INIT_CONTAINER_MOUNT_PATH, "models"))

    def __post_init__(self):
        # Custom checks or validations can be added here
        if self.model_uri == "" or self.model_uri is None:
            raise ValueError("model_uri cannot be empty.")

    @property
    def download_dir(self):
        return self.download_dir

    @download_dir.setter
    def download_dir(self, value):
        raise AttributeError("Cannot modify read-only field 'download_dir'")


@dataclass
class HuggingFaceTrainParams:
    training_parameters: TrainingArguments = field(default_factory=TrainingArguments)
    lora_config: LoraConfig = field(default_factory=LoraConfig)


class HuggingFace(modelProvider):
    def load_config(self, serialised_args):
        # implementation for loading the config
        self.config = HuggingFaceModelParams(**json.loads(serialised_args))

    def download_model_and_tokenizer(self):
        # implementation for downloading the model
        print("downloading model")
        transformer_type_class = getattr(transformers, self.config.transformer_type)
        parsed_uri = urlparse(self.config.model_uri)
        self.model = parsed_uri.netloc + parsed_uri.path
        transformer_type_class.from_pretrained(
            self.model,
            token=self.config.access_token,
            cache_dir=self.config.download_dir,
            trust_remote_code=True,
        )
        transformers.AutoTokenizer.from_pretrained(
            self.model, cache_dir=self.config.download_dir
        )


@dataclass
class HfDatasetParams:
    repo_id: str
    access_token: str = None
    allow_patterns: list[str] = None
    ignore_patterns: list[str] = None
    download_dir: str = field(
        default=os.path.join(INIT_CONTAINER_MOUNT_PATH, "datasets")
    )

    def __post_init__(self):
        # Custom checks or validations can be added here
        if self.repo_id == "" or self.repo_id is None:
            raise ValueError("repo_id is None")

    @property
    def download_dir(self):
        return self.download_dir

    @download_dir.setter
    def download_dir(self, value):
        raise AttributeError("Cannot modify read-only field 'download_dir'")


class HuggingFaceDataset(datasetProvider):
    def load_config(self, serialised_args):
        self.config = HfDatasetParams(**json.loads(serialised_args))

    def download_dataset(self):
        print("downloading dataset")

        if self.config.access_token:
            huggingface_hub.login(self.config.access_token)

        load_dataset(self.config.repo_id, cache_dir=self.config.download_dir)
