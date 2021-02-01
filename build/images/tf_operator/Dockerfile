FROM golang:1.13.5 AS build-image

ADD . /go/src/github.com/kubeflow/tf-operator

WORKDIR /go/src/github.com/kubeflow/tf-operator

RUN if [ "$(uname -m)" = "aarch64" ]; then \
        GO111MODULE="on" CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o tf-operator.v1 ./cmd/tf-operator.v1; \
    else \
        GO111MODULE="on" CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o tf-operator.v1 ./cmd/tf-operator.v1; \
    fi

FROM gcr.io/distroless/base-debian10

COPY third_party/library/license.txt /opt/license.txt

COPY --from=build-image /go/src/github.com/kubeflow/tf-operator/tf-operator.v1 /opt/

ENTRYPOINT ["/opt/tf-operator.v1"]
