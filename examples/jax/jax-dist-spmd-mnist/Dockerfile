FROM python:3.13

RUN pip install --upgrade pip
RUN pip install --upgrade jax absl-py

RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    git \
    libgoogle-glog-dev \
    libgflags-dev \
    libprotobuf-dev \
    protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

RUN git clone https://github.com/facebookincubator/gloo.git \
    && cd gloo \
    && git checkout 43b7acbf372cdce14075f3526e39153b7e433b53 \
    && mkdir build \
    && cd build \
    && cmake ../ \
    && make \
    && make install

WORKDIR /app

ADD datasets.py spmd_mnist_classifier_fromscratch.py /app

ENTRYPOINT ["python3", "spmd_mnist_classifier_fromscratch.py"]
