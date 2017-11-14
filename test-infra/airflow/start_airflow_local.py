#!/usr/bin/python
"""Run Airflow locally in a Docker container.
"""
import logging
import os
import subprocess

if __name__ == "__main__":
  logging.getLogger().setLevel(logging.INFO)
  # Remove a docker container named Airflow if one exists.
  subprocess.call(["docker", "rm", "-f", "airflow"])

  # Start Airflow in a container.
  command = ["docker", "run", "-d", "-e", "POSTGRES_HOST=postgre",
             "-p", "8080:8080", "--name=airflow"]

  credentials = os.getenv("GOOGLE_APPLICATION_CREDENTIALS", "")
  if credentials:
    logging.info("Using service account: %s", credentials)
    container_path = os.path.join("/keys", os.path.basename(credentials))
    command.extend([
    "-e", "GOOGLE_APPLICATION_CREDENTIALS=" + container_path,
    "-v", os.path.dirname(credentials) + ":/keys"])
  else:
    logging.info("Environment variable GOOGLE_APPLICATION_CREDENTIALS not set. "
                 "You will need to manually obtain credentials inside the "
                 "container. To avoid this set GOOGLE_APPLICATION_CREDENTIALS "
                 "to a valid a service account keyfile.")
  this_dir = os.path.abspath(os.path.dirname(__file__))
  dags_dir = os.path.join(this_dir, "dags")

  command.extend(["-v", dags_dir + ":/usr/local/airflow/dags",
                  "--network=airflow",
                  "gcr.io/mlkube-testing/airflow:latest", "webserver"])
  logging.info("Running:\n%s", " ".join(command))
  subprocess.check_call(command)
