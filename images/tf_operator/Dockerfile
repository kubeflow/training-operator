# TODO(jlewi): How can we make it work with golang:1.8.2-alpine
FROM golang:1.8.2

RUN mkdir -p /opt/mlkube
RUN mkdir -p /opt/mlkube/test
COPY tf_operator /opt/mlkube
COPY e2e /opt/mlkube/test
COPY grpc_tensorflow_server.py /opt/mlkube/grpc_tensorflow_server/grpc_tensorflow_server.py
RUN chmod a+x /opt/mlkube/tf_operator
RUN chmod a+x /opt/mlkube/test/e2e

# TODO(jlewi): Reduce log level.
ENTRYPOINT ["/opt/mlkube/tf_operator", "-alsologtostderr"]