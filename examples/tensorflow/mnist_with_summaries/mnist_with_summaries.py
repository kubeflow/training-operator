# Copyright 2015 The TensorFlow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the 'License');
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an 'AS IS' BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# ==============================================================================
"""A simple MNIST classifier which displays summaries in TensorBoard.
This is an unimpressive MNIST model, but it is a good example of using
tf.name_scope to make a graph legible in the TensorBoard graph explorer, and of
naming summary tags so that they are grouped meaningfully in TensorBoard.
It demonstrates the functionality of every TensorBoard dashboard.
"""
import argparse
import os

import mnist_utils as helper
import tensorflow as tf

args = None


def init_parser():
    global args
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--fake_data",
        nargs="?",
        const=True,
        type=bool,
        default=False,
        help="If true, uses fake data for unit testing.",
    )
    parser.add_argument(
        "--epochs", type=int, default=5, help="Number of epochs for training."
    )
    parser.add_argument(
        "--learning_rate", type=float, default=0.001, help="Initial learning rate"
    )
    parser.add_argument(
        "--batch_size", type=int, default=64, help="Training batch size"
    )
    parser.add_argument(
        "--dropout",
        type=float,
        default=0.9,
        help="Keep probability for training dropout.",
    )
    parser.add_argument(
        "--data_path",
        type=str,
        default="mnist.npz",
        help="Path where to cache the dataset locally (relative to ~/.keras/datasets).",
    )
    parser.add_argument(
        "--log_dir",
        type=str,
        default=os.path.join(
            os.getenv("TEST_TMPDIR", "/tmp"),
            "tensorflow/mnist/logs/mnist_with_summaries",
        ),
        help="Summaries log directory",
    )
    args = parser.parse_args()
    print(f"Run script with {args=}")


def main():
    """
    The main function to load data, preprocess it, build the model, and train it.
    """
    # Load and preprocess data
    train_ds, test_ds = helper.load_data(
        data_path=args.data_path, fake_data=args.fake_data
    )
    train_ds = helper.preprocess(ds=train_ds, batch_size=args.batch_size)
    test_ds = helper.preprocess(ds=test_ds, batch_size=args.batch_size)

    # Build model
    model = helper.build_model(dropout=args.dropout, learning_rate=args.learning_rate)

    # Setup TensorBoard
    tensorboard_callback = tf.keras.callbacks.TensorBoard(
        log_dir=args.log_dir, histogram_freq=1
    )

    # Train the model
    model.fit(
        train_ds,
        epochs=args.epochs,
        validation_data=test_ds,
        callbacks=[tensorboard_callback],
    )


if __name__ == "__main__":
    init_parser()
    main()
