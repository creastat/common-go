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

	// Use default embedding model
	model := "text-embedding-3-small"
	if s.provider.config.Model != "" {
		model = s.provider.config.Model
	}

	// For Yandex, prepend the folder_id to the model name
	if s.provider.name == "yandex" {
		if folderID, ok := s.provider.config.Options["folder_id"].(string); ok && folderID != "" {
			// Model format: emb://<folder_id>/<model_name>
			model = fmt.Sprintf("emb://%s/%s", folderID, model)
		}
	}

	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: model,
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
