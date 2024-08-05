# example ref:
# https://jax.readthedocs.io/en/latest/multi_process.html#running-multi-process-computations
# https://github.com/GoogleCloudPlatform/ai-on-gke/blob/main/tutorials-and-examples/gpu-examples/a100-jax/train.py # noqa

import os
import socket

import jax
from absl import app

jax.config.update("jax_cpu_collectives_implementation", "gloo")


def _main(argv):

    process_id = int(os.getenv("PROCESS_ID"))
    num_processes = int(os.getenv("NUM_PROCESSES"))
    coordinator_address = os.getenv("COORDINATOR_ADDRESS")
    coordinator_port = int(os.getenv("COORDINATOR_PORT"))
    coordinator_address = f"{coordinator_address}:{coordinator_port}"

    jax.distributed.initialize(
        coordinator_address=coordinator_address,
        num_processes=num_processes,
        process_id=process_id,
    )

    print(
        f"JAX process {jax.process_index()}/{jax.process_count()} initialized on "
        f"{socket.gethostname()}"
    )
    print(f"JAX global devices:{jax.devices()}")
    print(f"JAX local devices:{jax.local_devices()}")

    print(jax.device_count())
    print(jax.local_device_count())

    xs = jax.numpy.ones(jax.local_device_count())
    print(jax.pmap(lambda x: jax.lax.psum(x, "i"), axis_name="i")(xs))


if __name__ == "__main__":
    app.run(_main)
