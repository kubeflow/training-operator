# TODO(jlewi): How can we make it work with golang:1.8.2-alpine
FROM golang:1.8.2

RUN mkdir -p /opt/mlkube
RUN mkdir -p /opt/mlkube/test
RUN mkdir -p /opt/tensorflow_k8s/dashboard/
COPY tf_operator /opt/mlkube
COPY e2e /opt/mlkube/test
COPY backend /opt/tensorflow_k8s/dashboard/
COPY build /opt/tensorflow_k8s/dashboard/frontend/build
RUN chmod a+x /opt/mlkube/tf_operator
RUN chmod a+x /opt/mlkube/test/e2e
RUN chmod a+x /opt/tensorflow_k8s/dashboard/backend

# TODO(jlewi): Reduce log level.
ENTRYPOINT ["/opt/mlkube/tf_operator", "-alsologtostderr"]
