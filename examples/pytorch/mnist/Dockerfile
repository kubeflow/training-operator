FROM pytorch/pytorch:2.2.1-cuda12.1-cudnn8-runtime

RUN apt update \
    && apt install --no-install-recommends -y g++

RUN pip install tensorboardX==2.6.2
RUN mkdir -p /opt/mnist

WORKDIR /opt/mnist/src
ADD mnist.py /opt/mnist/src/mnist.py

RUN chgrp -R 0 /opt/mnist \
    && chmod -R g+rwX /opt/mnist

ENTRYPOINT ["python", "/opt/mnist/src/mnist.py"]
