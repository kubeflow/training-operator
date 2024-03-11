FROM pytorch/pytorch:2.2.1-cuda12.1-cudnn8-runtime

RUN mkdir -p /opt/mlkube
COPY examples/pytorch/smoke-dist/dist_sendrecv.py /opt/mlkube/
ENTRYPOINT ["python", "/opt/mlkube/dist_sendrecv.py"]
