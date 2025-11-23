package minimax

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"
	"github.com/creastat/common-go/pkg/types"

	"github.com/gorilla/websocket"
)

// MinimaxTTSService implements the TextToSpeechService interface for MiniMax
type MinimaxTTSService struct {
	provider *MinimaxProvider
	logger   types.Logger
}

// NewMinimaxTTSService creates a new MiniMax TTS service
func NewMinimaxTTSService(provider *MinimaxProvider) *MinimaxTTSService {
	return &MinimaxTTSService{
		provider: provider,
		logger:   provider.logger,
	}
}

// NewTTSClient creates a new TTS client for streaming synthesis
func (s *MinimaxTTSService) NewTTSClient(ctx context.Context, config models.TTSConfig) (interfaces.TTSClient, error) {
	if !s.provider.IsInitialized() {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Get provider config for defaults
	providerConfig := s.provider.GetConfig()

	// Check provider config options
	if providerConfig.Options != nil {
	} else {
	}

	// Set defaults if not provided
	if config.Model == "" {
		if providerConfig.Model != "" {
			config.Model = providerConfig.Model
		} else {
			config.Model = "speech-2.6-hd"
		}
	}
	if config.Language == "" {
		config.Language = "en"
	}
	if config.Voice == "" {
		// Use default voice for the language
		config.Voice = s.GetDefaultVoiceForLanguage(config.Language)
	}
	if config.SampleRate == 0 {
		// Try to get from provider config
		if providerConfig.Options != nil {
			// Try int first, then float64 (YAML unmarshaling can produce either)
			if sr, ok := providerConfig.Options["sample_rate"].(int); ok && sr > 0 {
				config.SampleRate = sr
			} else if sr, ok := providerConfig.Options["sample_rate"].(float64); ok && sr > 0 {
				config.SampleRate = int(sr)
			} else {
				config.SampleRate = 32000
			}
		} else {
			config.SampleRate = 32000
		}
	}
	if config.Encoding == "" {
		// Try to get from provider config
		if providerConfig.Options != nil {
			if format, ok := providerConfig.Options["format"].(string); ok && format != "" {
				config.Encoding = format
			} else {
				config.Encoding = "mp3"
			}
		} else {
			config.Encoding = "mp3"
		}
	}
	if config.Speed == 0 {
		// Try to get from provider config
		if providerConfig.Options != nil {
			if speed, ok := providerConfig.Options["speed"].(float64); ok && speed > 0 {
				config.Speed = speed
			} else if speed, ok := providerConfig.Options["speed"].(int); ok && speed > 0 {
				config.Speed = float64(speed)
			} else {
				config.Speed = 1.0
			}
		} else {
			config.Speed = 1.0
		}
	}
	if config.Volume == 0 {
		// Try to get from provider config
		if providerConfig.Options != nil {
			if vol, ok := providerConfig.Options["volume"].(float64); ok && vol > 0 {
				config.Volume = vol
			} else if vol, ok := providerConfig.Options["volume"].(int); ok && vol > 0 {
				config.Volume = float64(vol)
			} else {
				config.Volume = 1.0
			}
		} else {
			config.Volume = 1.0
		}
	}

	// Connect to MiniMax TTS WebSocket
	wsURL := "wss://api.minimax.io/ws/v1/t2a_v2"

	dialer := websocket.DefaultDialer
	header := make(map[string][]string)
	header["Authorization"] = []string{fmt.Sprintf("Bearer %s", s.provider.GetAPIKey())}

	conn, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MiniMax TTS: %w", err)
	}

	client := &minimaxTTSClient{
		conn:    conn,
		config:  config,
		audioCh: make(chan []byte, 10),
		errCh:   make(chan error, 1),
		doneCh:  make(chan struct{}),
		closed:  false,
		logger:  s.logger,
	}

	// Wait for connection success message
	if err := client.waitForConnection(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("connection failed: %w", err)
	}

	// Start task
	if err := client.startTask(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to start task: %w", err)
	}

	// Start reading messages in background
	go client.readMessages()

	return client, nil
}

