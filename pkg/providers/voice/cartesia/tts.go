package cartesia

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"
	"github.com/creastat/common-go/pkg/types"

	"github.com/gorilla/websocket"
)

// CartesiaTTSService implements the TextToSpeechService interface for Cartesia
type CartesiaTTSService struct {
	provider *CartesiaProvider
	logger   types.Logger
}

// NewCartesiaTTSService creates a new Cartesia TTS service
func NewCartesiaTTSService(provider *CartesiaProvider) *CartesiaTTSService {
	return &CartesiaTTSService{
		provider: provider,
		logger:   provider.logger,
	}
}

// NewTTSClient creates a new TTS client for streaming synthesis
func (s *CartesiaTTSService) NewTTSClient(ctx context.Context, config models.TTSConfig) (interfaces.TTSClient, error) {
	if !s.provider.IsInitialized() {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Set defaults if not provided
	if config.Model == "" {
		config.Model = "sonic-3"
	}
	if config.Voice == "" {
		config.Voice = "694f9389-aac1-45b6-b726-9d9369183238" // Default Sonic voice
	}
	if config.Language == "" {
		config.Language = "en"
	}
	if config.SampleRate == 0 {
		config.SampleRate = 16000
	}
	if config.Encoding == "" {
		config.Encoding = "pcm_s16le"
	}

	// Connect to Cartesia TTS WebSocket
	wsURL := "wss://api.cartesia.ai/tts/websocket"

	dialer := websocket.DefaultDialer
	header := make(map[string][]string)
	header["X-API-Key"] = []string{s.provider.GetAPIKey()}
	header["Cartesia-Version"] = []string{"2025-04-16"}

	conn, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Cartesia TTS: %w", err)
	}

	client := &cartesiaTTSClient{
		conn:    conn,
		config:  config,
		audioCh: make(chan []byte, 10),
		errCh:   make(chan error, 1),
		doneCh:  make(chan struct{}),
		closed:  false,
		logger:  s.logger,
	}

	// Start reading messages in background
	go client.readMessages()

	return client, nil
}

// Synthesize synthesizes text to audio (non-streaming)
func (s *CartesiaTTSService) Synthesize(ctx context.Context, text string, config models.TTSConfig) ([]byte, error) {
	// Create a streaming client
	client, err := s.NewTTSClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create TTS client: %w", err)
	}
	defer client.Close()

	// Send text for synthesis
	if err := client.Send(ctx, text); err != nil {
		return nil, fmt.Errorf("failed to send text: %w", err)
	}

	// Collect all audio chunks
	var audioData []byte
	for {
		chunk, err := client.Receive(ctx)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to receive audio: %w", err)
		}

		audioData = append(audioData, chunk...)
	}

	return audioData, nil
}

// GetVoices returns available voices
func (s *CartesiaTTSService) GetVoices(ctx context.Context) ([]models.Voice, error) {
	// Cartesia has many voices, here are some popular ones
	voices := []models.Voice{
		{
			ID:          "694f9389-aac1-45b6-b726-9d9369183238",
			Name:        "Sonic (Default)",
			Language:    "en",
			Gender:      "neutral",
			Description: "Default Sonic voice with natural tone",
		},
		{
			ID:          "a0e99841-438c-4a64-b679-ae501e7d6091",
			Name:        "Barbershop Man",
			Language:    "en",
			Gender:      "male",
			Description: "Friendly male voice",
		},
		{
			ID:          "79a125e8-cd45-4c13-8a67-188112f4dd22",
			Name:        "British Lady",
			Language:    "en",
			Gender:      "female",
			Description: "British accent female voice",
		},
		{
			ID:          "2ee87190-8f84-4925-97da-e52547f9462c",
			Name:        "Calm Lady",
			Language:    "en",
			Gender:      "female",
			Description: "Calm and soothing female voice",
		},
		{
			ID:          "41534374-4c8c-4e8f-a7d5-4b8e0d8e0e0e",
			Name:        "Professional Man",
			Language:    "en",
			Gender:      "male",
			Description: "Professional male voice",
		},
	}

	return voices, nil
}

