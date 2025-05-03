package service

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/api-gateway/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/api-gateway/pkg/discovery"
)

// ProxyService handles routing requests to the appropriate backend services
type ProxyService struct {
	servicesConfig config.ServicesConfig
	discovery      discovery.ServiceDiscovery
	httpProxies    map[string]*httputil.ReverseProxy
	httpEndpoints  map[string][]string
	mu             sync.RWMutex
	refreshTicker  *time.Ticker
	stopCh         chan struct{}
}

// NewProxyService creates a new proxy service
func NewProxyService(cfg config.ServicesConfig) *ProxyService {
	// Initialize service discovery based on configuration
	var disc discovery.ServiceDiscovery
	var err error

	switch cfg.Discovery {
	case "consul":
		disc, err = discovery.NewConsulDiscovery(cfg.ConsulAddress)
	case "etcd":
		disc, err = discovery.NewEtcdDiscovery(cfg.EtcdEndpoints)
	default:
		disc = discovery.NewStaticDiscovery()
	}

	if err != nil {
		log.Printf("Failed to initialize service discovery: %v. Falling back to static discovery", err)
		disc = discovery.NewStaticDiscovery()
	}

	// Create proxy service
	ps := &ProxyService{
		servicesConfig: cfg,
		discovery:      disc,
		httpProxies:    make(map[string]*httputil.ReverseProxy),
		httpEndpoints:  make(map[string][]string),
		stopCh:         make(chan struct{}),
	}

	// Initialize endpoints from static configuration
	for name, service := range cfg.Services {
		ps.httpEndpoints[name] = service.HTTPEndpoints
		for _, endpoint := range service.HTTPEndpoints {
			target, err := url.Parse(endpoint)
			if err != nil {
				log.Printf("Invalid endpoint URL for service %s: %v", name, err)
				continue
			}
			ps.httpProxies[endpoint] = httputil.NewSingleHostReverseProxy(target)
		}
	}

	// Start periodic refresh if using dynamic service discovery
	if cfg.Discovery != "static" {
		ps.refreshTicker = time.NewTicker(cfg.RefreshInterval)
		go ps.refreshEndpoints()
	}

	return ps
}

// refreshEndpoints periodically refreshes service endpoints from service discovery
func (ps *ProxyService) refreshEndpoints() {
	for {
		select {
		case <-ps.refreshTicker.C:
			ps.updateEndpoints()
		case <-ps.stopCh:
			ps.refreshTicker.Stop()
			return
		}
	}
}

// updateEndpoints updates service endpoints from service discovery
func (ps *ProxyService) updateEndpoints() {
	for name := range ps.servicesConfig.Services {
		endpoints, err := ps.discovery.GetService(name)
		if err != nil {
			log.Printf("Failed to get endpoints for service %s: %v", name, err)
			continue
		}

		// Update HTTP endpoints
		ps.mu.Lock()
		ps.httpEndpoints[name] = endpoints

		// Update HTTP proxies
		for _, endpoint := range endpoints {
			if _, exists := ps.httpProxies[endpoint]; !exists {
				target, err := url.Parse(endpoint)
				if err != nil {
					log.Printf("Invalid endpoint URL for service %s: %v", name, err)
					continue
				}
				ps.httpProxies[endpoint] = httputil.NewSingleHostReverseProxy(target)
			}
		}
		ps.mu.Unlock()

		log.Printf("Updated endpoints for service %s: %v", name, endpoints)
	}
}

// Stop stops the proxy service
func (ps *ProxyService) Stop() {
	close(ps.stopCh)
}

