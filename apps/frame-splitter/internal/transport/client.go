package transport

import (
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// BaseClient represents a base client for connecting to other services
type BaseClient struct {
	conn *grpc.ClientConn
}

// NewBaseClient creates a new base client
func NewBaseClient(address string) (*BaseClient, error) {
	// Setup connection options
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(10 * time.Second),
	}

	// Connect to service
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	return &BaseClient{
		conn: conn,
	}, nil
}

// Close closes the client connection
func (c *BaseClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetConnection returns the gRPC client connection
func (c *BaseClient) GetConnection() *grpc.ClientConn {
	return c.conn
}
