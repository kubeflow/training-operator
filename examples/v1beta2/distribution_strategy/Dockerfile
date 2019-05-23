FROM tensorflow/tensorflow:1.11.0

COPY keras_model_to_estimator.py /
ENTRYPOINT ["python", "/keras_model_to_estimator.py", "/tmp/tfkeras_example/"]
