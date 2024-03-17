FROM nvcr.io/nvidia/pytorch:24.01-py3

RUN mkdir -p /opt/mlkube
COPY examples/pytorch/smoke-dist/dist_sendrecv.py /opt/mlkube/
ENTRYPOINT ["python", "/opt/mlkube/dist_sendrecv.py"]
