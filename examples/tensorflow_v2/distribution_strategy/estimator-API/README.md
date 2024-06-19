# Distributed Training on Kubeflow

This is an example of running distributed training on Kubeflow. The source code is taken from
TensorFlow team's example [here](https://github.com/tensorflow/ecosystem/tree/master/distribution_strategy).

The directory contains the following files:
* Dockerfile: Builds the independent worker image.
* Makefile: For building the above image.
* keras_model_to_estimator.py: This is the model code to run multi-worker training. Identical to the TensorFlow example.
* distributed_tfjob.yaml: The TFJob spec.

To run the example, edit `distributed_tfjob.yaml` for your cluster's namespace. Then run
```
kubectl apply -f distributed_tfjob.yaml
```
to create the job.

Then use
```
kubectl -n ${NAMESPACE} describe tfjob distributed-training
```
to see the status.
