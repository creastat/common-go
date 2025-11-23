package deepgram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sync"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"
	"github.com/creastat/common-go/pkg/types"

	"github.com/gorilla/websocket"
)

// DeepgramSTTService implements the SpeechToTextService interface for Deepgram
type DeepgramSTTService struct {
	provider *DeepgramProvider
	logger   types.Logger
}

// NewDeepgramSTTService creates a new Deepgram STT service
func NewDeepgramSTTService(provider *DeepgramProvider) *DeepgramSTTService {
	return &DeepgramSTTService{
		provider: provider,
		logger:   provider.logger,
	}
}

// NewSTTClient creates a new STT client for streaming audio
func (s *DeepgramSTTService) NewSTTClient(ctx context.Context, config models.STTConfig) (interfaces.STTClient, error) {
	if !s.provider.IsInitialized() {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Set defaults if not provided
	if config.Model == "" {
		config.Model = "nova-3" // Use latest Nova 3 model by default
	}
	if config.Language == "" {
		config.Language = "en"
	}
	if config.SampleRate == 0 {
		config.SampleRate = 16000
	}
	if config.Encoding == "" {
		config.Encoding = "linear16"
	}

	// Map generic "raw" encoding to Deepgram's "linear16"
	if config.Encoding == "raw" {
		config.Encoding = "linear16"
	}

	// Extract Deepgram-specific options
	channels := 1
	multichannel := false
	smartFormat := true
	diarize := false
	utteranceEndMs := 0 // Disabled by default (requires interim_results)
	vadEvents := false  // Disabled by default

	if config.Options != nil {
		if ch, ok := config.Options["channels"].(int); ok {
			channels = ch
		}
		if mc, ok := config.Options["multichannel"].(bool); ok {
			multichannel = mc
		}
		if sf, ok := config.Options["smart_format"].(bool); ok {
			smartFormat = sf
		}
		if d, ok := config.Options["diarize"].(bool); ok {
			diarize = d
		}
		if uem, ok := config.Options["utterance_end_ms"].(int); ok {
			utteranceEndMs = uem
		}
		if ve, ok := config.Options["vad_events"].(bool); ok {
			vadEvents = ve
		}
	}

	// If utterance_end_ms is set, interim_results must be enabled
	if utteranceEndMs > 0 && !config.InterimResults {
		config.InterimResults = true
	}

	// Build WebSocket URL with query parameters
	u, _ := url.Parse("wss://api.deepgram.com/v1/listen")
	query := u.Query()
	query.Set("model", config.Model)
	query.Set("encoding", config.Encoding)
	query.Set("sample_rate", fmt.Sprintf("%d", config.SampleRate))
	query.Set("channels", fmt.Sprintf("%d", channels))
	query.Set("multichannel", fmt.Sprintf("%t", multichannel))
	query.Set("smart_format", fmt.Sprintf("%t", smartFormat))
	query.Set("diarize", fmt.Sprintf("%t", diarize))
	query.Set("interim_results", fmt.Sprintf("%t", config.InterimResults))

	// Only add utterance_end_ms if it's enabled (requires interim_results)
	if utteranceEndMs > 0 {
		query.Set("utterance_end_ms", fmt.Sprintf("%d", utteranceEndMs))
	}

	// Only add vad_events if enabled
	if vadEvents {
		query.Set("vad_events", "true")
	}

	if config.Language != "" {
		query.Set("language", config.Language)
	}

	if config.PunctuationEnabled {
		query.Set("punctuate", "true")
	}

	u.RawQuery = query.Encode()

	// Create WebSocket connection
	dialer := websocket.DefaultDialer
	header := make(map[string][]string)
	header["Authorization"] = []string{fmt.Sprintf("token %s", s.provider.GetAPIKey())}

	conn, resp, err := dialer.Dial(u.String(), header)
	if err != nil {
		if resp != nil {
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			resp.Body.Close()
			return nil, fmt.Errorf("failed to connect to Deepgram STT (status: %d): %s - %w", resp.StatusCode, string(body[:n]), err)
		}
		return nil, fmt.Errorf("failed to connect to Deepgram STT: %w", err)
	}

	client := &deepgramSTTClient{
		conn:     conn,
		config:   config,
		resultCh: make(chan *models.STTResult, 10),
		errCh:    make(chan error, 1),
		doneCh:   make(chan struct{}),
		closed:   false,
		logger:   s.logger,
	}

	s.logger.Debug("Connected to Deepgram STT",
		"model", config.Model,
		"language", config.Language,
		"sample_rate", config.SampleRate,
		"encoding", config.Encoding,
	)

	// Start reading messages in background
	go client.readMessages()

	return client, nil
}

// Transcribe transcribes audio data to text (non-streaming)
func (s *DeepgramSTTService) Transcribe(ctx context.Context, audio io.Reader, config models.STTConfig) (string, error) {
	// Create a streaming client
	client, err := s.NewSTTClient(ctx, config)
	if err != nil {
		return "", fmt.Errorf("failed to create STT client: %w", err)
	}
	defer client.Close()

	// Channel to collect results
	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// Start goroutine to collect results
	go func() {
		var fullText string
		for {
			result, err := client.Receive(ctx)
			if err != nil {
				if err == io.EOF {
					resultCh <- fullText
					return
				}
				errCh <- fmt.Errorf("failed to receive result: %w", err)
				return
			}

			if result.IsFinal {
				fullText += result.Text + " "
			}
		}
	}()

	// Read audio data and send to client
	buffer := make([]byte, 4096)
	totalBytes := 0
	for {
		n, err := audio.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("failed to read audio: %w", err)
		}

		if n > 0 {
			totalBytes += n
			if err := client.Send(ctx, buffer[:n]); err != nil {
				return "", fmt.Errorf("failed to send audio: %w", err)
			}
		}
	}

	// Finalize to signal end of audio stream
	// Cast to concrete type to access Finalize method
	if deepgramClient, ok := client.(*deepgramSTTClient); ok {
		if err := deepgramClient.Finalize(); err != nil {
			return "", fmt.Errorf("failed to finalize audio stream: %w", err)
		}
	}

	// Wait for results
	select {
	case text := <-resultCh:
		return text, nil
	case err := <-errCh:
		return "", err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// GetModels returns available STT models
func (s *DeepgramSTTService) GetModels(ctx context.Context) ([]models.Model, error) {
	modelsList := []models.Model{
		{
			ID:          "nova-2-general",
			Name:        "Nova 2 General",
			Description: "Latest general-purpose speech recognition model with high accuracy",
		},
		{
			ID:          "nova-2-phonecall",
			Name:        "Nova 2 Phone Call",
			Description: "Optimized for phone call audio with enhanced accuracy for telephony",
		},
		{
			ID:          "nova-2-meeting",
			Name:        "Nova 2 Meeting",
			Description: "Optimized for meeting and conference audio with multiple speakers",
		},
		{
			ID:          "whisper-large",
			Name:        "Whisper Large",
			Description: "OpenAI Whisper large model for high-accuracy transcription",
		},
	}

	return modelsList, nil
}

// deepgramSTTClient implements the STTClient interface
type deepgramSTTClient struct {
	conn     *websocket.Conn
	config   models.STTConfig
	resultCh chan *models.STTResult
	errCh    chan error
	doneCh   chan struct{}
	mu       sync.Mutex
	closed   bool
	logger   types.Logger
}

// Send sends audio data to the STT service
func (c *deepgramSTTClient) Send(ctx context.Context, audio []byte) error {
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
func (c *deepgramSTTClient) Receive(ctx context.Context) (*models.STTResult, error) {
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
func (c *deepgramSTTClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.doneCh)
	return c.conn.Close()
}

// Finalize sends a CloseStream message to complete the transcription
func (c *deepgramSTTClient) Finalize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("STT client is closed")
	}

	// Deepgram expects a CloseStream message, not Finalize
	closeMessage := map[string]any{
		"type": "CloseStream",
	}

	jsonData, err := json.Marshal(closeMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal close stream message: %w", err)
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		return fmt.Errorf("failed to send close stream message: %w", err)
	}

	return nil
}

// readMessages reads messages from STT WebSocket
func (c *deepgramSTTClient) readMessages() {
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
				// Check if it's a normal close (1000)
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					// Normal close - just signal done
					c.Close()
					return
				}

				// Other errors - send to error channel
				select {
				case c.errCh <- fmt.Errorf("STT read error: %w", err):
				default:
				}
				c.Close()
			}
			return
		}

		if messageType == websocket.TextMessage {
			// Parse the message
			var rawResult map[string]any
			if err := json.Unmarshal(message, &rawResult); err != nil {
				continue
			}

			msgType, _ := rawResult["type"].(string)

			// Handle different message types
			switch msgType {
			case "Results":
				result := c.parseResultsMessage(rawResult)
				if result != nil {
					// Log transcript at trace level
					if result.Text != "" {
						c.logger.Debug("Deepgram STT result",
							"text", result.Text,
							"is_final", result.IsFinal,
							"confidence", result.Confidence,
						)
					}

					select {
					case c.resultCh <- result:
					case <-c.doneCh:
						return
					}
				}

			case "Metadata":
				// Handle metadata separately if needed

			case "UtteranceEnd":
				// Handle utterance end

			case "SpeechStarted":
				// Handle speech started

			default:
			}
		}
	}
}

