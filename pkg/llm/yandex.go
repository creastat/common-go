package llm

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/creastat/common-go/pkg/types"
	"github.com/sashabaranov/go-openai"
)

const (
	ProviderYandex = "yandex"
)

// yandexTransport wraps an HTTP transport to add Yandex-specific headers
type yandexTransport struct {
	base     http.RoundTripper
	folderID string
}

func (t *yandexTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add Yandex folder ID header
	req.Header.Set("OpenAI-Project", t.folderID)
	return t.base.RoundTrip(req)
}

// YandexProvider implements the Provider interface for Yandex LLM API
type YandexProvider struct {
	name         string
	client       *openai.Client
	config       types.ProviderConfig
	capabilities []types.Capability
	initialized  bool
}

// NewYandexProvider creates a new Yandex provider instance
func NewYandexProvider() *YandexProvider {
	return &YandexProvider{
		name: ProviderYandex,
		capabilities: []types.Capability{
			types.CapabilityChat,
			types.CapabilityEmbedding,
		},
		initialized: false,
	}
}

// Name returns the provider name
func (p *YandexProvider) Name() string {
	return p.name
}

// Capabilities returns the list of capabilities
func (p *YandexProvider) Capabilities() []types.Capability {
	return p.capabilities
}

// Initialize initializes the provider
func (p *YandexProvider) Initialize(ctx context.Context, config types.ProviderConfig) error {
	if config.APIKey == "" {
		return fmt.Errorf("Yandex API key is required")
	}

	p.config = config

	// Create OpenAI client with Yandex base URL
	clientConfig := openai.DefaultConfig(config.APIKey)
	clientConfig.BaseURL = "https://llm.api.cloud.yandex.net/v1"

	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	// Add custom headers for folder_id if provided
	if config.Options != nil {
		if folderID, ok := config.Options["folder_id"].(string); ok && folderID != "" {
			clientConfig.HTTPClient = &http.Client{
				Transport: &yandexTransport{
					base:     http.DefaultTransport,
					folderID: folderID,
				},
				Timeout: 30 * time.Second,
			}
		}
	}

	p.client = openai.NewClientWithConfig(clientConfig)

	// Skip validation for Yandex as it uses a different API structure
	p.initialized = true
	return nil
}

// HealthCheck performs a health check
func (p *YandexProvider) HealthCheck(ctx context.Context) error {
	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}

	// Yandex doesn't support ListModels, so we skip health check
	return nil
}

// Close closes the provider
func (p *YandexProvider) Close() error {
	p.initialized = false
	return nil
}

// GetClient returns the underlying OpenAI client
func (p *YandexProvider) GetClient() *openai.Client {
	return p.client
}

// GetConfig returns the provider configuration
func (p *YandexProvider) GetConfig() types.ProviderConfig {
	return p.config
}

// IsInitialized returns whether the provider is initialized
func (p *YandexProvider) IsInitialized() bool {
	return p.initialized
}

// GenerateEmbedding generates an embedding for the given text
func (p *YandexProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if !p.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Get model from provider config
	model := p.config.Model
	if model == "" {
		model = "text-search-query/latest"
	}

	// For Yandex, prepend the folder_id to the model name
	if p.config.Options != nil {
		if folderID, ok := p.config.Options["folder_id"].(string); ok && folderID != "" {
			// Model format: emb://<folder_id>/<model_name>
			model = fmt.Sprintf("emb://%s/%s", folderID, model)
		}
	}

	req := openai.EmbeddingRequest{
		Input:          []string{text},
		Model:          model,
		EncodingFormat: openai.EmbeddingEncodingFormatFloat,
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
func (p *YandexProvider) ChatCompletion(ctx context.Context, messages []types.ChatMessage) (string, error) {
	model := p.config.Model
	if model == "" {
		model = "yandexgpt/latest"
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
