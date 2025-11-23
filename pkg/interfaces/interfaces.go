package interfaces

import (
	"context"

	"github.com/creastat/common-go/pkg/models"
	"github.com/creastat/common-go/pkg/types"
)

// BaseProvider defines common methods for all providers
type BaseProvider interface {
	Name() string
	Type() models.ProviderType
	Capabilities() []types.Capability
	Initialize(ctx context.Context, config models.ProviderConfig) error
	Close() error
	HealthCheck(ctx context.Context) error
}

// Provider is an alias for BaseProvider for backward compatibility
type Provider = BaseProvider

// AIProvider defines interface for AI models (LLM, Embedding)
type AIProvider interface {
	BaseProvider
	ChatService
	EmbeddingService
}

// SpeechProvider defines interface for Speech models (STT, TTS)
type SpeechProvider interface {
	BaseProvider
	STTService
	TTSService
}

// ChatService provides chat completion functionality
type ChatService interface {
	ChatCompletion(ctx context.Context, messages []types.ChatMessage, options map[string]any) (string, error)
	StreamChatCompletion(ctx context.Context, messages []types.ChatMessage, options map[string]any) (<-chan string, <-chan error)
	GetModels(ctx context.Context) ([]models.Model, error)
	StreamCompletion(ctx context.Context, req ChatRequest, stream ChatStream) error
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model       string              `json:"model"`
	Messages    []types.ChatMessage `json:"messages"`
	Temperature *float64            `json:"temperature,omitempty"`
	MaxTokens   *int                `json:"max_tokens,omitempty"`
	TopP        *float64            `json:"top_p,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
	Options     map[string]any      `json:"options,omitempty"`
}

// ChatChunk represents a chunk of a streaming chat response
type ChatChunk struct {
	Delta        string `json:"delta"`
	Content      string `json:"content"`
	Done         bool   `json:"done"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// ChatStream represents a streaming chat response handler
type ChatStream interface {
	Send(chunk ChatChunk) error
	Close() error
}

// EmbeddingService provides embedding generation functionality
type EmbeddingService interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// STTService provides speech-to-text functionality
type STTService interface {
	Transcribe(ctx context.Context, audioData []byte, options map[string]any) (string, error)
	StreamTranscribe(ctx context.Context, audioStream <-chan []byte, options map[string]any) (<-chan string, <-chan error)
	NewSTTClient(ctx context.Context, config models.STTConfig) (STTClient, error)
}

// TTSService provides text-to-speech functionality
type TTSService interface {
	Synthesize(ctx context.Context, text string, config models.TTSConfig) ([]byte, error)
	StreamSynthesize(ctx context.Context, textStream <-chan string, config models.TTSConfig) (<-chan []byte, <-chan error)
	NewTTSClient(ctx context.Context, config models.TTSConfig) (TTSClient, error)
	GetVoices(ctx context.Context) ([]models.Voice, error)
}

// TTSClient represents a TTS client interface
type TTSClient interface {
	Close() error
	GetVoices(ctx context.Context) ([]models.Voice, error)
	Send(ctx context.Context, text string) error
	Receive(ctx context.Context) ([]byte, error)
}

// STTClient represents an STTClient interface
type STTClient interface {
	Close() error
	Send(ctx context.Context, audioData []byte) error
	Receive(ctx context.Context) (*models.STTResult, error)
}
