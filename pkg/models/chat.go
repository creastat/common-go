package models

import (
	"time"
)

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Messages         []ChatMessage  `json:"messages"`
	Model            string         `json:"model,omitempty"`
	Temperature      float64        `json:"temperature,omitempty"`
	MaxTokens        int            `json:"max_tokens,omitempty"`
	TopP             float64        `json:"top_p,omitempty"`
	FrequencyPenalty float64        `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64        `json:"presence_penalty,omitempty"`
	Stop             []string       `json:"stop,omitempty"`
	Stream           bool           `json:"stream"`
	Context          map[string]any `json:"context,omitempty"`
}

// ChatMessage represents a single message in a chat conversation
type ChatMessage struct {
	Role    string `json:"role"` // user, assistant, system
	Content string `json:"content"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID           string      `json:"id"`
	Model        string      `json:"model"`
	Content      string      `json:"content"`
	Role         string      `json:"role"`
	FinishReason string      `json:"finish_reason,omitempty"`
	Usage        *TokenUsage `json:"usage,omitempty"`
	Timestamp    time.Time   `json:"timestamp"`
}

// TokenUsage represents token usage information
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatStream represents a streaming chat response interface
type ChatStream interface {
	Send(chunk string) error
	SendError(err error) error
	Close() error
}
