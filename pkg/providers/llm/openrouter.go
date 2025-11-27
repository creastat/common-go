package llm

import (
	"context"
	"fmt"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"
	"github.com/creastat/common-go/pkg/types"
	"github.com/sashabaranov/go-openai"
)

const (
	ProviderOpenRouter = "openrouter"
)

// OpenRouterProvider implements the Provider interface for OpenRouter
type OpenRouterProvider struct {
	name         string
	client       *openai.Client
	config       models.ProviderConfig
	capabilities []types.Capability
	initialized  bool
}

// NewOpenRouterProvider creates a new OpenRouter provider instance
func NewOpenRouterProvider() *OpenRouterProvider {
	return &OpenRouterProvider{
		name: ProviderOpenRouter,
		capabilities: []types.Capability{
			types.CapabilityChat,
			types.CapabilityEmbedding,
		},
		initialized: false,
	}
}

// Name returns the provider name
func (p *OpenRouterProvider) Name() string {
	return p.name
}

// Type returns the provider type
func (p *OpenRouterProvider) Type() interfaces.ProviderType {
	return interfaces.ProviderTypeAI
}

// Capabilities returns the list of capabilities
func (p *OpenRouterProvider) Capabilities() []interfaces.Capability {
	return p.capabilities
}

// Initialize initializes the provider
func (p *OpenRouterProvider) Initialize(ctx context.Context, config models.ProviderConfig) error {
	if config.APIKey == "" {
		return fmt.Errorf("OpenRouter API key is required")
	}

	p.config = config

	clientConfig := openai.DefaultConfig(config.APIKey)
	clientConfig.BaseURL = "https://openrouter.ai/api/v1"

	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	p.client = openai.NewClientWithConfig(clientConfig)
	p.initialized = true
	return nil
}

// HealthCheck performs a health check
func (p *OpenRouterProvider) HealthCheck(ctx context.Context) error {
	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}

	_, err := p.client.ListModels(ctx)
	return err
}

// Close closes the provider
func (p *OpenRouterProvider) Close() error {
	p.initialized = false
	return nil
}

// GetClient returns the underlying OpenAI client
func (p *OpenRouterProvider) GetClient() *openai.Client {
	return p.client
}

// GetConfig returns the provider configuration
func (p *OpenRouterProvider) GetConfig() models.ProviderConfig {
	return p.config
}

// IsInitialized returns whether the provider is initialized
func (p *OpenRouterProvider) IsInitialized() bool {
	return p.initialized
}

// GenerateEmbedding generates an embedding for the given text
func (p *OpenRouterProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if !p.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}

	model := "text-embedding-3-small"
	if p.config.Model != "" {
		model = p.config.Model
	}

	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: model,
	}

	resp, err := p.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return resp.Data[0].Embedding, nil
}

// ChatCompletion implements a simple chat completion wrapper
func (p *OpenRouterProvider) ChatCompletion(ctx context.Context, messages []types.ChatMessage) (string, error) {
	model := "openai/gpt-3.5-turbo"
	if p.config.Model != "" {
		model = p.config.Model
	}

	openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		openaiMessages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: openaiMessages,
	}

	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}
