#!/bin/sh

# Run conformance test and generate test report.
pytest test/e2e/ --namespace=kf-conformance -v | tee /tmp/training-operator-conformance.log

# Create the done file.
touch /tmp/training-operator-conformance.done
echo "Done..."

# Keep the container running so the test logs can be downloaded.
while true; do sleep 10000; done