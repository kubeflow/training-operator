FROM python:3.11-alpine

WORKDIR /workspace

# Copy the required Python modules.
COPY cmd/initializer_v2/dataset/requirements.txt .
COPY pkg/initializer_v2 pkg/initializer_v2

# Install the needed packages.
RUN pip install -r requirements.txt

# Copy and install the Kubeflow Training SDK for the configs.
COPY sdk_v2 sdk_v2
COPY LICENSE LICENSE
COPY README.md README.md
RUN pip install ./sdk_v2

ENTRYPOINT ["python", "-m", "pkg.initializer_v2.dataset"]