// cartesiaTTSClient implements the TTSClient interface
type cartesiaTTSClient struct {
	conn    *websocket.Conn
	config  models.TTSConfig
	audioCh chan []byte
	errCh   chan error
	doneCh  chan struct{}
	mu      sync.Mutex
	closed  bool
	logger  types.Logger
}

// Send sends text to be synthesized
func (c *cartesiaTTSClient) Send(ctx context.Context, text string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("TTS client is closed")
	}

	// Generate a unique context ID for this synthesis request
	contextID := fmt.Sprintf("ctx_%d", time.Now().UnixNano())

	// Build request according to Cartesia API v2025-04-16
	request := map[string]any{
		"model_id":   c.config.Model,
		"transcript": text,
		"voice": map[string]string{
			"mode": "id",
			"id":   c.config.Voice,
		},
		"output_format": map[string]any{
			"container":   "raw",
			"encoding":    c.config.Encoding,
			"sample_rate": c.config.SampleRate,
		},
		"language":   c.config.Language,
		"context_id": contextID,
	}

	// Add optional parameters
	if c.config.Speed > 0 {
		request["speed"] = c.config.Speed
	}

	if err := c.conn.WriteJSON(request); err != nil {
		return fmt.Errorf("failed to send TTS request: %w", err)
	}

	c.logger.Debug("Sent TTS request",
		"model", c.config.Model,
		"voice", c.config.Voice,
		"text_length", len(text),
		"context_id", contextID,
	)

	return nil
}

// Receive receives synthesized audio data
func (c *cartesiaTTSClient) Receive(ctx context.Context) ([]byte, error) {
	select {
	case audio := <-c.audioCh:
		return audio, nil
	case err := <-c.errCh:
		return nil, err
	case <-c.doneCh:
		return nil, io.EOF
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close closes the TTS client and releases resources
func (c *cartesiaTTSClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.doneCh)
	return c.conn.Close()
}

// readMessages reads messages from TTS WebSocket
func (c *cartesiaTTSClient) readMessages() {
	defer func() {
		c.mu.Lock()
		if !c.closed {
			close(c.doneCh)
		}
		c.mu.Unlock()
	}()

	for {
		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()

		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			wasClosed := c.closed
			c.mu.Unlock()

			if !wasClosed {

				select {
				case c.errCh <- fmt.Errorf("TTS read error: %w", err):
				default:
				}
				c.Close()
			}
			return
		}

		if messageType == websocket.BinaryMessage {
			// Legacy: Binary audio data (shouldn't happen with new API)
			select {
			case c.audioCh <- message:
			case <-c.doneCh:
				return
			}
		} else {
			// JSON message (chunk, done, error, etc.)
			var result map[string]any
			if err := json.Unmarshal(message, &result); err != nil {

				continue
			}

			msgType, _ := result["type"].(string)

			switch msgType {
			case "chunk":
				// Audio chunk with base64-encoded data
				if dataStr, ok := result["data"].(string); ok {
					// Decode base64 audio data
					audioData, err := base64.StdEncoding.DecodeString(dataStr)
					if err != nil {

						continue
					}

					// Send decoded audio to callback
					select {
					case c.audioCh <- audioData:
						c.logger.Debug("Received audio chunk",
							"size", len(audioData),
						)
					case <-c.doneCh:
						return
					}
				}

			case "done":
				c.Close()
				return

			case "error":
				errMsg := c.extractErrorMessage(result)
				select {
				case c.errCh <- fmt.Errorf("TTS error: %s", errMsg):
				default:
				}
				c.Close()
				return
			}
		}
	}
}

// extractErrorMessage extracts error message from raw result
func (c *cartesiaTTSClient) extractErrorMessage(raw map[string]any) string {
	if msg, ok := raw["error"].(string); ok && msg != "" {
		return msg
	}
	if msg, ok := raw["message"].(string); ok && msg != "" {
		return msg
	}
	return "Unknown TTS error"
}

// GetVoices is not supported on individual client instances
// Use the service-level GetVoices method instead
func (c *cartesiaTTSClient) GetVoices(ctx context.Context) ([]models.Voice, error) {
	return nil, fmt.Errorf("GetVoices not supported on client instance, use service-level method")
}
