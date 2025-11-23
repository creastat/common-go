package models

import (
	"time"

	"github.com/creastat/common-go/pkg/types"
)

// ProviderType represents the type of provider
type ProviderType string

const (
	ProviderTypeAI         ProviderType = "ai"
	ProviderTypeSpeech     ProviderType = "speech"
	ProviderTypeOpenAI     ProviderType = "openai"
	ProviderTypeGemini     ProviderType = "gemini"
	ProviderTypeOpenRouter ProviderType = "openrouter"
	ProviderTypeYandex     ProviderType = "yandex"
	ProviderTypeMinimax    ProviderType = "minimax"
	ProviderTypeCartesia   ProviderType = "cartesia"
	ProviderTypeDeepgram   ProviderType = "deepgram"
)

// Re-export Capability from types
type Capability = types.Capability

const (
	CapabilityChat      = types.CapabilityChat
	CapabilityEmbedding = types.CapabilityEmbedding
	CapabilitySTT       = types.CapabilitySTT
	CapabilityTTS       = types.CapabilityTTS
)

// ProviderInfo represents metadata about a provider
type ProviderInfo struct {
	Name         string             `json:"name"`
	Type         ProviderType       `json:"type"`
	Version      string             `json:"version,omitempty"`
	Description  string             `json:"description,omitempty"`
	Capabilities []Capability       `json:"capabilities"`
	Models       map[string][]Model `json:"models"` // capability -> models
	Available    bool               `json:"available"`
	HealthStatus HealthStatus       `json:"health_status"`
	Metadata     map[string]any     `json:"metadata,omitempty"`
	LastChecked  time.Time          `json:"last_checked,omitempty"`
}

// Model represents information about a specific model
type Model struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Capability  Capability     `json:"capability"`
	ContextSize int            `json:"context_size,omitempty"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Pricing     *ModelPricing  `json:"pricing,omitempty"`
	Features    []string       `json:"features,omitempty"`
	Deprecated  bool           `json:"deprecated"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ModelPricing represents pricing information for a model
type ModelPricing struct {
	InputCost  float64 `json:"input_cost"`  // Cost per 1K tokens
	OutputCost float64 `json:"output_cost"` // Cost per 1K tokens
	Currency   string  `json:"currency"`    // USD, EUR, etc.
}

// ProviderCapabilities represents the capabilities of a provider
type ProviderCapabilities struct {
	Chat      *ChatCapability      `json:"chat,omitempty"`
	Embedding *EmbeddingCapability `json:"embedding,omitempty"`
	STT       *STTCapability       `json:"stt,omitempty"`
	TTS       *TTSCapability       `json:"tts,omitempty"`
}

// ChatCapability represents chat-specific capabilities
type ChatCapability struct {
	Streaming        bool     `json:"streaming"`
	FunctionCalling  bool     `json:"function_calling"`
	VisionSupport    bool     `json:"vision_support"`
	MaxContextTokens int      `json:"max_context_tokens"`
	SupportedModels  []string `json:"supported_models"`
}

// EmbeddingCapability represents embedding-specific capabilities
type EmbeddingCapability struct {
	MaxInputLength  int      `json:"max_input_length"`
	Dimensions      int      `json:"dimensions"`
	SupportedModels []string `json:"supported_models"`
}

// STTCapability represents speech-to-text capabilities
type STTCapability struct {
	Streaming        bool     `json:"streaming"`
	Languages        []string `json:"languages"`
	AudioFormats     []string `json:"audio_formats"`
	MaxAudioDuration int      `json:"max_audio_duration"` // in seconds
	SupportedModels  []string `json:"supported_models"`
}

// TTSCapability represents text-to-speech capabilities
type TTSCapability struct {
	Streaming       bool     `json:"streaming"`
	Voices          []string `json:"voices"`
	Languages       []string `json:"languages"`
	AudioFormats    []string `json:"audio_formats"`
	SupportedModels []string `json:"supported_models"`
}

// HealthStatus represents the health status of a provider
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// ProviderConfig represents configuration for a provider
type ProviderConfig struct {
	Name        string         `json:"name"`
	Type        ProviderType   `json:"type"`
	APIKey      string         `json:"api_key,omitempty"`
	BaseURL     string         `json:"base_url,omitempty"`
	Model       string         `json:"model,omitempty"`
	Options     map[string]any `json:"options,omitempty"`
	Timeout     time.Duration  `json:"timeout,omitempty"`
	RetryPolicy *RetryPolicy   `json:"retry_policy,omitempty"`
	Enabled     bool           `json:"enabled"`
}

// RetryPolicy defines retry behavior for provider calls
type RetryPolicy struct {
	MaxAttempts     int           `json:"max_attempts"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	RetryableErrors []string      `json:"retryable_errors,omitempty"`
}

// FallbackConfig defines fallback behavior between providers
type FallbackConfig struct {
	PrimaryProvider  string        `json:"primary_provider"`
	FallbackProvider string        `json:"fallback_provider"`
	MaxRetries       int           `json:"max_retries"`
	RetryDelay       time.Duration `json:"retry_delay"`
	Conditions       []string      `json:"conditions,omitempty"` // When to fallback
}

// ProviderMetrics represents metrics for a provider
type ProviderMetrics struct {
	ProviderName    string        `json:"provider_name"`
	Capability      Capability    `json:"capability"`
	TotalRequests   int64         `json:"total_requests"`
	SuccessfulReqs  int64         `json:"successful_requests"`
	FailedReqs      int64         `json:"failed_requests"`
	AverageLatency  time.Duration `json:"average_latency"`
	ErrorRate       float64       `json:"error_rate"`
	LastRequestTime time.Time     `json:"last_request_time"`
	LastErrorTime   time.Time     `json:"last_error_time,omitempty"`
	LastError       string        `json:"last_error,omitempty"`
}

// NewProviderInfo creates a new ProviderInfo instance
func NewProviderInfo(name string, providerType ProviderType, capabilities []Capability) *ProviderInfo {
	return &ProviderInfo{
		Name:         name,
		Type:         providerType,
		Capabilities: capabilities,
		Models:       make(map[string][]Model),
		Available:    false,
		HealthStatus: HealthStatusUnknown,
		Metadata:     make(map[string]any),
		LastChecked:  time.Now(),
	}
}

// AddModel adds a model to the provider info
func (pi *ProviderInfo) AddModel(capability Capability, model Model) {
	capKey := string(capability)
	if pi.Models == nil {
		pi.Models = make(map[string][]Model)
	}
	pi.Models[capKey] = append(pi.Models[capKey], model)
}

// GetModels returns models for a specific capability
func (pi *ProviderInfo) GetModels(capability Capability) []Model {
	return pi.Models[string(capability)]
}

// HasCapability checks if the provider has a specific capability
func (pi *ProviderInfo) HasCapability(capability Capability) bool {
	for _, cap := range pi.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// UpdateHealthStatus updates the health status and last checked time
func (pi *ProviderInfo) UpdateHealthStatus(status HealthStatus) {
	pi.HealthStatus = status
	pi.LastChecked = time.Now()
}

// IsAvailable returns whether the provider is available and healthy
func (pi *ProviderInfo) IsAvailable() bool {
	return pi.Available && (pi.HealthStatus == HealthStatusHealthy || pi.HealthStatus == HealthStatusDegraded)
}
