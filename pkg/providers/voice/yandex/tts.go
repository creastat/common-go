package yandex

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"
	tts "github.com/creastat/common-go/pkg/providers/voice/yandex/proto/generated/tts"
	"github.com/creastat/common-go/pkg/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const (
	yandexTTSEndpoint = "tts.api.cloud.yandex.net:443"
)

// YandexTTSService implements the TextToSpeechService interface for Yandex SpeechKit
type YandexTTSService struct {
	provider *YandexProvider
	logger   types.Logger
}

// NewYandexTTSService creates a new Yandex TTS service
func NewYandexTTSService(provider *YandexProvider) *YandexTTSService {
	return &YandexTTSService{
		provider: provider,
		logger:   provider.logger,
	}
}

// NewTTSClient creates a new TTS client for streaming synthesis
// Note: Yandex TTS v3 doesn't support true bidirectional streaming
// This implementation synthesizes on Send() and buffers the result
func (s *YandexTTSService) NewTTSClient(ctx context.Context, config models.TTSConfig) (interfaces.TTSClient, error) {
	if !s.provider.IsInitialized() {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Set defaults if not provided
	if config.Voice == "" {
		config.Voice = "ermil"
	}
	if config.Language == "" {
		config.Language = "ru-RU"
	}
	if config.SampleRate == 0 {
		config.SampleRate = 22050
	}
	if config.Encoding == "" {
		config.Encoding = "linear16"
	}
	if config.Speed == 0 {
		config.Speed = 1.0
	}
	if config.Volume == 0 {
		// Default volume for LUFS normalization (range: -145 to 0)
		config.Volume = -19.0
	}

	// Create gRPC connection
	creds := credentials.NewTLS(&tls.Config{})
	conn, err := grpc.NewClient(yandexTTSEndpoint, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Yandex TTS: %w", err)
	}

	// Create client
	client := &yandexTTSClient{
		conn:     conn,
		config:   config,
		provider: s.provider,
		audioCh:  make(chan []byte, 100),
		errCh:    make(chan error, 1),
		doneCh:   make(chan struct{}),
		closed:   false,
		ctx:      ctx,
		logger:   s.logger,
	}

	return client, nil
}

// Synthesize synthesizes text to audio (non-streaming)
func (s *YandexTTSService) Synthesize(ctx context.Context, text string, config models.TTSConfig) ([]byte, error) {
	if !s.provider.IsInitialized() {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Set defaults
	if config.Voice == "" {
		config.Voice = "alena"
	}
	if config.SampleRate == 0 {
		config.SampleRate = 22050
	}
	if config.Encoding == "" {
		config.Encoding = "linear16"
	}
	if config.Speed == 0 {
		config.Speed = 1.0
	}
	if config.Volume == 0 {
		// Default volume for LUFS normalization (range: -145 to 0)
		config.Volume = -19.0
	}

	s.logger.Debug("Starting TTS synthesis",
		"text_length", len(text),
	)

	// Create gRPC connection
	creds := credentials.NewTLS(&tls.Config{})
	conn, err := grpc.NewClient(yandexTTSEndpoint, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Yandex TTS: %w", err)
	}
	defer conn.Close()

	// Add authorization metadata with folder_id
	md := metadata.New(map[string]string{
		"authorization": fmt.Sprintf("Api-Key %s", s.provider.GetAPIKey()),
		"x-folder-id":   s.provider.GetFolderId(),
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Create synthesizer client
	synthesizerClient := tts.NewSynthesizerClient(conn)

	// Build request
	req := s.buildUtteranceRequest(text, config)

	// Call synthesis
	stream, err := synthesizerClient.UtteranceSynthesis(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to start synthesis: %w", err)
	}

	// Collect audio data
	var audioData []byte
	chunkCount := 0
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			s.logger.Debug("TTS synthesis completed",
				"chunks", chunkCount,
				"total_bytes", len(audioData),
			)
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to receive audio: %w", err)
		}

		if resp.AudioChunk != nil && len(resp.AudioChunk.Data) > 0 {
			chunkCount++
			audioData = append(audioData, resp.AudioChunk.Data...)
			if chunkCount%100 == 0 {
				s.logger.Debug("Received TTS chunks",
					"chunks", chunkCount,
					"bytes", len(audioData),
				)
			}
		}
	}

	return audioData, nil
}

// GetVoices returns available voices
func (s *YandexTTSService) GetVoices(ctx context.Context) ([]models.Voice, error) {
	voices := []models.Voice{
		{
			ID:          "alena",
			Name:        "Alena",
			Language:    "ru-RU",
			Gender:      "female",
			Description: "Russian female voice with neutral tone",
		},
		{
			ID:          "filipp",
			Name:        "Filipp",
			Language:    "ru-RU",
			Gender:      "male",
			Description: "Russian male voice with neutral tone",
		},
		{
			ID:          "ermil",
			Name:        "Ermil",
			Language:    "ru-RU",
			Gender:      "male",
			Description: "Russian male voice with emotional tone",
		},
		{
			ID:          "jane",
			Name:        "Jane",
			Language:    "ru-RU",
			Gender:      "female",
			Description: "Russian female voice with emotional tone",
		},
		{
			ID:          "omazh",
			Name:        "Omazh",
			Language:    "ru-RU",
			Gender:      "female",
			Description: "Russian female voice with calm tone",
		},
		{
			ID:          "zahar",
			Name:        "Zahar",
			Language:    "ru-RU",
			Gender:      "male",
			Description: "Russian male voice with calm tone",
		},
		{
			ID:          "john",
			Name:        "John",
			Language:    "en-US",
			Gender:      "male",
			Description: "English male voice",
		},
		{
			ID:          "amira",
			Name:        "Amira",
			Language:    "kk-KK",
			Gender:      "female",
			Description: "Kazakh female voice",
		},
		{
			ID:          "madi",
			Name:        "Madi",
			Language:    "kk-KK",
			Gender:      "male",
			Description: "Kazakh male voice",
		},
		{
			ID:          "nigora",
			Name:        "Nigora",
			Language:    "uz-UZ",
			Gender:      "female",
			Description: "Uzbek female voice",
		},
	}

	return voices, nil
}

// buildUtteranceRequest creates an utterance synthesis request
func (s *YandexTTSService) buildUtteranceRequest(text string, config models.TTSConfig) *tts.UtteranceSynthesisRequest {
	// Map encoding
	audioEncoding := tts.RawAudio_LINEAR16_PCM

	// Build audio format options
	audioSpec := &tts.AudioFormatOptions{
		AudioFormat: &tts.AudioFormatOptions_RawAudio{
			RawAudio: &tts.RawAudio{
				AudioEncoding:   audioEncoding,
				SampleRateHertz: int64(config.SampleRate),
			},
		},
	}

	// Build hints
	hints := []*tts.Hints{
		{
			Hint: &tts.Hints_Voice{
				Voice: config.Voice,
			},
		},
		{
			Hint: &tts.Hints_Speed{
				Speed: config.Speed,
			},
		},
		{
			Hint: &tts.Hints_Volume{
				Volume: config.Volume,
			},
		},
	}

	// Add pitch if specified
	if config.Pitch != 0 {
		hints = append(hints, &tts.Hints{
			Hint: &tts.Hints_PitchShift{
				PitchShift: config.Pitch,
			},
		})
	}

	// Add role if specified in options
	if config.Options != nil {
		if role, ok := config.Options["role"].(string); ok && role != "" {
			hints = append(hints, &tts.Hints{
				Hint: &tts.Hints_Role{
					Role: role,
				},
			})
		}
	}

	// Determine loudness normalization type
	loudnessType := tts.UtteranceSynthesisRequest_LUFS
	if config.Options != nil {
		if normType, ok := config.Options["loudness_normalization"].(string); ok {
			if normType == "max_peak" {
				loudnessType = tts.UtteranceSynthesisRequest_MAX_PEAK
			}
		}
	}

	// Adjust volume based on normalization type
	volume := config.Volume
	if loudnessType == tts.UtteranceSynthesisRequest_LUFS {
		// LUFS: range [-145, 0), default -19
		if volume > 0 {
			// User provided MAX_PEAK style volume (0-1), convert to LUFS
			volume = -19.0
		}
		if volume < -145 {
			volume = -145
		}
	} else {
		// MAX_PEAK: range (0, 1], default 0.7
		if volume <= 0 {
			// User provided LUFS style volume, convert to MAX_PEAK
			volume = 0.7
		}
		if volume > 1 {
			volume = 1.0
		}
	}

	// Update volume hint with adjusted value
	for i, hint := range hints {
		if _, ok := hint.Hint.(*tts.Hints_Volume); ok {
			hints[i] = &tts.Hints{
				Hint: &tts.Hints_Volume{
					Volume: volume,
				},
			}
			break
		}
	}

	return &tts.UtteranceSynthesisRequest{
		Model: config.Model,
		Utterance: &tts.UtteranceSynthesisRequest_Text{
			Text: text,
		},
		Hints:                     hints,
		OutputAudioSpec:           audioSpec,
		LoudnessNormalizationType: loudnessType,
		UnsafeMode:                false,
	}
}

// yandexTTSClient implements the TTSClient interface using StreamSynthesis
type yandexTTSClient struct {
	conn      *grpc.ClientConn
	config    models.TTSConfig
	provider  *YandexProvider
	stream    tts.Synthesizer_StreamSynthesisClient
	audioCh   chan []byte
	errCh     chan error
	doneCh    chan struct{}
	mu        sync.Mutex
	closed    bool
	ctx       context.Context
	wg        sync.WaitGroup // Track receiver goroutine
	closeOnce sync.Once      // Ensure audioCh is closed only once
	logger    types.Logger
}

// Send sends text to be synthesized using StreamSynthesis API for low latency
func (c *yandexTTSClient) Send(ctx context.Context, text string) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("TTS client is closed - cannot start new synthesis")
	}

	// Initialize stream on first Send
	if c.stream == nil {
		if err := c.initStream(); err != nil {
			c.mu.Unlock()
			return fmt.Errorf("failed to initialize stream: %w", err)
		}
	}
	c.mu.Unlock()

	if text == "" {
		return nil
	}

	c.logger.Debug("Sending TTS text",
		"length", len(text),
		"text", text,
	)

	// Send text input to stream
	req := &tts.StreamSynthesisRequest{
		Event: &tts.StreamSynthesisRequest_SynthesisInput{
			SynthesisInput: &tts.SynthesisInput{
				Text: text,
			},
		},
	}

	if err := c.stream.Send(req); err != nil {
		return fmt.Errorf("failed to send text: %w", err)
	}

	return nil
}

