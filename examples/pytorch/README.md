## Installation & deployment tips 
1. You need to configure your node to utilize GPU. In order this can be done the following way: 
    * Install [nvidia-docker2](https://github.com/NVIDIA/nvidia-docker)
    * Connect to your MasterNode and set nvidia as the default run in `/etc/docker/daemon.json`:
        ```
        {
            "default-runtime": "nvidia",
            "runtimes": {
                "nvidia": {
                    "path": "/usr/bin/nvidia-container-runtime",
                    "runtimeArgs": []
                }
            }
        }
        ```
    * After that deploy nvidia-daemon to kubernetes: 
        ```bash
        kubectl create -f https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/v1.11/nvidia-device-plugin.yml
        ```
        
2. NVIDIA GPUs can now be consumed via container level resource requirements using the resource name nvidia.com/gpu:
      ```
      resources:
        limits:
            nvidia.com/gpu: 2 # requesting 2 GPUs
      ```

3. Building image. Each example has prebuilt images that are stored on google cloud resources (GCR). If you want to create your own image we recommend using dockerhub. Each example has its own Dockerfile that we strongly advise to use. To build your custom image follow instruction on [TechRepublic](https://www.techrepublic.com/article/how-to-create-a-docker-image-and-push-it-to-docker-hub/).

4. To deploy your job we recommend using official [kubeflow documentation](https://www.kubeflow.org/docs/guides/components/pytorch/). Each example has example yaml files for two versions of apis. Feel free to modify them, e.g. image or number of GPUs.
