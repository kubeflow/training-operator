# Dockerfile used by out prow jobs.
# The sole purpose of this image is to customize the command run.
FROM python:3.6.5-slim
MAINTAINER kubeflow-team

RUN pip install flask requests
RUN mkdir /opt/kubeflow
COPY test_app.py  /opt/kubeflow
RUN chmod a+x /opt/kubeflow

# Just for testing - remove
ENV TF_CONFIG={"cluster":{"chief":["mytfjob-chief-0:2222"],"ps":["mytfjob-ps-0:2222"],"worker":["mytfjob-worker-0:2222"]},"task":{"type":"chief","index":0},"environment":"cloud"}

ENTRYPOINT ["python", "/opt/kubeflow/test_app.py"]
