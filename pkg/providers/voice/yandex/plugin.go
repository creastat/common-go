package yandex

import (
	"context"
	"fmt"
	"time"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"
	"github.com/creastat/common-go/pkg/types"
)

// YandexProvider implements the Provider interface for Yandex SpeechKit
type YandexProvider struct {
	name         string
	apiKey       string
	folderId     string
	config       models.ProviderConfig
	capabilities []types.Capability
	initialized  bool
	logger       types.Logger
}

// NewYandexProvider creates a new Yandex provider instance
func NewYandexProvider(logger types.Logger) *YandexProvider {
	if logger == nil {
		logger = &types.NoOpLogger{}
	}
	return &YandexProvider{
		name: "yandex",
		capabilities: []types.Capability{
			types.CapabilitySTT,
			types.CapabilityTTS,
		},
		initialized: false,
		logger:      logger,
	}
}

// Name returns the provider name
func (p *YandexProvider) Name() string {
	return p.name
}

// Type returns the provider type
func (p *YandexProvider) Type() models.ProviderType {
	return models.ProviderTypeSpeech
}

// Capabilities returns the list of capabilities this provider supports
func (p *YandexProvider) Capabilities() []types.Capability {
	return p.capabilities
}

// Initialize initializes the provider with the given configuration
func (p *YandexProvider) Initialize(ctx context.Context, config models.ProviderConfig) error {
	if config.APIKey == "" {
		return fmt.Errorf("Yandex API key is required")
	}

	// Extract folder ID from options
	folderId := ""
	if config.Options != nil {
		if id, ok := config.Options["folder_id"].(string); ok {
			folderId = id
		}
	}

	if folderId == "" {
		return fmt.Errorf("Yandex folder_id is required in options")
	}

	// Store configuration
	p.config = config
	p.apiKey = config.APIKey
	p.folderId = folderId

	// Mark as initialized
	p.initialized = true

	return nil
}

// HealthCheck performs a health check on the provider
func (p *YandexProvider) HealthCheck(ctx context.Context) error {
	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}

	// Create a context with timeout for health check
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// For Yandex, we'll just verify the configuration is valid
	// A full health check would require making an actual API call
	if p.apiKey == "" || p.folderId == "" {
		return fmt.Errorf("health check failed: invalid configuration")
	}

	_ = healthCtx // Use the context to avoid unused variable warning

	return nil
}

// Close closes the provider and releases any resources
func (p *YandexProvider) Close() error {
	p.initialized = false
	return nil
}

// GetAPIKey returns the API key
func (p *YandexProvider) GetAPIKey() string {
	return p.apiKey
}

// GetFolderId returns the folder ID
func (p *YandexProvider) GetFolderId() string {
	return p.folderId
}

// GetConfig returns the provider configuration
func (p *YandexProvider) GetConfig() models.ProviderConfig {
	return p.config
}

// IsInitialized returns whether the provider is initialized
func (p *YandexProvider) IsInitialized() bool {
	return p.initialized
}

// GetSTTProvider returns a provider wrapper that implements SpeechToTextService
func (p *YandexProvider) GetSTTProvider() interfaces.Provider {
	return &YandexSTTServiceWrapper{provider: p}
}

// GetTTSProvider returns a provider wrapper that implements TextToSpeechService
func (p *YandexProvider) GetTTSProvider() interfaces.Provider {
	return &YandexTTSServiceWrapper{provider: p}
}

// GetProviderInfo returns metadata about the Yandex provider
func (p *YandexProvider) GetProviderInfo() *models.ProviderInfo {
	info := models.NewProviderInfo(p.name, models.ProviderTypeYandex, []models.Capability{
		models.CapabilitySTT,
		models.CapabilityTTS,
	})

	info.Description = "Yandex SpeechKit API provider for speech-to-text and text-to-speech capabilities"
	info.Available = p.initialized

	// Add STT models
	sttModels := []models.Model{
		{
			ID:          "general",
			Name:        "General",
			Description: "General-purpose speech recognition model",
			Capability:  models.CapabilitySTT,
			Features:    []string{"streaming", "word-level-timestamps", "multi-language", "normalization"},
			Metadata: map[string]any{
				"sample_rate": 8000,
				"encoding":    "linear16",
				"languages":   []string{"ru-RU", "en-US", "tr-TR", "kk-KZ", "uz-UZ"},
			},
		},
		{
			ID:          "general:rc",
			Name:        "General RC",
			Description: "Release candidate version of general model with latest improvements",
			Capability:  models.CapabilitySTT,
			Features:    []string{"streaming", "word-level-timestamps", "multi-language", "normalization"},
			Metadata: map[string]any{
				"sample_rate": 8000,
				"encoding":    "linear16",
				"languages":   []string{"ru-RU", "en-US", "tr-TR", "kk-KZ", "uz-UZ"},
			},
		},
		{
			ID:          "deferred-general",
			Name:        "Deferred General",
			Description: "Asynchronous recognition for pre-recorded audio files",
			Capability:  models.CapabilitySTT,
			Features:    []string{"async", "word-level-timestamps", "multi-language", "normalization", "speaker-labeling"},
			Metadata: map[string]any{
				"sample_rate": 8000,
				"encoding":    "linear16",
				"languages":   []string{"ru-RU", "en-US", "tr-TR", "kk-KZ", "uz-UZ"},
			},
		},
	}

	for _, model := range sttModels {
		info.AddModel(models.CapabilitySTT, model)
	}

	// Add TTS voices as models
	ttsModels := []models.Model{
		{
			ID:          "alena",
			Name:        "Alena",
			Description: "Russian female voice with neutral tone",
			Capability:  models.CapabilityTTS,
			Features:    []string{"streaming", "speed-control", "pitch-control", "volume-control"},
			Metadata: map[string]any{
				"language": "ru-RU",
				"gender":   "female",
			},
		},
		{
			ID:          "filipp",
			Name:        "Filipp",
			Description: "Russian male voice with neutral tone",
			Capability:  models.CapabilityTTS,
			Features:    []string{"streaming", "speed-control", "pitch-control", "volume-control"},
			Metadata: map[string]any{
				"language": "ru-RU",
				"gender":   "male",
			},
		},
		{
			ID:          "ermil",
			Name:        "Ermil",
			Description: "Russian male voice with emotional tone",
			Capability:  models.CapabilityTTS,
			Features:    []string{"streaming", "speed-control", "pitch-control", "volume-control", "emotional"},
			Metadata: map[string]any{
				"language": "ru-RU",
				"gender":   "male",
			},
		},
		{
			ID:          "jane",
			Name:        "Jane",
			Description: "Russian female voice with emotional tone",
			Capability:  models.CapabilityTTS,
			Features:    []string{"streaming", "speed-control", "pitch-control", "volume-control", "emotional"},
			Metadata: map[string]any{
				"language": "ru-RU",
				"gender":   "female",
			},
		},
		{
			ID:          "john",
			Name:        "John",
			Description: "English male voice",
			Capability:  models.CapabilityTTS,
			Features:    []string{"streaming", "speed-control", "pitch-control", "volume-control"},
			Metadata: map[string]any{
				"language": "en-US",
				"gender":   "male",
			},
		},
	}

	for _, model := range ttsModels {
		info.AddModel(models.CapabilityTTS, model)
	}

	if p.initialized {
		info.HealthStatus = models.HealthStatusHealthy
	} else {
		info.HealthStatus = models.HealthStatusUnknown
	}

	return info
}
