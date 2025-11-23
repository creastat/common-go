package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"
	"github.com/creastat/common-go/pkg/types"

	"google.golang.org/genai"
)

// GeminiProvider implements the Provider interface for Google Gemini
type GeminiProvider struct {
	name         string
	client       *genai.Client
	config       models.ProviderConfig
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

// Type returns the provider type
func (p *GeminiProvider) Type() models.ProviderType {
	return models.ProviderTypeGemini
}

// Capabilities returns the list of capabilities
func (p *GeminiProvider) Capabilities() []types.Capability {
	return p.capabilities
}

// Initialize initializes the provider
func (p *GeminiProvider) Initialize(ctx context.Context, config models.ProviderConfig) error {
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
func (p *GeminiProvider) GetConfig() models.ProviderConfig {
	return p.config
}

// IsInitialized returns whether the provider is initialized
func (p *GeminiProvider) IsInitialized() bool {
	return p.initialized
}

// GetProviderInfo returns metadata about the provider
func (p *GeminiProvider) GetProviderInfo() *models.ProviderInfo {
	info := models.NewProviderInfo(p.name, models.ProviderTypeGemini, []models.Capability{
		models.CapabilityChat,
		models.CapabilityEmbedding,
	})

	info.Description = "Google Gemini API provider"
	info.Available = p.initialized

	chatModels := []models.Model{
		{
			ID:          "gemini-2.0-flash-exp",
			Name:        "Gemini 2.0 Flash Experimental",
			Description: "Latest experimental flash model",
			Capability:  models.CapabilityChat,
			ContextSize: 1048576,
			MaxTokens:   8192,
		},
		{
			ID:          "gemini-1.5-flash",
			Name:        "Gemini 1.5 Flash",
			Description: "Fast and efficient model",
			Capability:  models.CapabilityChat,
			ContextSize: 1048576,
			MaxTokens:   8192,
		},
		{
			ID:          "gemini-1.5-pro",
			Name:        "Gemini 1.5 Pro",
			Description: "Advanced model for complex tasks",
			Capability:  models.CapabilityChat,
			ContextSize: 2097152,
			MaxTokens:   8192,
		},
	}

	for _, model := range chatModels {
		info.AddModel(models.CapabilityChat, model)
	}

	if p.initialized {
		info.HealthStatus = models.HealthStatusHealthy
	} else {
		info.HealthStatus = models.HealthStatusUnknown
	}

	return info
}

// ChatCompletion implements ChatService interface
func (p *GeminiProvider) ChatCompletion(ctx context.Context, messages []types.ChatMessage, options map[string]any) (string, error) {
	if !p.initialized {
		return "", fmt.Errorf("provider not initialized")
	}
	return "", fmt.Errorf("Gemini chat completion not yet implemented")
}

// StreamChatCompletion implements ChatService interface
func (p *GeminiProvider) StreamChatCompletion(ctx context.Context, messages []types.ChatMessage, options map[string]any) (<-chan string, <-chan error) {
	contentChan := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		defer close(contentChan)
		defer close(errChan)
		errChan <- fmt.Errorf("Gemini streaming chat completion not yet implemented")
	}()

	return contentChan, errChan
}

// StreamCompletion implements ChatService interface
func (p *GeminiProvider) StreamCompletion(ctx context.Context, req interfaces.ChatRequest, stream interfaces.ChatStream) error {
	// Gemini implementation would go here
	return fmt.Errorf("Gemini streaming not yet implemented")
}

// GetModels implements ChatService interface
func (p *GeminiProvider) GetModels(ctx context.Context) ([]models.Model, error) {
	if !p.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Return predefined models for Gemini
	result := []models.Model{
		{ID: "gemini-2.0-flash-exp", Name: "Gemini 2.0 Flash Experimental"},
		{ID: "gemini-1.5-flash", Name: "Gemini 1.5 Flash"},
		{ID: "gemini-1.5-pro", Name: "Gemini 1.5 Pro"},
	}

	return result, nil
}

// GenerateEmbedding implements EmbeddingService interface
func (p *GeminiProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("Gemini embeddings not yet implemented")
}

// GetDimensions implements EmbeddingService interface
func (p *GeminiProvider) GetDimensions() int {
	return 768
}
