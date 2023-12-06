import argparse
from hugging_face import HuggingFace


def model_factory(model_provider, model_provider_args):
    match model_provider:
        case "hf":
            hf = HuggingFace()
            hf.load_config(model_provider_args)
            hf.download_model_and_tokenizer()
        case _:
            return "This is the default case"


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="script for downloading model and datasets to PVC."
    )
    parser.add_argument("model_provider", type=str, help="name of model provider")
    parser.add_argument(
        "model_provider_args", type=str, help="model provider serialised arguments"
    )

    parser.add_argument("dataset_provider", type=str, help="name of dataset provider")
    parser.add_argument(
        "dataset_provider_args", type=str, help="dataset provider serialised arguments"
    )
    args = parser.parse_args()

    model_factory(args.model_provider, args.model_provider_args)
