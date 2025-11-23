package factory

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/creastat/common-go/pkg/models"
)

// ProviderBuilder provides a fluent interface for building provider instances
type ProviderBuilder struct {
	providerType string
	config       models.ProviderConfig
	httpClient   *http.Client
	ctx          context.Context
	err          error
}

// NewProviderBuilder creates a new provider builder
func NewProviderBuilder(providerType string) *ProviderBuilder {
	return &ProviderBuilder{
		providerType: providerType,
		ctx:          context.Background(),
		config: models.ProviderConfig{
			Name:    providerType,
			Timeout: 30 * time.Second,
			RetryPolicy: &models.RetryPolicy{
				MaxAttempts:  3,
				InitialDelay: 1 * time.Second,
			},
			Options: make(map[string]any),
		},
	}
}

// WithContext sets the context for provider initialization
func (b *ProviderBuilder) WithContext(ctx context.Context) *ProviderBuilder {
	if b.err != nil {
		return b
	}
	b.ctx = ctx
	return b
}

// WithAPIKey sets the API key for the provider
func (b *ProviderBuilder) WithAPIKey(apiKey string) *ProviderBuilder {
	if b.err != nil {
		return b
	}
	if apiKey == "" {
		b.err = fmt.Errorf("API key cannot be empty for provider %s", b.providerType)
		return b
	}
	b.config.APIKey = apiKey
	return b
}

// WithBaseURL sets the base URL for the provider API
func (b *ProviderBuilder) WithBaseURL(baseURL string) *ProviderBuilder {
	if b.err != nil {
		return b
	}
	b.config.BaseURL = baseURL
	return b
}

// WithModel sets the default model for the provider
func (b *ProviderBuilder) WithModel(model string) *ProviderBuilder {
	if b.err != nil {
		return b
	}
	b.config.Model = model
	return b
}

// WithTimeout sets the request timeout
func (b *ProviderBuilder) WithTimeout(timeout time.Duration) *ProviderBuilder {
	if b.err != nil {
		return b
	}
	if timeout <= 0 {
		b.err = fmt.Errorf("timeout must be positive")
		return b
	}
	b.config.Timeout = timeout
	return b
}

// WithRetryPolicy sets the retry configuration
func (b *ProviderBuilder) WithRetryPolicy(maxRetries int, retryDelay time.Duration) *ProviderBuilder {
	if b.err != nil {
		return b
	}
	if maxRetries < 0 {
		b.err = fmt.Errorf("max retries cannot be negative")
		return b
	}
	if retryDelay < 0 {
		b.err = fmt.Errorf("retry delay cannot be negative")
		return b
	}
	if b.config.RetryPolicy == nil {
		b.config.RetryPolicy = &models.RetryPolicy{}
	}
	b.config.RetryPolicy.MaxAttempts = maxRetries
	b.config.RetryPolicy.InitialDelay = retryDelay
	return b
}

// WithOption sets a provider-specific option
func (b *ProviderBuilder) WithOption(key string, value any) *ProviderBuilder {
	if b.err != nil {
		return b
	}
	if b.config.Options == nil {
		b.config.Options = make(map[string]any)
	}
	b.config.Options[key] = value
	return b
}

// WithOptions sets multiple provider-specific options
func (b *ProviderBuilder) WithOptions(options map[string]any) *ProviderBuilder {
	if b.err != nil {
		return b
	}
	if b.config.Options == nil {
		b.config.Options = make(map[string]any)
	}
	for key, value := range options {
		b.config.Options[key] = value
	}
	return b
}

// WithHTTPClient sets a custom HTTP client
func (b *ProviderBuilder) WithHTTPClient(client *http.Client) *ProviderBuilder {
	if b.err != nil {
		return b
	}
	b.httpClient = client
	return b
}

// WithConfig applies configuration from a config.ProviderConfig
func (b *ProviderBuilder) WithConfig(cfg models.ProviderConfig) *ProviderBuilder {
	if b.err != nil {
		return b
	}

	if cfg.APIKey != "" {
		b.config.APIKey = cfg.APIKey
	}
	if cfg.BaseURL != "" {
		b.config.BaseURL = cfg.BaseURL
	}
	if cfg.Model != "" {
		b.config.Model = cfg.Model
	}
	if cfg.Timeout > 0 {
		b.config.Timeout = cfg.Timeout
	}
	if cfg.RetryPolicy != nil {
		if b.config.RetryPolicy == nil {
			b.config.RetryPolicy = &models.RetryPolicy{}
		}
		if cfg.RetryPolicy.MaxAttempts > 0 {
			b.config.RetryPolicy.MaxAttempts = cfg.RetryPolicy.MaxAttempts
		}
		if cfg.RetryPolicy.InitialDelay > 0 {
			b.config.RetryPolicy.InitialDelay = cfg.RetryPolicy.InitialDelay
		}
	}
	if cfg.Options != nil {
		if b.config.Options == nil {
			b.config.Options = make(map[string]any)
		}
		for key, value := range cfg.Options {
			b.config.Options[key] = value
		}
	}

	return b
}

