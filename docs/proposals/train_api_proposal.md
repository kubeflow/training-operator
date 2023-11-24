**<h1>Train API Proposal for LLMs</h1>**

**<h3>Authors:</h3>**

* Deepanker Gupta (**[@deepanker13](https://github.com/deepanker13)**), Nutanix
* Johnu George (**[@johnugeorge](https://github.com/johnugeorge)**), Nutanix

**<h3>Status</h3>**

* 10 Nov 2023 (v1)

**<h3>Abstract</h3>**

LLMs are being widely used for generative AI tasks and as their adoption is increasing across various domains, the need for LLMOps in the Kubernetes environment has also risen. There is a need to fine-tune these large models on task-specific datasets. The user/data scientist should be able to do this simplistically in the Kubernetes environment using the Kubeflow training operator SDK without any infrastructure knowledge.

**<h3>Background</h3>**

Currently, there are two flows for data scientists to start using Training operator for their distributed training needs. 

**<h4>Traditional method</h4>**

1. The end-user has to write a custom script that includes data fetching, model loading, and a training loop using pytorch libraries.
2. Create an image and push it to a registry.
3. Using the Kubeflow train sdk to create worker spec and master spec which uses the above image and using these two specs create a pytorch job.

This method requires infrastructure knowledge including image creation, registry access and training job spec. This makes it difficult for data scientists to purely focus at the application layer.

**<h4>Higher level SDK</h4>**

To provide a better user experience, a new higher level SDK was added in[ https://github.com/kubeflow/training-operator/pull/1659](https://github.com/kubeflow/training-operator/pull/1659) which avoided the need of image creation. Instead of creating the image, we can directly pass the custom script to train_func parameter and final pytorch job is created

```python
training_client.create_job(
   name=pytorchjob_name,
   train_func=train_function,
   num_worker_replicas=3, # How many PyTorch Workers will be created.
)
```

The above method reduces the time taken but still the end user has to write the train_function method.

In many cases like in LLM, training core code template is a standard and it doesn’t require any custom modification. For faster experimentation of such models, an easier api should exist to train/finetune models without writing custom Python code. HuggingFace Trainer class is a good example but it is not easy to set it up in a Kubernetes environment. Training operator can handle this complexity and provide a better abstracted layer to the user.

**<h3>New Train API proposal:</h3>**

```python
from kubeflow.training import TrainingClient
# Arguments related to the model provider including credentials
model_args = modelProviderClass()
# Arguments related to the dataset provider including credentials
dataset_args = datasetProviderClass()
# Arguments related to the trainer code
parameters = {key:value pairs}
trainingClient.train(
   nodes=1, 
   nprocs_per_node=4, 
   model='provider://repo/model_name', 
   dataset= 'provider://train_dataset_path',
   eval_dataset = "provider://eval_dataset_path"
   model_args, 
   dataset_args, 
   parameters
)
```

Example: 

```python
@dataclass
class HuggingFace:
    access_token = field()
    peft_config = field()
    transformerClass = field()

@dataclass
class S3:
    access_token = field()
    region = field()
 
trainingClient.train(
   nodes=1, 
   nprocs_per_node=4, 
   model='hf://openchat/openchat_3.5', 
   dataset= 's3://doc-example-bucket1/train_dataset',
   eval_dataset = "s3://doc-example-bucket1/eval_dataset"
   HuggingFace(access_token = "hf_..." , peft_config = {lora_alpha: 16}, transformerClass="Trainer"), 
   s3(access_token = "s3 access token", region="us-west-2"), 
   {learning_rate=0.1}
)
```

The new proposed API takes following arguments 

1. System parameters - Number of nodes, number of procs per node(GPUs per node).
2. Model parameters - Model provider and repository details.
3. Dataset parameters - Dataset provider and dataset details.
4. Training parameters - Training specific parameters like learning rate etc.

**<h3>Implementation</h3>**

1. Setup **init** **containers** that download the model and dataset to a PVC. Based on the specified model provider, corresponding training utility functions will be used. Eg: For Huggingface provider, Huggingface trainer can be used. For this **get_pytorchjob_template** function in the sdk needs to be changed to add init containers spec.. Inorder to download models and data sets, we need to support different providers like kaggle, hugging face, s3 or git lfs. The data can be stored in a shared volume between the init container and the main container.
A new folder containing the code for downloading model and dataset can be added to generate the images for init_containers. Abstract classes will be used as base to create when dataset download, model download and training loop is written for base images.

```
sdk/python
        -> kubeflow
            -> model_init_container_images
                -> abstract_model_provider.py
                -> hugging_face.py
            -> dataset_init_container_images
                -> abstract_data_provider.py
                -> s3.py
```
```python
# code present in abstract_model_provider.py
class modelProvider():
    @abstractmethod
    def download_model(self):
        pass

# code present in hugging_face.py
class HuggingFace(modelProvider):
    def download_model(self):
        # implementation for downloading the model

```

2. Currently, **create_job** api doesn’t support **num_of_nodes** and **gpus_per_node.** We need to add support for that as well, so that the pytorch job with the spec mentioned in [https://github.com/kubeflow/training-operator/issues/1872#issue comment-1659445716](https://github.com/kubeflow/training-operator/issues/1872#issuecomment-1659445716) can be created.

```python
training_client.create_job(name="pytorchjob_name",train_func=custom_training_function, num_of_nodes=1, gpus_per_node = 4)
```

3. We can provide the training function as a **custom_training_function** argument or inside the **base_image** argument of the **create_job** API directly. In case of Hugging Face models, we can use Hugging Face Transformer library’s Trainer class as the training function. 

4. The launch command of the training job needs to be changed to torchrun to take **nnodes** and **nproc_per_node**  into effect inside **get_pod_template_spec** function in the training operator SDK.      

```python
exec_script = textwrap.dedent(
   """
   program_path=$(mktemp -d)
   read -r -d '' SCRIPT << EOM\n
   {func_code}
   EOM
   printf "%s" \"$SCRIPT\" > \"$program_path/ephemeral_script.py\"
   "torchrun", "$program_path/ephemeral_script.py\
   """"
   )
```

**<h3>Limitations</h3>**
1. The dataset is assumed to be preprocessed by the user.

2. Currently only pytorch framework will be supported for running distributed training.