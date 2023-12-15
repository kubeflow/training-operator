from .abstract_dataset_provider import datasetProvider
from dataclasses import dataclass, field
import json
import boto3
from urllib.parse import urlparse
import os

@dataclass
class S3DatasetParams:
    endpoint_url: str
    bucket_name: str
    file_key: str
    region_name: str = None
    access_key: str = None
    secret_key: str = None
    download_dir: str = field(default="/workspace/datasets")

    def is_valid_url(self, url):
        try:
            parsed_url = urlparse(url)
            print(parsed_url)
            return all([parsed_url.scheme, parsed_url.netloc])
        except ValueError:
            return False

    def __post_init__(self):
        # Custom checks or validations can be added here
        if (
            self.bucket_name is None
            or self.endpoint_url is None
            or self.file_key is None
        ):
            raise ValueError("bucket_name or endpoint_url or file_key is None")
        self.is_valid_url(self.endpoint_url)


class S3(datasetProvider):
    def load_config(self, serialised_args):
        self.config = S3DatasetParams(**json.loads(serialised_args))

    def download_dataset(self):
        # Create an S3 client for Nutanix Object Store/S3
        s3_client = boto3.client(
            "s3",
            aws_access_key_id=self.config.access_key,
            aws_secret_access_key=self.config.secret_key,
            endpoint_url=self.config.endpoint_url,
            region_name=self.config.region_name,
        )

        # Download the file
        s3_client.download_file(
            self.config.bucket_name, self.config.file_key, os.path.join(self.config.download_dir,self.config.file_key)
        )
        print(f"File downloaded to: {self.config.download_dir}")