// initStream initializes the streaming synthesis connection
func (c *yandexTTSClient) initStream() error {
	// Add authorization metadata
	md := metadata.New(map[string]string{
		"authorization": fmt.Sprintf("Api-Key %s", c.provider.GetAPIKey()),
		"x-folder-id":   c.provider.GetFolderId(),
	})
	streamCtx := metadata.NewOutgoingContext(c.ctx, md)

	// Create synthesizer client
	synthesizerClient := tts.NewSynthesizerClient(c.conn)

	// Start bidirectional stream
	stream, err := synthesizerClient.StreamSynthesis(streamCtx)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}
	c.stream = stream

	// Send initial options
	opts := c.buildSynthesisOptions()
	req := &tts.StreamSynthesisRequest{
		Event: &tts.StreamSynthesisRequest_Options{
			Options: opts,
		},
	}

	if err := stream.Send(req); err != nil {
		return fmt.Errorf("failed to send options: %w", err)
	}

	c.logger.Debug("TTS stream initialized",
		"voice", c.config.Voice,
		"speed", c.config.Speed,
		"volume", c.config.Volume,
	)

	// Start receiver goroutine
	c.wg.Add(1)
	go c.receiveAudio()

	return nil
}

// receiveAudio receives audio chunks from the stream
func (c *yandexTTSClient) receiveAudio() {
	defer c.wg.Done()

	chunkCount := 0
	totalBytes := 0

	for {
		resp, err := c.stream.Recv()
		if err == io.EOF {
			c.logger.Debug("TTS stream ended",
				"chunks", chunkCount,
				"total_bytes", totalBytes,
			)
			return
		}
		if err != nil {
			select {
			case c.errCh <- fmt.Errorf("failed to receive audio: %w", err):
			default:
			}
			return
		}

		if resp.AudioChunk != nil && len(resp.AudioChunk.Data) > 0 {
			chunkCount++
			totalBytes += len(resp.AudioChunk.Data)
			c.logger.Debug("Received TTS audio chunk",
				"chunk_number", chunkCount,
				"size", len(resp.AudioChunk.Data),
			)

			// Send audio chunk
			select {
			case c.audioCh <- resp.AudioChunk.Data:
			case <-time.After(5 * time.Second):
				c.logger.Warn("Timeout sending TTS chunk to channel",
					"chunk_number", chunkCount,
				)
				return
			}
		}
	}
}

