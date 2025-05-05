// apps/websocket-signaling/internal/metrics/metrics.go
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector defines the interface for metrics collection
type Collector interface {
	// Client metrics
	ClientConnected(streamID, userID string)
	ClientDisconnected(streamID, userID string)
	ClientError(streamID, userID, errorType string)

	// Stream metrics
	StreamRegistered(streamID string)
	StreamUnregistered(streamID string)

	// Signaling metrics
	SignalingMessageReceived(streamID, messageType string, sizeBytes int)
	SignalingMessageSent(streamID, messageType string, sizeBytes int)
	SignalingMessageError(streamID, messageType, errorType string)

	// Handler returns an HTTP handler for metrics endpoint
	Handler() http.Handler
}

// PrometheusCollector implements the Collector interface using Prometheus
type PrometheusCollector struct {
	// Client metrics
	activeClients     prometheus.Gauge
	clientConnections *prometheus.CounterVec
	clientDisconnects *prometheus.CounterVec
	clientErrors      *prometheus.CounterVec

	// Stream metrics
	activeStreams         prometheus.Gauge
	streamRegistrations   *prometheus.CounterVec
	streamUnregistrations *prometheus.CounterVec

	// Signaling metrics
	messagesReceived *prometheus.CounterVec
	messagesSent     *prometheus.CounterVec
	messagesErrors   *prometheus.CounterVec
	messageSize      *prometheus.HistogramVec
}

// NewPrometheusCollector creates a new PrometheusCollector
func NewPrometheusCollector() *PrometheusCollector {
	return &PrometheusCollector{
		// Client metrics
		activeClients: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "signaling_active_clients",
			Help: "Number of active WebSocket clients",
		}),

		clientConnections: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "signaling_client_connections_total",
				Help: "Total number of WebSocket client connections",
			},
			[]string{"stream_id", "user_id"},
		),

		clientDisconnects: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "signaling_client_disconnects_total",
				Help: "Total number of WebSocket client disconnections",
			},
			[]string{"stream_id", "user_id"},
		),

		clientErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "signaling_client_errors_total",
				Help: "Total number of WebSocket client errors",
			},
			[]string{"stream_id", "user_id", "error_type"},
		),

		// Stream metrics
		activeStreams: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "signaling_active_streams",
			Help: "Number of active streams",
		}),

		streamRegistrations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "signaling_stream_registrations_total",
				Help: "Total number of stream registrations",
			},
			[]string{"stream_id"},
		),

		streamUnregistrations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "signaling_stream_unregistrations_total",
				Help: "Total number of stream unregistrations",
			},
			[]string{"stream_id"},
		),

		// Signaling metrics
		messagesReceived: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "signaling_messages_received_total",
				Help: "Total number of WebSocket messages received",
			},
			[]string{"stream_id", "message_type"},
		),

		messagesSent: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "signaling_messages_sent_total",
				Help: "Total number of WebSocket messages sent",
			},
			[]string{"stream_id", "message_type"},
		),

		messagesErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "signaling_message_errors_total",
				Help: "Total number of WebSocket message errors",
			},
			[]string{"stream_id", "message_type", "error_type"},
		),

		messageSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "signaling_message_size_bytes",
				Help:    "Size of WebSocket messages in bytes",
				Buckets: prometheus.ExponentialBuckets(64, 2, 10), // 64B to 32KB
			},
			[]string{"stream_id", "message_type", "direction"},
		),
	}
}

// ClientConnected records a client connection
func (c *PrometheusCollector) ClientConnected(streamID, userID string) {
	c.clientConnections.WithLabelValues(streamID, userID).Inc()
	c.activeClients.Inc()
}

// ClientDisconnected records a client disconnection
func (c *PrometheusCollector) ClientDisconnected(streamID, userID string) {
	c.clientDisconnects.WithLabelValues(streamID, userID).Inc()
	c.activeClients.Dec()
}

// ClientError records a client error
func (c *PrometheusCollector) ClientError(streamID, userID, errorType string) {
	c.clientErrors.WithLabelValues(streamID, userID, errorType).Inc()
}

// StreamRegistered records a stream registration
func (c *PrometheusCollector) StreamRegistered(streamID string) {
	c.streamRegistrations.WithLabelValues(streamID).Inc()
	c.activeStreams.Inc()
}

// StreamUnregistered records a stream unregistration
func (c *PrometheusCollector) StreamUnregistered(streamID string) {
	c.streamUnregistrations.WithLabelValues(streamID).Inc()
	c.activeStreams.Dec()
}

// SignalingMessageReceived records a signaling message being received
func (c *PrometheusCollector) SignalingMessageReceived(streamID, messageType string, sizeBytes int) {
	c.messagesReceived.WithLabelValues(streamID, messageType).Inc()
	c.messageSize.WithLabelValues(streamID, messageType, "received").Observe(float64(sizeBytes))
}

// SignalingMessageSent records a signaling message being sent
func (c *PrometheusCollector) SignalingMessageSent(streamID, messageType string, sizeBytes int) {
	c.messagesSent.WithLabelValues(streamID, messageType).Inc()
	c.messageSize.WithLabelValues(streamID, messageType, "sent").Observe(float64(sizeBytes))
}

// SignalingMessageError records a signaling message error
func (c *PrometheusCollector) SignalingMessageError(streamID, messageType, errorType string) {
	c.messagesErrors.WithLabelValues(streamID, messageType, errorType).Inc()
}

// Handler returns an HTTP handler for metrics endpoint
func (c *PrometheusCollector) Handler() http.Handler {
	return promhttp.Handler()
}
