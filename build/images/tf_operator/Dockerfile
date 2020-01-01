FROM golang:1.10 AS build-image

ADD . /go/src/github.com/kubeflow/tf-operator

WORKDIR /go/src/github.com/kubeflow/tf-operator

RUN go build -o tf-operator.v1 ./cmd/tf-operator.v1

FROM registry.access.redhat.com/ubi8/ubi:latest

# TODO(jlewi): We should probably change the directory to /opt/kubeflow.
RUN mkdir -p /opt/kubeflow/samples

#COPY tf_smoke.py /opt/kubeflow/samples/
#RUN chmod a+x /opt/kubeflow/samples/*

COPY --from=build-image /go/src/github.com/kubeflow/tf-operator/tf-operator.v1 /opt/kubeflow

RUN chmod a+x /opt/kubeflow/tf-operator.v1

ENTRYPOINT ["/opt/kubeflow/tf-operator.v1"]
