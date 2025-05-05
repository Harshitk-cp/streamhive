package util

import (
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/model"
)

// GenerateID generates a unique ID for frames
func GenerateID() string {
	// Generate random bytes
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to time-based ID if random generation fails
		return fmt.Sprintf("frm_%d", time.Now().UnixNano())
	}

	// Create ID with timestamp and random component
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	return fmt.Sprintf("frm_%d_%s", timestamp, hex.EncodeToString(randomBytes))
}

// GetFrameTypeStr converts a model.FrameType to a string representation
func GetFrameTypeStr(frameType model.FrameType) string {
	switch frameType {
	case model.FrameTypeVideo:
		return "VIDEO"
	case model.FrameTypeAudio:
		return "AUDIO"
	case model.FrameTypeMetadata:
		return "METADATA"
	default:
		return "UNKNOWN"
	}
}

// MatchesFilter checks if a frame matches the specified filter
func MatchesFilter(frame model.Frame, filter string) bool {
	// Empty filter matches all frames
	if filter == "" {
		return true
	}

	// Parse and apply filter
	switch filter {
	case "frame.type == VIDEO && frame.is_key_frame == true":
		return frame.Type == model.FrameTypeVideo && frame.IsKeyFrame
	case "frame.type == AUDIO":
		return frame.Type == model.FrameTypeAudio
	case "frame.type == VIDEO":
		return frame.Type == model.FrameTypeVideo
	case "frame.type == METADATA":
		return frame.Type == model.FrameTypeMetadata
	default:
		// For more complex filters, implement a proper filter parser
		return evaluateComplexFilter(frame, filter)
	}
}

// evaluateComplexFilter evaluates more complex filters
func evaluateComplexFilter(frame model.Frame, filter string) bool {
	// Simple implementation for handling basic filter combinations
	// In production, you would use a proper filter parser/evaluator

	// Handle AND conditions
	if strings.Contains(filter, "&&") {
		parts := strings.Split(filter, "&&")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if !evaluateSimpleFilter(frame, part) {
				return false
			}
		}
		return true
	}

	// Handle OR conditions
	if strings.Contains(filter, "||") {
		parts := strings.Split(filter, "||")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if evaluateSimpleFilter(frame, part) {
				return true
			}
		}
		return false
	}

	// Single condition
	return evaluateSimpleFilter(frame, filter)
}

// evaluateSimpleFilter evaluates a simple filter condition
func evaluateSimpleFilter(frame model.Frame, filter string) bool {
	filter = strings.TrimSpace(filter)

	// Check for frame type condition
	if strings.Contains(filter, "frame.type") {
		if strings.Contains(filter, "== VIDEO") {
			return frame.Type == model.FrameTypeVideo
		}
		if strings.Contains(filter, "== AUDIO") {
			return frame.Type == model.FrameTypeAudio
		}
		if strings.Contains(filter, "== METADATA") {
			return frame.Type == model.FrameTypeMetadata
		}
	}

	// Check for key frame condition
	if strings.Contains(filter, "frame.is_key_frame") {
		if strings.Contains(filter, "== true") {
			return frame.IsKeyFrame
		}
		if strings.Contains(filter, "== false") {
			return !frame.IsKeyFrame
		}
	}

	// Check for sequence conditions
	if strings.Contains(filter, "frame.sequence") {
		// Extract comparison operator and value
		parts := strings.Split(filter, "frame.sequence")
		if len(parts) > 1 {
			opAndVal := strings.TrimSpace(parts[1])

			// Greater than
			if strings.HasPrefix(opAndVal, ">") {
				valStr := strings.TrimSpace(strings.TrimPrefix(opAndVal, ">"))
				if val, err := strconv.ParseInt(valStr, 10, 64); err == nil {
					return frame.Sequence > val
				}
			}

			// Less than
			if strings.HasPrefix(opAndVal, "<") {
				valStr := strings.TrimSpace(strings.TrimPrefix(opAndVal, "<"))
				if val, err := strconv.ParseInt(valStr, 10, 64); err == nil {
					return frame.Sequence < val
				}
			}

			// Equals
			if strings.HasPrefix(opAndVal, "==") {
				valStr := strings.TrimSpace(strings.TrimPrefix(opAndVal, "=="))
				if val, err := strconv.ParseInt(valStr, 10, 64); err == nil {
					return frame.Sequence == val
				}
			}
		}
	}

	// Check for metadata conditions
	if strings.Contains(filter, "frame.metadata[") {
		// Extract key and value
		keyStart := strings.Index(filter, "[") + 1
		keyEnd := strings.Index(filter, "]")
		if keyStart > 0 && keyEnd > keyStart {
			key := filter[keyStart:keyEnd]
			key = strings.Trim(key, "'\"") // Remove quotes

			// Get the comparison part
			compPart := filter[keyEnd+1:]
			compPart = strings.TrimSpace(compPart)

			if strings.HasPrefix(compPart, "==") {
				valStr := strings.TrimSpace(strings.TrimPrefix(compPart, "=="))
				valStr = strings.Trim(valStr, "'\"") // Remove quotes

				// Check if the metadata key exists with the specified value
				if actualVal, ok := frame.Metadata[key]; ok {
					return actualVal == valStr
				}
				return false
			}
		}
	}

	// Default: unknown filter
	return false
}

// CompressData compresses data using gzip
func CompressData(data []byte, level int) ([]byte, error) {
	var b strings.Builder
	gzWriter, err := gzip.NewWriterLevel(&b, level)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	_, err = gzWriter.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed to write data to gzip writer: %w", err)
	}

	err = gzWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return []byte(b.String()), nil
}

// DecompressData decompresses data using gzip
func DecompressData(data []byte) ([]byte, error) {
	gzReader, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read decompressed data: %w", err)
	}

	return decompressed, nil
}

// EnsureDirectoryExists ensures that a directory exists
func EnsureDirectoryExists(path string) error {
	return os.MkdirAll(path, 0755)
}

// GetDirSize returns the size of a directory in bytes
func GetDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// RemoveOldFiles removes files older than the specified time
func RemoveOldFiles(path string, olderThan time.Time) error {
	return filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.ModTime().Before(olderThan) {
			return os.Remove(filePath)
		}
		return nil
	})
}

// UniqueStrings returns a slice with duplicate strings removed
func UniqueStrings(slice []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(slice))

	for _, str := range slice {
		if _, exists := seen[str]; !exists {
			seen[str] = struct{}{}
			result = append(result, str)
		}
	}

	return result
}
