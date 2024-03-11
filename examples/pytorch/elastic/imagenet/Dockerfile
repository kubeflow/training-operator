ARG BASE_IMAGE=pytorch/pytorch:2.2.1-cuda12.1-cudnn8-runtime
FROM $BASE_IMAGE

WORKDIR /workspace

# download imagenet tiny for data
RUN apt-get -q update && apt-get -q install -y wget unzip g++
RUN wget -q http://cs231n.stanford.edu/tiny-imagenet-200.zip && unzip -q tiny-imagenet-200.zip -d data && rm tiny-imagenet-200.zip

COPY examples/pytorch/elastic/imagenet/ ./examples

USER root
ENTRYPOINT ["python", "-m", "torch.distributed.run"]
CMD ["--help"]
