// Package observability owns the production telemetry exporters and the
// internal administrative HTTP surface.
package observability

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
)

// Metrics is a process-local OpenTelemetry meter provider backed by an
// isolated Prometheus registry. It deliberately does not mutate the global
// Prometheus registry or OpenTelemetry provider.
type Metrics struct {
	provider *sdkmetric.MeterProvider
	handler  http.Handler
}

func NewMetrics() (*Metrics, error) {
	registry := prometheus.NewRegistry()
	exporter, err := otelprometheus.New(
		otelprometheus.WithRegisterer(registry),
		otelprometheus.WithoutScopeInfo(),
		otelprometheus.WithoutTargetInfo(),
	)
	if err != nil {
		return nil, err
	}
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	return &Metrics{
		provider: provider,
		handler:  promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
	}, nil
}

func (m *Metrics) Provider() *sdkmetric.MeterProvider {
	if m == nil {
		return nil
	}
	return m.provider
}

func (m *Metrics) Handler() http.Handler {
	if m == nil {
		return http.NotFoundHandler()
	}
	return m.handler
}

func (m *Metrics) Close(ctx context.Context) error {
	if m == nil || m.provider == nil {
		return nil
	}
	return m.provider.Shutdown(ctx)
}

// NewAdminHandler exposes only non-secret process administration endpoints.
// Callers must bind it to an internal listener and keep that listener out of
// public ingress.
func NewAdminHandler(metrics http.Handler, readiness func(context.Context) idp.ReadinessReport) (http.Handler, error) {
	if metrics == nil || readiness == nil {
		return nil, errors.New("metrics handler and readiness reporter are required")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status":"alive"}` + "\n"))
	})
	mux.HandleFunc("/readyz", func(writer http.ResponseWriter, request *http.Request) {
		report := readiness(request.Context())
		writer.Header().Set("Content-Type", "application/json")
		status := http.StatusOK
		if !report.Ready {
			status = http.StatusServiceUnavailable
		}
		writer.WriteHeader(status)
		_ = json.NewEncoder(writer).Encode(report)
	})
	mux.Handle("/metrics", metrics)
	return mux, nil
}
