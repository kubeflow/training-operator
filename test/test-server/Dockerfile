# Dockerfile used by out prow jobs.
# The sole purpose of this image is to customize the command run.
FROM python:3.6.5-slim
MAINTAINER kubeflow-team

RUN pip install flask requests
RUN mkdir /opt/kubeflow
COPY test_app.py  /opt/kubeflow
RUN chmod a+x /opt/kubeflow

ENTRYPOINT ["python", "/opt/kubeflow/test_app.py"]