// parseResultsMessage parses a Results message into STTResult
func (c *deepgramSTTClient) parseResultsMessage(raw map[string]any) *models.STTResult {
	result := &models.STTResult{
		Metadata: make(map[string]any),
	}

	// Extract is_final and speech_final flags
	if isFinal, ok := raw["is_final"].(bool); ok {
		result.IsFinal = isFinal
	}
	if speechFinal, ok := raw["speech_final"].(bool); ok {
		result.Metadata["speech_final"] = speechFinal
	}

	// Extract duration and start time
	if duration, ok := raw["duration"].(float64); ok {
		result.EndTime = duration
		result.Metadata["duration"] = duration
	}
	if start, ok := raw["start"].(float64); ok {
		result.StartTime = start
	}

	// Extract channel data - can be either object or array depending on Deepgram response format
	var channelMap map[string]any

	// Try as object first (newer format)
	if channel, ok := raw["channel"].(map[string]any); ok {
		channelMap = channel
	} else if channelData, ok := raw["channel"].([]any); ok && len(channelData) > 0 {
		// Try as array (older format)
		if ch, ok := channelData[0].(map[string]any); ok {
			channelMap = ch
		}
	}

	if channelMap != nil {
		if alternatives, ok := channelMap["alternatives"].([]any); ok && len(alternatives) > 0 {
			if alt, ok := alternatives[0].(map[string]any); ok {
				// Extract transcript
				if transcript, ok := alt["transcript"].(string); ok {
					result.Text = transcript
				}

				// Extract confidence
				if confidence, ok := alt["confidence"].(float64); ok {
					result.Confidence = confidence
				}

				// Extract words with timing information
				if words, ok := alt["words"].([]any); ok {
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
							if confidence, ok := wordMap["confidence"].(float64); ok {
								word.Confidence = confidence
							}
							result.Words = append(result.Words, word)
						}
					}
				}
			}
		}
	}

	return result
}
