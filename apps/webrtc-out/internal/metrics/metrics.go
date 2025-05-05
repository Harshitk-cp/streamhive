package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector defines the interface for metrics collection
type Collector interface {
	// WebRTC connection metrics
	PeerConnected(streamID, userID string)
	PeerDisconnected(streamID, userID string)
	PeerFailure(streamID, userID, reason string)

	// Frame processing metrics
	FrameReceived(streamID, frameType string, isKeyFrame bool, sizeBytes int)
	FrameProcessed(streamID, frameType string, processingTime time.Duration)
	FrameSent(streamID, userID, frameType string, sizeBytes int, duration time.Duration)
	FrameDropped(streamID, userID, frameType string, reason string)

	// Bandwidth metrics
	BandwidthEstimation(streamID, userID string, bitrateBps float64)
	NetworkStats(streamID, userID string, rtt float64, packetsLost int, fractionLost float64)

	// Error metrics
	ErrorOccurred(streamID, userID, errorType string)

	// HTTP handler for metrics endpoint
	Handler() http.Handler
}

// PrometheusCollector implements the Collector interface using Prometheus
type PrometheusCollector struct {
	// Peer connection metrics
	activePeers     prometheus.Gauge
	peerConnections *prometheus.CounterVec
	peerDisconnects *prometheus.CounterVec
	peerFailures    *prometheus.CounterVec

	// Frame metrics
	framesReceived  *prometheus.CounterVec
	framesProcessed *prometheus.CounterVec
	framesSent      *prometheus.CounterVec
	framesDropped   *prometheus.CounterVec

	// Size metrics
	frameSizeBytes *prometheus.HistogramVec

	// Timing metrics
	frameProcessingTime *prometheus.HistogramVec
	frameSendTime       *prometheus.HistogramVec

	// Bandwidth metrics
	bitrateEstimation *prometheus.GaugeVec
	roundTripTime     *prometheus.GaugeVec
	packetsLost       *prometheus.CounterVec
	fractionLost      *prometheus.GaugeVec

	// Error metrics
	errors *prometheus.CounterVec
}

// NewPrometheusCollector creates a new PrometheusCollector
func NewPrometheusCollector() *PrometheusCollector {
	return &PrometheusCollector{
		// Peer connection metrics
		activePeers: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "webrtc_active_peers",
			Help: "Number of active WebRTC peers",
		}),

		peerConnections: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webrtc_peer_connections_total",
				Help: "Total number of WebRTC peer connections",
			},
			[]string{"stream_id", "user_id"},
		),

		peerDisconnects: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webrtc_peer_disconnects_total",
				Help: "Total number of WebRTC peer disconnections",
			},
			[]string{"stream_id", "user_id"},
		),

		peerFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webrtc_peer_failures_total",
				Help: "Total number of WebRTC peer connection failures",
			},
			[]string{"stream_id", "user_id", "reason"},
		),

		// Frame metrics
		framesReceived: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webrtc_frames_received_total",
				Help: "Total number of frames received",
			},
			[]string{"stream_id", "frame_type", "key_frame"},
		),

		framesProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webrtc_frames_processed_total",
				Help: "Total number of frames processed",
			},
			[]string{"stream_id", "frame_type"},
		),

		framesSent: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webrtc_frames_sent_total",
				Help: "Total number of frames sent",
			},
			[]string{"stream_id", "user_id", "frame_type"},
		),

		framesDropped: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webrtc_frames_dropped_total",
				Help: "Total number of frames dropped",
			},
			[]string{"stream_id", "user_id", "frame_type", "reason"},
		),

		// Size metrics
		frameSizeBytes: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "webrtc_frame_size_bytes",
				Help:    "Size of frames in bytes",
				Buckets: prometheus.ExponentialBuckets(1024, 2, 10), // 1KB to 1MB
			},
			[]string{"stream_id", "frame_type"},
		),

		// Timing metrics
		frameProcessingTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "webrtc_frame_processing_seconds",
				Help:    "Time taken to process frames",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to 1s
			},
			[]string{"stream_id", "frame_type"},
		),

		frameSendTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "webrtc_frame_send_seconds",
				Help:    "Time taken to send frames",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 8), // 1ms to 256ms
			},
			[]string{"stream_id", "user_id", "frame_type"},
		),

		// Bandwidth metrics
		bitrateEstimation: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "webrtc_bitrate_bps",
				Help: "Estimated bitrate in bits per second",
			},
			[]string{"stream_id", "user_id"},
		),

		roundTripTime: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "webrtc_round_trip_time_seconds",
				Help: "WebRTC round trip time in seconds",
			},
			[]string{"stream_id", "user_id"},
		),

		packetsLost: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webrtc_packets_lost_total",
				Help: "Total number of packets lost",
			},
			[]string{"stream_id", "user_id"},
		),

		fractionLost: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "webrtc_fraction_lost",
				Help: "Fraction of packets lost (0.0 to 1.0)",
			},
			[]string{"stream_id", "user_id"},
		),

		// Error metrics
		errors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webrtc_errors_total",
				Help: "Total number of errors",
			},
			[]string{"stream_id", "user_id", "error_type"},
		),
	}
}

