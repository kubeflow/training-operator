FROM python:3.11-alpine

WORKDIR /workspace

# Copy the required Python modules.
COPY cmd/initiailizer_v2/dataset/requirements.txt .
COPY sdk/python/kubeflow sdk/python/kubeflow
COPY pkg/initiailizer_v2 pkg/initiailizer_v2

# Install the needed packages.
RUN pip install -r requirements.txt

ENTRYPOINT ["python", "-m", "pkg.initiailizer_v2.dataset"]
