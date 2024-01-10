from dataclasses import dataclass, field
import transformers
from peft import LoraConfig
from urllib.parse import urlparse
import json, os
from typing import Union
from .constants import INIT_CONTAINER_MOUNT_PATH
from .abstract_model_provider import modelProvider
from .abstract_dataset_provider import datasetProvider


TRANSFORMER_TYPES = Union[
    transformers.AutoModelForSequenceClassification,
    transformers.AutoModelForTokenClassification,
    transformers.AutoModelForQuestionAnswering,
    transformers.AutoModelForCausalLM,
    transformers.AutoModelForMaskedLM,
    transformers.AutoModelForImageClassification,
]


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
    training_parameters: transformers.TrainingArguments = field(
        default_factory=transformers.TrainingArguments
    )
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
        import huggingface_hub
        from datasets import load_dataset

        if self.config.access_token:
            huggingface_hub.login(self.config.access_token)

        load_dataset(self.config.repo_id, cache_dir=self.config.download_dir)
