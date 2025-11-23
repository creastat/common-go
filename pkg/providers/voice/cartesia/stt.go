package cartesia

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"

	"github.com/gorilla/websocket"
)

// CartesiaSTTService implements the SpeechToTextService interface for Cartesia
type CartesiaSTTService struct {
	provider *CartesiaProvider
}

// NewCartesiaSTTService creates a new Cartesia STT service
func NewCartesiaSTTService(provider *CartesiaProvider) *CartesiaSTTService {
	return &CartesiaSTTService{
		provider: provider,
	}
}

// NewSTTClient creates a new STT client for streaming audio
func (s *CartesiaSTTService) NewSTTClient(ctx context.Context, config models.STTConfig) (interfaces.STTClient, error) {
	if !s.provider.IsInitialized() {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Set defaults if not provided
	if config.Model == "" {
		config.Model = "ink-whisper"
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

	// Extract Cartesia-specific options
	minVolume := 0.05         // Default: 5% threshold for speech detection
	maxSilenceDuration := 1.0 // Default: 1 second of silence before finalizing

	if config.Options != nil {
		if mv, ok := config.Options["min_volume"].(float64); ok {
			minVolume = mv
		}
		if msd, ok := config.Options["max_silence_duration_secs"].(float64); ok {
			maxSilenceDuration = msd
		}
	}

	// Build WebSocket URL with query parameters
	wsURL := fmt.Sprintf(
		"wss://api.cartesia.ai/stt/websocket?model=%s&language=%s&encoding=%s&sample_rate=%d&min_volume=%f&max_silence_duration_secs=%f",
		config.Model,
		config.Language,
		config.Encoding,
		config.SampleRate,
		minVolume,
		maxSilenceDuration,
	)

	// Create WebSocket connection
	dialer := websocket.DefaultDialer
	header := make(map[string][]string)
	header["X-API-Key"] = []string{s.provider.GetAPIKey()}
	header["Cartesia-Version"] = []string{"2024-06-10"}

	conn, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Cartesia STT: %w", err)
	}

	client := &cartesiaSTTClient{
		conn:     conn,
		config:   config,
		resultCh: make(chan *models.STTResult, 10),
		errCh:    make(chan error, 1),
		doneCh:   make(chan struct{}),
		closed:   false,
	}

	// Start reading messages in background
	go client.readMessages()

	return client, nil
}

// Transcribe transcribes audio data to text (non-streaming)
func (s *CartesiaSTTService) Transcribe(ctx context.Context, audio io.Reader, config models.STTConfig) (string, error) {
	// Create a streaming client
	client, err := s.NewSTTClient(ctx, config)
	if err != nil {
		return "", fmt.Errorf("failed to create STT client: %w", err)
	}
	defer client.Close()

	// Read audio data and send to client
	buffer := make([]byte, 4096)
	for {
		n, err := audio.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("failed to read audio: %w", err)
		}

		if n > 0 {
			if err := client.Send(ctx, buffer[:n]); err != nil {
				return "", fmt.Errorf("failed to send audio: %w", err)
			}
		}
	}

	// Collect all results
	var fullText string
	for {
		result, err := client.Receive(ctx)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("failed to receive result: %w", err)
		}

		if result.IsFinal {
			fullText += result.Text + " "
		}
	}

	return fullText, nil
}

// GetModels returns available STT models
func (s *CartesiaSTTService) GetModels(ctx context.Context) ([]models.Model, error) {
	modelsList := []models.Model{
		{
			ID:          "sonic",
			Name:        "Sonic",
			Description: "Fast and accurate speech recognition model",
		},
	}

	return modelsList, nil
}

// cartesiaSTTClient implements the STTClient interface
type cartesiaSTTClient struct {
	conn     *websocket.Conn
	config   models.STTConfig
	resultCh chan *models.STTResult
	errCh    chan error
	doneCh   chan struct{}
	mu       sync.Mutex
	closed   bool
}

