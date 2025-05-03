package model

import "time"

// Health represents the health status of the service
type Health struct {
	Status    string            `json:"status"`
	Uptime    int64             `json:"uptime"`
	StartTime time.Time         `json:"start_time"`
	Checks    map[string]string `json:"checks"`
	Version   string            `json:"version"`
	BuildInfo BuildInfo         `json:"build_info"`
}

// BuildInfo contains information about the build
type BuildInfo struct {
	Version   string    `json:"version"`
	Commit    string    `json:"commit"`
	BuildTime time.Time `json:"build_time"`
	GoVersion string    `json:"go_version"`
}

// HealthStatus represents possible health statuses
type HealthStatus string

const (
	// HealthStatusOK means the service is healthy
	HealthStatusOK HealthStatus = "ok"

	// HealthStatusDegraded means the service is functioning but with issues
	HealthStatusDegraded HealthStatus = "degraded"

	// HealthStatusUnhealthy means the service is not functioning properly
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// ComponentStatus represents the status of a component
type ComponentStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}