// Synthesize synthesizes text to audio (non-streaming)
func (s *MinimaxTTSService) Synthesize(ctx context.Context, text string, config models.TTSConfig) ([]byte, error) {

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
func (s *MinimaxTTSService) GetVoices(ctx context.Context) ([]models.Voice, error) {
	// Try to get voices from config first
	if voices := s.getVoicesFromConfig(); len(voices) > 0 {
		return voices, nil
	}

	// Fallback to hardcoded voices
	voices := []models.Voice{
		{
			ID:          "male-qn-qingse",
			Name:        "Male Qingse",
			Language:    "zh",
			Gender:      "male",
			Description: "Clear male voice with natural tone",
		},
		{
			ID:          "female-shaonv",
			Name:        "Female Shaonv",
			Language:    "zh",
			Gender:      "female",
			Description: "Young female voice",
		},
		{
			ID:          "female-yujie",
			Name:        "Female Yujie",
			Language:    "zh",
			Gender:      "female",
			Description: "Mature female voice",
		},
		{
			ID:          "male-qn-jingying",
			Name:        "Male Jingying",
			Language:    "zh",
			Gender:      "male",
			Description: "Professional male voice",
		},
		{
			ID:          "presenter_male",
			Name:        "Presenter Male",
			Language:    "en",
			Gender:      "male",
			Description: "Professional presenter voice",
		},
		{
			ID:          "presenter_female",
			Name:        "Presenter Female",
			Language:    "en",
			Gender:      "female",
			Description: "Professional female presenter voice",
		},
	}

	return voices, nil
}

// getVoicesFromConfig extracts voices from provider config
func (s *MinimaxTTSService) getVoicesFromConfig() []models.Voice {
	config := s.provider.GetConfig()
	if config.Options == nil {
		return nil
	}

	voicesConfig, ok := config.Options["voices"].(map[string]any)
	if !ok {
		return nil
	}

	var voices []models.Voice

	// Iterate through languages
	for lang, langVoices := range voicesConfig {
		voiceList, ok := langVoices.([]any)
		if !ok {
			continue
		}

		// Parse each voice
		for _, v := range voiceList {
			voiceMap, ok := v.(map[string]any)
			if !ok {
				continue
			}

			voice := models.Voice{
				Language: lang,
			}

			if id, ok := voiceMap["id"].(string); ok {
				voice.ID = id
			}
			if name, ok := voiceMap["name"].(string); ok {
				voice.Name = name
			}
			if gender, ok := voiceMap["gender"].(string); ok {
				voice.Gender = gender
			}
			if desc, ok := voiceMap["description"].(string); ok {
				voice.Description = desc
			}

			if voice.ID != "" {
				voices = append(voices, voice)
			}
		}
	}

	return voices
}

// GetVoicesByLanguage returns voices filtered by language
func (s *MinimaxTTSService) GetVoicesByLanguage(ctx context.Context, language string) ([]models.Voice, error) {
	allVoices, err := s.GetVoices(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []models.Voice
	for _, voice := range allVoices {
		if voice.Language == language {
			filtered = append(filtered, voice)
		}
	}

	return filtered, nil
}

// GetDefaultVoiceForLanguage returns the default voice for a language
func (s *MinimaxTTSService) GetDefaultVoiceForLanguage(language string) string {
	config := s.provider.GetConfig()
	if config.Options == nil {
		return s.getHardcodedDefaultVoice(language)
	}

	defaultVoices, ok := config.Options["default_voices"].(map[string]any)
	if !ok {
		return s.getHardcodedDefaultVoice(language)
	}

	if voiceID, ok := defaultVoices[language].(string); ok && voiceID != "" {
		return voiceID
	}

	return s.getHardcodedDefaultVoice(language)
}

// getHardcodedDefaultVoice returns hardcoded default voices
func (s *MinimaxTTSService) getHardcodedDefaultVoice(language string) string {
	defaults := map[string]string{
		"en": "presenter_male",
		"zh": "male-qn-qingse",
		"ru": "Russian_ReliableMan", // Russian default
	}

	if voice, ok := defaults[language]; ok {
		return voice
	}

	return "male-qn-qingse" // Global default
}

// minimaxTTSClient implements the TTSClient interface
type minimaxTTSClient struct {
	conn    *websocket.Conn
	config  models.TTSConfig
	audioCh chan []byte
	errCh   chan error
	doneCh  chan struct{}
	mu      sync.Mutex
	closed  bool
	logger  types.Logger
}

// waitForConnection waits for the connection success message
func (c *minimaxTTSClient) waitForConnection() error {
	_, message, err := c.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read connection message: %w", err)
	}

	var response map[string]any
	if err := json.Unmarshal(message, &response); err != nil {
		return fmt.Errorf("failed to parse connection message: %w", err)
	}

	if event, ok := response["event"].(string); !ok || event != "connected_success" {
		return fmt.Errorf("unexpected connection response: %v", response)
	}

	return nil
}

// startTask sends the task_start message
func (c *minimaxTTSClient) startTask() error {
	// Build task_start request
	request := map[string]any{
		"event": "task_start",
		"model": c.config.Model,
		"voice_setting": map[string]any{
			"voice_id":              c.config.Voice,
			"speed":                 c.config.Speed,
			"vol":                   c.config.Volume,
			"pitch":                 c.config.Pitch,
			"english_normalization": false,
		},
		"audio_setting": map[string]any{
			"sample_rate": c.config.SampleRate,
			"bitrate":     128000,
			"format":      c.config.Encoding,
			"channel":     1,
		},
	}

	if err := c.conn.WriteJSON(request); err != nil {
		return fmt.Errorf("failed to send task_start: %w", err)
	}

	// Wait for task_started response
	_, message, err := c.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read task_started message: %w", err)
	}

	var response map[string]any
	if err := json.Unmarshal(message, &response); err != nil {
		return fmt.Errorf("failed to parse task_started message: %w", err)
	}

	if event, ok := response["event"].(string); !ok || event != "task_started" {
		return fmt.Errorf("unexpected task_started response: %v", response)
	}

	return nil
}

