package observability

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/areasong/areaflow/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry         *prometheus.Registry
	requests         *prometheus.CounterVec
	duration         *prometheus.HistogramVec
	inflight         prometheus.Gauge
	commands         *prometheus.CounterVec
	oidcRequests     *prometheus.CounterVec
	sseConnections   prometheus.Gauge
	dependencyReady  *prometheus.GaugeVec
	dependencyOps    *prometheus.CounterVec
	dependencyTime   *prometheus.HistogramVec
	auditWrites      prometheus.Counter
	databaseAttached bool
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	metrics := &Metrics{
		registry:        registry,
		requests:        prometheus.NewCounterVec(prometheus.CounterOpts{Name: "areaflow_http_requests_total", Help: "AreaFlow HTTP requests."}, []string{"method", "route", "status"}),
		duration:        prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "areaflow_http_request_duration_seconds", Help: "AreaFlow HTTP request duration.", Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}}, []string{"method", "route"}),
		inflight:        prometheus.NewGauge(prometheus.GaugeOpts{Name: "areaflow_http_inflight_requests", Help: "Current AreaFlow HTTP requests."}),
		commands:        prometheus.NewCounterVec(prometheus.CounterOpts{Name: "areaflow_command_requests_total", Help: "AreaFlow mutating command HTTP requests."}, []string{"route", "status"}),
		oidcRequests:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "areaflow_oidc_requests_total", Help: "AreaFlow OIDC login and callback requests."}, []string{"route", "status"}),
		sseConnections:  prometheus.NewGauge(prometheus.GaugeOpts{Name: "areaflow_sse_connections", Help: "Current AreaFlow SSE connections."}),
		dependencyReady: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "areaflow_dependency_ready", Help: "AreaFlow dependency readiness, where 1 is ready."}, []string{"dependency"}),
		dependencyOps:   prometheus.NewCounterVec(prometheus.CounterOpts{Name: "areaflow_dependency_operations_total", Help: "AreaFlow database and artifact store operations."}, []string{"dependency", "operation", "status"}),
		dependencyTime:  prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "areaflow_dependency_operation_duration_seconds", Help: "AreaFlow dependency operation duration.", Buckets: prometheus.DefBuckets}, []string{"dependency", "operation"}),
		auditWrites:     prometheus.NewCounter(prometheus.CounterOpts{Name: "areaflow_audit_writes_total", Help: "Successful audit event insert statements observed by the API database pool."}),
	}
	registry.MustRegister(metrics.requests, metrics.duration, metrics.inflight, metrics.commands, metrics.oidcRequests, metrics.sseConnections, metrics.dependencyReady, metrics.dependencyOps, metrics.dependencyTime, metrics.auditWrites, prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	return metrics
}

func (m *Metrics) ObserveDatabaseQuery(operation string, err error, duration time.Duration, auditWrite bool) {
	m.observeDependencyOperation("database", operation, err, duration)
	if auditWrite {
		m.auditWrites.Inc()
	}
}

func (m *Metrics) ObserveArtifactOperation(operation string, err error, duration time.Duration) {
	m.observeDependencyOperation("artifact_store", operation, err, duration)
}

func (m *Metrics) observeDependencyOperation(dependency, operation string, err error, duration time.Duration) {
	status := "ok"
	if err != nil {
		status = "error"
	}
	m.dependencyOps.WithLabelValues(dependency, operation, status).Inc()
	m.dependencyTime.WithLabelValues(dependency, operation).Observe(duration.Seconds())
}

func (m *Metrics) AttachDatabase(pool *pgxpool.Pool) {
	if pool == nil || m.databaseAttached {
		return
	}
	m.databaseAttached = true
	gauges := []prometheus.Collector{
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "areaflow_database_pool_acquired_connections", Help: "Acquired PostgreSQL pool connections."}, func() float64 { return float64(pool.Stat().AcquiredConns()) }),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "areaflow_database_pool_idle_connections", Help: "Idle PostgreSQL pool connections."}, func() float64 { return float64(pool.Stat().IdleConns()) }),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "areaflow_database_pool_total_connections", Help: "Total PostgreSQL pool connections."}, func() float64 { return float64(pool.Stat().TotalConns()) }),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "areaflow_database_pool_max_connections", Help: "Configured PostgreSQL pool maximum."}, func() float64 { return float64(pool.Config().MaxConns) }),
	}
	m.registry.MustRegister(gauges...)
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{EnableOpenMetrics: true})
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		m.inflight.Inc()
		defer m.inflight.Dec()
		writer := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		isSSE := strings.Contains(r.URL.Path, "/events/stream")
		if isSSE {
			m.sseConnections.Inc()
			defer m.sseConnections.Dec()
		}
		next.ServeHTTP(writer, r)
		route := routeLabel(r.URL.Path)
		m.requests.WithLabelValues(r.Method, route, strconv.Itoa(writer.status)).Inc()
		m.duration.WithLabelValues(r.Method, route).Observe(time.Since(started).Seconds())
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			m.commands.WithLabelValues(route, strconv.Itoa(writer.status)).Inc()
		}
		if strings.Contains(r.URL.Path, "/auth/oidc/") {
			m.oidcRequests.WithLabelValues(route, strconv.Itoa(writer.status)).Inc()
		}
	})
}

func (m *Metrics) ObserveDependency(name string, ready bool) {
	value := 0.0
	if ready {
		value = 1
	}
	m.dependencyReady.WithLabelValues(name).Set(value)
}

func StartMetricsServer(ctx context.Context, cfg config.ObservabilityConfig, metrics *Metrics) error {
	listener, err := net.Listen("tcp", net.JoinHostPort(cfg.MetricsHost, cfg.MetricsPort))
	if err != nil {
		return fmt.Errorf("listen metrics: %w", err)
	}
	server := &http.Server{Handler: metrics.Handler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	go func() { _ = server.Serve(listener) }()
	return nil
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *statusWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func routeLabel(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for index, part := range parts {
		if _, err := strconv.ParseInt(part, 10, 64); err == nil {
			parts[index] = "{id}"
		}
		if index > 0 {
			switch parts[index-1] {
			case "projects":
				parts[index] = "{project_key}"
			case "workflow-versions":
				parts[index] = "{version}"
			case "workers":
				parts[index] = "{worker}"
			}
		}
	}
	return "/" + strings.Join(parts, "/")
}
