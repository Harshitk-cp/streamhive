package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector defines the interface for collecting metrics
type Collector interface {
	// StreamStarted records that a stream has started
	StreamStarted(streamID string, isBackup bool)

	// StreamEnded records that a stream has ended
	StreamEnded(streamID string, bytesReceived, frameCount int64, duration time.Duration)

	// StreamTimeout records that a stream has timed out
	StreamTimeout(streamID string)

	// PacketReceived records that a packet has been received
	PacketReceived(streamID string, bytes int, isKeyFrame bool)

	// FrameProcessed records that a frame has been processed
	FrameProcessed(streamID string, bytes int, processingTime time.Duration)

	// Error records an error
	Error(streamID, errorType string)

	// HTTPHandler returns the HTTP handler for exposing metrics
	HTTPHandler() http.Handler
}

// PrometheusCollector is a Prometheus implementation of the Collector interface
type PrometheusCollector struct {
	// Stream metrics
	activeStreams     prometheus.Gauge
	streamStartsTotal *prometheus.CounterVec
	streamEndsTotal   *prometheus.CounterVec
	streamBytesTotal  *prometheus.CounterVec
	streamFramesTotal *prometheus.CounterVec
	streamDuration    *prometheus.HistogramVec

	// Packet metrics
	packetBytesTotal *prometheus.CounterVec
	keyFramesTotal   *prometheus.CounterVec

	// Processing metrics
	frameProcessingTime *prometheus.HistogramVec

	// Error metrics
	errorsTotal *prometheus.CounterVec
}

// NewPrometheusCollector creates a new PrometheusCollector
func NewPrometheusCollector() *PrometheusCollector {
	return &PrometheusCollector{
		activeStreams: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "rtmp_active_streams",
			Help: "Number of active RTMP streams",
		}),

		streamStartsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rtmp_stream_starts_total",
				Help: "Total number of stream starts",
			},
			[]string{"stream_id", "backup"},
		),

		streamEndsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rtmp_stream_ends_total",
				Help: "Total number of stream ends",
			},
			[]string{"stream_id", "reason"},
		),

		streamBytesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rtmp_stream_bytes_total",
				Help: "Total bytes received per stream",
			},
			[]string{"stream_id"},
		),

		streamFramesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rtmp_stream_frames_total",
				Help: "Total frames received per stream",
			},
			[]string{"stream_id"},
		),

		streamDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "rtmp_stream_duration_seconds",
				Help:    "Stream duration in seconds",
				Buckets: prometheus.ExponentialBuckets(60, 2, 10), // 1min to ~14hours
			},
			[]string{"stream_id"},
		),

		packetBytesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rtmp_packet_bytes_total",
				Help: "Total bytes received in packets",
			},
			[]string{"stream_id"},
		),

		keyFramesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rtmp_key_frames_total",
				Help: "Total number of key frames received",
			},
			[]string{"stream_id"},
		),

		frameProcessingTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "rtmp_frame_processing_seconds",
				Help:    "Time spent processing frames in seconds",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
			},
			[]string{"stream_id"},
		),

		errorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rtmp_errors_total",
				Help: "Total number of errors",
			},
			[]string{"stream_id", "type"},
		),
	}
}

// StreamStarted records that a stream has started
func (c *PrometheusCollector) StreamStarted(streamID string, isBackup bool) {
	c.activeStreams.Inc()

	backup := "false"
	if isBackup {
		backup = "true"
	}

	c.streamStartsTotal.WithLabelValues(streamID, backup).Inc()
}

// StreamEnded records that a stream has ended
func (c *PrometheusCollector) StreamEnded(streamID string, bytesReceived, frameCount int64, duration time.Duration) {
	c.activeStreams.Dec()

	c.streamEndsTotal.WithLabelValues(streamID, "normal").Inc()
	c.streamBytesTotal.WithLabelValues(streamID).Add(float64(bytesReceived))
	c.streamFramesTotal.WithLabelValues(streamID).Add(float64(frameCount))
	c.streamDuration.WithLabelValues(streamID).Observe(duration.Seconds())
}

// StreamTimeout records that a stream has timed out
func (c *PrometheusCollector) StreamTimeout(streamID string) {
	c.activeStreams.Dec()
	c.streamEndsTotal.WithLabelValues(streamID, "timeout").Inc()
}

// PacketReceived records that a packet has been received
func (c *PrometheusCollector) PacketReceived(streamID string, bytes int, isKeyFrame bool) {
	c.packetBytesTotal.WithLabelValues(streamID).Add(float64(bytes))

	if isKeyFrame {
		c.keyFramesTotal.WithLabelValues(streamID).Inc()
	}
}

// FrameProcessed records that a frame has been processed
func (c *PrometheusCollector) FrameProcessed(streamID string, bytes int, processingTime time.Duration) {
	c.frameProcessingTime.WithLabelValues(streamID).Observe(processingTime.Seconds())
}

// Error records an error
func (c *PrometheusCollector) Error(streamID, errorType string) {
	c.errorsTotal.WithLabelValues(streamID, errorType).Inc()
}

// HTTPHandler returns the HTTP handler for exposing metrics
func (c *PrometheusCollector) HTTPHandler() http.Handler {
	return promhttp.Handler()
}

// NoOpCollector is a no-op implementation of the Collector interface
type NoOpCollector struct{}

// StreamStarted is a no-op
func (c *NoOpCollector) StreamStarted(streamID string, isBackup bool) {}

// StreamEnded is a no-op
func (c *NoOpCollector) StreamEnded(streamID string, bytesReceived, frameCount int64, duration time.Duration) {
}

// StreamTimeout is a no-op
func (c *NoOpCollector) StreamTimeout(streamID string) {}

// PacketReceived is a no-op
func (c *NoOpCollector) PacketReceived(streamID string, bytes int, isKeyFrame bool) {}

// FrameProcessed is a no-op
func (c *NoOpCollector) FrameProcessed(streamID string, bytes int, processingTime time.Duration) {}

// Error is a no-op
func (c *NoOpCollector) Error(streamID, errorType string) {}

// HTTPHandler returns a no-op HTTP handler
func (c *NoOpCollector) HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Metrics collection is disabled"))
	})
}