// Send sends audio data to the STT service
func (c *cartesiaSTTClient) Send(ctx context.Context, audio []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("STT client is closed")
	}

	if err := c.conn.WriteMessage(websocket.BinaryMessage, audio); err != nil {

		return fmt.Errorf("failed to send audio: %w", err)
	}

	return nil
}

// Receive receives transcription results from the STT service
func (c *cartesiaSTTClient) Receive(ctx context.Context) (*models.STTResult, error) {
	select {
	case result := <-c.resultCh:
		return result, nil
	case err := <-c.errCh:
		return nil, err
	case <-c.doneCh:
		return nil, io.EOF
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close closes the STT client and releases resources
func (c *cartesiaSTTClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.doneCh)
	return c.conn.Close()
}

// Finalize flushes any buffered audio and forces Cartesia to send transcript
// without closing the connection
func (c *cartesiaSTTClient) Finalize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, []byte("finalize")); err != nil {

		return fmt.Errorf("failed to send finalize command: %w", err)
	}

	return nil
}

// Flush signals end of audio stream by sending 'done' command
func (c *cartesiaSTTClient) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, []byte("done")); err != nil {

		return fmt.Errorf("failed to send done command: %w", err)
	}

	c.closed = true
	return nil
}

// readMessages reads messages from STT WebSocket
func (c *cartesiaSTTClient) readMessages() {
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

		_, message, err := c.conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			wasClosed := c.closed
			c.mu.Unlock()

			if !wasClosed {

				select {
				case c.errCh <- fmt.Errorf("STT read error: %w", err):
				default:
				}
				c.Close()
			}
			return
		}

		// Parse the message
		var rawResult map[string]any
		if err := json.Unmarshal(message, &rawResult); err != nil {
			continue
		}

		msgType, _ := rawResult["type"].(string)

		// Handle different message types
		switch msgType {
		case "transcript":
			result := c.parseTranscriptResult(rawResult)
			select {
			case c.resultCh <- result:
			case <-c.doneCh:
				return
			}

		case "error":
			errMsg := c.extractErrorMessage(rawResult)

			select {
			case c.errCh <- fmt.Errorf("Cartesia STT error: %s", errMsg):
			default:
			}
			c.Close()
			return

		case "flush_done":
			// Connection stays open, continue processing

		case "done":
			c.Close()
			return
		}
	}
}

// parseTranscriptResult parses a transcript message into STTResult
func (c *cartesiaSTTClient) parseTranscriptResult(raw map[string]any) *models.STTResult {
	result := &models.STTResult{
		Metadata: make(map[string]any),
	}

	if text, ok := raw["text"].(string); ok {
		result.Text = text
	}

	if isFinal, ok := raw["is_final"].(bool); ok {
		result.IsFinal = isFinal
	} else if isFinal, ok := raw["isFinal"].(bool); ok {
		// Handle camelCase variant
		result.IsFinal = isFinal
	}

	// Parse word-level timestamps
	if words, ok := raw["words"].([]any); ok {
		result.Words = make([]models.WordInfo, 0, len(words))
		for _, w := range words {
			if wordMap, ok := w.(map[string]any); ok {
				word := models.WordInfo{}
				if wordText, ok := wordMap["word"].(string); ok {
					word.Word = wordText
				}
				if start, ok := wordMap["start"].(float64); ok {
					word.StartTime = start
				}
				if end, ok := wordMap["end"].(float64); ok {
					word.EndTime = end
				}
				result.Words = append(result.Words, word)
			}
		}
	}

	return result
}

// extractErrorMessage extracts error message from raw result
func (c *cartesiaSTTClient) extractErrorMessage(raw map[string]any) string {
	if msg, ok := raw["message"].(string); ok && msg != "" {
		return msg
	}
	if err, ok := raw["error"].(string); ok && err != "" {
		return err
	}
	return "Unknown STT error"
}
