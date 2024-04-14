# We need to use the nvcr.io/nvidia/pytorch image as a base image to support both linux/amd64 and linux_arm64 platforms.
# PyTorch=2.2.0, cuda=12.3.2
# Ref: https://docs.nvidia.com/deeplearning/frameworks/pytorch-release-notes/rel-24-01.html#rel-24-01
ARG BASE_IMAGE=nvcr.io/nvidia/pytorch:24.01-py3
FROM $BASE_IMAGE

WORKDIR /workspace

# download imagenet tiny for data
RUN apt-get -q update && apt-get -q install -y wget unzip
RUN wget -q http://cs231n.stanford.edu/tiny-imagenet-200.zip && unzip -q tiny-imagenet-200.zip -d data && rm tiny-imagenet-200.zip

# install dependent library
RUN pip install --upgrade pip && pip install python-etcd

COPY examples/pytorch/elastic/imagenet/ ./examples

USER root
ENTRYPOINT ["python", "-m", "torch.distributed.run"]
CMD ["--help"]
