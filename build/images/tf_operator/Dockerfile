FROM golang:1.13.5 AS build-image

ADD . /go/src/github.com/kubeflow/tf-operator

WORKDIR /go/src/github.com/kubeflow/tf-operator

RUN go build -o tf-operator.v1 ./cmd/tf-operator.v1

FROM gcr.io/distroless/base-debian10

COPY --from=build-image /go/src/github.com/kubeflow/tf-operator/tf-operator.v1 /opt/

ENTRYPOINT ["/opt/tf-operator.v1"]
