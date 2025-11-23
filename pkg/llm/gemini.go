package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/creastat/common-go/pkg/types"
	"google.golang.org/genai"
)

// GeminiProvider implements the Provider interface for Google Gemini
type GeminiProvider struct {
	name         string
	client       *genai.Client
	config       types.ProviderConfig
	capabilities []types.Capability
	initialized  bool
}

// NewGeminiProvider creates a new Gemini provider instance
func NewGeminiProvider() *GeminiProvider {
	return &GeminiProvider{
		name: "gemini",
		capabilities: []types.Capability{
			types.CapabilityChat,
			types.CapabilityEmbedding,
		},
		initialized: false,
	}
}

// Name returns the provider name
func (p *GeminiProvider) Name() string {
	return p.name
}

// Capabilities returns the list of capabilities
func (p *GeminiProvider) Capabilities() []types.Capability {
	return p.capabilities
}

// Initialize initializes the provider
func (p *GeminiProvider) Initialize(ctx context.Context, config types.ProviderConfig) error {
	if config.APIKey == "" {
		return fmt.Errorf("Gemini API key is required")
	}

	p.config = config

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: config.APIKey,
	})
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}

	p.client = client

	if err := p.validateAPIKey(ctx); err != nil {
		return fmt.Errorf("failed to validate Gemini API key: %w", err)
	}

	p.initialized = true
	return nil
}

// validateAPIKey validates the API key
func (p *GeminiProvider) validateAPIKey(ctx context.Context) error {
	validateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := p.client.Models.List(validateCtx, nil)
	if err != nil {
		return fmt.Errorf("API key validation failed: %w", err)
	}

	return nil
}

// HealthCheck performs a health check
func (p *GeminiProvider) HealthCheck(ctx context.Context) error {
	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}

	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := p.client.Models.List(healthCtx, nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// Close closes the provider
func (p *GeminiProvider) Close() error {
	p.initialized = false
	return nil
}

// GetClient returns the underlying Gemini client
func (p *GeminiProvider) GetClient() *genai.Client {
	return p.client
}

// GetConfig returns the provider configuration
func (p *GeminiProvider) GetConfig() types.ProviderConfig {
	return p.config
}

// IsInitialized returns whether the provider is initialized
func (p *GeminiProvider) IsInitialized() bool {
	return p.initialized
}

// GenerateEmbedding generates an embedding for the given text
func (p *GeminiProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if !p.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Use the embedding model (default to text-embedding-004 or similar if not specified)
	model := "text-embedding-004"
	if p.config.Model != "" {
		model = p.config.Model
	}

	// Create content from text
	content := genai.Text(text)

	res, err := p.client.Models.EmbedContent(ctx, model, content, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(res.Embeddings) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	return res.Embeddings[0].Values, nil
}
