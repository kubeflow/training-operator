# Docker image for Airflow.
# This was based on https://github.com/puckel/docker-airflow.
# However, we modified it because we wanted to use Python2.7 since some
# of our client libraries require it.

FROM python:2.7-slim
MAINTAINER Jeremy Lewi

# Never prompts the user for choices on installation/configuration of packages
ENV DEBIAN_FRONTEND noninteractive
ENV TERM linux

# Airflow
ARG AIRFLOW_HOME=/usr/local/airflow

# Define en_US.
ENV LANGUAGE=en_US.UTF-8 \
    LANG=en_US.UTF-8 \
    LC_ALL=en_US.UTF-8 \
    LC_CTYPE=en_US.UTF-8 \
    LC_MESSAGES=en_US.UTF-8 \
    LC_ALL=en_US.UTF-8


# buildDeps should be packages needed only to build some other packages as
# these packages are purged in a later step.
RUN set -ex \
    && buildDeps=' \
        python2.7-dev \
        libkrb5-dev \
        libsasl2-dev \
        libssl-dev \
        libffi-dev \
        build-essential \
        libblas-dev \
        liblapack-dev \
        libpq-dev \
        libxml2 \
        libxslt-dev libxml2-dev \
    ' \
    && apt-get update -yqq \
    && apt-get install -yqq --no-install-recommends \
        $buildDeps \
        apt-utils \
        curl \
        netcat \
        locales \
        wget \
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
        python2.7 \


    && sed -i 's/^# en_US.UTF-8 UTF-8$/en_US.UTF-8 UTF-8/g' /etc/locale.gen \
    && locale-gen \
    && update-locale LANG=en_US.UTF-8 LC_ALL=en_US.UTF-8 \
    && useradd -ms /bin/bash -d ${AIRFLOW_HOME} airflow \
    && easy_install pip \
    && python -m pip install -U pip setuptools wheel \
    && pip install Cython \
    && pip install pytz \
    && pip install pyOpenSSL \
    && pip install ndg-httpsclient \
    && pip install pyasn1 psycopg2 \
    # Install airflow
    # We need a new version of Airflow to pick up the latest changes supporting
    # Airflow's experimental API.
    && git clone https://github.com/apache/incubator-airflow.git /opt/airflow \
    && cd /opt/airflow \
    && git checkout f87d8aca93cf2c6df21bea7b13b6703d91f09865 \
    && python setup.py install \
    && pip install --upgrade six pyyaml google-api-python-client \
       google-cloud-storage google-auth-httplib2 pylint kubernetes mock \
    # Airflow 1.9.0rc02 requires funcsigs 1.0.0; newer versions cause airflow
    # scheduler to complain; it looks like mock might pull in a newer version.
    # so we force a particular version.
    && pip install funcsigs==1.0.0 \
    && apt-get purge --auto-remove -yqq $buildDeps \
    # Airflow depends on numpy which depends on blas and lapack.
    # Above we installed the dev versions to compile things but then
    # we removed them. So now we install non-dev versions to be used.
    && apt-get install -yqq --no-install-recommends libblas3 liblapack3 \
    && apt-get clean \
    && rm -rf \
        /var/lib/apt/lists/* \
        /tmp/* \
        /var/tmp/* \
        /usr/share/man \
        /usr/share/doc \
        /usr/share/doc-base

# Install  Node.js
RUN curl -sL https://deb.nodesource.com/setup_8.x | bash - \
    && apt-get install -y nodejs

# Install yarn
RUN curl -sS http://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - \
    && echo "deb http://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list \
    && apt-get update -yqq \
    && apt-get install -yqq --no-install-recommends yarn

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

RUN chown -R airflow: ${AIRFLOW_HOME}

EXPOSE 8080 5555 8793

USER airflow
WORKDIR ${AIRFLOW_HOME}

ENTRYPOINT ["/entrypoint.sh"]