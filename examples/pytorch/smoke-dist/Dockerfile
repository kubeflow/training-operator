# We need to use the nvcr.io/nvidia/pytorch image as a base image to support both linux/amd64 and linux_arm64 platforms.
# PyTorch=2.2.0, cuda=12.3.2
# Ref: https://docs.nvidia.com/deeplearning/frameworks/pytorch-release-notes/rel-24-01.html#rel-24-01
FROM nvcr.io/nvidia/pytorch:24.01-py3

RUN mkdir -p /opt/mlkube
COPY examples/pytorch/smoke-dist/dist_sendrecv.py /opt/mlkube/
ENTRYPOINT ["python", "/opt/mlkube/dist_sendrecv.py"]
