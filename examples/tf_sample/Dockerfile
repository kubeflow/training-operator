FROM gcr.io/tensorflow/tensorflow:latest
RUN mkdir -p /opt/cmle
COPY tf_sample/tf_smoke.py /opt/cmle/
ENTRYPOINT ["python", "/opt/cmle/tf_smoke.py"]