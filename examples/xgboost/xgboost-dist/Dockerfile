# Install python 3.6.
FROM python:3.6

RUN mkdir -p /opt/mlkube

# Install requirements
COPY requirements.txt /opt/mlkube/
RUN pip install -r /opt/mlkube/requirements.txt

COPY *.py /opt/mlkube/

ENTRYPOINT ["python", "/opt/mlkube/main.py"]
