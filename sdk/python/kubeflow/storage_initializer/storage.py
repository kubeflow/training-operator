import argparse
from .hugging_face import HuggingFace, HuggingFaceDataset
from .s3 import S3


def model_factory(model_provider, model_provider_parameters):
    if model_provider == "hf":
        hf = HuggingFace()
        hf.load_config(model_provider_parameters)
        hf.download_model_and_tokenizer()
    else:
        return "This is the default case"


def dataset_factory(dataset_provider, dataset_provider_parameters):
    if dataset_provider == "s3":
        s3 = S3()
        s3.load_config(dataset_provider_parameters)
        s3.download_dataset()
    elif dataset_provider == "hf":
        hf = HuggingFaceDataset()
        hf.load_config(dataset_provider_parameters)
        hf.download_dataset()
    else:
        return "This is the default case"


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="script for downloading model and datasets to PVC."
    )
    parser.add_argument("--model_provider", type=str, help="name of model provider")
    parser.add_argument(
        "--model_provider_parameters",
        type=str,
        help="model provider serialised arguments",
    )

    parser.add_argument("--dataset_provider", type=str, help="name of dataset provider")
    parser.add_argument(
        "--dataset_provider_parameters",
        type=str,
        help="dataset provider serialized arguments",
    )
    args = parser.parse_args()

    model_factory(args.model_provider, args.model_provider_parameters)
    dataset_factory(args.dataset_provider, args.dataset_provider_parameters)
