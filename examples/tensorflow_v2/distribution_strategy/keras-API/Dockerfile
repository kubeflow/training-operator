FROM python:3.9

RUN pip install tensorflow==2.11.0 tensorflow_datasets==4.7.0

COPY examples/tensorflow/distribution_strategy/keras-API/multi_worker_strategy-with-keras.py /
ENTRYPOINT ["python", "/multi_worker_strategy-with-keras.py", "--saved_model_dir", "/train/saved_model/", "--checkpoint_dir", "/train/checkpoint"]
