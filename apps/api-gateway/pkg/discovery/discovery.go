package discovery

import (
	"errors"
	"fmt"
	"sync"
)

// ServiceDiscovery defines the interface for service discovery
type ServiceDiscovery interface {
	// GetService returns a list of endpoints for a service
	GetService(name string) ([]string, error)

	// RegisterService registers a service endpoint
	RegisterService(name, endpoint string) error

	// DeregisterService removes a service endpoint
	DeregisterService(name, endpoint string) error

	// WatchService watches for changes in service endpoints
	WatchService(name string, callback func([]string)) error

	// Close closes the service discovery client
	Close() error
}

// StaticDiscovery implements a static service discovery
type StaticDiscovery struct {
	services map[string][]string
	mu       sync.RWMutex
}

// NewStaticDiscovery creates a new static service discovery
func NewStaticDiscovery() *StaticDiscovery {
	return &StaticDiscovery{
		services: make(map[string][]string),
	}
}

// GetService returns a list of endpoints for a service
func (sd *StaticDiscovery) GetService(name string) ([]string, error) {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	endpoints, ok := sd.services[name]
	if !ok || len(endpoints) == 0 {
		return nil, fmt.Errorf("service not found: %s", name)
	}

	// Return a copy of the endpoints
	result := make([]string, len(endpoints))
	copy(result, endpoints)

	return result, nil
}

// RegisterService registers a service endpoint
func (sd *StaticDiscovery) RegisterService(name, endpoint string) error {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	// Check if service exists
	endpoints, ok := sd.services[name]
	if !ok {
		// Create new service
		sd.services[name] = []string{endpoint}
		return nil
	}

	// Check if endpoint already exists
	for _, ep := range endpoints {
		if ep == endpoint {
			return nil // Already registered
		}
	}

	// Add endpoint to service
	sd.services[name] = append(endpoints, endpoint)

	return nil
}

// DeregisterService removes a service endpoint
func (sd *StaticDiscovery) DeregisterService(name, endpoint string) error {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	// Check if service exists
	endpoints, ok := sd.services[name]
	if !ok {
		return nil // Service not found, nothing to deregister
	}

	// Find endpoint
	for i, ep := range endpoints {
		if ep == endpoint {
			// Remove endpoint
			sd.services[name] = append(endpoints[:i], endpoints[i+1:]...)
			return nil
		}
	}

	return nil // Endpoint not found, nothing to deregister
}

// WatchService watches for changes in service endpoints
func (sd *StaticDiscovery) WatchService(name string, callback func([]string)) error {
	// Static discovery doesn't support watching
	return errors.New("watching not supported in static discovery")
}

// Close closes the service discovery client
func (sd *StaticDiscovery) Close() error {
	// Nothing to close in static discovery
	return nil
}

// ConsulDiscovery implements service discovery using Consul
type ConsulDiscovery struct {
	address string
	// consul client would be added here in a real implementation
}

// NewConsulDiscovery creates a new Consul service discovery
func NewConsulDiscovery(address string) (*ConsulDiscovery, error) {
	// In a real implementation, this would create a Consul client
	// For now, just return a stub with the address

	return &ConsulDiscovery{
		address: address,
	}, nil
}

// GetService returns a list of endpoints for a service
func (cd *ConsulDiscovery) GetService(name string) ([]string, error) {
	// In a real implementation, this would query Consul
	// For now, just return a stub error

	return nil, errors.New("consul service discovery not implemented")
}

// RegisterService registers a service endpoint
func (cd *ConsulDiscovery) RegisterService(name, endpoint string) error {
	// In a real implementation, this would register with Consul
	// For now, just return a stub error

	return errors.New("consul service discovery not implemented")
}

// DeregisterService removes a service endpoint
func (cd *ConsulDiscovery) DeregisterService(name, endpoint string) error {
	// In a real implementation, this would deregister from Consul
	// For now, just return a stub error

	return errors.New("consul service discovery not implemented")
}

// WatchService watches for changes in service endpoints
func (cd *ConsulDiscovery) WatchService(name string, callback func([]string)) error {
	// In a real implementation, this would set up a watch in Consul
	// For now, just return a stub error

	return errors.New("consul service discovery not implemented")
}

// Close closes the service discovery client
func (cd *ConsulDiscovery) Close() error {
	// In a real implementation, this would close the Consul client
	// For now, just return nil

	return nil
}

// EtcdDiscovery implements service discovery using etcd
type EtcdDiscovery struct {
	endpoints []string
	// etcd client would be added here in a real implementation
}

// NewEtcdDiscovery creates a new etcd service discovery
func NewEtcdDiscovery(endpoints []string) (*EtcdDiscovery, error) {
	// In a real implementation, this would create an etcd client
	// For now, just return a stub with the endpoints

	return &EtcdDiscovery{
		endpoints: endpoints,
	}, nil
}

// GetService returns a list of endpoints for a service
func (ed *EtcdDiscovery) GetService(name string) ([]string, error) {
	// In a real implementation, this would query etcd
	// For now, just return a stub error

	return nil, errors.New("etcd service discovery not implemented")
}

// RegisterService registers a service endpoint
func (ed *EtcdDiscovery) RegisterService(name, endpoint string) error {
	// In a real implementation, this would register with etcd
	// For now, just return a stub error

	return errors.New("etcd service discovery not implemented")
}

// DeregisterService removes a service endpoint
func (ed *EtcdDiscovery) DeregisterService(name, endpoint string) error {
	// In a real implementation, this would deregister from etcd
	// For now, just return a stub error

	return errors.New("etcd service discovery not implemented")
}

// WatchService watches for changes in service endpoints
func (ed *EtcdDiscovery) WatchService(name string, callback func([]string)) error {
	// In a real implementation, this would set up a watch in etcd
	// For now, just return a stub error

	return errors.New("etcd service discovery not implemented")
}

// Close closes the service discovery client
func (ed *EtcdDiscovery) Close() error {
	// In a real implementation, this would close the etcd client
	// For now, just return nil

	return nil
}
