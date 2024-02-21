FROM pytorch/pytorch:1.2-cuda10.0-cudnn7-runtime

RUN pip install tensorboardX==1.6.0
RUN mkdir -p /opt/mnist

WORKDIR /opt/mnist/src
ADD mnist.py /opt/mnist/src/mnist.py

RUN  chgrp -R 0 /opt/mnist \
  && chmod -R g+rwX /opt/mnist

ENTRYPOINT ["python", "/opt/mnist/src/mnist.py"]
