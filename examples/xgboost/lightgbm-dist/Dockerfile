# inspired from https://github.com/microsoft/LightGBM/blob/v4.1.0/docker/dockerfile-cli
FROM ubuntu:20.04 as builder

ENV \
    DEBIAN_FRONTEND=noninteractive \
    LANG=C.UTF-8 \
    LC_ALL=C.UTF-8

RUN apt-get update -y && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        cmake \
        build-essential \
        gcc \
        g++ \
        git \
        libomp-dev && \
    rm -rf /var/lib/apt/lists/*

RUN git clone \
        --recursive \
        --branch v4.1.0 \
        --depth 1 \
        https://github.com/Microsoft/LightGBM && \
    mkdir LightGBM/build && \
    cd LightGBM/build && \
    cmake .. && \
    make -j4 && \
    make install && \
    cd "${HOME}" && \
    rm -rf LightGBM

FROM python:3.7

COPY requirements.txt .
RUN pip install -r requirements.txt
COPY --from=builder /usr/local/bin/lightgbm /usr/local/bin/lightgbm

WORKDIR /app

# Download the example data
RUN mkdir data
ADD https://raw.githubusercontent.com/microsoft/LightGBM/stable/examples/parallel_learning/binary.train data/.
ADD https://raw.githubusercontent.com/microsoft/LightGBM/stable/examples/parallel_learning/binary.test data/.
COPY *.py ./

ENTRYPOINT [ "python", "/app/main.py" ]
