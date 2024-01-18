import logging
import time

from kubeflow.training import TrainingClient
from kubeflow.training.constants import constants
from test.e2e.constants import TEST_GANG_SCHEDULER_NAME_SCHEDULER_PLUGINS
from test.e2e.constants import DEFAULT_SCHEDULER_PLUGINS_NAME
from test.e2e.constants import TEST_GANG_SCHEDULER_NAME_VOLCANO

logging.basicConfig(format="%(message)s")
logging.getLogger().setLevel(logging.INFO)


def verify_unschedulable_job_e2e(client: TrainingClient, name: str, namespace: str):
    """Verify unschedulable Training Job e2e test."""
    logging.info(f"\n\n\n{client.job_kind} is creating")
    job = client.wait_for_job_conditions(
        name, namespace, expected_conditions={constants.JOB_CONDITION_CREATED}
    )

    logging.info("Checking 3 times that pods are not scheduled")
    for num in range(3):
        logging.info(f"Number of attempts: {int(num)+1}/3")

        # Job should have correct conditions
        if not client.is_job_created(job=job) or client.is_job_running(job=job):
            raise Exception(
                f"{client.job_kind} should be in Created condition. "
                f"{client.job_kind} should not be in Running condition."
            )

        logging.info("Sleeping 5 seconds...")
        time.sleep(5)


def verify_job_e2e(
    client: TrainingClient,
    name: str,
    namespace: str,
    wait_timeout: int = 600,
):
    """Verify Training Job e2e test."""

    # Wait until Job is Succeeded.
    logging.info(f"\n\n\n{client.job_kind} is running")
    job = client.wait_for_job_conditions(name, namespace, wait_timeout=wait_timeout)

    # Job should have Created, Running, and Succeeded conditions.
    conditions = client.get_job_conditions(job=job)
    # If Job is complete fast, it has 2 conditions: Created and Succeeded.
    if len(conditions) < 2:
        raise Exception(f"{client.job_kind} conditions are invalid: {conditions}")

    # Job should have correct conditions.
    if (
        not client.is_job_created(job=job)
        or not client.is_job_succeeded(job=job)
        or client.is_job_running(job=job)
        or client.is_job_restarting(job=job)
        or client.is_job_failed(job=job)
    ):
        raise Exception(
            f"{client.job_kind} should be in Succeeded and Created conditions. "
            f"{client.job_kind} should not be in Running, Restarting, or Failed conditions."
        )


def get_pod_spec_scheduler_name(gang_scheduler_name: str) -> str:
    if gang_scheduler_name == TEST_GANG_SCHEDULER_NAME_SCHEDULER_PLUGINS:
        return DEFAULT_SCHEDULER_PLUGINS_NAME
    elif gang_scheduler_name == TEST_GANG_SCHEDULER_NAME_VOLCANO:
        return TEST_GANG_SCHEDULER_NAME_VOLCANO

    return ""


def print_job_results(client: TrainingClient, name: str, namespace: str):
    # Print Job.
    logging.info(f"\n\n\n{client.job_kind} info")
    logging.info(client.get_job(name, namespace))

    # Print Job pod names.
    logging.info(f"\n\n\n{client.job_kind} pod names")
    logging.info(client.get_job_pod_names(name, namespace))

    # Print Job logs.
    logging.info(f"\n\n\n{client.job_kind} logs")
    client.get_job_logs(name, namespace)
