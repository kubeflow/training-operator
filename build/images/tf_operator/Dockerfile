FROM centos:7

ENV GOLANG_VERSION 1.8.2
ENV GOLANG_DOWNLOAD_URL https://golang.org/dl/go$GOLANG_VERSION.linux-amd64.tar.gz
ENV GOLANG_DOWNLOAD_SHA256 5477d6c9a4f96fa120847fafa88319d7b56b5d5068e41c3587eebe248b939be7
RUN curl -fsSL "$GOLANG_DOWNLOAD_URL" -o golang.tar.gz \
	&& echo "$GOLANG_DOWNLOAD_SHA256  golang.tar.gz" | sha256sum -c - \
	&& tar -C /usr/local -xzf golang.tar.gz \
	&& rm golang.tar.gz
ENV PATH /usr/local/go/bin:$PATH

# TODO(jlewi): We should probably change the directory to /opt/kubeflow.
RUN mkdir -p /opt/mlkube
RUN mkdir -p /opt/mlkube/test
RUN mkdir -p /opt/tensorflow_k8s/dashboard/
RUN mkdir -p /opt/kubeflow/samples

COPY tf_smoke.py /opt/kubeflow/samples/
RUN chmod a+x /opt/kubeflow/samples/*
COPY tf-operator /opt/mlkube
COPY e2e /opt/mlkube/test
COPY backend /opt/tensorflow_k8s/dashboard/
COPY build /opt/tensorflow_k8s/dashboard/frontend/build
RUN chmod a+x /opt/mlkube/tf-operator
RUN chmod a+x /opt/mlkube/test/e2e
RUN chmod a+x /opt/tensorflow_k8s/dashboard/backend

# TODO(jlewi): Reduce log level.
ENTRYPOINT ["/opt/mlkube/tf-operator", "-alsologtostderr"]
