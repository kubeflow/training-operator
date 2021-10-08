# Prometheus Monitoring for TFJob

## Available Metrics

Currently available metrics to monitor are listed below.

### Metrics for Each Component Container for TFJob

Component Containers:

- tf-operator
- tf-chief
- tf-ps
- tf-worker

#### Each Container Reports on its:

Use prometheus graph to run the following example commands to visualize metrics.

_Note_: These metrics are derived from [cAdvisor](https://github.com/google/cadvisor) kubelet integration which reports to Prometheus through our prometheus-operator installation. You may see a complete list of metrics available in `\metrics` page of your Prometheus web UI which you can further use to compose your own queries.

**CPU usage**

```
sum (rate (container_cpu_usage_seconds_total{pod_name=~"tfjob-name-.*"}[1m])) by (pod_name)
```

**GPU Usage**

```
sum (rate (container_accelerator_memory_used_bytes{pod_name=~"tfjob-name-.*"}[1m])) by (pod_name)
```

**Memory Usage**

```
sum (rate (container_memory_usage_bytes{pod_name=~"tfjob-name-.*"}[1m])) by (pod_name)
```

**Network Usage**

```
sum (rate (container_network_transmit_bytes_total{pod_name=~"tfjob-name-.*"}[1m])) by (pod_name)
```

**I/O Usage**

```
sum (rate (container_fs_write_seconds_total{pod_name=~"tfjob-name-.*"}[1m])) by (pod_name)
```

**Keep-Alive check**

```
up
```

This is maintained by Prometheus on its own with its `up` metric detailed in the documentation [here](https://prometheus.io/docs/concepts/jobs_instances/#automatically-generated-labels-and-time-series).

**Is Leader check**

```
tf_operator_is_leader
```

_Note_: Replace `tfjob-name` with your own TF Job name you want to monitor for the example queries above.

### Report TFJob metrics:

_Note_: If you are using release v1 tf-operator, these TFJob metrics don't have suffix `total`. So you have to use metric name like `tf_operator_jobs_created` to get your metrics. See [PR](https://github.com/kubeflow/training-operator/pull/1055) to get more information.

**Job Creation**

```
tf_operator_jobs_created_total
```

**Job Creation**

```
sum (rate (tf_operator_jobs_created_total[60m]))
```

**Job Deletion**

```
tf_operator_jobs_deleted_total
```

**Successful Job Completions**

```
tf_operator_jobs_successful_total
```

**Failed Jobs**

```
tf_operator_jobs_failed_total
```

**Restarted Jobs**

```
tf_operator_jobs_restarted_total
```
