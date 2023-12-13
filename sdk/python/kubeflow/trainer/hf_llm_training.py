import argparse
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


def setup_model_and_tokenizer(token_dir, model_dir):
    # Set up the model and tokenizer
    tokenizer = AutoTokenizer.from_pretrained(
        token_dir, use_fast=False, trust_remote_code=True
    )
    tokenizer.pad_token = tokenizer.eos_token
    tokenizer.add_pad_token = True

    model = AutoModelForCausalLM.from_pretrained(
        model_dir,
        device_map="auto",
        trust_remote_code=True,
    )

    # Freeze model parameters
    for param in model.parameters():
        param.requires_grad = False

    return model, tokenizer


def load_and_preprocess_data(dataset_dir, tokenizer):
    # Load and preprocess the dataset
    train_data = load_dataset(dataset_dir, split="train").map(
        lambda x: tokenizer(x["text"]), batched=True
    )
    train_data = train_data.train_test_split(shuffle=True, test_size=0.1)

    try:
        eval_data = load_dataset(dataset_dir, split="eval")
    except Exception as err:
        eval_data = None

    return train_data, eval_data


def setup_peft_model(model, lora_config):
    # Set up the PEFT model
    lora_config = LoraConfig(**lora_config)
    model = get_peft_model(model, lora_config)
    return model


def train_model(model, train_data, eval_data, tokenizer, train_params):
    # Train the model
    trainer = Trainer(
        model=model,
        train_dataset=train_data,
        eval_dataset=eval_data,
        tokenizer=tokenizer,
        args=TrainingArguments(
            **train_params,
            data_collator=DataCollatorForLanguageModeling(
                tokenizer, pad_to_multiple_of=8, return_tensors="pt", mlm=False
            )
        ),
    )

    trainer.train()


def parse_arguments():
    parser = argparse.ArgumentParser(
        description="Script for training a model with PEFT configuration."
    )
    parser.add_argument("--model_dir", help="directory containing model")
    parser.add_argument("--token_dir", help="directory containing tokenizer")
    parser.add_argument("--dataset_dir", help="directory contaning dataset")
    parser.add_argument("--peft_config", help="peft_config")
    parser.add_argument("--train_params", help="hugging face training parameters")

    return parser.parse_args()


if __name__ == "__main__":
    args = parse_arguments()
    model, tokenizer = setup_model_and_tokenizer(args.token_dir, args.model_dir)
    train_data, eval_data = load_and_preprocess_data(args.dataset_dir, tokenizer)
    model = setup_peft_model(model, args.peft_config)
    train_model(model, train_data, eval_data, tokenizer, args)
