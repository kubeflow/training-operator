FROM python:3.11-slim-bookworm

WORKDIR /workspace

# Copy the required Python modules.
COPY cmd/initializer_v2/dataset/requirements.txt .
COPY sdk/python/kubeflow sdk/python/kubeflow
COPY pkg/initializer_v2 pkg/initializer_v2

# Install the needed packages.
RUN pip install -r requirements.txt

ENTRYPOINT ["python", "-m", "pkg.initializer_v2.dataset"]
