FROM pytorch/pytorch:1.0-cuda10.0-cudnn7-runtime

RUN mkdir -p /opt/mlkube
COPY examples/pytorch/smoke-dist/dist_sendrecv.py /opt/mlkube/
ENTRYPOINT ["python", "/opt/mlkube/dist_sendrecv.py"]
