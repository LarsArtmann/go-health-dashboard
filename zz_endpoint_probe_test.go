package dashboard

import (
	"context"
	"os"
	"testing"
)

func TestZZEndpointProbe(t *testing.T) {
	vm := goldenViewModel(t)
	vm.ExportURL = "/health/export"
	vm.TrendURL = "/health/trend"
	vm.MetricsURL = "/health/metrics"

	file, err := os.Create("/tmp/endpoint-" + os.Getenv("PROBE_TAG") + ".html")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer file.Close()

	if err := dashboardContent(vm).Render(context.Background(), file); err != nil {
		t.Fatalf("render: %v", err)
	}
}
