package types

import (
	"context"
	"time"
)

// Capability represents a provider capability type
type Capability string

const (
	CapabilityChat      Capability = "chat"
	CapabilityEmbedding Capability = "embedding"
	CapabilitySTT       Capability = "stt"
	CapabilityTTS       Capability = "tts"
)

// ChatMessage represents a message in a chat conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Provider represents a generic AI provider that can offer one or more capabilities
type Provider interface {
	// Name returns the unique identifier for this provider
	Name() string

	// Capabilities returns the list of capabilities this provider supports
	Capabilities() []Capability

	// Initialize initializes the provider with the given configuration
	Initialize(ctx context.Context, config ProviderConfig) error

	// HealthCheck performs a health check on the provider
	HealthCheck(ctx context.Context) error

	// Close closes the provider and releases any resources
	Close() error
}

// ProviderConfig contains configuration for a provider
type ProviderConfig struct {
	// Name is the provider identifier
	Name string

	// APIKey is the authentication key for the provider
	APIKey string

	// BaseURL is the base URL for the provider API (optional, uses default if empty)
	BaseURL string

	// Model is the default model to use for this provider
	Model string

	// Timeout is the request timeout duration
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration

	// Options contains provider-specific configuration options
	Options map[string]any
}
