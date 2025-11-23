package llm

import (
	"context"
	"fmt"
	"io"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"

	"github.com/sashabaranov/go-openai"
)

// ChatService provides chat completion functionality
type ChatService struct {
	provider *OpenAICompatibleProvider
}

// NewChatService creates a new chat service
func NewChatService(provider *OpenAICompatibleProvider) *ChatService {
	return &ChatService{
		provider: provider,
	}
}

// StreamCompletion streams chat completion responses
func (s *ChatService) StreamCompletion(ctx context.Context, req interfaces.ChatRequest, stream interfaces.ChatStream) error {
	if !s.provider.IsInitialized() {
		return fmt.Errorf("provider not initialized")
	}

	// Convert to OpenAI request
	openaiReq := s.convertToOpenAIRequest(req)

	// Create stream
	openaiStream, err := s.provider.client.CreateChatCompletionStream(ctx, openaiReq)
	if err != nil {
		return fmt.Errorf("failed to create chat completion stream: %w", err)
	}
	defer openaiStream.Close()

	// Stream responses
	for {
		// Check if context is cancelled (e.g., by break signal)
		select {
		case <-ctx.Done():
			// Context cancelled, stop streaming
			return ctx.Err()
		default:
			// Continue processing
		}

		response, err := openaiStream.Recv()
		if err == io.EOF {
			// Send final chunk with Done flag
			if err := stream.Send(interfaces.ChatChunk{Done: true}); err != nil {
				return fmt.Errorf("failed to send final chunk: %w", err)
			}
			break
		}
		if err != nil {
			return fmt.Errorf("stream error: %w", err)
		}

		// Convert and send chunk
		chunk := s.convertFromOpenAIResponse(response)
		if err := stream.Send(chunk); err != nil {
			return fmt.Errorf("failed to send chunk: %w", err)
		}
	}

	return nil
}

// GetModels returns available models
func (s *ChatService) GetModels(ctx context.Context) ([]models.Model, error) {
	if !s.provider.IsInitialized() {
		return nil, fmt.Errorf("provider not initialized")
	}

	resp, err := s.provider.client.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	result := make([]models.Model, len(resp.Models))
	for i, model := range resp.Models {
		result[i] = models.Model{
			ID:   model.ID,
			Name: model.ID,
		}
	}

	return result, nil
}

// convertToOpenAIRequest converts interface request to OpenAI request
func (s *ChatService) convertToOpenAIRequest(req interfaces.ChatRequest) openai.ChatCompletionRequest {
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// For Yandex, prepend the folder_id to the model name
	model := req.Model
	if s.provider.name == "yandex" {
		if folderID, ok := s.provider.config.Options["folder_id"].(string); ok && folderID != "" {
			// Model format: gpt://<folder_id>/<model_name>
			model = fmt.Sprintf("gpt://%s/%s", folderID, req.Model)
			fmt.Printf("DEBUG: Yandex model URI: %s (folder_id: %s, original model: %s)\n", model, folderID, req.Model)
		} else {
			fmt.Printf("DEBUG: Yandex folder_id not found in options: %+v\n", s.provider.config.Options)
		}
	}

	openaiReq := openai.ChatCompletionRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	if req.Temperature != nil && *req.Temperature > 0 {
		openaiReq.Temperature = float32(*req.Temperature)
	}
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		openaiReq.MaxTokens = *req.MaxTokens
	}
	if req.TopP != nil && *req.TopP > 0 {
		openaiReq.TopP = float32(*req.TopP)
	}

	return openaiReq
}

// convertFromOpenAIResponse converts OpenAI response to interface chunk
func (s *ChatService) convertFromOpenAIResponse(resp openai.ChatCompletionStreamResponse) interfaces.ChatChunk {
	chunk := interfaces.ChatChunk{}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		chunk.Delta = choice.Delta.Content
		chunk.Content = choice.Delta.Content
		chunk.FinishReason = string(choice.FinishReason)

		if choice.FinishReason == "stop" || choice.FinishReason == "length" {
			chunk.Done = true
		}
	}

	return chunk
}
