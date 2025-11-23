package llm

import (
	"context"
	"fmt"

	"github.com/creastat/common-go/pkg/types"
	"github.com/sashabaranov/go-openai"
)

const (
	ProviderOpenRouter = "openrouter"
)

type OpenRouterProvider struct {
	client *openai.Client
	config types.ProviderConfig
}

func NewOpenRouterProvider() *OpenRouterProvider {
	return &OpenRouterProvider{}
}

func (p *OpenRouterProvider) Name() string {
	return ProviderOpenRouter
}

func (p *OpenRouterProvider) Capabilities() []types.Capability {
	return []types.Capability{types.CapabilityChat, types.CapabilityEmbedding}
}

func (p *OpenRouterProvider) Initialize(ctx context.Context, config types.ProviderConfig) error {
	p.config = config

	clientConfig := openai.DefaultConfig(config.APIKey)
	clientConfig.BaseURL = "https://openrouter.ai/api/v1"

	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	p.client = openai.NewClientWithConfig(clientConfig)
	return nil
}

func (p *OpenRouterProvider) HealthCheck(ctx context.Context) error {
	// Simple model list check
	_, err := p.client.ListModels(ctx)
	return err
}

func (p *OpenRouterProvider) Close() error {
	return nil
}

func (p *OpenRouterProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	model := openai.SmallEmbedding3
	if p.config.Model != "" {
		model = openai.EmbeddingModel(p.config.Model)
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
	model := "openai/gpt-3.5-turbo" // Default OpenRouter model
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
