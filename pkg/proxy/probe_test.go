package proxy

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"monis.app/mlog"

	"github.com/Azure/azure-workload-identity/pkg/webhook"
)

func TestProbe(t *testing.T) {
	os.Setenv(webhook.AzureTenantIDEnvVar, "tenant_id")
	defer os.Unsetenv(webhook.AzureTenantIDEnvVar)

	p := &proxy{logger: mlog.New()}
	handler := buildProxyHandler(p.readyzHandler, nil, nil)

	server := httptest.NewServer(handler)
	defer server.Close()

	if err := probe(server.URL + "/readyz"); err != nil {
		t.Errorf("probe() = %v, want nil", err)
	}
}

func TestProbeError(t *testing.T) {
	os.Setenv(webhook.AzureTenantIDEnvVar, "tenant_id")
	defer os.Unsetenv(webhook.AzureTenantIDEnvVar)

	errHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "oh noes", http.StatusTeapot)
	})

	server := httptest.NewServer(errHandler)
	defer server.Close()

	if err := probe(server.URL + "/readyz"); err == nil {
		t.Errorf("probe() = nil, want error")
	}
}
