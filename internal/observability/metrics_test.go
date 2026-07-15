package observability

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDependencyAndAuditMetrics(t *testing.T) {
	metrics := NewMetrics()
	metrics.ObserveDatabaseQuery("insert", nil, 10*time.Millisecond, true)
	metrics.ObserveArtifactOperation("get", errors.New("unavailable"), 20*time.Millisecond)

	response := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))
	body := response.Body.String()
	for _, expected := range []string{
		`areaflow_dependency_operations_total{dependency="database",operation="insert",status="ok"} 1`,
		`areaflow_dependency_operations_total{dependency="artifact_store",operation="get",status="error"} 1`,
		`areaflow_audit_writes_total 1`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics output missing %q", expected)
		}
	}
}
