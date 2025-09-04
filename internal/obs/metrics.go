// Copyright (c) 2024 Abhishek2095
// SPDX-License-Identifier: MIT

package obs

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// Command metrics
	CommandsTotal    *prometheus.CounterVec
	CommandDuration  *prometheus.HistogramVec
	CommandsInFlight prometheus.Gauge

	// Connection metrics
	ConnectionsTotal   prometheus.Counter
	ConnectionsCurrent prometheus.Gauge

	// Storage metrics
	KeysTotal        prometheus.Gauge
	ExpiredKeysTotal prometheus.Counter
	MemoryUsage      prometheus.Gauge

	// Server metrics
	UptimeSeconds prometheus.Gauge

	registry *prometheus.Registry
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()

	m := &Metrics{
		CommandsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kvstash_commands_total",
				Help: "Total number of commands processed",
			},
			[]string{"command", "status"},
		),
		CommandDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kvstash_command_duration_seconds",
				Help:    "Command processing duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
			},
			[]string{"command"},
		),
		CommandsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "kvstash_commands_in_flight",
				Help: "Number of commands currently being processed",
			},
		),
		ConnectionsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "kvstash_connections_total",
				Help: "Total number of connections accepted",
			},
		),
		ConnectionsCurrent: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "kvstash_connections_current",
				Help: "Current number of open connections",
			},
		),
		KeysTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "kvstash_keys_total",
				Help: "Total number of keys in the database",
			},
		),
		ExpiredKeysTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "kvstash_expired_keys_total",
				Help: "Total number of keys that have expired",
			},
		),
		MemoryUsage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "kvstash_memory_usage_bytes",
				Help: "Memory usage in bytes",
			},
		),
		UptimeSeconds: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "kvstash_uptime_seconds",
				Help: "Server uptime in seconds",
			},
		),
		registry: registry,
	}

	// Register all metrics
	registry.MustRegister(
		m.CommandsTotal,
		m.CommandDuration,
		m.CommandsInFlight,
		m.ConnectionsTotal,
		m.ConnectionsCurrent,
		m.KeysTotal,
		m.ExpiredKeysTotal,
		m.MemoryUsage,
		m.UptimeSeconds,
	)

	return m
}

// RecordCommand records metrics for a command execution
func (m *Metrics) RecordCommand(command string, duration time.Duration, success bool) {
	status := "success"
	if !success {
		status = "error"
	}

	m.CommandsTotal.WithLabelValues(command, status).Inc()
	m.CommandDuration.WithLabelValues(command).Observe(duration.Seconds())
}

// IncCommandsInFlight increments commands in flight
func (m *Metrics) IncCommandsInFlight() {
	m.CommandsInFlight.Inc()
}

// DecCommandsInFlight decrements commands in flight
func (m *Metrics) DecCommandsInFlight() {
	m.CommandsInFlight.Dec()
}

// IncConnections increments connection counters
func (m *Metrics) IncConnections() {
	m.ConnectionsTotal.Inc()
	m.ConnectionsCurrent.Inc()
}

// DecConnections decrements current connections
func (m *Metrics) DecConnections() {
	m.ConnectionsCurrent.Dec()
}

// SetKeys updates the total number of keys
func (m *Metrics) SetKeys(count int64) {
	m.KeysTotal.Set(float64(count))
}

// IncExpiredKeys increments expired keys counter
func (m *Metrics) IncExpiredKeys() {
	m.ExpiredKeysTotal.Inc()
}

// SetMemoryUsage updates memory usage metric
func (m *Metrics) SetMemoryUsage(bytes int64) {
	m.MemoryUsage.Set(float64(bytes))
}

// SetUptime updates uptime metric
func (m *Metrics) SetUptime(uptime time.Duration) {
	m.UptimeSeconds.Set(uptime.Seconds())
}

// Handler returns the HTTP handler for metrics
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// StartMetricsServer starts the Prometheus metrics HTTP server
func (m *Metrics) StartMetricsServer(addr string, logger *Logger) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", m.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	logger.Info("Starting metrics server", "addr", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return server.ListenAndServe()
}
