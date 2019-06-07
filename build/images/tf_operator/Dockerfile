FROM registry.access.redhat.com/ubi8/ubi:latest

ARG GOLANG_VERSION=1.8.3
ENV GOROOT=/usr/local/go PATH=$PATH:/usr/local/go/bin

RUN if [ "$(uname -m)" = "ppc64le" ]; then \
        curl -fksSL https://dl.google.com/go/go$GOLANG_VERSION.linux-ppc64le.tar.gz | tar -xz -C /usr/local/; \
    else \
        curl -fksSL https://dl.google.com/go/go$GOLANG_VERSION.linux-amd64.tar.gz | tar -xz -C /usr/local/; \
    fi

# TODO(jlewi): We should probably change the directory to /opt/kubeflow.
RUN mkdir -p /opt/tensorflow_k8s/dashboard/
RUN mkdir -p /opt/kubeflow/samples

COPY tf_smoke.py /opt/kubeflow/samples/
RUN chmod a+x /opt/kubeflow/samples/*
COPY tf-operator.v1beta2 /opt/kubeflow
COPY tf-operator.v1 /opt/kubeflow
COPY backend /opt/tensorflow_k8s/dashboard/
COPY build /opt/tensorflow_k8s/dashboard/frontend/build

RUN chmod a+x /opt/kubeflow/tf-operator.v1beta2
RUN chmod a+x /opt/kubeflow/tf-operator.v1
RUN chmod a+x /opt/tensorflow_k8s/dashboard/backend

ENTRYPOINT ["/opt/kubeflow/tf-operator.v1"]
