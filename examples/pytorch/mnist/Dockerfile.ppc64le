FROM ppc64le/ubuntu:18.04
WORKDIR /root/
RUN apt-get update && apt-get -y install gcc g++ libjpeg-dev zlib1g-dev cmake
RUN apt-get -y install python3 python3-pip git
RUN pip3 install numpy pyyaml
RUN git clone --recursive https://github.com/pytorch/pytorch && cd pytorch && python3 setup.py install
RUN pip3 install torchvision tensorboardX==1.6.0
WORKDIR /var
ADD mnist.py /var

ENTRYPOINT ["python3", "/var/mnist.py"]
