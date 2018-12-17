FROM centos:7

ARG GOLANG_VERSION=1.8.3
ARG GOLANG_ARCH=amd64
ENV GOROOT=/usr/local/go PATH=$PATH:/usr/local/go/bin

RUN curl -fksSL https://dl.google.com/go/go$GOLANG_VERSION.linux-$GOLANG_ARCH.tar.gz | tar -xz -C /usr/local/

# TODO(jlewi): We should probably change the directory to /opt/kubeflow.
RUN mkdir -p /opt/tensorflow_k8s/dashboard/
RUN mkdir -p /opt/kubeflow/samples

COPY tf_smoke.py /opt/kubeflow/samples/
RUN chmod a+x /opt/kubeflow/samples/*
COPY tf-operator.v1beta1 /opt/kubeflow
COPY tf-operator.v1beta2 /opt/kubeflow
COPY backend /opt/tensorflow_k8s/dashboard/
COPY build /opt/tensorflow_k8s/dashboard/frontend/build

RUN chmod a+x /opt/kubeflow/tf-operator.v1beta1
RUN chmod a+x /opt/kubeflow/tf-operator.v1beta2
RUN chmod a+x /opt/tensorflow_k8s/dashboard/backend

ENTRYPOINT ["/opt/kubeflow/tf-operator.v1beta1"]
