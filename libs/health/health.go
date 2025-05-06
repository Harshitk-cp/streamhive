package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Status represents the health status of a component
type Status string

const (
	// StatusUp indicates that the component is healthy
	StatusUp Status = "up"
	// StatusDown indicates that the component is unhealthy
	StatusDown Status = "down"
	// StatusDegraded indicates that the component is partially healthy
	StatusDegraded Status = "degraded"
)

// Component represents a component that can be health checked
type Component struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Checker represents a health checker
type Checker struct {
	components  map[string]*Component
	updatedAt   time.Time
	mu          sync.RWMutex
	checkFuncs  map[string]func(ctx context.Context) (Status, error)
	checkPeriod time.Duration
	stopChan    chan struct{}
}

// NewChecker creates a new health checker
func NewChecker() *Checker {
	return &Checker{
		components:  make(map[string]*Component),
		updatedAt:   time.Now(),
		checkFuncs:  make(map[string]func(ctx context.Context) (Status, error)),
		checkPeriod: 30 * time.Second,
		stopChan:    make(chan struct{}),
	}
}

// RegisterComponent registers a component with the health checker
func (c *Checker) RegisterComponent(name string, checkFunc func(ctx context.Context) (Status, error)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.components[name] = &Component{
		Name:   name,
		Status: StatusDown,
	}
	c.checkFuncs[name] = checkFunc
}

// Start starts the health checker
func (c *Checker) Start() {
	ticker := time.NewTicker(c.checkPeriod)
	go func() {
		// Initial check
		c.checkAll()

		for {
			select {
			case <-ticker.C:
				c.checkAll()
			case <-c.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the health checker
func (c *Checker) Stop() {
	close(c.stopChan)
}

// checkAll runs health checks for all components
func (c *Checker) checkAll() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.updatedAt = time.Now()

	// Collect components to check
	checks := make(map[string]func(ctx context.Context) (Status, error))
	for name, checkFunc := range c.checkFuncs {
		checks[name] = checkFunc
	}

	// Run checks in parallel
	var wg sync.WaitGroup
	var checkMu sync.Mutex

	for name, checkFunc := range checks {
		wg.Add(1)
		go func(name string, checkFunc func(ctx context.Context) (Status, error)) {
			defer wg.Done()

			status, err := checkFunc(ctx)

			checkMu.Lock()
			defer checkMu.Unlock()

			component, exists := c.components[name]
			if !exists {
				// Component was removed during check
				return
			}

			component.Status = status
			if err != nil {
				component.Error = err.Error()
			} else {
				component.Error = ""
			}
		}(name, checkFunc)
	}

	wg.Wait()
}

// GetComponentStatus gets the status of a component
func (c *Checker) GetComponentStatus(name string) (Component, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	component, exists := c.components[name]
	if !exists {
		return Component{}, fmt.Errorf("component not found: %s", name)
	}

	return *component, nil
}

// GetAllComponentStatuses gets the status of all components
func (c *Checker) GetAllComponentStatuses() []Component {
	c.mu.RLock()
	defer c.mu.RUnlock()

	statuses := make([]Component, 0, len(c.components))
	for _, component := range c.components {
		statuses = append(statuses, *component)
	}

	return statuses
}

// GetOverallStatus gets the overall health status
func (c *Checker) GetOverallStatus() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.components) == 0 {
		return StatusDown
	}

	hasDown := false
	for _, component := range c.components {
		if component.Status == StatusDown {
			hasDown = true
			break
		}
	}

	if hasDown {
		return StatusDegraded
	}

	return StatusUp
}

// HTTPHandler returns an HTTP handler for health checks
func (c *Checker) HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Get query parameters
		format := r.URL.Query().Get("format")
		component := r.URL.Query().Get("component")

		// If component is specified, return the status of that component
		if component != "" {
			componentStatus, err := c.GetComponentStatus(component)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, "Component not found: %s", component)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(componentStatus)
			return
		}

		// Otherwise, return the status of all components
		allStatuses := c.GetAllComponentStatuses()
		overallStatus := c.GetOverallStatus()

		// If format is simple, return a simple status
		if format == "simple" {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "%s", overallStatus)
			return
		}

		// Default to JSON format
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     overallStatus,
			"components": allStatuses,
			"updated_at": c.updatedAt.Format(time.RFC3339),
		})
	})
}

// SimpleCheck creates a simple health check function that makes an HTTP request
func SimpleCheck(url string) func(ctx context.Context) (Status, error) {
	return func(ctx context.Context) (Status, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return StatusDown, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return StatusDown, err
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			return StatusDown, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		return StatusUp, nil
	}
}

// GRPCCheck creates a health check function for a gRPC service
func GRPCCheck(address string, checkFunc func(ctx context.Context, address string) error) func(ctx context.Context) (Status, error) {
	return func(ctx context.Context) (Status, error) {
		err := checkFunc(ctx, address)
		if err != nil {
			return StatusDown, err
		}
		return StatusUp, nil
	}
}
