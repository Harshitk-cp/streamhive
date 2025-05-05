package processor

import (
	"context"
	"log"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/metrics"
	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/model"
)

// RTMPHandler handles frames from RTMP Ingestor
type RTMPHandler struct {
	processor        *FrameProcessor
	metricsCollector metrics.Collector
}

// NewRTMPHandler creates a new RTMP handler
func NewRTMPHandler(processor *FrameProcessor, metricsCollector metrics.Collector) *RTMPHandler {
	return &RTMPHandler{
		processor:        processor,
		metricsCollector: metricsCollector,
	}
}

// HandleVideoFrame handles a video frame from RTMP Ingestor
func (h *RTMPHandler) HandleVideoFrame(ctx context.Context, streamID string, frameID string, data []byte, timestamp time.Time, isKeyFrame bool, sequence int64, metadata map[string]string) error {
	startTime := time.Now()

	// Create frame
	frame := model.Frame{
		StreamID:   streamID,
		FrameID:    frameID,
		Timestamp:  timestamp,
		Type:       model.FrameTypeVideo,
		Data:       data,
		Metadata:   metadata,
		Sequence:   sequence,
		IsKeyFrame: isKeyFrame,
	}

	// Process frame
	result, err := h.processor.ProcessFrame(ctx, frame)
	if err != nil {
		h.metricsCollector.ErrorOccurred(streamID, "video_frame_processing_failed")
		return err
	}

	// Log frame processing
	log.Printf("Processed video frame %s for stream %s: destinations=%v, took=%v",
		frameID, streamID, result.Destinations, time.Since(startTime))

	// Update metrics
	h.metricsCollector.FrameProcessed(streamID, string(model.FrameTypeVideo), time.Since(startTime), true)

	return nil
}

// HandleAudioFrame handles an audio frame from RTMP Ingestor
func (h *RTMPHandler) HandleAudioFrame(ctx context.Context, streamID string, frameID string, data []byte, timestamp time.Time, sequence int64, metadata map[string]string) error {
	startTime := time.Now()

	// Create frame
	frame := model.Frame{
		StreamID:   streamID,
		FrameID:    frameID,
		Timestamp:  timestamp,
		Type:       model.FrameTypeAudio,
		Data:       data,
		Metadata:   metadata,
		Sequence:   sequence,
		IsKeyFrame: false,
	}

	// Process frame
	result, err := h.processor.ProcessFrame(ctx, frame)
	if err != nil {
		h.metricsCollector.ErrorOccurred(streamID, "audio_frame_processing_failed")
		return err
	}

	// Log frame processing
	log.Printf("Processed audio frame %s for stream %s: destinations=%v, took=%v",
		frameID, streamID, result.Destinations, time.Since(startTime))

	// Update metrics
	h.metricsCollector.FrameProcessed(streamID, string(model.FrameTypeAudio), time.Since(startTime), true)

	return nil
}

// HandleMetadataFrame handles a metadata frame from RTMP Ingestor
func (h *RTMPHandler) HandleMetadataFrame(ctx context.Context, streamID string, frameID string, data []byte, timestamp time.Time, sequence int64) error {
	startTime := time.Now()

	// Create frame
	frame := model.Frame{
		StreamID:   streamID,
		FrameID:    frameID,
		Timestamp:  timestamp,
		Type:       model.FrameTypeMetadata,
		Data:       data,
		Sequence:   sequence,
		IsKeyFrame: false,
	}

	// Process frame
	result, err := h.processor.ProcessFrame(ctx, frame)
	if err != nil {
		h.metricsCollector.ErrorOccurred(streamID, "metadata_frame_processing_failed")
		return err
	}

	// Log frame processing
	log.Printf("Processed metadata frame %s for stream %s: destinations=%v, took=%v",
		frameID, streamID, result.Destinations, time.Since(startTime))

	// Update metrics
	h.metricsCollector.FrameProcessed(streamID, string(model.FrameTypeMetadata), time.Since(startTime), true)

	return nil
}
