ARG BASE_IMAGE=nvcr.io/nvidia/pytorch:24.01-py3
FROM $BASE_IMAGE

WORKDIR /workspace

# download imagenet tiny for data
RUN apt-get -q update && apt-get -q install -y wget unzip
RUN wget -q http://cs231n.stanford.edu/tiny-imagenet-200.zip && unzip -q tiny-imagenet-200.zip -d data && rm tiny-imagenet-200.zip

COPY examples/pytorch/elastic/imagenet/ ./examples

USER root
ENTRYPOINT ["python", "-m", "torch.distributed.run"]
CMD ["--help"]