// GetConfig returns the built provider configuration
func (b *ProviderBuilder) GetConfig() (models.ProviderConfig, error) {
	if b.err != nil {
		return models.ProviderConfig{}, b.err
	}

	// Validate required fields
	if b.config.APIKey == "" {
		return models.ProviderConfig{}, fmt.Errorf("API key is required for provider %s", b.providerType)
	}

	return b.config, nil
}

// GetHTTPClient returns the HTTP client, creating a default one if not set
func (b *ProviderBuilder) GetHTTPClient() *http.Client {
	if b.httpClient != nil {
		return b.httpClient
	}

	// Create default HTTP client with timeout
	return &http.Client{
		Timeout: b.config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

// Build validates the configuration and returns any errors
func (b *ProviderBuilder) Build() error {
	if b.err != nil {
		return b.err
	}

	// Validate required fields
	if b.config.APIKey == "" {
		return fmt.Errorf("API key is required for provider %s", b.providerType)
	}

	return nil
}

// ProviderBuilderFromConfig creates a provider builder from a config
func ProviderBuilderFromConfig(providerName string, cfg models.ProviderConfig) *ProviderBuilder {
	builder := NewProviderBuilder(providerName)
	return builder.WithConfig(cfg)
}

// CredentialManager manages provider credentials
type CredentialManager struct {
	credentials map[string]string
}

// NewCredentialManager creates a new credential manager
func NewCredentialManager() *CredentialManager {
	return &CredentialManager{
		credentials: make(map[string]string),
	}
}

// SetCredential sets a credential for a provider
func (cm *CredentialManager) SetCredential(providerName, apiKey string) {
	cm.credentials[providerName] = apiKey
}

// GetCredential retrieves a credential for a provider
func (cm *CredentialManager) GetCredential(providerName string) (string, bool) {
	apiKey, exists := cm.credentials[providerName]
	return apiKey, exists
}

// HasCredential checks if a credential exists for a provider
func (cm *CredentialManager) HasCredential(providerName string) bool {
	_, exists := cm.credentials[providerName]
	return exists
}

// RemoveCredential removes a credential for a provider
func (cm *CredentialManager) RemoveCredential(providerName string) {
	delete(cm.credentials, providerName)
}

// LoadFromConfig loads credentials from configuration
// LoadFromConfig loads credentials from configuration
// This method is removed to avoid dependency on service-specific config package
// Use SetCredential directly instead

// APIClientInitializer handles initialization of API clients for providers
type APIClientInitializer struct {
	httpClient *http.Client
	timeout    time.Duration
}

// NewAPIClientInitializer creates a new API client initializer
func NewAPIClientInitializer(timeout time.Duration) *APIClientInitializer {
	return &APIClientInitializer{
		timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
			},
		},
	}
}

// GetHTTPClient returns the configured HTTP client
func (i *APIClientInitializer) GetHTTPClient() *http.Client {
	return i.httpClient
}

// WithCustomTransport sets a custom HTTP transport
func (i *APIClientInitializer) WithCustomTransport(transport http.RoundTripper) *APIClientInitializer {
	i.httpClient.Transport = transport
	return i
}

// WithTimeout sets the HTTP client timeout
func (i *APIClientInitializer) WithTimeout(timeout time.Duration) *APIClientInitializer {
	i.timeout = timeout
	i.httpClient.Timeout = timeout
	return i
}

// ConfigValidator validates provider configurations
type ConfigValidator struct{}

// NewConfigValidator creates a new config validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// ValidateProviderConfig validates a provider configuration
func (v *ConfigValidator) ValidateProviderConfig(cfg models.ProviderConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("provider name is required")
	}

	if cfg.APIKey == "" {
		return fmt.Errorf("API key is required for provider %s", cfg.Name)
	}

	if cfg.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive for provider %s", cfg.Name)
	}

	if cfg.RetryPolicy != nil {
		if cfg.RetryPolicy.MaxAttempts < 0 {
			return fmt.Errorf("max retries cannot be negative for provider %s", cfg.Name)
		}
		if cfg.RetryPolicy.InitialDelay < 0 {
			return fmt.Errorf("retry delay cannot be negative for provider %s", cfg.Name)
		}
	}

	return nil
}

// ValidateCapabilityConfig validates a capability configuration
// ValidateCapabilityConfig validates a capability configuration
// This method is removed to avoid dependency on service-specific config package
