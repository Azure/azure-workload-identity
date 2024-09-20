package webhook

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	requestDurationMetricName = "azwi_mutation_request"

	namespaceKey = "namespace"
)

var (
	req metric.Float64Histogram
	// if service.name is not specified, the default is "unknown_service:<exe name>"
	// xref: https://opentelemetry.io/docs/reference/specification/resource/semantic_conventions/#service
	labels = []attribute.KeyValue{attribute.String("service.name", "webhook")}
)

func registerMetrics() error {
	var err error
	meter := otel.Meter("webhook")

	req, err = meter.Float64Histogram(
		requestDurationMetricName,
		metric.WithDescription("Distribution of how long it took for the azure-workload-identity mutation request"))

	return err
}

// ReportRequest reports the request duration for the given namespace.
func ReportRequest(ctx context.Context, namespace string, duration time.Duration) {
	l := append(labels, attribute.String(namespaceKey, namespace))
	req.Record(ctx, duration.Seconds(), metric.WithAttributes(l...))
}
