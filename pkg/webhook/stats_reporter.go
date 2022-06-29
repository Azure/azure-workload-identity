package webhook

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

const (
	requestDurationMetricName = "azwi_mutation_request"

	namespaceKey = "namespace"
)

var (
	req metric.Float64ValueRecorder
	// if service.name is not specified, the default is "unknown_service:<exe name>"
	// xref: https://opentelemetry.io/docs/reference/specification/resource/semantic_conventions/#service
	labels = []attribute.KeyValue{attribute.String("service.name", "webhook")}
)

// reporter implements StatsReporter.
type reporter struct {
	metric.Meter
}

// StatsReporter reports webhook metrics.
type StatsReporter interface {
	ReportRequest(ctx context.Context, namespace string, duration time.Duration)
}

func newStatsReporter() StatsReporter {
	meter := global.Meter("azure-workload-identity")
	req = metric.Must(meter).NewFloat64ValueRecorder(requestDurationMetricName,
		metric.WithDescription("Distribution of how long it took for the azure-workload-identity mutation request"))
	return &reporter{meter}
}

// ReportRequest reports the request duration for the given namespace.
func (r *reporter) ReportRequest(ctx context.Context, namespace string, duration time.Duration) {
	l := append(labels, attribute.String(namespaceKey, namespace))
	r.RecordBatch(ctx, l, req.Measurement(duration.Seconds()))
}