// buildSynthesisOptions creates synthesis options for StreamSynthesis
func (c *yandexTTSClient) buildSynthesisOptions() *tts.SynthesisOptions {
	// Map encoding
	audioEncoding := tts.RawAudio_LINEAR16_PCM

	// Build audio format options
	audioSpec := &tts.AudioFormatOptions{
		AudioFormat: &tts.AudioFormatOptions_RawAudio{
			RawAudio: &tts.RawAudio{
				AudioEncoding:   audioEncoding,
				SampleRateHertz: int64(c.config.SampleRate),
			},
		},
	}

	// Determine loudness normalization type
	loudnessType := tts.LoudnessNormalizationType_LUFS
	if c.config.Options != nil {
		if normType, ok := c.config.Options["loudness_normalization"].(string); ok {
			if normType == "max_peak" {
				loudnessType = tts.LoudnessNormalizationType_MAX_PEAK
			}
		}
	}

	// Adjust volume based on normalization type
	volume := c.config.Volume
	if loudnessType == tts.LoudnessNormalizationType_LUFS {
		if volume > 0 {
			volume = -19.0
		}
		if volume < -145 {
			volume = -145
		}
	} else {
		if volume <= 0 {
			volume = 0.7
		}
		if volume > 1 {
			volume = 1.0
		}
	}

	// Get role if specified
	role := ""
	if c.config.Options != nil {
		if r, ok := c.config.Options["role"].(string); ok {
			role = r
		}
	}

	return &tts.SynthesisOptions{
		Model:                     c.config.Model,
		Voice:                     c.config.Voice,
		Role:                      role,
		Speed:                     c.config.Speed,
		Volume:                    volume,
		PitchShift:                c.config.Pitch,
		OutputAudioSpec:           audioSpec,
		LoudnessNormalizationType: loudnessType,
	}
}

