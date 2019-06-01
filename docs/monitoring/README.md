# Prometheus Monitoring for TF operator

## Available Metrics

Currently available metrics to monitor are listed below.

### Metrics for Each Component Container for TF operator

Component Containers:
* tf-operator
* tf-chief
* tf-ps
* tf-worker

#### Each Container Reports on its:

Use prometheus graph to run the following example commands to visualize metrics.

*Note*: These metrics are derived from [cAdvisor](https://github.com/google/cadvisor) kubelet integration which reports to Prometheus through our prometheus-operator installation. You may see a complete list of metrics available in `\metrics` page of your Prometheus web UI which you can further use to compose your own queries.

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

*Note*: Replace `tfjob-name` with your own TF Job name you want to monitor for the example queries above.

### Report TFJob metrics:

**Job Creation**
```
tf_operator_jobs_created
```

**Job Creation**
```
sum (rate (tf_operator_jobs_created[60m]))
```

**Job Deletion**
```
tf_operator_jobs_deleted
```

**Successful Job Completions**
```
tf_operator_jobs_successful
```

**Failed Jobs**
```
tf_operator_jobs_failed
```
