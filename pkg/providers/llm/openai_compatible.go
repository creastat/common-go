package llm

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"
	"github.com/creastat/common-go/pkg/types"

	"github.com/sashabaranov/go-openai"
)

// yandexTransport wraps an HTTP transport to add Yandex-specific headers
type yandexTransport struct {
	base     http.RoundTripper
	folderID string
}

func (t *yandexTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add Yandex folder ID header (OpenAI-Project is the header name for folder_id)
	req.Header.Set("OpenAI-Project", t.folderID)
	return t.base.RoundTrip(req)
}

// OpenAICompatibleProvider is a universal provider for OpenAI-compatible APIs
type OpenAICompatibleProvider struct {
	name         string
	providerType models.ProviderType
	client       *openai.Client
	config       models.ProviderConfig
	capabilities []types.Capability
	initialized  bool
	modelInfo    []models.Model
}

// ProviderConfig holds provider-specific configuration
type ProviderConfig struct {
	Name         string
	Type         models.ProviderType
	BaseURL      string
	Models       []models.Model
	DefaultModel string
}

// Predefined provider configurations
var (
	OpenAIConfig = ProviderConfig{
		Name:         "openai",
		Type:         models.ProviderTypeOpenAI,
		BaseURL:      "https://api.openai.com/v1",
		DefaultModel: "gpt-4o-mini",
		Models: []models.Model{
			{
				ID:          "gpt-4o-mini",
				Name:        "GPT-4o Mini",
				Description: "Fast and affordable model",
				Capability:  models.CapabilityChat,
				ContextSize: 128000,
				MaxTokens:   16384,
			},
			{
				ID:          "gpt-4o",
				Name:        "GPT-4o",
				Description: "High-intelligence flagship model",
				Capability:  models.CapabilityChat,
				ContextSize: 128000,
				MaxTokens:   16384,
			},
			{
				ID:          "text-embedding-3-small",
				Name:        "Text Embedding 3 Small",
				Description: "Small embedding model",
				Capability:  models.CapabilityEmbedding,
				Metadata:    map[string]any{"dimensions": 1536},
			},
		},
	}

	OpenRouterConfig = ProviderConfig{
		Name:         "openrouter",
		Type:         models.ProviderTypeOpenRouter,
		BaseURL:      "https://openrouter.ai/api/v1",
		DefaultModel: "openai/gpt-4o-mini",
		Models: []models.Model{
			{
				ID:          "openai/gpt-4o-mini",
				Name:        "GPT-4o Mini",
				Description: "OpenAI GPT-4o Mini via OpenRouter",
				Capability:  models.CapabilityChat,
				ContextSize: 128000,
				MaxTokens:   16384,
			},
			{
				ID:          "openai/gpt-4o",
				Name:        "GPT-4o",
				Description: "OpenAI GPT-4o via OpenRouter",
				Capability:  models.CapabilityChat,
				ContextSize: 128000,
				MaxTokens:   16384,
			},
			{
				ID:          "anthropic/claude-3.5-sonnet",
				Name:        "Claude 3.5 Sonnet",
				Description: "Anthropic Claude 3.5 Sonnet via OpenRouter",
				Capability:  models.CapabilityChat,
				ContextSize: 200000,
				MaxTokens:   8192,
			},
			{
				ID:          "google/gemini-pro-1.5",
				Name:        "Gemini Pro 1.5",
				Description: "Google Gemini Pro 1.5 via OpenRouter",
				Capability:  models.CapabilityChat,
				ContextSize: 1000000,
				MaxTokens:   8192,
			},
		},
	}

	YandexConfig = ProviderConfig{
		Name:         "yandex",
		Type:         models.ProviderTypeYandex,
		BaseURL:      "https://llm.api.cloud.yandex.net/v1",
		DefaultModel: "yandexgpt/latest",
		Models: []models.Model{
			{
				ID:          "yandexgpt/latest",
				Name:        "YandexGPT Latest",
				Description: "Latest YandexGPT model",
				Capability:  models.CapabilityChat,
				ContextSize: 8000,
				MaxTokens:   2000,
			},
			{
				ID:          "yandexgpt-lite/latest",
				Name:        "YandexGPT Lite Latest",
				Description: "Latest lightweight YandexGPT model",
				Capability:  models.CapabilityChat,
				ContextSize: 8000,
				MaxTokens:   2000,
			},
			{
				ID:          "yandexgpt-32k/latest",
				Name:        "YandexGPT 32K Latest",
				Description: "Latest YandexGPT with extended context",
				Capability:  models.CapabilityChat,
				ContextSize: 32000,
				MaxTokens:   2000,
			},
			{
				ID:          "text-search-doc/latest",
				Name:        "Text Search Doc Latest",
				Description: "Latest embedding model for document search",
				Capability:  models.CapabilityEmbedding,
				Metadata:    map[string]any{"dimensions": 256},
			},
			{
				ID:          "text-search-query/latest",
				Name:        "Text Search Query Latest",
				Description: "Latest embedding model for search queries",
				Capability:  models.CapabilityEmbedding,
				Metadata:    map[string]any{"dimensions": 256},
			},
		},
	}

	MinimaxLLMConfig = ProviderConfig{
		Name:         "minimax-llm",
		Type:         models.ProviderTypeMinimax,
		BaseURL:      "https://api.minimax.chat/v1",
		DefaultModel: "abab6.5s-chat",
		Models: []models.Model{
			{
				ID:          "abab6.5s-chat",
				Name:        "MiniMax abab6.5s",
				Description: "MiniMax chat model",
				Capability:  models.CapabilityChat,
				ContextSize: 245760,
				MaxTokens:   8192,
			},
		},
	}
)

