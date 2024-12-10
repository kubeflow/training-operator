## An MNIST example with single-program multiple-data (SPMD) data parallelism.

The aim here is to illustrate how to use JAX's [`pmap`](https://jax.readthedocs.io/en/latest/_autosummary/jax.pmap.html) to express and execute
[SPMD](https://jax.readthedocs.io/en/latest/glossary.html#term-SPMD) programs for data parallelism along a batch dimension, while also
minimizing dependencies by avoiding the use of higher-level layers and
optimizers libraries.

Adapted from https://github.com/jax-ml/jax/blob/main/examples/spmd_mnist_classifier_fromscratch.py.

```bash
$ kubectl apply -f examples/jax/jax-dist-spmd-mnist/jaxjob_dist_spmd_mnist_gloo.yaml
```

---

```bash
$ kubectl get pods -n kubeflow -l training.kubeflow.org/job-name=jaxjob-mnist
```
---
```bash
$ PODNAME=$(kubectl get pods -l training.kubeflow.org/job-name=jaxjob-mnist,training.kubeflow.org/replica-type=worker,training.kubeflow.org/replica-index=0 -o
name -n kubeflow)
$ kubectl logs -f ${PODNAME} -n kubeflow
```

---

```bash
$ kubectl get -o yaml jaxjobs jaxjob-mnist -n kubeflow
```
