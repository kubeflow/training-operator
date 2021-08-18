FROM mxnet/python:gpu

RUN apt-get update && \
    apt-get install -y git && \
    git clone https://github.com/apache/incubator-mxnet.git -b v1.6.x

ENTRYPOINT ["python", "/incubator-mxnet/example/image-classification/train_mnist.py"]
