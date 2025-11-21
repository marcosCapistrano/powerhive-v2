package vnish

import (
	"time"

	"github.com/powerhive/powerhive-v2/pkg/miner"
)

// ClientFactory creates VNish HTTP clients.
// It implements miner.ClientFactory for integration with discovery.
type ClientFactory struct {
	auth    *AuthManager
	timeout time.Duration
}

// FactoryOption configures a ClientFactory.
type FactoryOption func(*ClientFactory)

// WithFactoryTimeout sets the HTTP timeout for created clients.
func WithFactoryTimeout(timeout time.Duration) FactoryOption {
	return func(f *ClientFactory) {
		f.timeout = timeout
	}
}

// NewClientFactory creates a new VNish client factory.
func NewClientFactory(auth *AuthManager, opts ...FactoryOption) *ClientFactory {
	f := &ClientFactory{
		auth:    auth,
		timeout: 30 * time.Second,
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// NewClient creates a new VNish HTTP client for the given host.
// Implements miner.ClientFactory.
func (f *ClientFactory) NewClient(host string) miner.Client {
	return NewClient(host, f.auth, WithTimeout(f.timeout))
}

// NewVNishClient creates a new VNish-specific client with full API access.
// Use this when you need vnish-specific functionality beyond miner.Client.
func (f *ClientFactory) NewVNishClient(host string) *HTTPClient {
	return NewClient(host, f.auth, WithTimeout(f.timeout))
}

// Ensure ClientFactory implements miner.ClientFactory.
var _ miner.ClientFactory = (*ClientFactory)(nil)