// buildUtteranceRequest creates an utterance synthesis request
func (c *yandexTTSClient) buildUtteranceRequest(text string) *tts.UtteranceSynthesisRequest {
	// Map encoding
	audioEncoding := tts.RawAudio_LINEAR16_PCM

	// Build audio format options
	audioSpec := &tts.AudioFormatOptions{
		AudioFormat: &tts.AudioFormatOptions_RawAudio{
			RawAudio: &tts.RawAudio{
				AudioEncoding:   audioEncoding,
				SampleRateHertz: int64(c.config.SampleRate),
			},
		},
	}

	// Build hints
	hints := []*tts.Hints{
		{
			Hint: &tts.Hints_Voice{
				Voice: c.config.Voice,
			},
		},
		{
			Hint: &tts.Hints_Speed{
				Speed: c.config.Speed,
			},
		},
		{
			Hint: &tts.Hints_Volume{
				Volume: c.config.Volume,
			},
		},
	}

	// Add pitch if specified
	if c.config.Pitch != 0 {
		hints = append(hints, &tts.Hints{
			Hint: &tts.Hints_PitchShift{
				PitchShift: c.config.Pitch,
			},
		})
	}

	// Add role if specified in options
	if c.config.Options != nil {
		if role, ok := c.config.Options["role"].(string); ok && role != "" {
			hints = append(hints, &tts.Hints{
				Hint: &tts.Hints_Role{
					Role: role,
				},
			})
		}
	}

	// Determine loudness normalization type
	loudnessType := tts.UtteranceSynthesisRequest_LUFS
	if c.config.Options != nil {
		if normType, ok := c.config.Options["loudness_normalization"].(string); ok {
			if normType == "max_peak" {
				loudnessType = tts.UtteranceSynthesisRequest_MAX_PEAK
			}
		}
	}

	// Adjust volume based on normalization type
	volume := c.config.Volume
	if loudnessType == tts.UtteranceSynthesisRequest_LUFS {
		// LUFS: range [-145, 0), default -19
		if volume > 0 {
			// User provided MAX_PEAK style volume (0-1), convert to LUFS
			volume = -19.0
		}
		if volume < -145 {
			volume = -145
		}
	} else {
		// MAX_PEAK: range (0, 1], default 0.7
		if volume <= 0 {
			// User provided LUFS style volume, convert to MAX_PEAK
			volume = 0.7
		}
		if volume > 1 {
			volume = 1.0
		}
	}

	// Update volume hint with adjusted value
	for i, hint := range hints {
		if _, ok := hint.Hint.(*tts.Hints_Volume); ok {
			hints[i] = &tts.Hints{
				Hint: &tts.Hints_Volume{
					Volume: volume,
				},
			}
			break
		}
	}

	// Create the request with proper protobuf types
	req := &tts.UtteranceSynthesisRequest{
		Model: "", // Empty for standard voices per Yandex docs
		Utterance: &tts.UtteranceSynthesisRequest_Text{
			Text: text,
		},
		Hints:                     hints,
		OutputAudioSpec:           audioSpec,
		LoudnessNormalizationType: loudnessType,
		UnsafeMode:                false,
	}

	return req
}

// Receive receives synthesized audio data
func (c *yandexTTSClient) Receive(ctx context.Context) ([]byte, error) {
	select {
	case audio, ok := <-c.audioCh:
		if !ok {
			// Channel closed, EOF
			return nil, io.EOF
		}
		return audio, nil
	case err := <-c.errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close closes the TTS client and releases resources
func (c *yandexTTSClient) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	// Close the send side of the stream if it exists
	if c.stream != nil {
		if err := c.stream.CloseSend(); err != nil {
			c.logger.Warn("Error closing TTS stream send",
				"error", err,
			)
		}
	}

	// Wait for receiver goroutine to finish
	c.wg.Wait()

	// Close audioCh only once after receiver is done
	c.closeOnce.Do(func() {
		close(c.audioCh)
	})

	// Signal done
	close(c.doneCh)

	// Close the connection
	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

// GetVoices is not supported on individual client instances
// Use the service-level GetVoices method instead
func (c *yandexTTSClient) GetVoices(ctx context.Context) ([]models.Voice, error) {
	return nil, fmt.Errorf("GetVoices not supported on client instance, use service-level method")
}
