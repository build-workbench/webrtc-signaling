package observability

import (
	"github.com/prometheus/client_golang/prometheus"
)

const metricsNamespace = "signal"

var (
	WSConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "ws_connections",
		Help:      "Current number of active WebSocket connections.",
	})

	RoomsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "rooms",
		Help:      "Current number of active rooms.",
	})

	ParticipantsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "participants",
		Help:      "Current number of participants across all rooms.",
	})

	MessagesInTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Name:      "messages_in_total",
		Help:      "Total number of messages received from clients.",
	})

	MessagesOutTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Name:      "messages_out_total",
		Help:      "Total number of messages sent to clients.",
	})

	ErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Name:      "errors_total",
		Help:      "Total number of errors.",
	}, []string{"code"})

	MessageLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metricsNamespace,
		Name:      "message_latency_seconds",
		Help:      "Latency of message processing in seconds.",
		Buckets:   prometheus.DefBuckets,
	})
)

func init() {
	prometheus.MustRegister(WSConnections, RoomsGauge, ParticipantsGauge, MessagesInTotal, MessagesOutTotal, ErrorsTotal, MessageLatency)
}
