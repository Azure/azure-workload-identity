package proxy

import (
	"testing"

	"monis.app/mlog"
)

func TestProbe(t *testing.T) {
	setup()
	defer teardown()

	p := &proxy{logger: mlog.New()}
	rtr.PathPrefix("/readyz").HandlerFunc(p.readyzHandler)

	if err := probe(server.URL + "/readyz"); err != nil {
		t.Errorf("probe() = %v, want nil", err)
	}
}

func TestProbeError(t *testing.T) {
	setup()
	defer teardown()

	if err := probe(server.URL + "/readyz"); err == nil {
		t.Errorf("probe() = nil, want error")
	}
}
