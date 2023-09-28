FROM nvcr.io/nvidia/mxnet:23.09-py3

# COPY custom code to container
COPY start-job.py /
COPY auto-tuning.py /

ENTRYPOINT ["python3", "/start-job.py"]
