package observability

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics defines the interface for recording observability metrics.
// This interface allows for dependency injection and easier testing.
type Metrics interface {
	// Connection metrics
	IncConnections()
	DecConnections()

	// Room metrics
	SetRooms(count int)
	IncParticipants()
	DecParticipants()

	// Message metrics
	IncMessagesIn()
	IncMessagesOut()

	// Error metrics
	IncError(code int)

	// Latency metrics
	RecordLatency(seconds float64)
}

// PrometheusMetrics implements Metrics using Prometheus collectors.
type PrometheusMetrics struct {
	wsConnections  prometheus.Gauge
	roomsGauge     prometheus.Gauge
	participants   prometheus.Gauge
	messagesIn     prometheus.Counter
	messagesOut    prometheus.Counter
	errorsTotal    *prometheus.CounterVec
	messageLatency prometheus.Histogram
}

// NewPrometheusMetrics creates a new PrometheusMetrics instance.
// It registers all metrics with the default Prometheus registry.
func NewPrometheusMetrics() *PrometheusMetrics {
	const ns = "signal"

	m := &PrometheusMetrics{
		wsConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "ws_connections",
			Help:      "Current number of active WebSocket connections.",
		}),
		roomsGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "rooms",
			Help:      "Current number of active rooms.",
		}),
		participants: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "participants",
			Help:      "Current number of participants across all rooms.",
		}),
		messagesIn: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "messages_in_total",
			Help:      "Total number of messages received from clients.",
		}),
		messagesOut: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "messages_out_total",
			Help:      "Total number of messages sent to clients.",
		}),
		errorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "errors_total",
			Help:      "Total number of errors.",
		}, []string{"code"}),
		messageLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "message_latency_seconds",
			Help:      "Latency of message processing in seconds.",
			Buckets:   prometheus.DefBuckets,
		}),
	}

	prometheus.MustRegister(
		m.wsConnections,
		m.roomsGauge,
		m.participants,
		m.messagesIn,
		m.messagesOut,
		m.errorsTotal,
		m.messageLatency,
	)

	return m
}

func (m *PrometheusMetrics) IncConnections()     { m.wsConnections.Inc() }
func (m *PrometheusMetrics) DecConnections()     { m.wsConnections.Dec() }
func (m *PrometheusMetrics) SetRooms(count int)  { m.roomsGauge.Set(float64(count)) }
func (m *PrometheusMetrics) IncParticipants()    { m.participants.Inc() }
func (m *PrometheusMetrics) DecParticipants()    { m.participants.Dec() }
func (m *PrometheusMetrics) IncMessagesIn()      { m.messagesIn.Inc() }
func (m *PrometheusMetrics) IncMessagesOut()     { m.messagesOut.Inc() }
func (m *PrometheusMetrics) IncError(code int) { m.errorsTotal.WithLabelValues(strconv.Itoa(code)).Inc() }
func (m *PrometheusMetrics) RecordLatency(s float64) { m.messageLatency.Observe(s) }

// NoopMetrics implements Metrics with no-op operations, useful for testing.
type NoopMetrics struct{}

func NewNoopMetrics() *NoopMetrics { return &NoopMetrics{} }

func (m *NoopMetrics) IncConnections()    {}
func (m *NoopMetrics) DecConnections()    {}
func (m *NoopMetrics) SetRooms(_ int)     {}
func (m *NoopMetrics) IncParticipants()   {}
func (m *NoopMetrics) DecParticipants()   {}
func (m *NoopMetrics) IncMessagesIn()     {}
func (m *NoopMetrics) IncMessagesOut()    {}
func (m *NoopMetrics) IncError(_ int)     {}
func (m *NoopMetrics) RecordLatency(_ float64) {}
