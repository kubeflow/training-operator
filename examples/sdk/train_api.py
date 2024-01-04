from kubeflow.training.api.training_client import TrainingClient
from kubeflow.storage_init_container.hugging_face import (
    HuggingFaceModelParams,
    HuggingFaceTrainParams,
    HfDatasetParams,
    TRANSFORMER_TYPES,
)
from kubeflow.storage_init_container.s3 import S3DatasetParams
from peft import LoraConfig
from transformers import TrainingArguments
import json

client = TrainingClient(
    config_file="/Users/deepanker/Downloads/deepanker-test-kubectl.cfg"
)

client.train(
    name="deepanker-test",
    namespace="test",
    num_workers=2,
    num_procs_per_worker=0,
    storage_config={
        "size": "10Gi",
        "storage_class": "deepanker-test",
    },
    model_provider_parameters=HuggingFaceModelParams(
        model_uri="hf://Jedalc/codeparrot-gp2-finetune",
        transformer_type=TRANSFORMER_TYPES.AutoModelForCausalLM,
    ),
    dataset_provider_parameters=HfDatasetParams(
        repo_id="imdatta0/ultrachat_10k",
        access_token="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",
    ),
    # dataset_provider_parameters=S3DatasetParams(endpoint_url="http://10.117.63.3", bucket_name="deepanker-test", file_key="list_roles_response.txt", access_key="qEMHyz8wNwLpUWkvfZmQZrj60TE6zX4p", secret_key="qIp_QNLPKI0LJ5X0F8NrypoSMSsw_Gfe" ),
    train_parameters=HuggingFaceTrainParams(
        lora_config=LoraConfig(
            r=8,
            lora_alpha=8,
            target_modules=["c_attn", "c_proj", "w1", "w2"],
            layers_to_transform=list(range(30, 40)),
            # layers_pattern=['lm_head'],
            lora_dropout=0.1,
            bias="none",
            task_type="CAUSAL_LM",
        ),
        training_parameters=TrainingArguments(
            num_train_epochs=2,
            per_device_train_batch_size=1,
            gradient_accumulation_steps=1,
            gradient_checkpointing=True,
            warmup_steps=0.01,
            # max_steps=50, #20,
            learning_rate=1,
            lr_scheduler_type="cosine",
            bf16=False,
            logging_steps=0.01,
            output_dir="",
            optim=f"paged_adamw_32bit",
            save_steps=0.01,
            save_total_limit=3,
            disable_tqdm=False,
            resume_from_checkpoint=True,
            remove_unused_columns=True,
            evaluation_strategy="steps",
            eval_steps=0.01,
            # eval_accumulation_steps=1,
            per_device_eval_batch_size=1,
            # load_best_model_at_end=True,
            report_to="wandb",
            run_name=f"{1}",
        ),
    ),
    resources_per_worker={"gpu": 0, "cpu": 8, "memory": "8Gi"},
)
