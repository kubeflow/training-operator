FROM python:3.12

RUN pip install jax absl-py kubernetes

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
    && mkdir build \
    && cd build \
    && cmake ../ \
    && make \
    && make install

WORKDIR /app

ADD train.py /app
