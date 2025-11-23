package minimax

import (
	"context"
	"fmt"
	"time"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"
	"github.com/creastat/common-go/pkg/types"
)

// MinimaxProvider implements the Provider interface for MiniMax
type MinimaxProvider struct {
	name         string
	apiKey       string
	config       models.ProviderConfig
	capabilities []types.Capability
	initialized  bool
	logger       types.Logger
}

// NewMinimaxProvider creates a new MiniMax provider instance
func NewMinimaxProvider(logger types.Logger) *MinimaxProvider {
	if logger == nil {
		logger = &types.NoOpLogger{}
	}
	return &MinimaxProvider{
		name: "minimax",
		capabilities: []types.Capability{
			types.CapabilityTTS,
		},
		initialized: false,
		logger:      logger,
	}
}

// Name returns the provider name
func (p *MinimaxProvider) Name() string {
	return p.name
}

// Type returns the provider type
func (p *MinimaxProvider) Type() models.ProviderType {
	return models.ProviderTypeSpeech
}

// Capabilities returns the list of capabilities this provider supports
func (p *MinimaxProvider) Capabilities() []types.Capability {
	return p.capabilities
}

// Initialize initializes the provider with the given configuration
func (p *MinimaxProvider) Initialize(ctx context.Context, config models.ProviderConfig) error {
	if config.APIKey == "" {
		return fmt.Errorf("MiniMax API key is required")
	}

	// Store configuration
	p.config = config
	p.apiKey = config.APIKey

	// Mark as initialized - API key will be validated on first use
	p.initialized = true

	return nil
}

// validateAPIKey validates the API key by making a test connection
func (p *MinimaxProvider) validateAPIKey(ctx context.Context) error {
	// Create a context with timeout for validation
	validateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// For MiniMax, we'll validate by attempting to create a TTS client
	// and immediately closing it. This verifies the API key is valid.
	ttsService := NewMinimaxTTSService(p)

	// Create a minimal config for validation
	testConfig := models.TTSConfig{
		Voice:      "male-qn-qingse", // Default voice
		Language:   "en",
		SampleRate: 32000,
		Encoding:   "mp3",
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
func (p *MinimaxProvider) HealthCheck(ctx context.Context) error {
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
func (p *MinimaxProvider) Close() error {
	// MiniMax doesn't require explicit cleanup
	p.initialized = false
	return nil
}

// GetAPIKey returns the API key
// This is used by the TTS service
func (p *MinimaxProvider) GetAPIKey() string {
	return p.apiKey
}

// GetConfig returns the provider configuration
func (p *MinimaxProvider) GetConfig() models.ProviderConfig {
	return p.config
}

// IsInitialized returns whether the provider is initialized
func (p *MinimaxProvider) IsInitialized() bool {
	return p.initialized
}

// NewTTSClient creates a new TTS client for streaming synthesis
func (p *MinimaxProvider) NewTTSClient(ctx context.Context, config models.TTSConfig) (interfaces.TTSClient, error) {
	ttsService := NewMinimaxTTSService(p)
	return ttsService.NewTTSClient(ctx, config)
}

// Synthesize synthesizes text to audio (non-streaming)
func (p *MinimaxProvider) Synthesize(ctx context.Context, text string, config models.TTSConfig) ([]byte, error) {
	ttsService := NewMinimaxTTSService(p)
	return ttsService.Synthesize(ctx, text, config)
}

// StreamSynthesize streams text-to-speech synthesis
func (p *MinimaxProvider) StreamSynthesize(ctx context.Context, textStream <-chan string, config models.TTSConfig) (<-chan []byte, <-chan error) {
	// Create output channels
	audioChan := make(chan []byte)
	errChan := make(chan error, 1)

	go func() {
		defer close(audioChan)
		defer close(errChan)

		// For now, return an error as streaming is handled via NewClient
		errChan <- fmt.Errorf("use NewClient for streaming synthesis")
	}()

	return audioChan, errChan
}

// GetVoices returns available voices
func (p *MinimaxProvider) GetVoices(ctx context.Context) ([]models.Voice, error) {
	ttsService := NewMinimaxTTSService(p)
	return ttsService.GetVoices(ctx)
}

// GetVoicesByLanguage returns voices filtered by language
func (p *MinimaxProvider) GetVoicesByLanguage(ctx context.Context, language string) ([]models.Voice, error) {
	ttsService := NewMinimaxTTSService(p)
	return ttsService.GetVoicesByLanguage(ctx, language)
}

// GetDefaultVoiceForLanguage returns the default voice for a language
func (p *MinimaxProvider) GetDefaultVoiceForLanguage(language string) string {
	ttsService := NewMinimaxTTSService(p)
	return ttsService.GetDefaultVoiceForLanguage(language)
}

// GetProviderInfo returns metadata about the MiniMax provider
func (p *MinimaxProvider) GetProviderInfo() *models.ProviderInfo {
	info := models.NewProviderInfo(p.name, models.ProviderTypeMinimax, []models.Capability{
		models.CapabilityTTS,
	})

	info.Description = "MiniMax API provider for text-to-speech capabilities"
	info.Available = p.initialized

	// Add TTS models
	ttsModels := []models.Model{
		{
			ID:          "speech-2.6-hd",
			Name:        "Speech 2.6 HD",
			Description: "High-quality text-to-speech model with natural voice quality",
			Capability:  models.CapabilityTTS,
			Features:    []string{"streaming", "multi-language", "high-quality"},
			Metadata: map[string]any{
				"sample_rate":   32000,
				"bitrate":       128000,
				"formats":       []string{"mp3", "wav", "pcm"},
				"languages":     []string{"en", "zh"},
				"default_voice": "male-qn-qingse",
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
