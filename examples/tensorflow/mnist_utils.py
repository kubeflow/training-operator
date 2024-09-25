# Copyright 2024 The TensorFlow Authors. All Rights Reserved.
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
"""
Utility functions for loading, preprocessing, and building models for MNIST data.

This module provides functions to load the MNIST dataset, preprocess it into TensorFlow datasets,
and build a simple neural network model using TensorFlow's Keras API.
"""

import numpy as np
import tensorflow as tf
from tensorflow.keras.datasets import mnist


def load_data(fake_data=False, data_path=None, repeat=False):
    """
     Loads the MNIST dataset and converts it into TensorFlow datasets.

    Args:
         fake_data (bool): If `True`, loads a fake dataset for testing purposes.
                           If `False`, loads the real MNIST dataset.
         data_path (str, optional): Path where to cache the dataset locally.
                           If `None`, the dataset is loaded to the default location.
         repeat (bool, optional): If `True`, makes the dataset repeat indefinitely.

     Returns:
         train_ds (tf.data.Dataset): Dataset containing the training data (images and labels).
         test_ds (tf.data.Dataset): Dataset containing the test data (images and labels).
    """
    if fake_data:
        (x_train, y_train), (x_test, y_test) = load_fake_data()
    else:
        (x_train, y_train), (x_test, y_test) = (
            mnist.load_data(path=data_path) if data_path else mnist.load_data()
        )
    # Create TensorFlow datasets from the NumPy arrays
    train_ds = tf.data.Dataset.from_tensor_slices((x_train, y_train))
    test_ds = tf.data.Dataset.from_tensor_slices((x_test, y_test))
    if repeat:
        return train_ds.repeat(), test_ds.repeat()
    return train_ds, test_ds


def load_fake_data():
    x_train = np.random.randint(0, 256, (60000, 28, 28)).astype(np.uint8)
    y_train = np.random.randint(0, 10, (60000,)).astype(np.uint8)
    x_test = np.random.randint(0, 256, (10000, 28, 28)).astype(np.uint8)
    y_test = np.random.randint(0, 10, (10000,)).astype(np.uint8)

    return (x_train, y_train), (x_test, y_test)


def build_model(dropout=0.9, learning_rate=0.001):
    """
    Builds a simple neural network model using Keras Sequential API.

    Args:
        dropout (float, optional): Keep probability for training dropout.
        learning_rate (float, optional): The learning rate for the Adam optimizer.

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
                1 - dropout
            ),  # Dropout layer to prevent overfitting
            tf.keras.layers.Dense(
                10, activation="softmax"
            ),  # Output layer with 10 neurons (one for each class)
        ]
    )
    # Define an optimizer with a specific learning rate
    optimizer = tf.keras.optimizers.Adam(learning_rate=learning_rate)
    # Compile the model with Adam optimizer and sparse categorical crossentropy loss
    model.compile(
        optimizer=optimizer,
        loss="sparse_categorical_crossentropy",
        metrics=["accuracy"],
    )
    return model


def preprocess(ds, batch_size):
    """
    Preprocesses the dataset by normalizing the images, shuffling, batching, and prefetching.

    Args:
        ds (tf.data.Dataset): The dataset to preprocess (either training or testing data).
        batch_size (int): The number of samples per batch of data.


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
    ds = ds.shuffle(
        buffer_size=10000
    )  # Shuffle the dataset with a buffer size of 10,000
    ds = ds.batch(batch_size)  # Batch the dataset
    ds = ds.prefetch(
        buffer_size=tf.data.experimental.AUTOTUNE
    )  # Prefetch to improve performance.
    return ds
