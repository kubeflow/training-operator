# TODO(jlewi): How can we make it work with golang:1.8.2-alpine
FROM golang:1.8.2

RUN mkdir -p /opt/cmle
COPY tf_operator /opt/cmle
RUN chmod a+x /opt/cmle/tf_operator
# TODO(jlewi): Reduce log level.
ENTRYPOINT ["/opt/cmle/tf_operator", "-alsologtostderr"]