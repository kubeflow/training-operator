from dataclasses import dataclass, field
from urllib.parse import urlparse
import json, os
from datasets import load_dataset
from peft import LoraConfig
import transformers
from transformers import TrainingArguments
import enum
import huggingface_hub
from typing import Union

TRANSFORMER_TYPES = Union[
    transformers.AutoModelForSequenceClassification,
    transformers.AutoModelForTokenClassification,
    transformers.AutoModelForQuestionAnswering,
    transformers.AutoModelForCausalLM,
    transformers.AutoModelForMaskedLM,
    transformers.AutoModelForImageClassification,
]
