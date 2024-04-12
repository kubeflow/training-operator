from dataclasses import dataclass, field
import json
import os
from urllib.parse import urlparse
from .abstract_dataset_provider import datasetProvider
from .constants import VOLUME_PATH_DATASET


@dataclass
class S3DatasetParams:
    endpoint_url: str
    bucket_name: str
    file_key: str
    region_name: str = None
    access_key: str = None
    secret_key: str = None

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
        import boto3

        # Create an S3 client for Nutanix Object Store/S3
        s3_client = boto3.Session(
            aws_access_key_id=self.config.access_key,
            aws_secret_access_key=self.config.secret_key,
            region_name=self.config.region_name,
        )
        s3_resource = s3_client.resource("s3", endpoint_url=self.config.endpoint_url)
        # Get the bucket object
        bucket = s3_resource.Bucket(self.config.bucket_name)

        # Filter objects with the specified prefix
        objects = bucket.objects.filter(Prefix=self.config.file_key)
        # Iterate over filtered objects
        for obj in objects:
            # Extract the object key (filename)
            obj_key = obj.key
            path_components = obj_key.split(os.path.sep)
            path_excluded_first_last_parts = os.path.sep.join(path_components[1:-1])

            # Create directories if they don't exist
            os.makedirs(
                os.path.join(VOLUME_PATH_DATASET, path_excluded_first_last_parts),
                exist_ok=True,
            )

            # Download the file
            file_path = os.path.sep.join(path_components[1:])
            bucket.download_file(obj_key, os.path.join(VOLUME_PATH_DATASET, file_path))
        print(f"Files downloaded")
