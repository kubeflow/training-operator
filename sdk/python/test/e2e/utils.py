import logging

from kubeflow.training import TrainingClient


logging.basicConfig(format="%(message)s")
logging.getLogger().setLevel(logging.INFO)


def verify_job_e2e(
    client: TrainingClient, name: str, namespace: str, job_kind: str, container: str
):
    """Verify Training Job e2e test."""

    # Wait until Job is Succeeded.
    logging.info(f"\n\n\n{job_kind} is running")
    client.wait_for_job_conditions(name, namespace, job_kind)

    # Job should have Created, Running, and Succeeded conditions.
    conditions = client.get_job_conditions(name, namespace, job_kind)
    if len(conditions) != 3:
        raise Exception(f"{job_kind} conditions are invalid: {conditions}")

    # Job should have correct conditions.
    if not client.is_job_created(name, namespace, job_kind):
        raise Exception(f"{job_kind} should be in Created condition")

    if client.is_job_running(name, namespace, job_kind):
        raise Exception(f"{job_kind} should not be in Running condition")

    if client.is_job_restarting(name, namespace, job_kind):
        raise Exception(f"{job_kind} should not be in Restarting condition")

    if not client.is_job_succeeded(name, namespace, job_kind):
        raise Exception(f"{job_kind} should be in Succeeded condition")

    if client.is_job_failed(name, namespace, job_kind):
        raise Exception(f"{job_kind} should not be in Failed condition")

    # Print Job pod names.
    logging.info(f"\n\n\n{job_kind} pod names")
    logging.info(client.get_job_pod_names(name, namespace))

    # Print Job logs.
    logging.info(f"\n\n\n{job_kind} logs")
    client.get_job_logs(name, namespace, container=container)
