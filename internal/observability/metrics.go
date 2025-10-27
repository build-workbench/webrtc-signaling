package observability

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	WSConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ws_connections",
		Help: "Current number of active WebSocket connections.",
	})

	RoomsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rooms",
		Help: "Current number of active rooms.",
	})

	MessagesInTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "messages_in_total",
		Help: "Total number of messages received from clients.",
	})

	MessagesOutTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "messages_out_total",
		Help: "Total number of messages sent to clients.",
	})

	ErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "errors_total",
		Help: "Total number of errors.",
	}, []string{"code"})
)

func init() {
	prometheus.MustRegister(WSConnections, RoomsGauge, MessagesInTotal, MessagesOutTotal, ErrorsTotal)
}
