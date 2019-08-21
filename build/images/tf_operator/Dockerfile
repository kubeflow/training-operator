FROM golang:1.10 AS build-image

ADD . /go/src/github.com/kubeflow/tf-operator

WORKDIR /go/src/github.com/kubeflow/tf-operator
RUN ls
RUN go build -o tf-operator.v1beta2 ./cmd/tf-operator.v1beta2
RUN go build -o tf-operator.v1 ./cmd/tf-operator.v1
RUN go build -o backend ./dashboard/backend

FROM registry.access.redhat.com/ubi8/ubi:latest

# TODO(jlewi): We should probably change the directory to /opt/kubeflow.
RUN mkdir -p /opt/tensorflow_k8s/dashboard/ \
    &&  mkdir -p /opt/kubeflow/samples

COPY tf_smoke.py /opt/kubeflow/samples/
RUN chmod a+x /opt/kubeflow/samples/*
COPY --from=build-image /go/src/github.com/kubeflow/tf-operator/tf-operator.v1beta2 /opt/kubeflow
COPY --from=build-image /go/src/github.com/kubeflow/tf-operator/tf-operator.v1 /opt/kubeflow
COPY --from=build-image /go/src/github.com/kubeflow/tf-operator/backend /opt/tensorflow_k8s/dashboard/
COPY build /opt/tensorflow_k8s/dashboard/frontend/build

RUN chmod a+x /opt/kubeflow/tf-operator.v1beta2 \
    && chmod a+x /opt/kubeflow/tf-operator.v1 \
    && chmod a+x /opt/tensorflow_k8s/dashboard/backend

ENTRYPOINT ["/opt/kubeflow/tf-operator.v1"]