func (ps *ProxyService) ProxyHTTP(serviceName string, w http.ResponseWriter, r *http.Request) error {
	// Get endpoint for service
	ps.mu.RLock()
	endpoints, ok := ps.httpEndpoints[serviceName]
	ps.mu.RUnlock()

	if !ok || len(endpoints) == 0 {
		return fmt.Errorf("no endpoints available for service: %s", serviceName)
	}

	// Simple round-robin load balancing
	endpoint := endpoints[time.Now().UnixNano()%int64(len(endpoints))]

	// Create full URL
	targetURL := fmt.Sprintf("%s%s", endpoint, r.URL.Path)
	log.Printf("Forwarding request to: %s", targetURL)

	// The body might have already been read by handlers, so we need to
	// create a new request with a new body for forwarding
	var bodyData []byte
	var err error

	if r.Body != nil {
		bodyData, err = io.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("failed to read request body: %w", err)
		}
		// Replace the request body for potential future use
		r.Body = io.NopCloser(bytes.NewBuffer(bodyData))
	}

	// Log the body for debugging
	log.Printf("Request body: %s", string(bodyData))

	// Create a new request with the collected body
	proxyReq, err := http.NewRequest(r.Method, targetURL, bytes.NewBuffer(bodyData))
	if err != nil {
		return fmt.Errorf("failed to create proxy request: %w", err)
	}

	// Copy all headers
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Make sure Content-Type is set
	if proxyReq.Header.Get("Content-Type") == "" {
		proxyReq.Header.Set("Content-Type", "application/json")
	}

	// Set the Content-Length header
	proxyReq.ContentLength = int64(len(bodyData))

	// Create a client and execute the request
	client := &http.Client{
		Timeout: ps.servicesConfig.Services[serviceName].Timeout,
	}

	// Send the request
	resp, err := client.Do(proxyReq)
	if err != nil {
		return fmt.Errorf("proxy request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Log the response for debugging
	log.Printf("Response from %s: %s", serviceName, string(respBody))

	// Copy all response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set the response status code
	w.WriteHeader(resp.StatusCode)

	// Write the response body
	w.Write(respBody)

	return nil
}

// GetServiceEndpoint returns a service endpoint for direct access
func (ps *ProxyService) GetServiceEndpoint(serviceName string) (string, error) {
	ps.mu.RLock()
	endpoints, ok := ps.httpEndpoints[serviceName]
	ps.mu.RUnlock()

	if !ok || len(endpoints) == 0 {
		return "", fmt.Errorf("no endpoints available for service: %s", serviceName)
	}

	// Simple round-robin load balancing
	endpoint := endpoints[time.Now().UnixNano()%int64(len(endpoints))]
	return endpoint, nil
}

// GetGRPCServiceEndpoint returns a gRPC service endpoint
func (ps *ProxyService) GetGRPCServiceEndpoint(serviceName string) (string, error) {
	ps.mu.RLock()
	serviceConfig, ok := ps.servicesConfig.Services[serviceName]
	ps.mu.RUnlock()

	if !ok || len(serviceConfig.GRPCEndpoints) == 0 {
		return "", fmt.Errorf("no gRPC endpoints available for service: %s", serviceName)
	}

	// Simple round-robin load balancing
	endpoints := serviceConfig.GRPCEndpoints
	endpoint := endpoints[time.Now().UnixNano()%int64(len(endpoints))]
	return endpoint, nil
}

// CheckServiceHealth checks the health of a service
func (ps *ProxyService) CheckServiceHealth(serviceName string) error {
	serviceConfig, ok := ps.servicesConfig.Services[serviceName]
	if !ok {
		return fmt.Errorf("service not found: %s", serviceName)
	}

	if serviceConfig.HealthCheck == "" {
		return errors.New("no health check endpoint defined for service")
	}

	// Get an endpoint for the service
	endpoint, err := ps.GetServiceEndpoint(serviceName)
	if err != nil {
		return err
	}

	// Create health check URL
	healthURL := fmt.Sprintf("%s%s", endpoint, serviceConfig.HealthCheck)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: serviceConfig.Timeout,
	}

	// Send health check request
	resp, err := client.Get(healthURL)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unhealthy service, status code: %d", resp.StatusCode)
	}

	return nil
}
