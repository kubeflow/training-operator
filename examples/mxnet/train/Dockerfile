FROM mxnet/python:1.9.1_gpu_cu112_py3

RUN apt-get update -y && \
    apt-get install -y git && \
    git clone https://github.com/apache/mxnet.git -b v1.9.x && \
    rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["python3", "/mxnet/mxnet/example/image-classification/train_mnist.py"]
