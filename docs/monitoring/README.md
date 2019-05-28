# Prometheus Monitoring for TF operator

## Install Prometheus in your Kubernetes Cluster
To install the chart with the release name `my-release`:

```console
$ helm install --name my-release stable/prometheus-operator
```

Follow instructions in this [link](https://github.com/helm/charts/blob/master/stable/prometheus-operator/README.md#installing-the-chart) for elaborate instructions.

*Note*: This [link](https://github.com/coreos/prometheus-operator/blob/master/Documentation/troubleshooting.md) helps in troubleshooting your setup.

## Available Metrics

Currently available metrics to monitor are listed below.

### Metrics for Each Component Container for TF operator

Component Containers:
* tf-operator
* tf-master
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

**Is Leader check**

*Note*: Replace `tfjob-name` with your own TF Job name you want to monitor for the example queries above.

### Report TFJob metrics:

**Job Creation**

**Job Deletion**

**Jobs Created per Hour**

**Successful Job Completions**
