package metrics

import (
	"net/http"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector defines the interface for metrics collection
type Collector interface {
	// Frame processing metrics
	FrameReceived(streamID string, frameType string, isKeyFrame bool, sizeBytes int)
	FrameProcessed(streamID string, frameType string, processingTime time.Duration, isSuccess bool)
	FrameRouted(streamID string, destination string, batchSize int, routingTime time.Duration)
	FrameDropped(streamID string, frameType string, reason string)
	FrameBatchProcessed(streamID string, batchSize int, processingTime time.Duration)

	// Queue metrics
	QueueSize(streamID string, size int)

	// Backup metrics
	FrameBackedUp(streamID string, frameType string, sizeBytes int, backupTime time.Duration)
	BackupSpaceUsed(bytes int64)

	// Error metrics
	ErrorOccurred(streamID string, errorType string)

	// HTTP handler for metrics endpoint
	Handler() http.Handler

	UpdateStreamMetrics(streamID string, stats model.StreamStats)
}

// PrometheusCollector implements the Collector interface using Prometheus
type PrometheusCollector struct {
	// Frame metrics
	framesReceived       *prometheus.CounterVec
	framesProcessed      *prometheus.CounterVec
	framesRouted         *prometheus.CounterVec
	framesDropped        *prometheus.CounterVec
	framesBatchProcessed *prometheus.CounterVec

	// Size metrics
	frameSizeBytes *prometheus.HistogramVec

	// Timing metrics
	frameProcessingTime *prometheus.HistogramVec
	frameRoutingTime    *prometheus.HistogramVec
	frameBackupTime     *prometheus.HistogramVec

	// Queue metrics
	queueSize *prometheus.GaugeVec

	// Backup metrics
	frameBackups    *prometheus.CounterVec
	backupSizeBytes prometheus.Gauge

	// Error metrics
	errors *prometheus.CounterVec
}

func (c *PrometheusCollector) UpdateStreamMetrics(streamID string, stats model.StreamStats) {
	// Update frames processed counter
	if stats.FramesProcessed > 0 {
		// We don't have a direct counter for this, but we can approximate by incrementing
		// based on change since last measurement if needed
		c.framesBatchProcessed.WithLabelValues(streamID).Inc()
	}

	// Update frames dropped counter
	if stats.FramesDropped > 0 {
		c.framesDropped.WithLabelValues(streamID, "all", "system").Add(float64(stats.FramesDropped))
	}

	// Update bytes processed via size histogram if available
	if stats.BytesProcessed > 0 {
		c.frameSizeBytes.WithLabelValues(streamID, "all").Observe(float64(stats.BytesProcessed))
	}

	// Update processing time if available
	if stats.AvgProcessingTime > 0 {
		c.frameProcessingTime.WithLabelValues(streamID, "all").Observe(stats.AvgProcessingTime.Seconds())
	}


}

// NewPrometheusCollector creates a new PrometheusCollector
func NewPrometheusCollector() *PrometheusCollector {
	collector := &PrometheusCollector{
		// Frame metrics
		framesReceived: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "frame_splitter_frames_received_total",
				Help: "Total number of frames received",
			},
			[]string{"stream_id", "frame_type", "key_frame"},
		),

		framesProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "frame_splitter_frames_processed_total",
				Help: "Total number of frames processed",
			},
			[]string{"stream_id", "frame_type", "success"},
		),

		framesRouted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "frame_splitter_frames_routed_total",
				Help: "Total number of frames routed to destinations",
			},
			[]string{"stream_id", "destination"},
		),

		framesDropped: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "frame_splitter_frames_dropped_total",
				Help: "Total number of frames dropped",
			},
			[]string{"stream_id", "frame_type", "reason"},
		),

		framesBatchProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "frame_splitter_frame_batches_processed_total",
				Help: "Total number of frame batches processed",
			},
			[]string{"stream_id"},
		),

		// Size metrics
		frameSizeBytes: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "frame_splitter_frame_size_bytes",
				Help:    "Size of frames in bytes",
				Buckets: prometheus.ExponentialBuckets(1024, 2, 10), // 1KB to 1MB
			},
			[]string{"stream_id", "frame_type"},
		),

		// Timing metrics
		frameProcessingTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "frame_splitter_frame_processing_seconds",
				Help:    "Time taken to process frames",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to 1s
			},
			[]string{"stream_id", "frame_type"},
		),

		frameRoutingTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "frame_splitter_frame_routing_seconds",
				Help:    "Time taken to route frames",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to 1s
			},
			[]string{"stream_id", "destination"},
		),

		frameBackupTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "frame_splitter_frame_backup_seconds",
				Help:    "Time taken to backup frames",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to 1s
			},
			[]string{"stream_id"},
		),

		// Queue metrics
		queueSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "frame_splitter_queue_size",
				Help: "Current size of the frame queue",
			},
			[]string{"stream_id"},
		),

		// Backup metrics
		frameBackups: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "frame_splitter_frame_backups_total",
				Help: "Total number of frames backed up",
			},
			[]string{"stream_id", "frame_type"},
		),

		backupSizeBytes: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "frame_splitter_backup_size_bytes",
				Help: "Total size of backed up frames in bytes",
			},
		),

		// Error metrics
		errors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "frame_splitter_errors_total",
				Help: "Total number of errors",
			},
			[]string{"stream_id", "error_type"},
		),
	}

	return collector

}

