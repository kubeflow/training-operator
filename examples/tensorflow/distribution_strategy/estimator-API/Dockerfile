FROM tensorflow/tensorflow:1.11.0

COPY examples/tensorflow/distribution_strategy/estimator-API/keras_model_to_estimator.py /
ENTRYPOINT ["python", "/keras_model_to_estimator.py", "/tmp/tfkeras_example/"]
