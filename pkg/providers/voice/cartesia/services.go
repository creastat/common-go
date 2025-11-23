package cartesia

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"
	"github.com/creastat/common-go/pkg/types"
)

// CartesiaSTTServiceWrapper wraps the provider to implement both Provider and SpeechToTextService
type CartesiaSTTServiceWrapper struct {
	provider *CartesiaProvider
}

// Provider interface methods
func (w *CartesiaSTTServiceWrapper) Name() string { return "cartesia-stt" }
func (w *CartesiaSTTServiceWrapper) Type() models.ProviderType {
	return models.ProviderTypeSpeech
}
func (w *CartesiaSTTServiceWrapper) Capabilities() []types.Capability {
	return []types.Capability{types.CapabilitySTT}
}
func (w *CartesiaSTTServiceWrapper) Initialize(ctx context.Context, config models.ProviderConfig) error {
	return nil
}
func (w *CartesiaSTTServiceWrapper) HealthCheck(ctx context.Context) error {
	return w.provider.HealthCheck(ctx)
}
func (w *CartesiaSTTServiceWrapper) Close() error { return nil }
func (w *CartesiaSTTServiceWrapper) GetProviderInfo() *models.ProviderInfo {
	return w.provider.GetProviderInfo()
}

// STTService interface methods (new interface with different signatures)
func (w *CartesiaSTTServiceWrapper) Transcribe(ctx context.Context, audioData []byte, options map[string]any) (string, error) {
	// Convert []byte to io.Reader for the old implementation
	sttService := NewCartesiaSTTService(w.provider)
	return sttService.Transcribe(ctx, io.NopCloser(bytes.NewReader(audioData)), models.STTConfig{
		Options: options,
	})
}

func (w *CartesiaSTTServiceWrapper) StreamTranscribe(ctx context.Context, audioStream <-chan []byte, options map[string]any) (<-chan string, <-chan error) {
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

func (w *CartesiaSTTServiceWrapper) NewSTTClient(ctx context.Context, config models.STTConfig) (interfaces.STTClient, error) {
	sttService := NewCartesiaSTTService(w.provider)
	return sttService.NewSTTClient(ctx, config)
}

// CartesiaTTSServiceWrapper wraps the provider to implement both Provider and TextToSpeechService
type CartesiaTTSServiceWrapper struct {
	provider *CartesiaProvider
}

// Provider interface methods
func (w *CartesiaTTSServiceWrapper) Name() string { return "cartesia-tts" }
func (w *CartesiaTTSServiceWrapper) Type() models.ProviderType {
	return models.ProviderTypeSpeech
}
func (w *CartesiaTTSServiceWrapper) Capabilities() []types.Capability {
	return []types.Capability{types.CapabilityTTS}
}
func (w *CartesiaTTSServiceWrapper) Initialize(ctx context.Context, config models.ProviderConfig) error {
	return nil
}
func (w *CartesiaTTSServiceWrapper) HealthCheck(ctx context.Context) error {
	return w.provider.HealthCheck(ctx)
}
func (w *CartesiaTTSServiceWrapper) Close() error { return nil }
func (w *CartesiaTTSServiceWrapper) GetProviderInfo() *models.ProviderInfo {
	return w.provider.GetProviderInfo()
}

// TTSService interface methods
func (w *CartesiaTTSServiceWrapper) Synthesize(ctx context.Context, text string, config models.TTSConfig) ([]byte, error) {
	ttsService := NewCartesiaTTSService(w.provider)
	return ttsService.Synthesize(ctx, text, config)
}

func (w *CartesiaTTSServiceWrapper) StreamSynthesize(ctx context.Context, textStream <-chan string, config models.TTSConfig) (<-chan []byte, <-chan error) {
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

func (w *CartesiaTTSServiceWrapper) NewTTSClient(ctx context.Context, config models.TTSConfig) (interfaces.TTSClient, error) {
	ttsService := NewCartesiaTTSService(w.provider)
	return ttsService.NewTTSClient(ctx, config)
}

func (w *CartesiaTTSServiceWrapper) GetVoices(ctx context.Context) ([]models.Voice, error) {
	ttsService := NewCartesiaTTSService(w.provider)
	return ttsService.GetVoices(ctx)
}