// Send sends text to be synthesized
func (c *minimaxTTSClient) Send(ctx context.Context, text string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("TTS client is closed")
	}

	// Build task_continue request
	request := map[string]any{
		"event": "task_continue",
		"text":  text,
	}

	if err := c.conn.WriteJSON(request); err != nil {
		return fmt.Errorf("failed to send TTS request: %w", err)
	}

	c.logger.Debug("Sent TTS request",
		"model", c.config.Model,
		"voice", c.config.Voice,
		"text_length", len(text),
	)

	return nil
}

// Receive receives synthesized audio data
func (c *minimaxTTSClient) Receive(ctx context.Context) ([]byte, error) {
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
func (c *minimaxTTSClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	// Send task_finish message but DON'T set closed=true yet
	// Let readMessages handle the final cleanup when it receives task_finished
	finishMsg := map[string]any{
		"event": "task_finish",
	}
	if err := c.conn.WriteJSON(finishMsg); err != nil {
		c.closed = true
		close(c.doneCh)
		return c.conn.Close()
	}

	// Don't close the connection or set closed=true
	// The readMessages goroutine will handle cleanup when it receives task_finished
	return nil
}

// readMessages reads messages from TTS WebSocket
func (c *minimaxTTSClient) readMessages() {
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
				case c.errCh <- fmt.Errorf("TTS read error: %w", err):
				default:
				}
				c.Close()
			}
			return
		}

		// Parse JSON message
		var response map[string]any
		if err := json.Unmarshal(message, &response); err != nil {
			continue
		}

		// Check event type
		event, _ := response["event"].(string)

		// Handle different event types
		switch event {
		case "task_continued":
			// Check for audio data
			if data, ok := response["data"].(map[string]any); ok {
				if audioHex, ok := data["audio"].(string); ok && audioHex != "" {
					// Decode hex audio data
					audioData, err := hex.DecodeString(audioHex)
					if err != nil {
						continue
					}

					// Send decoded audio to channel
					select {
					case c.audioCh <- audioData:
						c.logger.Debug("Received audio chunk",
							"size", len(audioData),
						)
					case <-c.doneCh:
						return
					}
				}
			}

		case "task_finished":
			c.mu.Lock()
			if !c.closed {
				c.closed = true
				close(c.doneCh)
			}
			c.mu.Unlock()
			c.conn.Close()
			return

		case "task_failed":
			errMsg := c.extractErrorMessage(response)
			c.mu.Lock()
			if !c.closed {
				c.closed = true
				select {
				case c.errCh <- fmt.Errorf("TTS task failed: %s", errMsg):
				default:
				}
				close(c.doneCh)
			}
			c.mu.Unlock()
			c.conn.Close()
			return
		}
	}
}

// extractErrorMessage extracts error message from response
func (c *minimaxTTSClient) extractErrorMessage(response map[string]any) string {
	if msg, ok := response["error"].(string); ok && msg != "" {
		return msg
	}
	if msg, ok := response["message"].(string); ok && msg != "" {
		return msg
	}
	return "Unknown TTS error"
}

// GetVoices is not supported on individual client instances
// Use the service-level GetVoices method instead
func (c *minimaxTTSClient) GetVoices(ctx context.Context) ([]models.Voice, error) {
	return nil, fmt.Errorf("GetVoices not supported on client instance, use service-level method")
}
