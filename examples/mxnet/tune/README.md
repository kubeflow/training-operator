[mx_job_tune_gpu_v1.yaml](mx_job_tune_gpu_v1.yaml) will pull sample image and run it.

In the sample image, [Apache TVM](https://tvm.apache.org/) and [Apache MXNet](https://mxnet.apache.org/) are pre-installed, you can check out the [Dockerfile](Dockerfile) to get some information.

There two customized scripts in the sample image, [start-job.py](start-job.py) and [auto-tuning.py](auto-tuning.py).
* [start-job.py](start-job.py) is a script tell you how to read the environment variable MX_CONFIG.
* [auto-tuning.py](auto-tuning.py) is a sample script of autotvm, which will tune a `resnet-18` network.
