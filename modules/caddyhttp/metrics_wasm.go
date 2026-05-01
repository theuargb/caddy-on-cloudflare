//go:build js && wasm

package caddyhttp

import (
	"context"
	"sync"

	"github.com/caddyserver/caddy/v2"
)

// Metrics is a no-op placeholder for Workers builds.
type Metrics struct {
	PerHost              bool `json:"per_host,omitempty"`
	ObserveCatchallHosts bool `json:"observe_catchall_hosts,omitempty"`
	OTLP                 bool `json:"otlp,omitempty"`

	init        sync.Once
	httpMetrics *httpMetrics
}

type httpMetrics struct{}

func (m *Metrics) provisionOTLP(caddy.Context) error { return nil }
func (m *Metrics) shutdown(context.Context) error    { return nil }
func (m *Metrics) scanConfigForHosts(*App)           {}

func newMetricsInstrumentedRoute(_ caddy.Context, _ string, next Handler, _ *Metrics) Handler {
	return next
}
