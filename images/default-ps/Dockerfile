ARG BASE_IMAGE=tensorflow/tensorflow
FROM $BASE_IMAGE
RUN mkdir /app
COPY ./main.py /app/main.py
CMD ["python", "/app/main.py"]