// FrameReceived records a frame being received
func (c *PrometheusCollector) FrameReceived(streamID string, frameType string, isKeyFrame bool, sizeBytes int) {
	keyFrame := "false"
	if isKeyFrame {
		keyFrame = "true"
	}

	c.framesReceived.WithLabelValues(streamID, frameType, keyFrame).Inc()
	c.frameSizeBytes.WithLabelValues(streamID, frameType).Observe(float64(sizeBytes))
}

// FrameProcessed records a frame being processed
func (c *PrometheusCollector) FrameProcessed(streamID string, frameType string, processingTime time.Duration, isSuccess bool) {
	success := "false"
	if isSuccess {
		success = "true"
	}

	c.framesProcessed.WithLabelValues(streamID, frameType, success).Inc()
	c.frameProcessingTime.WithLabelValues(streamID, frameType).Observe(processingTime.Seconds())
}

// FrameRouted records a frame being routed to a destination
func (c *PrometheusCollector) FrameRouted(streamID string, destination string, batchSize int, routingTime time.Duration) {
	c.framesRouted.WithLabelValues(streamID, destination).Add(float64(batchSize))
	c.frameRoutingTime.WithLabelValues(streamID, destination).Observe(routingTime.Seconds())
}

// FrameDropped records a frame being dropped
func (c *PrometheusCollector) FrameDropped(streamID string, frameType string, reason string) {
	c.framesDropped.WithLabelValues(streamID, frameType, reason).Inc()
}

// FrameBatchProcessed records a batch of frames being processed
func (c *PrometheusCollector) FrameBatchProcessed(streamID string, batchSize int, processingTime time.Duration) {
	c.framesBatchProcessed.WithLabelValues(streamID).Inc()
}

// QueueSize records the current queue size
func (c *PrometheusCollector) QueueSize(streamID string, size int) {
	c.queueSize.WithLabelValues(streamID).Set(float64(size))
}

// FrameBackedUp records a frame being backed up
func (c *PrometheusCollector) FrameBackedUp(streamID string, frameType string, sizeBytes int, backupTime time.Duration) {
	c.frameBackups.WithLabelValues(streamID, frameType).Inc()
	c.frameBackupTime.WithLabelValues(streamID).Observe(backupTime.Seconds())
}

// BackupSpaceUsed records the total backup space used
func (c *PrometheusCollector) BackupSpaceUsed(bytes int64) {
	c.backupSizeBytes.Set(float64(bytes))
}

// ErrorOccurred records an error
func (c *PrometheusCollector) ErrorOccurred(streamID string, errorType string) {
	c.errors.WithLabelValues(streamID, errorType).Inc()
}

// Handler returns an HTTP handler for the metrics endpoint
func (c *PrometheusCollector) Handler() http.Handler {
	return promhttp.Handler()
}

// NoOpCollector is a no-op implementation of the Collector interface
type NoOpCollector struct{}

// FrameReceived is a no-op
func (c *NoOpCollector) FrameReceived(streamID string, frameType string, isKeyFrame bool, sizeBytes int) {
}

// FrameProcessed is a no-op
func (c *NoOpCollector) FrameProcessed(streamID string, frameType string, processingTime time.Duration, isSuccess bool) {
}

// FrameRouted is a no-op
func (c *NoOpCollector) FrameRouted(streamID string, destination string, batchSize int, routingTime time.Duration) {
}

// FrameDropped is a no-op
func (c *NoOpCollector) FrameDropped(streamID string, frameType string, reason string) {}

// FrameBatchProcessed is a no-op
func (c *NoOpCollector) FrameBatchProcessed(streamID string, batchSize int, processingTime time.Duration) {
}

// QueueSize is a no-op
func (c *NoOpCollector) QueueSize(streamID string, size int) {}

// FrameBackedUp is a no-op
func (c *NoOpCollector) FrameBackedUp(streamID string, frameType string, sizeBytes int, backupTime time.Duration) {
}

// BackupSpaceUsed is a no-op
func (c *NoOpCollector) BackupSpaceUsed(bytes int64) {}

// ErrorOccurred is a no-op
func (c *NoOpCollector) ErrorOccurred(streamID string, errorType string) {}

// Handler returns a no-op HTTP handler
func (c *NoOpCollector) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Metrics collection is disabled"))
	})
}
