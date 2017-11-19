# TODO(jlewi): How can we make it work with golang:1.8.2-alpine
FROM golang:1.8.2

RUN mkdir -p /opt/mlkube
RUN mkdir -p /opt/mlkube/test
RUN mkdir -p /opt/mlkube/dashboard/
COPY tf_operator /opt/mlkube
COPY e2e /opt/mlkube/test
COPY grpc_tensorflow_server.py /opt/mlkube/grpc_tensorflow_server/grpc_tensorflow_server.py
COPY backend /opt/mlkube/dashboard/
COPY build /opt/mlkube/dashboard/frontend/build
RUN chmod a+x /opt/mlkube/tf_operator
RUN chmod a+x /opt/mlkube/test/e2e
RUN chmod a+x /opt/mlkube/dashboard/backend

# TODO(jlewi): Reduce log level.
ENTRYPOINT ["/opt/mlkube/tf_operator", "-alsologtostderr"]