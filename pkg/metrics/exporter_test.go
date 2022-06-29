package metrics

import "testing"

func TestInitMetricsExporter(t *testing.T) {
	tests := []struct {
		name           string
		metricsBackend string
	}{
		{
			name:           "prometheus",
			metricsBackend: "prometheus",
		},
		{
			name:           "Prometheus",
			metricsBackend: "Prometheus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := InitMetricsExporter(tt.metricsBackend); err != nil {
				t.Errorf("InitMetricsExporter() error = %v, expected nil", err)
			}
		})
	}
}

func TestInitMetricsExporterError(t *testing.T) {
	if err := InitMetricsExporter("unknown"); err == nil {
		t.Errorf("InitMetricsExporter() error = nil, expected error")
	}
}