// NewOpenAICompatibleProvider creates a new OpenAI-compatible provider
func NewOpenAICompatibleProvider(providerConfig ProviderConfig) *OpenAICompatibleProvider {
	return &OpenAICompatibleProvider{
		name:         providerConfig.Name,
		providerType: providerConfig.Type,
		capabilities: []types.Capability{
			types.CapabilityChat,
			types.CapabilityEmbedding,
		},
		modelInfo:   providerConfig.Models,
		initialized: false,
	}
}

// Name returns the provider name
func (p *OpenAICompatibleProvider) Name() string {
	return p.name
}

// Type returns the provider type
func (p *OpenAICompatibleProvider) Type() models.ProviderType {
	return models.ProviderTypeOpenAI
}

// Capabilities returns the list of capabilities
func (p *OpenAICompatibleProvider) Capabilities() []types.Capability {
	return p.capabilities
}

// Initialize initializes the provider
func (p *OpenAICompatibleProvider) Initialize(ctx context.Context, config models.ProviderConfig) error {
	if config.APIKey == "" {
		return fmt.Errorf("%s API key is required", p.name)
	}

	p.config = config

	// Create OpenAI client with custom base URL
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	// Add custom headers for Yandex (folder_id)
	if p.name == "yandex" && config.Options != nil {
		if folderID, ok := config.Options["folder_id"].(string); ok && folderID != "" {
			// Create a custom HTTP client with Yandex transport
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

	// Validate by listing models (skip for Yandex as it uses a different API structure)
	if p.name != "yandex" {
		if err := p.validateAPIKey(ctx); err != nil {
			return fmt.Errorf("failed to validate %s API key: %w", p.name, err)
		}
	}

	p.initialized = true
	return nil
}

// validateAPIKey validates the API key
func (p *OpenAICompatibleProvider) validateAPIKey(ctx context.Context) error {
	validateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := p.client.ListModels(validateCtx)
	if err != nil {
		return fmt.Errorf("API key validation failed: %w", err)
	}

	return nil
}

// HealthCheck performs a health check
func (p *OpenAICompatibleProvider) HealthCheck(ctx context.Context) error {
	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}

	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := p.client.ListModels(healthCtx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// Close closes the provider
func (p *OpenAICompatibleProvider) Close() error {
	p.initialized = false
	return nil
}

// GetClient returns the underlying OpenAI client
func (p *OpenAICompatibleProvider) GetClient() *openai.Client {
	return p.client
}

// GetConfig returns the provider configuration
func (p *OpenAICompatibleProvider) GetConfig() models.ProviderConfig {
	return p.config
}

// IsInitialized returns whether the provider is initialized
func (p *OpenAICompatibleProvider) IsInitialized() bool {
	return p.initialized
}

// GetProviderInfo returns metadata about the provider
func (p *OpenAICompatibleProvider) GetProviderInfo() *models.ProviderInfo {
	info := models.NewProviderInfo(p.name, p.providerType, []models.Capability{
		models.CapabilityChat,
		models.CapabilityEmbedding,
	})

	info.Description = fmt.Sprintf("%s provider using OpenAI-compatible API", p.name)
	info.Available = p.initialized

	// Add models
	for _, model := range p.modelInfo {
		info.AddModel(model.Capability, model)
	}

	if p.initialized {
		info.HealthStatus = models.HealthStatusHealthy
	} else {
		info.HealthStatus = models.HealthStatusUnknown
	}

	return info
}

// ChatCompletion implements ChatService interface
func (p *OpenAICompatibleProvider) ChatCompletion(ctx context.Context, messages []types.ChatMessage, options map[string]any) (string, error) {
	if !p.initialized {
		return "", fmt.Errorf("provider not initialized")
	}

	// Convert messages
	openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		openaiMessages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Get model from options or use default
	model := p.config.Model
	if modelOpt, ok := options["model"].(string); ok && modelOpt != "" {
		model = modelOpt
	}

	// For Yandex, prepend the folder_id to the model name
	if p.name == "yandex" {
		if folderID, ok := p.config.Options["folder_id"].(string); ok && folderID != "" {
			model = fmt.Sprintf("gpt://%s/%s", folderID, model)
		}
	}

	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: openaiMessages,
	}

	// Apply options
	if temp, ok := options["temperature"].(float64); ok {
		req.Temperature = float32(temp)
	}
	if maxTokens, ok := options["max_tokens"].(int); ok {
		req.MaxTokens = maxTokens
	}
	if topP, ok := options["top_p"].(float64); ok {
		req.TopP = float32(topP)
	}

	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	return resp.Choices[0].Message.Content, nil
}

// StreamChatCompletion implements ChatService interface
func (p *OpenAICompatibleProvider) StreamChatCompletion(ctx context.Context, messages []types.ChatMessage, options map[string]any) (<-chan string, <-chan error) {
	contentChan := make(chan string, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(contentChan)
		defer close(errChan)

		if !p.initialized {
			errChan <- fmt.Errorf("provider not initialized")
			return
		}

		// Convert messages
		openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
		for i, msg := range messages {
			openaiMessages[i] = openai.ChatCompletionMessage{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}

		// Get model from options or use default
		model := p.config.Model
		if modelOpt, ok := options["model"].(string); ok && modelOpt != "" {
			model = modelOpt
		}

		// For Yandex, prepend the folder_id to the model name
		if p.name == "yandex" {
			if folderID, ok := p.config.Options["folder_id"].(string); ok && folderID != "" {
				model = fmt.Sprintf("gpt://%s/%s", folderID, model)
			}
		}

		req := openai.ChatCompletionRequest{
			Model:    model,
			Messages: openaiMessages,
			Stream:   true,
		}

		// Apply options
		if temp, ok := options["temperature"].(float64); ok {
			req.Temperature = float32(temp)
		}
		if maxTokens, ok := options["max_tokens"].(int); ok {
			req.MaxTokens = maxTokens
		}
		if topP, ok := options["top_p"].(float64); ok {
			req.TopP = float32(topP)
		}

		stream, err := p.client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			errChan <- fmt.Errorf("failed to create stream: %w", err)
			return
		}
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if err != nil {
				if err.Error() == "EOF" {
					return
				}
				errChan <- fmt.Errorf("stream error: %w", err)
				return
			}

			if len(response.Choices) > 0 {
				content := response.Choices[0].Delta.Content
				if content != "" {
					contentChan <- content
				}
			}
		}
	}()

	return contentChan, errChan
}

// StreamCompletion implements ChatService interface
func (p *OpenAICompatibleProvider) StreamCompletion(ctx context.Context, req interfaces.ChatRequest, stream interfaces.ChatStream) error {
	chatService := NewChatService(p)
	return chatService.StreamCompletion(ctx, req, stream)
}

// GetModels implements ChatService interface
func (p *OpenAICompatibleProvider) GetModels(ctx context.Context) ([]models.Model, error) {
	chatService := NewChatService(p)
	return chatService.GetModels(ctx)
}

// GenerateEmbedding implements EmbeddingService interface
func (p *OpenAICompatibleProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embeddingService := NewEmbeddingService(p)
	return embeddingService.GenerateEmbedding(ctx, text)
}

// GetDimensions implements EmbeddingService interface
func (p *OpenAICompatibleProvider) GetDimensions() int {
	return 1536 // Default OpenAI embedding dimensions
}
