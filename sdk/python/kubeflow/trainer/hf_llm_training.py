import argparse
import transformers
from transformers import (
    AutoModelForCausalLM,
    AutoTokenizer,
    AutoConfig,
    TrainingArguments,
    DataCollatorForLanguageModeling,
    Trainer,
)
import torch
from datasets import load_dataset
from peft import LoraConfig, get_peft_model
from urllib.parse import urlparse
import os
import json


def setup_model_and_tokenizer(model_uri, transformer_type, model_dir, train_args):
    # Set up the model and tokenizer
    parsed_uri = urlparse(model_uri)
    model_name = parsed_uri.netloc + parsed_uri.path
    transformer_type_class = getattr(transformers, transformer_type)

    model = transformer_type_class.from_pretrained(
        pretrained_model_name_or_path=model_name,
        cache_dir=model_dir,
        local_files_only=True,
        device_map="auto",
        trust_remote_code=True,
    )

    tokenizer = transformers.AutoTokenizer.from_pretrained(
        pretrained_model_name_or_path=model_name,
        cache_dir=model_dir,
        local_files_only=True,
        device_map="auto",
    )

    tokenizer.pad_token = tokenizer.eos_token
    tokenizer.add_pad_token = True

    # Freeze model parameters
    for param in model.parameters():
        param.requires_grad = False

    return model, tokenizer


def load_and_preprocess_data(dataset_name, dataset_dir, transformer_type, tokenizer):
    # Load and preprocess the dataset
    print("loading dataset")
    transformer_type_class = getattr(transformers, transformer_type)
    if transformer_type_class != transformers.AutoModelForImageClassification:
        dataset = load_dataset(dataset_name, cache_dir=dataset_dir).map(
            lambda x: tokenizer(x["text"]), batched=True
        )
    else:
        dataset = load_dataset(dataset_name, cache_dir=dataset_dir)

    train_data = dataset["train"]

    try:
        eval_data = dataset["eval"]
    except Exception as err:
        eval_data = None
        print("Evaluation dataset is not found")

    return train_data, eval_data


def setup_peft_model(model, lora_config):
    # Set up the PEFT model
    lora_config = LoraConfig(**json.loads(lora_config))
    model.enable_input_require_grads()
    model = get_peft_model(model, lora_config)
    return model


def train_model(model, train_data, eval_data, tokenizer, train_args):
    # Train the model
    trainer = Trainer(
        model=model,
        train_dataset=train_data,
        eval_dataset=eval_data,
        tokenizer=tokenizer,
        args=train_args,
        data_collator=DataCollatorForLanguageModeling(
            tokenizer, pad_to_multiple_of=8, mlm=False
        ),
    )
    trainer.train()
    print("training done")


def parse_arguments():
    parser = argparse.ArgumentParser(
        description="Script for training a model with PEFT configuration."
    )

    parser.add_argument("--model_uri", help="model uri")
    parser.add_argument("--transformer_type", help="model transformer type")
    parser.add_argument("--model_dir", help="directory containing model")
    parser.add_argument("--dataset_dir", help="directory contaning dataset")
    parser.add_argument("--dataset_name", help="dataset name")
    parser.add_argument("--lora_config", help="lora_config")
    parser.add_argument(
        "--training_parameters", help="hugging face training parameters"
    )

    return parser.parse_args()


if __name__ == "__main__":
    args = parse_arguments()
    train_args = TrainingArguments(**json.loads(args.training_parameters))
    model, tokenizer = setup_model_and_tokenizer(
        args.model_uri, args.transformer_type, args.model_dir, train_args
    )
    train_data, eval_data = load_and_preprocess_data(
        args.dataset_name, args.dataset_dir, args.transformer_type, tokenizer
    )
    model = setup_peft_model(model, args.lora_config)
    train_model(model, train_data, eval_data, tokenizer, train_args)
