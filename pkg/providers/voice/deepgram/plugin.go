package deepgram

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"
	"github.com/creastat/common-go/pkg/types"
)

// DeepgramProvider implements the Provider interface for Deepgram
type DeepgramProvider struct {
	name         string
	apiKey       string
	config       models.ProviderConfig
	capabilities []types.Capability
	initialized  bool
	logger       types.Logger
}

// NewDeepgramProvider creates a new Deepgram provider instance
func NewDeepgramProvider(logger types.Logger) *DeepgramProvider {
	if logger == nil {
		logger = &types.NoOpLogger{}
	}
	return &DeepgramProvider{
		name: "deepgram",
		capabilities: []types.Capability{
			types.CapabilitySTT,
		},
		initialized: false,
		logger:      logger,
	}
}

// Name returns the provider name
func (p *DeepgramProvider) Name() string {
	return p.name
}

// Type returns the provider type
func (p *DeepgramProvider) Type() models.ProviderType {
	return models.ProviderTypeSpeech
}

// Capabilities returns the list of capabilities this provider supports
func (p *DeepgramProvider) Capabilities() []types.Capability {
	return p.capabilities
}

// Initialize initializes the provider with the given configuration
func (p *DeepgramProvider) Initialize(ctx context.Context, config models.ProviderConfig) error {
	if config.APIKey == "" {
		return fmt.Errorf("Deepgram API key is required")
	}

	// Store configuration
	p.config = config
	p.apiKey = config.APIKey

	// Mark as initialized - API key will be validated on first use
	p.initialized = true

	return nil
}

// validateAPIKey validates the API key by making a test connection
func (p *DeepgramProvider) validateAPIKey(ctx context.Context) error {
	// Create a context with timeout for validation
	validateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// For Deepgram, we'll validate by attempting to create an STT client
	// and immediately closing it. This verifies the API key is valid.
	sttService := NewDeepgramSTTService(p)

	// Create a minimal config for validation
	testConfig := models.STTConfig{
		Model:          "nova-3-general",
		Language:       "en",
		SampleRate:     16000,
		Encoding:       "linear16",
		InterimResults: false,
	}

	// Try to create a client (this will validate the API key)
	client, err := sttService.NewSTTClient(validateCtx, testConfig)
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
func (p *DeepgramProvider) HealthCheck(ctx context.Context) error {
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
func (p *DeepgramProvider) Close() error {
	// Deepgram doesn't require explicit cleanup
	p.initialized = false
	return nil
}

// GetAPIKey returns the API key
// This is used by the STT service
func (p *DeepgramProvider) GetAPIKey() string {
	return p.apiKey
}

// GetConfig returns the provider configuration
func (p *DeepgramProvider) GetConfig() models.ProviderConfig {
	return p.config
}

// IsInitialized returns whether the provider is initialized
func (p *DeepgramProvider) IsInitialized() bool {
	return p.initialized
}

// STTService interface methods (new interface with different signatures)
func (p *DeepgramProvider) Transcribe(ctx context.Context, audioData []byte, options map[string]any) (string, error) {
	// Convert []byte to io.Reader for the old implementation
	sttService := NewDeepgramSTTService(p)
	return sttService.Transcribe(ctx, io.NopCloser(bytes.NewReader(audioData)), models.STTConfig{
		Options: options,
	})
}

func (p *DeepgramProvider) StreamTranscribe(ctx context.Context, audioStream <-chan []byte, options map[string]any) (<-chan string, <-chan error) {
	// Create output channels
	resultChan := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		defer close(resultChan)
		defer close(errChan)

		// For now, return an error as streaming is handled via NewClient
		errChan <- fmt.Errorf("use NewClient for streaming transcription")
	}()

	return resultChan, errChan
}

// NewSTTClient creates a new STT client for streaming audio
// This makes DeepgramProvider implement the SpeechToTextService interface
func (p *DeepgramProvider) NewSTTClient(ctx context.Context, config models.STTConfig) (interfaces.STTClient, error) {
	sttService := NewDeepgramSTTService(p)
	return sttService.NewSTTClient(ctx, config)
}

// GetModels returns available STT models
func (p *DeepgramProvider) GetModels(ctx context.Context) ([]models.Model, error) {
	sttService := NewDeepgramSTTService(p)
	return sttService.GetModels(ctx)
}

// GetProviderInfo returns metadata about the Deepgram provider
func (p *DeepgramProvider) GetProviderInfo() *models.ProviderInfo {
	info := models.NewProviderInfo(p.name, models.ProviderTypeDeepgram, []models.Capability{
		models.CapabilitySTT,
	})

	info.Description = "Deepgram API provider for speech-to-text capabilities"
	info.Available = p.initialized

	// Add STT models
	sttModels := []models.Model{
		{
			ID:          "nova-2-general",
			Name:        "Nova 2 General",
			Description: "Latest general-purpose speech recognition model with high accuracy",
			Capability:  models.CapabilitySTT,
			Features:    []string{"streaming", "word-level-timestamps", "multi-language", "diarization", "smart-formatting"},
			Metadata: map[string]any{
				"sample_rate": 16000,
				"encoding":    "linear16",
				"languages":   []string{"en", "es", "fr", "de", "it", "pt", "nl", "pl", "ru", "zh", "ja", "ko", "hi", "ar"},
			},
		},
		{
			ID:          "nova-2-phonecall",
			Name:        "Nova 2 Phone Call",
			Description: "Optimized for phone call audio with enhanced accuracy for telephony",
			Capability:  models.CapabilitySTT,
			Features:    []string{"streaming", "word-level-timestamps", "telephony-optimized"},
			Metadata: map[string]any{
				"sample_rate": 8000,
				"encoding":    "linear16",
				"languages":   []string{"en"},
			},
		},
		{
			ID:          "nova-2-meeting",
			Name:        "Nova 2 Meeting",
			Description: "Optimized for meeting and conference audio with multiple speakers",
			Capability:  models.CapabilitySTT,
			Features:    []string{"streaming", "word-level-timestamps", "diarization", "multi-speaker"},
			Metadata: map[string]any{
				"sample_rate": 16000,
				"encoding":    "linear16",
				"languages":   []string{"en"},
			},
		},
		{
			ID:          "whisper-large",
			Name:        "Whisper Large",
			Description: "OpenAI Whisper large model for high-accuracy transcription",
			Capability:  models.CapabilitySTT,
			Features:    []string{"streaming", "word-level-timestamps", "multi-language", "translation"},
			Metadata: map[string]any{
				"sample_rate": 16000,
				"encoding":    "linear16",
				"languages":   []string{"en", "es", "fr", "de", "it", "pt", "nl", "pl", "ru", "zh", "ja", "ko", "hi", "ar"},
			},
		},
	}

	for _, model := range sttModels {
		info.AddModel(models.CapabilitySTT, model)
	}

	if p.initialized {
		info.HealthStatus = models.HealthStatusHealthy
	} else {
		info.HealthStatus = models.HealthStatusUnknown
	}

	return info
}
