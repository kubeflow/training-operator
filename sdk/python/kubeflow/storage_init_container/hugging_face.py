from abstract_model_provider import modelProvider
from abstract_dataset_provider import datasetProvider
from dataclasses import dataclass, field
from typing import Literal
from urllib.parse import urlparse
import json
from typing import Dict, Any

TRANSFORMER_TYPES = [
    "AutoModelForSequenceClassification",
    "AutoModelForTokenClassification",
    "AutoModelForQuestionAnswering",
    "AutoModelForCausalLM",
    "AutoModelForMaskedLM",
    "AutoModelForImageClassification",
]


@dataclass
class HuggingFaceModelParams:
    access_token: str
    model_uri: str
    transformer_type: Literal[*TRANSFORMER_TYPES]
    download_dir: str = field(default="/workspace/models")

    def __post_init__(self):
        # Custom checks or validations can be added here
        if self.transformer_type not in TRANSFORMER_TYPES:
            raise ValueError("transformer_type must be one of %s", TRANSFORMER_TYPES)
        if self.model_uri is None:
            raise ValueError("model_uri cannot be none.")


@dataclass
class HuggingFaceTrainParams:
    additional_data: Dict[str, Any] = field(default_factory=dict)
    peft_config: Dict[str, Any] = field(default_factory=dict)


class HuggingFace(modelProvider):
    def load_config(self, serialised_args):
        # implementation for loading the config
        self.config = HuggingFaceModelParams(**json.loads(serialised_args))

    def download_model_and_tokenizer(self):
        # implementation for downloading the model
        print("downloading model")
        import transformers

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
    download_dir: str = field(default="/workspace/datasets")

    def __post_init__(self):
        # Custom checks or validations can be added here
        if self.repo_id is None:
            raise ValueError("repo_id is None")


class HuggingFaceDataset(datasetProvider):
    def load_config(self, serialised_args):
        self.config = HfDatasetParams(**json.loads(serialised_args))

    def download_dataset(self):
        print("downloading dataset")
        import huggingface_hub
        from huggingface_hub import snapshot_download

        if self.config.access_token:
            huggingface_hub.login(self.config.access_token)
        snapshot_download(
            repo_id=self.config.repo_id,
            repo_type="dataset",
            allow_patterns=self.config.allow_patterns,
            ignore_patterns=self.config.ignore_patterns,
            local_dir=self.config.download_dir,
        )
