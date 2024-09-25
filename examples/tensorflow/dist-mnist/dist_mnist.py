# Copyright 2016 The TensorFlow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# ==============================================================================
"""Distributed MNIST training and validation, with model replicas, using Parameter Server Strategy.

A Sequential model with a Flatten layer, a Dense layer (128 ReLU units),
Dropout for regularization, and a final Dense layer with 10 softmax units for classification.
The parameters (weights and biases) are located on one parameter server (ps), while the ops
are executed on two worker nodes by default. The TF sessions also run on the
worker node.
This script can be run with multiple workers and parameter servers, with at least
one chief, one worker, and one parameter server.

The coordination between the multiple worker invocations occurs due to
the definition of the parameters on the same ps devices. The parameter updates
from one worker is visible to all other workers. As such, the workers can
perform forward computation and gradient calculation in parallel, which
should lead to increased training speed for the simple model.
"""

import argparse
import os
import time

import mnist_utils as helper
import tensorflow as tf

args = None


def init_parser():
    global args
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--data_path",
        type=str,
        default="mnist.npz",
        help="Path where to cache the dataset locally (relative to ~/.keras/datasets).",
    )
    parser.add_argument(
        "--dropout",
        type=float,
        default=0.9,
        help="Keep probability for training dropout",
    )
    parser.add_argument(
        "--batch_size", type=int, default=100, help="Training batch size"
    )
    parser.add_argument(
        "--learning_rate", type=float, default=0.001, help="Learning rate"
    )
    parser.add_argument(
        "--epochs", type=int, default=5, help="Number of epochs for training"
    )
    parser.add_argument(
        "--fake_data",
        nargs="?",
        const=True,
        type=bool,
        default=False,
        help="If true, uses fake data for unit testing.",
    )
    args = parser.parse_args()
    print(f"Run script with {args=}")


def main():
    # Set the environment variable to allow reporting worker and ps failure to the
    # coordinator. This is a workaround and won't be necessary in the future.
    os.environ["GRPC_FAIL_FAST"] = "use_caller"

    cluster_resolver = tf.distribute.cluster_resolver.TFConfigClusterResolver()
    print(f"{cluster_resolver=}")

    # Get the cluster specification
    cluster_spec = cluster_resolver.cluster_spec()

    # Get the number of PS replicas (parameter servers)
    if "ps" in cluster_spec.jobs:
        num_ps = cluster_spec.num_tasks("ps")
        print(f"Number of PS replicas: {num_ps}")
    else:
        raise Exception("No PS replicas found in the cluster configuration.")

    if cluster_resolver.task_type in ("worker", "ps"):
        # Start a TensorFlow server and wait.
        server = tf.distribute.Server(
            cluster_spec,
            job_name=cluster_resolver.task_type,
            task_index=cluster_resolver.task_id,
            protocol=cluster_resolver.rpc_layer or "grpc",
            start=True,
        )
        server.join()
    else:
        # Run the coordinator.

        # Configure ParameterServerStrategy
        variable_partitioner = (
            tf.distribute.experimental.partitioners.MinSizePartitioner(
                min_shard_bytes=(256 << 10), max_shards=num_ps
            )
        )

        strategy = tf.distribute.ParameterServerStrategy(
            cluster_resolver, variable_partitioner=variable_partitioner
        )

        # Load and preprocess data
        train_ds, test_ds = helper.load_data(
            fake_data=args.fake_data, data_path=args.data_path, repeat=True
        )
        train_ds = helper.preprocess(ds=train_ds, batch_size=args.batch_size)
        test_ds = helper.preprocess(ds=test_ds, batch_size=args.batch_size)

        # Distribute training across workers
        with strategy.scope():
            model = helper.build_model(
                dropout=args.dropout,
                learning_rate=args.learning_rate,
            )

        # Start training
        time_begin = time.time()
        print(f"Training begins @ {time.ctime(time_begin)}")

        model.fit(
            train_ds,
            batch_size=args.batch_size,
            epochs=args.epochs,
            steps_per_epoch=6000 // args.batch_size * 2,
        )

        time_end = time.time()
        print(f"Training ends @ {time.ctime(time_end)}")
        training_time = time_end - time_begin
        print(f"Training elapsed time: {training_time} s")

        # Validation
        coordinator = tf.distribute.coordinator.ClusterCoordinator(strategy)
        with strategy.scope():
            eval_accuracy = tf.keras.metrics.Accuracy()

        @tf.function
        def eval_step(iterator):
            """
            Perform an evaluation step across replicas.

            Args:
                iterator: An iterator for the evaluation dataset.
            """

            def replica_fn(batch_data, labels):
                # Generates output predictions
                pred = model(batch_data, training=False)
                # Get the predicted class by taking the argmax over the class probabilities (axis=1)
                predicted_class = tf.argmax(pred, axis=1, output_type=tf.int64)
                eval_accuracy.update_state(labels, predicted_class)

            batch_data, labels = next(iterator)
            # Run the function on all workers using strategy.run
            strategy.run(replica_fn, args=(batch_data, labels))

        # Prepare the per-worker evaluation dataset and iterator
        per_worker_eval_dataset = coordinator.create_per_worker_dataset(test_ds)
        per_worker_eval_iterator = iter(per_worker_eval_dataset)

        # Calculate evaluation steps per epoch (e.g., based on dataset size and batch size)
        eval_steps_per_epoch = 10000 // args.batch_size * 2

        # Loop through the evaluation steps, scheduling them across the workers
        for _ in range(eval_steps_per_epoch):
            coordinator.schedule(eval_step, args=(per_worker_eval_iterator,))

        # Wait for all scheduled evaluation steps to complete
        coordinator.join()

        # Print the evaluation result (accuracy)
        print("Evaluation accuracy: %f" % eval_accuracy.result())


if __name__ == "__main__":
    init_parser()
    main()
