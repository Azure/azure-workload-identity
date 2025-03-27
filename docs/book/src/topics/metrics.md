# Metrics provided by Azure Workload Identity

The Azure Workload Identity mutating admission webhook uses [opentelemetry](https://opentelemetry.io/) for reporting metrics.

Prometheus is the only exporter that's currently supported.

## List of metrics provided by Azure Workload Identity

| Metric                         | Description                                                                       | Tags        |
| ------------------------------ | --------------------------------------------------------------------------------- | ----------- |
| `azwi_mutation_request_bucket` | Distribution of how long it took for the azure-workload-identity mutation request | `namespace` |

Metrics are served from port 8095, but this port is not exposed outside the pod by default. Use kubectl port-forward to access the metrics over localhost:

```bash
kubectl port-forward deploy/azure-wi-webhook-controller-manager -n azure-workload-identity-system 8095:8095 &
curl localhost:8095/metrics
```

### Sample Metrics output

```shell
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.001"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.002"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.003"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.004"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.005"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.006"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.007"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.008"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.009"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.01"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.02"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.03"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.04"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.05"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.06"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.07"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.08"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.09"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.1"} 0
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.2"} 1
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.3"} 1
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.4"} 1
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.5"} 1
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.6"} 1
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.7"} 1
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.8"} 1
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="0.9"} 1
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="1"} 1
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="1.5"} 1
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="2"} 1
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="2.5"} 1
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="3"} 1
azwi_mutation_request_bucket{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0",le="+Inf"} 1
azwi_mutation_request_sum{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0"} 0.104694953
azwi_mutation_request_count{namespace="default",service_name="webhook",telemetry_sdk_language="go",telemetry_sdk_name="opentelemetry",telemetry_sdk_version="0.20.0"} 1
```

Please note that there are many webhook specific metrics that may be useful to monitor as well.  Here are some webhook controller examples:

- `controller_runtime_webhook_requests_total`
- `controller_runtime_webhook_latency_seconds`
- `workqueue_depth`
- `certwatcher_read_certificate_total`
- `certwatcher_read_certificate_errors_total`
- `workqueue_retries_total`

To learn more about these metrics, please see [Default Exported Metrics References](https://book.kubebuilder.io/reference/metrics-reference.html)
