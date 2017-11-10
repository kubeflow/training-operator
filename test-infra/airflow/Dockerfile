FROM puckel/docker-airflow

USER root

RUN apt-get update && apt-get install -y --no-install-recommends \
    wget \
    curl \
    file \
    rsync \
    ca-certificates \
    build-essential \
    openssh-client \
    git \
    pkg-config \
    zip \
    unzip \
    xz-utils \
    zlib1g-dev \
    python \
    python-setuptools \
    python-openssl \
    python2.7

# Install go
RUN cd /tmp && \
    wget -O /tmp/go.tar.gz https://redirector.gvt1.com/edgedl/go/go1.9.2.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go.tar.gz

# Install glide
RUN cd /tmp && \
    wget -O glide-v0.13.0-linux-amd64.tar.gz \
            https://github.com/Masterminds/glide/releases/download/v0.13.0/glide-v0.13.0-linux-amd64.tar.gz && \
    tar -xvf glide-v0.13.0-linux-amd64.tar.gz && \
    mv ./linux-amd64/glide /usr/local/bin/

# Install gcloud

ENV PATH=/google-cloud-sdk/bin:/workspace:${PATH} \
    CLOUDSDK_CORE_DISABLE_PROMPTS=1

RUN wget -q https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz && \
    tar xzf google-cloud-sdk.tar.gz -C / && \
    rm google-cloud-sdk.tar.gz && \
    /google-cloud-sdk/install.sh \
        --disable-installation-options \
        --bash-completion=false \
        --path-update=false \
        --usage-reporting=false && \
    gcloud components install alpha beta kubectl

# Install Helm
RUN wget -O /tmp/get_helm.sh \
    https://raw.githubusercontent.com/kubernetes/helm/master/scripts/get && \
    chmod 700 /tmp/get_helm.sh && \
    /tmp/get_helm.sh && \
    rm /tmp/get_helm.sh

# Initialize helm
RUN helm init --client-only

RUN easy_install pip
RUN pip install --upgrade six pyyaml google-api-python-client \
    google-cloud-storage pylint kubernetes

RUN mkdir -p /opt/tensorflow_k8s

COPY py /opt/tensorflow_k8s/py/
RUN chmod -R a+rw /opt/tensorflow_k8s

ENV PYTHONPATH $PYTHONAPTH:/opt/tensorflow_k8s
ENV PATH $PATH:/usr/local/go/bin

COPY dags/ /usr/local/airflow/dags
COPY airflow.cfg /usr/local/airflow/
RUN mkdir -p /usr/local/airflow/logs
COPY entrypoint.sh /entrypoint.sh
RUN chmod a+rx /entrypoint.sh && \
    chown airflow /entrypoint.sh && \
    chown -R airflow /usr/local/airflow
USER airflow
ENTRYPOINT ["/entrypoint.sh"]