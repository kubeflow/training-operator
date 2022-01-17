# Install python 3。6.
FROM python:3.6

RUN apt-get update
RUN apt-get install -y git make g++ cmake

RUN mkdir -p /opt/mlkube

# Download the rabit tracker and xgboost code.

COPY requirements.txt /opt/mlkube/

# Install requirements

RUN pip install -r /opt/mlkube/requirements.txt

# Build XGBoost.
RUN git clone --recursive https://github.com/dmlc/xgboost && \
    cd xgboost && \
    make -j$(nproc) && \
    cd python-package; python setup.py install

COPY *.py /opt/mlkube/

ENTRYPOINT ["python", "/opt/mlkube/main.py"]