// PeerConnected records a peer connection
func (c *PrometheusCollector) PeerConnected(streamID, userID string) {
	c.peerConnections.WithLabelValues(streamID, userID).Inc()
	c.activePeers.Inc()
}

// PeerDisconnected records a peer disconnection
func (c *PrometheusCollector) PeerDisconnected(streamID, userID string) {
	c.peerDisconnects.WithLabelValues(streamID, userID).Inc()
	c.activePeers.Dec()
}

// PeerFailure records a peer connection failure
func (c *PrometheusCollector) PeerFailure(streamID, userID, reason string) {
	c.peerFailures.WithLabelValues(streamID, userID, reason).Inc()
}

// FrameReceived records a frame being received
func (c *PrometheusCollector) FrameReceived(streamID, frameType string, isKeyFrame bool, sizeBytes int) {
	keyFrame := "false"
	if isKeyFrame {
		keyFrame = "true"
	}
	c.framesReceived.WithLabelValues(streamID, frameType, keyFrame).Inc()
	c.frameSizeBytes.WithLabelValues(streamID, frameType).Observe(float64(sizeBytes))
}

// FrameProcessed records a frame being processed
func (c *PrometheusCollector) FrameProcessed(streamID, frameType string, processingTime time.Duration) {
	c.framesProcessed.WithLabelValues(streamID, frameType).Inc()
	c.frameProcessingTime.WithLabelValues(streamID, frameType).Observe(processingTime.Seconds())
}

// FrameSent records a frame being sent
func (c *PrometheusCollector) FrameSent(streamID, userID, frameType string, sizeBytes int, duration time.Duration) {
	c.framesSent.WithLabelValues(streamID, userID, frameType).Inc()
	c.frameSendTime.WithLabelValues(streamID, userID, frameType).Observe(duration.Seconds())
}

// FrameDropped records a frame being dropped
func (c *PrometheusCollector) FrameDropped(streamID, userID, frameType string, reason string) {
	c.framesDropped.WithLabelValues(streamID, userID, frameType, reason).Inc()
}

// BandwidthEstimation records bandwidth estimation
func (c *PrometheusCollector) BandwidthEstimation(streamID, userID string, bitrateBps float64) {
	c.bitrateEstimation.WithLabelValues(streamID, userID).Set(bitrateBps)
}

// NetworkStats records network statistics
func (c *PrometheusCollector) NetworkStats(streamID, userID string, rtt float64, packetsLostDelta int, fractionLost float64) {
	c.roundTripTime.WithLabelValues(streamID, userID).Set(rtt)
	c.packetsLost.WithLabelValues(streamID, userID).Add(float64(packetsLostDelta))
	c.fractionLost.WithLabelValues(streamID, userID).Set(fractionLost)
}

// ErrorOccurred records an error
func (c *PrometheusCollector) ErrorOccurred(streamID, userID, errorType string) {
	c.errors.WithLabelValues(streamID, userID, errorType).Inc()
}

// Handler returns an HTTP handler for metrics endpoint
func (c *PrometheusCollector) Handler() http.Handler {
	return promhttp.Handler()
}
