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

const (
	ProviderGemini = "gemini"
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
		name: ProviderGemini,
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
func (p *GeminiProvider) Type() interfaces.ProviderType {
	return interfaces.ProviderTypeAI
}

// Capabilities returns the list of capabilities
func (p *GeminiProvider) Capabilities() []interfaces.Capability {
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

// GenerateEmbedding generates an embedding for the given text
func (p *GeminiProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if !p.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}

	model := "text-embedding-004"
	if p.config.Model != "" {
		model = p.config.Model
	}

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

// ChatCompletion implements a simple chat completion wrapper
func (p *GeminiProvider) ChatCompletion(ctx context.Context, messages []types.ChatMessage) (string, error) {
	model := "gemini-pro"
	if p.config.Model != "" {
		model = p.config.Model
	}

	genaiMessages := make([]*genai.Content, len(messages))
	for i, msg := range messages {
		genaiMessages[i] = &genai.Content{
			Role: msg.Role,
			Parts: []genai.Part{
				genai.Text(msg.Content),
			},
		}
	}

	session := p.client.GenerativeModel(model).StartChat()
	session.History = genaiMessages

	resp, err := session.SendMessage(ctx, genai.Text(""))
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates returned")
	}

	if len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content parts returned")
	}

	return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), nil
}
