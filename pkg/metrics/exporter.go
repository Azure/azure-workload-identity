package metrics

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-workload-identity/pkg/metrics/exporters/prometheus"
)

func InitMetricsExporter(metricsBackend string) error {
	mb := strings.ToLower(metricsBackend)
	switch mb {
	// Prometheus is the only exporter for now
	case prometheus.ExporterName:
		return prometheus.InitExporter()
	default:
		return fmt.Errorf("unsupported metrics backend: %v", metricsBackend)
	}
}
