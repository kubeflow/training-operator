FROM centos:7

ENV GOLANG_VERSION 1.8.3

RUN yum -y install go && yum clean all

# TODO(jlewi): We should probably change the directory to /opt/kubeflow.
RUN mkdir -p /opt/mlkube
RUN mkdir -p /opt/mlkube/test
RUN mkdir -p /opt/tensorflow_k8s/dashboard/
RUN mkdir -p /opt/kubeflow/samples

COPY tf_smoke.py /opt/kubeflow/samples/
RUN chmod a+x /opt/kubeflow/samples/*
COPY tf-operator /opt/mlkube
COPY tf-operator.v2 /opt/kubeflow
COPY e2e /opt/mlkube/test
COPY backend /opt/tensorflow_k8s/dashboard/
COPY build /opt/tensorflow_k8s/dashboard/frontend/build

RUN chmod a+x /opt/kubeflow/tf-operator.v2
RUN chmod a+x /opt/mlkube/tf-operator
RUN chmod a+x /opt/mlkube/test/e2e
RUN chmod a+x /opt/tensorflow_k8s/dashboard/backend

# TODO(jlewi): Reduce log level.
ENTRYPOINT ["/opt/mlkube/tf-operator", "-alsologtostderr"]
