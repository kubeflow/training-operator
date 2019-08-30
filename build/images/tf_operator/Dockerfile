FROM registry.access.redhat.com/ubi8/ubi:latest

# TODO(jlewi): We should probably change the directory to /opt/kubeflow.
RUN mkdir -p /opt/tensorflow_k8s/dashboard/ \
    &&  mkdir -p /opt/kubeflow/samples

COPY tf_smoke.py /opt/kubeflow/samples/
RUN chmod a+x /opt/kubeflow/samples/*
COPY tf-operator.v1 /opt/kubeflow
COPY backend /opt/tensorflow_k8s/dashboard/
COPY build /opt/tensorflow_k8s/dashboard/frontend/build

RUN chmod a+x /opt/kubeflow/tf-operator.v1 \
    && chmod a+x /opt/tensorflow_k8s/dashboard/backend

ENTRYPOINT ["/opt/kubeflow/tf-operator.v1"]
