package cartesia

import (
	"context"
	"fmt"
	"time"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"
	"github.com/creastat/common-go/pkg/types"
)

// CartesiaProvider implements the Provider interface for Cartesia
type CartesiaProvider struct {
	name         string
	apiKey       string
	config       models.ProviderConfig
	capabilities []types.Capability
	initialized  bool
	logger       types.Logger
}

// NewCartesiaProvider creates a new Cartesia provider instance
func NewCartesiaProvider(logger types.Logger) *CartesiaProvider {
	if logger == nil {
		logger = &types.NoOpLogger{}
	}
	return &CartesiaProvider{
		name: "cartesia",
		capabilities: []types.Capability{
			types.CapabilitySTT,
			types.CapabilityTTS,
		},
		initialized: false,
		logger:      logger,
	}
}

// Name returns the provider name
func (p *CartesiaProvider) Name() string {
	return p.name
}

// Type returns the provider type
func (p *CartesiaProvider) Type() models.ProviderType {
	return models.ProviderTypeSpeech
}

// Capabilities returns the list of capabilities this provider supports
func (p *CartesiaProvider) Capabilities() []types.Capability {
	return p.capabilities
}

// Initialize initializes the provider with the given configuration
func (p *CartesiaProvider) Initialize(ctx context.Context, config models.ProviderConfig) error {
	if config.APIKey == "" {
		return fmt.Errorf("Cartesia API key is required")
	}

	// Store configuration
	p.config = config
	p.apiKey = config.APIKey

	// Mark as initialized - API key will be validated on first use
	p.initialized = true

	return nil
}

// validateAPIKey validates the API key by making a test connection
func (p *CartesiaProvider) validateAPIKey(ctx context.Context) error {
	// Create a context with timeout for validation
	validateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// For Cartesia, we'll validate by attempting to create a TTS client
	// and immediately closing it. This verifies the API key is valid.
	ttsService := NewCartesiaTTSService(p)

	// Create a minimal config for validation
	testConfig := models.TTSConfig{
		Voice:      "694f9389-aac1-45b6-b726-9d9369183238", // Default Sonic voice
		Language:   "en",
		SampleRate: 16000,
		Encoding:   "pcm_s16le",
	}

	// Try to create a client (this will validate the API key)
	client, err := ttsService.NewTTSClient(validateCtx, testConfig)
	if err != nil {
		return fmt.Errorf("API key validation failed: %w", err)
	}

	// Close the test client immediately
	if client != nil {
		client.Close()
	}

	return nil
}

// HealthCheck performs a health check on the provider
func (p *CartesiaProvider) HealthCheck(ctx context.Context) error {
	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}

	// Create a context with timeout for health check
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Perform a lightweight health check by validating the API key
	if err := p.validateAPIKey(healthCtx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// Close closes the provider and releases any resources
func (p *CartesiaProvider) Close() error {
	// Cartesia doesn't require explicit cleanup
	p.initialized = false
	return nil
}

// GetAPIKey returns the API key
// This is used by the STT and TTS services
func (p *CartesiaProvider) GetAPIKey() string {
	return p.apiKey
}

// GetConfig returns the provider configuration
func (p *CartesiaProvider) GetConfig() models.ProviderConfig {
	return p.config
}

// IsInitialized returns whether the provider is initialized
func (p *CartesiaProvider) IsInitialized() bool {
	return p.initialized
}

// GetSTTProvider returns a provider wrapper that implements SpeechToTextService
func (p *CartesiaProvider) GetSTTProvider() interfaces.BaseProvider {
	return &CartesiaSTTServiceWrapper{provider: p}
}

// GetTTSProvider returns a provider wrapper that implements TextToSpeechService
func (p *CartesiaProvider) GetTTSProvider() interfaces.BaseProvider {
	return &CartesiaTTSServiceWrapper{provider: p}
}

// GetProviderInfo returns metadata about the Cartesia provider
func (p *CartesiaProvider) GetProviderInfo() *models.ProviderInfo {
	info := models.NewProviderInfo(p.name, models.ProviderTypeCartesia, []models.Capability{
		models.CapabilitySTT,
		models.CapabilityTTS,
	})

	info.Description = "Cartesia API provider for speech-to-text and text-to-speech capabilities"
	info.Available = p.initialized

	// Add STT models
	sttModels := []models.Model{
		{
			ID:          "ink-whisper",
			Name:        "Ink Whisper",
			Description: "High-quality speech-to-text model with low latency",
			Capability:  models.CapabilitySTT,
			Features:    []string{"streaming", "word-level-timestamps", "multi-language"},
			Metadata: map[string]any{
				"sample_rate": 16000,
				"encoding":    "pcm_s16le",
				"languages":   []string{"en", "es", "fr", "de", "it", "pt", "nl", "pl", "ru", "zh", "ja", "ko"},
			},
		},
	}

	for _, model := range sttModels {
		info.AddModel(models.CapabilitySTT, model)
	}

	// Add TTS models
	ttsModels := []models.Model{
		{
			ID:          "sonic-3",
			Name:        "Sonic 3",
			Description: "Ultra-fast text-to-speech with natural voice quality",
			Capability:  models.CapabilityTTS,
			Features:    []string{"streaming", "low-latency", "multi-language", "voice-cloning"},
			Metadata: map[string]any{
				"sample_rate":   16000,
				"encoding":      "pcm_s16le",
				"languages":     []string{"en", "es", "fr", "de", "it", "pt", "nl", "pl", "ru", "zh", "ja", "ko"},
				"default_voice": "694f9389-aac1-45b6-b726-9d9369183238",
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
