package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
)

func TestAdminHandlerHealthReadinessAndMetrics(t *testing.T) {
	metrics, err := NewMetrics()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, metrics.Close(context.Background())) })

	meter := metrics.Provider().Meter("test")
	counter, err := meter.Int64Counter("tinyidp.test.requests")
	require.NoError(t, err)
	counter.Add(context.Background(), 1)

	ready := true
	handler, err := NewAdminHandler(metrics.Handler(), func(context.Context) idp.ReadinessReport {
		return idp.ReadinessReport{Ready: ready, Checks: []idp.ReadinessCheck{{
			Name: "test", Ready: ready, CheckedAt: time.Unix(1, 0).UTC(),
		}}}
	})
	require.NoError(t, err)

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, health.Code)
	require.JSONEq(t, `{"status":"alive"}`, health.Body.String())

	readiness := httptest.NewRecorder()
	handler.ServeHTTP(readiness, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	require.Equal(t, http.StatusOK, readiness.Code)

	ready = false
	readiness = httptest.NewRecorder()
	handler.ServeHTTP(readiness, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	require.Equal(t, http.StatusServiceUnavailable, readiness.Code)

	scrape := httptest.NewRecorder()
	handler.ServeHTTP(scrape, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, scrape.Code)
	require.True(t, strings.Contains(scrape.Body.String(), "tinyidp_test_requests_total 1"))
}

func TestAdminHandlerRequiresDependencies(t *testing.T) {
	_, err := NewAdminHandler(nil, func(context.Context) idp.ReadinessReport { return idp.ReadinessReport{} })
	require.Error(t, err)
}
