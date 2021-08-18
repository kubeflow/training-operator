FROM nvidia/cuda:10.0-cudnn7-devel-ubuntu16.04

# Download usefull tools and mxnet, tvm
WORKDIR /home/scripts
RUN apt-get update && apt-get install -y git vim cmake wget sed && \
    git clone --recursive https://github.com/dmlc/tvm && \
    git clone --recursive https://github.com/apache/incubator-mxnet mxnet

# Download necessary dependence
RUN apt-get update && \
    apt-get install -y python3 python3-dev python3-setuptools gcc libtinfo-dev zlib1g-dev && \
    apt-get install -y python3-pip && \
    apt-get install -y build-essential

# mxnet dependence
RUN apt-get install -y libopenblas-dev liblapack-dev && \
    apt-get install -y libopencv-dev

# tvm dependence
RUN pip3 install --user numpy decorator && \
    pip3 install --user tornado psutil xgboost

# get llvm 4.0.0 for tvm
RUN wget http://releases.llvm.org/4.0.0/clang+llvm-4.0.0-x86_64-linux-gnu-ubuntu-16.04.tar.xz && \
    tar -xf clang+llvm-4.0.0-x86_64-linux-gnu-ubuntu-16.04.tar.xz && \
    mv clang+llvm-4.0.0-x86_64-linux-gnu-ubuntu-16.04 llvm && \
    rm clang+llvm-4.0.0-x86_64-linux-gnu-ubuntu-16.04.tar.xz

# Compile mxnet
RUN cd mxnet && \
    make clean && \
    make -j $(nproc) USE_OPENCV=1 USE_BLAS=openblas USE_DIST_KVSTORE=1 USE_CUDA=1 USE_CUDA_PATH=/usr/local/cuda USE_CUDNN=1

# Install mxnet
RUN cd mxnet/python && \
    pip3 install -e .

# Compile tvm
RUN cd tvm && \
    mkdir build && \
    cp cmake/config.cmake build && \
    cd build && \
    sed -i 's/set(USE_CUDA OFF)/set(USE_CUDA ON)/g' config.cmake && \
    sed -i 's/set(USE_CUDNN OFF)/set(USE_CUDNN ON)/g' config.cmake && \
    sed -i 's/set(USE_CUBLAS OFF)/set(USE_CUBLAS ON)/g' config.cmake && \
    sed -i 's/set(USE_LLVM OFF)/set(USE_LLVM ..\/..\/llvm\/bin\/llvm-config)/g' config.cmake && \
    cmake .. && \
    make -j $(nproc)

# Install tvm
RUN cd tvm && \
    cd python; python3 setup.py install --user; cd .. && \
    cd topi/python; python3 setup.py install --user; cd ../.. && \
    cd nnvm/python; python3 setup.py install --user; cd ../..

# COPY custom code to container
COPY start-job.py .
COPY auto-tuning.py .

# Change working path
WORKDIR /home/log
