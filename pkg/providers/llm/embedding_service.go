package llm

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// EmbeddingService provides embedding functionality
type EmbeddingService struct {
	provider *OpenAICompatibleProvider
}

// NewEmbeddingService creates a new embedding service
func NewEmbeddingService(provider *OpenAICompatibleProvider) *EmbeddingService {
	return &EmbeddingService{
		provider: provider,
	}
}

// GenerateEmbedding generates an embedding for the given text
func (s *EmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if !s.provider.IsInitialized() {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Get model from provider config
	model := s.provider.config.Model

	// For Yandex, allow empty model and use default
	if s.provider.name == "yandex" {
		if model == "" {
			model = "text-search-query/latest"
		}

		if folderID, ok := s.provider.config.Options["folder_id"].(string); ok && folderID != "" {
			// Model format: emb://<folder_id>/<model_name>
			model = fmt.Sprintf("emb://%s/%s", folderID, model)
		}
	} else {
		// For other providers, model must be configured
		if model == "" {
			return nil, fmt.Errorf("no embedding model configured for provider %s", s.provider.name)
		}
	}

	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(model),
	}

	// Check for dimensions in options
	if dims, ok := s.provider.config.Options["dimensions"]; ok {
		switch v := dims.(type) {
		case int:
			req.Dimensions = v
		case float64:
			req.Dimensions = int(v)
		}
	}

	// Check for encoding_format in options
	if format, ok := s.provider.config.Options["encoding_format"].(string); ok && format != "" {
		req.EncodingFormat = openai.EmbeddingEncodingFormat(format)
	}

	// For Yandex, force encoding format to float if not specified
	if s.provider.name == "yandex" && req.EncodingFormat == "" {
		req.EncodingFormat = openai.EmbeddingEncodingFormatFloat
	}

	resp, err := s.provider.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return resp.Data[0].Embedding, nil
}

// GetDimensions returns the embedding dimensions
func (s *EmbeddingService) GetDimensions() int {
	return s.provider.GetDimensions()
}
