# Install python 3.6
FROM python:3.6

RUN mkdir -p /opt/mlkube

# Download the rabit tracker and xgboost code.
COPY tracker.py /opt/mlkube/
COPY xgboost_smoke_test.py /opt/mlkube/
COPY requirements.txt /opt/mlkube/

# Install requirements
RUN pip install -r /opt/mlkube/requirements.txt

ENTRYPOINT ["python", "/opt/mlkube/xgboost_smoke_test.py"]
