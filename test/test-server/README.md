# Test Server

This directory contains a simply python test server. This server is intended
for use in E2E tests. The server is intended to run as the program invoked in the TFJob replicas.
The server provides handlers like "/quit" that all allow the test harness to control what the
process does (e.g. exit). This allows the test runner to create conditions intended to test
various behaviors like restarts.