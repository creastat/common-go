package yandex

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"
	"github.com/creastat/common-go/pkg/types"
)

// YandexSTTServiceWrapper wraps the provider to implement both Provider and SpeechToTextService
type YandexSTTServiceWrapper struct {
	provider *YandexProvider
}

// Provider interface methods
func (w *YandexSTTServiceWrapper) Name() string { return "yandex-stt" }
func (w *YandexSTTServiceWrapper) Type() models.ProviderType {
	return models.ProviderTypeSpeech
}
func (w *YandexSTTServiceWrapper) Capabilities() []types.Capability {
	return []types.Capability{types.CapabilitySTT}
}
func (w *YandexSTTServiceWrapper) Initialize(ctx context.Context, config models.ProviderConfig) error {
	return nil
}
func (w *YandexSTTServiceWrapper) HealthCheck(ctx context.Context) error {
	return w.provider.HealthCheck(ctx)
}
func (w *YandexSTTServiceWrapper) Close() error { return nil }
func (w *YandexSTTServiceWrapper) GetProviderInfo() *models.ProviderInfo {
	return w.provider.GetProviderInfo()
}

// STTService interface methods (new interface with different signatures)
func (w *YandexSTTServiceWrapper) Transcribe(ctx context.Context, audioData []byte, options map[string]any) (string, error) {
	// Convert []byte to io.Reader for the old implementation
	sttService := NewYandexSTTService(w.provider)
	return sttService.Transcribe(ctx, io.NopCloser(bytes.NewReader(audioData)), models.STTConfig{
		Options: options,
	})
}

func (w *YandexSTTServiceWrapper) StreamTranscribe(ctx context.Context, audioStream <-chan []byte, options map[string]any) (<-chan string, <-chan error) {
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

func (w *YandexSTTServiceWrapper) NewSTTClient(ctx context.Context, config models.STTConfig) (interfaces.STTClient, error) {
	sttService := NewYandexSTTService(w.provider)
	return sttService.NewSTTClient(ctx, config)
}

// YandexTTSServiceWrapper wraps the provider to implement both Provider and TextToSpeechService
type YandexTTSServiceWrapper struct {
	provider *YandexProvider
}

// Provider interface methods
func (w *YandexTTSServiceWrapper) Name() string { return "yandex-tts" }
func (w *YandexTTSServiceWrapper) Type() models.ProviderType {
	return models.ProviderTypeSpeech
}
func (w *YandexTTSServiceWrapper) Capabilities() []types.Capability {
	return []types.Capability{types.CapabilityTTS}
}
func (w *YandexTTSServiceWrapper) Initialize(ctx context.Context, config models.ProviderConfig) error {
	return nil
}
func (w *YandexTTSServiceWrapper) HealthCheck(ctx context.Context) error {
	return w.provider.HealthCheck(ctx)
}
func (w *YandexTTSServiceWrapper) Close() error { return nil }
func (w *YandexTTSServiceWrapper) GetProviderInfo() *models.ProviderInfo {
	return w.provider.GetProviderInfo()
}

// TTSService interface methods
func (w *YandexTTSServiceWrapper) Synthesize(ctx context.Context, text string, config models.TTSConfig) ([]byte, error) {
	ttsService := NewYandexTTSService(w.provider)
	return ttsService.Synthesize(ctx, text, config)
}

func (w *YandexTTSServiceWrapper) StreamSynthesize(ctx context.Context, textStream <-chan string, config models.TTSConfig) (<-chan []byte, <-chan error) {
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

func (w *YandexTTSServiceWrapper) NewTTSClient(ctx context.Context, config models.TTSConfig) (interfaces.TTSClient, error) {
	ttsService := NewYandexTTSService(w.provider)
	return ttsService.NewTTSClient(ctx, config)
}

func (w *YandexTTSServiceWrapper) GetVoices(ctx context.Context) ([]models.Voice, error) {
	ttsService := NewYandexTTSService(w.provider)
	return ttsService.GetVoices(ctx)
}
