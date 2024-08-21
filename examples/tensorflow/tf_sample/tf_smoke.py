"""
Run a distributed TensorFlow program using
MultiWorkerMirroredStrategy to verify we can execute ops.

The program does a simple matrix multiplication.

With MultiWorkerMirroredStrategy, the operations are distributed across multiple workers,
and each worker performs the matrix multiplication. The strategy handles the distribution
of operations and aggregation of results.

This way we can verify that distributed training is working by executing ops on all devices.
"""

import argparse
import time

import numpy as np
import retrying
import tensorflow as tf

# Set up the MultiWorkerMirroredStrategy to distribute computation across multiple workers.
strategy = tf.distribute.MultiWorkerMirroredStrategy()


def parse_args():
    """Parse the command line arguments."""
    parser = argparse.ArgumentParser()

    parser.add_argument(
        "--sleep_secs", default=0, type=int, help="Amount of time to sleep at the end"
    )

    # TODO(jlewi): We ignore unknown arguments because the backend is currently
    # setting some flags to empty values like metadata path.
    args, _ = parser.parse_known_args()
    return args


# Add retries to deal with things like gRPC errors that result in
# UnavailableError.
@retrying.retry(
    wait_exponential_multiplier=1000,
    wait_exponential_max=10000,
    stop_max_delay=60 * 3 * 1000,
)
def matrix_multiplication_fn():
    """
    Perform matrix multiplication on two example matrices using TensorFlow.

    Returns:
        tf.Tensor: The result of the matrix multiplication.
    """
    width = 10
    height = 10
    a = np.arange(width * height).reshape(height, width).astype(np.float32)
    b = np.arange(width * height).reshape(height, width).astype(np.float32)

    # Perform matrix multiplication
    c = tf.matmul(a, b)
    tf.print(f"Result for this device: {c}")

    return c


def run():
    """
    Run the distributed matrix multiplication operation across multiple devices.
    """
    with strategy.scope():
        tf.print(f"Number of devices: {strategy.num_replicas_in_sync}")

        result = strategy.run(matrix_multiplication_fn)

        # Reduce results across devices to get a single result
        reduced_result = strategy.reduce(tf.distribute.ReduceOp.SUM, result, axis=None)
        tf.print(
            "Summed result of matrix multiplication across all devices:", reduced_result
        )


if __name__ == "__main__":
    args = parse_args()

    # Execute the distributed matrix multiplication.
    run()
    if args.sleep_secs:
        print(f"Sleeping for {args.sleep_secs} seconds")
        time.sleep(args.sleep_secs)
