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

import numpy as np
import tensorflow as tf
from tensorflow.keras.datasets import mnist


def load_data(fake_data=False):
    """
     Loads the MNIST dataset and converts it into TensorFlow datasets.

    Args:
         fake_data (bool): If `True`, loads a fake dataset for testing purposes.
                           If `False`, loads the real MNIST dataset.

     Returns:
         train_ds (tf.data.Dataset): Dataset containing the training data (images and labels).
         test_ds (tf.data.Dataset): Dataset containing the test data (images and labels).
    """
    if fake_data:
        (x_train, y_train), (x_test, y_test) = load_fake_data()
    else:
        (x_train, y_train), (x_test, y_test) = mnist.load_data(path=FLAGS.data_path)
    # Create TensorFlow datasets from the NumPy arrays
    train_ds = tf.data.Dataset.from_tensor_slices((x_train, y_train))
    test_ds = tf.data.Dataset.from_tensor_slices((x_test, y_test))
    return train_ds, test_ds


def load_fake_data():
    x_train = np.random.randint(0, 256, (60000, 28, 28)).astype(np.uint8)
    y_train = np.random.randint(0, 10, (60000,)).astype(np.uint8)
    x_test = np.random.randint(0, 256, (10000, 28, 28)).astype(np.uint8)
    y_test = np.random.randint(0, 10, (10000,)).astype(np.uint8)

    return (x_train, y_train), (x_test, y_test)


def preprocess(ds):
    """
    Preprocesses the dataset by normalizing the images, shuffling, batching, and prefetching.

    Args:
        ds (tf.data.Dataset): The dataset to preprocess (either training or testing data).

    Returns:
        ds (tf.data.Dataset): The preprocessed dataset.
    """

    def normalize_img(image, label):
        """
        Normalizes images by scaling pixel values from the range [0, 255] to [0, 1].

        Args:
            image (tf.Tensor): The image tensor.
            label (tf.Tensor): The corresponding label tensor.

        Returns:
            tuple: The normalized image and the corresponding label.
        """
        image = tf.cast(image, tf.float32) / 255.0
        return image, label

    # Map the normalization function across the dataset
    ds = ds.map(normalize_img, num_parallel_calls=tf.data.experimental.AUTOTUNE)
    ds = ds.cache()  # Cache the dataset to improve performance
    ds = ds.shuffle(
        buffer_size=10000
    )  # Shuffle the dataset with a buffer size of 10,000
    ds = ds.batch(FLAGS.batch_size)  # Batch the dataset
    ds = ds.prefetch(
        buffer_size=tf.data.experimental.AUTOTUNE
    )  # Prefetch to improve performance
    return ds


def build_model():
    """
    Builds a simple neural network model using Keras Sequential API.

    Returns:
        model (tf.keras.Model): The compiled Keras model.
    """
    model = tf.keras.Sequential(
        [
            tf.keras.layers.Input(
                shape=(28, 28, 1)
            ),  # Input layer with the shape of MNIST images
            tf.keras.layers.Flatten(),
            tf.keras.layers.Dense(
                128, activation="relu"
            ),  # Dense layer with 128 neurons and ReLU activation
            tf.keras.layers.Dropout(
                1 - FLAGS.dropout
            ),  # Dropout layer to prevent overfitting
            tf.keras.layers.Dense(
                10, activation="softmax"
            ),  # Output layer with 10 neurons (one for each class)
        ]
    )
    # Define an optimizer with a specific learning rate
    optimizer = tf.keras.optimizers.Adam(learning_rate=FLAGS.learning_rate)
    # Compile the model with Adam optimizer and sparse categorical crossentropy loss
    model.compile(
        optimizer=optimizer,
        loss="sparse_categorical_crossentropy",
        metrics=["accuracy"],
    )
    return model


def main():
    """
    The main function to load data, preprocess it, build the model, and train it.
    """
    # Load and preprocess data
    train_ds, test_ds = load_data(fake_data=FLAGS.fake_data)
    train_ds = preprocess(train_ds)
    test_ds = preprocess(test_ds)

    # Build model
    model = build_model()

    # Setup TensorBoard
    tensorboard_callback = tf.keras.callbacks.TensorBoard(
        log_dir=FLAGS.log_dir, histogram_freq=1
    )

    # Train the model
    model.fit(
        train_ds,
        epochs=FLAGS.epochs,
        validation_data=test_ds,
        callbacks=[tensorboard_callback],
    )


if __name__ == "__main__":
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
    FLAGS, _ = parser.parse_known_args()
    print(f"Run script with {FLAGS=}")
    main()
