from .abstract_model_provider import modelProvider
from dataclasses import dataclass, field
from typing import Literal
import transformers
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
    model_uri: str
    transformer_type: Literal[
        "AutoModelForSequenceClassification",
        "AutoModelForTokenClassification",
        "AutoModelForQuestionAnswering",
        "AutoModelForCausalLM",
        "AutoModelForMaskedLM",
        "AutoModelForImageClassification",
    ]
    access_token: str = None
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
