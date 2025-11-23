package yandex

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"sync"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"
	stt "github.com/creastat/common-go/pkg/providers/voice/yandex/proto/generated/stt"
	"github.com/creastat/common-go/pkg/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const (
	yandexSTTEndpoint = "stt.api.cloud.yandex.net:443"
)

// YandexSTTService implements the SpeechToTextService interface for Yandex SpeechKit
type YandexSTTService struct {
	provider *YandexProvider
	logger   types.Logger
}

// NewYandexSTTService creates a new Yandex STT service
func NewYandexSTTService(provider *YandexProvider) *YandexSTTService {
	return &YandexSTTService{
		provider: provider,
		logger:   provider.logger,
	}
}

// NewSTTClient creates a new STT client for streaming audio
func (s *YandexSTTService) NewSTTClient(ctx context.Context, config models.STTConfig) (interfaces.STTClient, error) {
	if !s.provider.IsInitialized() {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Set defaults if not provided
	if config.Model == "" {
		config.Model = "general"
	}
	if config.Language == "" {
		config.Language = "ru-RU"
	}
	if config.SampleRate == 0 {
		config.SampleRate = 8000
	}
	if config.Encoding == "" {
		config.Encoding = "linear16"
	}
	if config.Channels == 0 {
		config.Channels = 1
	}

	// Create gRPC connection
	creds := credentials.NewTLS(&tls.Config{})
	conn, err := grpc.NewClient(
		yandexSTTEndpoint,
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(10*1024*1024)), // 10MB max receive size
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Yandex STT: %w", err)
	}

	// Create streaming client
	client := &yandexSTTClient{
		conn:     conn,
		config:   config,
		provider: s.provider,
		resultCh: make(chan *models.STTResult, 10),
		errCh:    make(chan error, 1),
		doneCh:   make(chan struct{}),
		closed:   false,
		logger:   s.logger,
	}

	// Initialize the stream
	if err := client.initStream(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize stream: %w", err)
	}

	return client, nil
}

// Transcribe transcribes audio data to text (non-streaming)
func (s *YandexSTTService) Transcribe(ctx context.Context, audio io.Reader, config models.STTConfig) (string, error) {
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

	// Close the client to signal end of audio
	client.Close()

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
func (s *YandexSTTService) GetModels(ctx context.Context) ([]models.Model, error) {
	modelsList := []models.Model{
		{
			ID:          "general",
			Name:        "General",
			Description: "General-purpose speech recognition model",
		},
		{
			ID:          "general:rc",
			Name:        "General RC",
			Description: "Release candidate of general model with latest improvements",
		},
		{
			ID:          "general:deprecated",
			Name:        "General (Deprecated)",
			Description: "Older version of general model",
		},
		{
			ID:          "general:rc:deprecated",
			Name:        "General RC (Deprecated)",
			Description: "Older release candidate version",
		},
	}

	return modelsList, nil
}

// yandexSTTClient implements the STTClient interface
type yandexSTTClient struct {
	conn     *grpc.ClientConn
	stream   stt.Recognizer_RecognizeStreamingClient
	config   models.STTConfig
	provider *YandexProvider
	resultCh chan *models.STTResult
	errCh    chan error
	doneCh   chan struct{}
	mu       sync.Mutex
	closed   bool
	logger   types.Logger
}

// initStream initializes the bidirectional streaming connection
func (c *yandexSTTClient) initStream(ctx context.Context) error {
	fmt.Printf("[YANDEX STT] Initializing stream with model=%s, language=%s, sample_rate=%d\n",
		c.config.Model, c.config.Language, c.config.SampleRate)

	// Add authorization metadata
	md := metadata.New(map[string]string{
		"authorization": fmt.Sprintf("Api-Key %s", c.provider.GetAPIKey()),
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Create the recognizer client
	recognizerClient := stt.NewRecognizerClient(c.conn)

	// Start bidirectional stream
	fmt.Println("[YANDEX STT] Starting RecognizeStreaming RPC")
	stream, err := recognizerClient.RecognizeStreaming(ctx)
	if err != nil {
		fmt.Printf("[YANDEX STT] Failed to start streaming: %v\n", err)
		return fmt.Errorf("failed to start streaming: %w", err)
	}

	c.stream = stream
	fmt.Println("[YANDEX STT] Stream created successfully")

	// Send session options as first message
	sessionOptions := c.buildSessionOptions()

	// Validate that recognition model is set
	if sessionOptions == nil || sessionOptions.RecognitionModel == nil {
		return fmt.Errorf("recognition model is not set in session options")
	}
	if sessionOptions.RecognitionModel.Model == "" {
		return fmt.Errorf("recognition model name is empty")
	}
	if sessionOptions.RecognitionModel.AudioFormat == nil {
		return fmt.Errorf("audio format is not set in recognition model")
	}

	req := &stt.StreamingRequest{
		Event: &stt.StreamingRequest_SessionOptions{
			SessionOptions: sessionOptions,
		},
	}

	c.logger.Debug("Sending Yandex STT session options",
		"model", sessionOptions.RecognitionModel.Model,
		"sample_rate", int(sessionOptions.RecognitionModel.AudioFormat.GetRawAudio().SampleRateHertz),
	)

	if err := stream.Send(req); err != nil {
		return fmt.Errorf("failed to send session options: %w", err)
	}

	fmt.Println("[YANDEX STT] Session options sent, starting message reader goroutine")
	// Start reading responses in background
	go c.readMessages()

	return nil
}

// buildSessionOptions creates the session options from config
func (c *yandexSTTClient) buildSessionOptions() *stt.StreamingOptions {
	// Map encoding
	audioEncoding := stt.RawAudio_LINEAR16_PCM
	if c.config.Encoding == "opus" {
		// For OPUS, we'd use ContainerAudio instead
	}

	// Build audio format options with proper union type
	audioFormatOptions := &stt.AudioFormatOptions{
		AudioFormat: &stt.AudioFormatOptions_RawAudio{
			RawAudio: &stt.RawAudio{
				AudioEncoding:     audioEncoding,
				SampleRateHertz:   int64(c.config.SampleRate),
				AudioChannelCount: int64(c.config.Channels),
			},
		},
	}

	// Build recognition model options
	recognitionModel := &stt.RecognitionModelOptions{
		Model:               c.config.Model,
		AudioFormat:         audioFormatOptions,
		AudioProcessingType: stt.RecognitionModelOptions_REAL_TIME,
	}

	// Add language restriction if specified
	if c.config.Language != "" {
		// Normalize language code to Yandex format
		normalizedLang := c.normalizeLanguageCode(c.config.Language)
		fmt.Printf("[YANDEX STT] Language code: %s -> %s\n", c.config.Language, normalizedLang)

		recognitionModel.LanguageRestriction = &stt.LanguageRestrictionOptions{
			RestrictionType: stt.LanguageRestrictionOptions_WHITELIST,
			LanguageCode:    []string{normalizedLang},
		}
	}

	// Add text normalization options
	if c.config.PunctuationEnabled {
		recognitionModel.TextNormalization = &stt.TextNormalizationOptions{
			TextNormalization: stt.TextNormalizationOptions_TEXT_NORMALIZATION_ENABLED,
			ProfanityFilter:   false,
			LiteratureText:    false,
		}
	}

	// Build EOU classifier options
	eouClassifier := &stt.EouClassifierOptions{
		Classifier: &stt.EouClassifierOptions_DefaultClassifier{
			DefaultClassifier: &stt.DefaultEouClassifier{
				Type:                       stt.DefaultEouClassifier_DEFAULT,
				MaxPauseBetweenWordsHintMs: 1000,
			},
		},
	}

	return &stt.StreamingOptions{
		RecognitionModel: recognitionModel,
		EouClassifier:    eouClassifier,
	}
}

// Send sends audio data to the STT service
func (c *yandexSTTClient) Send(ctx context.Context, audio []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		fmt.Println("[YANDEX STT] Attempted to send audio on closed client")
		return fmt.Errorf("STT client is closed")
	}

	// Send audio chunk
	req := &stt.StreamingRequest{
		Event: &stt.StreamingRequest_Chunk{
			Chunk: &stt.AudioChunk{
				Data: audio,
			},
		},
	}

	if err := c.stream.Send(req); err != nil {
		return fmt.Errorf("failed to send audio: %w", err)
	}

	return nil
}

// Receive receives transcription results from the STT service
func (c *yandexSTTClient) Receive(ctx context.Context) (*models.STTResult, error) {
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

// Finalize finalizes the STT stream by sending end-of-stream marker
func (c *yandexSTTClient) Finalize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		fmt.Println("[YANDEX STT] Finalize called on already closed client")
		return fmt.Errorf("STT client is closed")
	}

	fmt.Println("[YANDEX STT] Finalizing STT stream")

	if c.stream != nil {
		fmt.Println("[YANDEX STT] Sending CloseSend to signal end of audio")
		if err := c.stream.CloseSend(); err != nil {
			fmt.Printf("[YANDEX STT] Error during CloseSend: %v\n", err)
			return fmt.Errorf("failed to finalize stream: %w", err)
		}
		fmt.Println("[YANDEX STT] CloseSend completed successfully")
	}

	return nil
}

// Close closes the STT client and releases resources
func (c *yandexSTTClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		fmt.Println("[YANDEX STT] Close called on already closed client")
		return nil
	}

	fmt.Println("[YANDEX STT] Closing STT client")
	c.closed = true
	close(c.doneCh)

	if c.stream != nil {
		fmt.Println("[YANDEX STT] Closing stream send")
		c.stream.CloseSend()
	}

	if c.conn != nil {
		fmt.Println("[YANDEX STT] Closing gRPC connection")
		return c.conn.Close()
	}

	fmt.Println("[YANDEX STT] Client closed successfully")
	return nil
}

// readMessages reads messages from the STT stream
func (c *yandexSTTClient) readMessages() {
	defer func() {
		c.mu.Lock()
		if !c.closed {
			close(c.doneCh)
		}
		c.mu.Unlock()
	}()

	messageCount := 0
	for {
		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			return
		}
		stream := c.stream
		c.mu.Unlock()

		messageCount++
		resp, err := stream.Recv()
		if err != nil {
			c.mu.Lock()
			wasClosed := c.closed
			c.mu.Unlock()

			if !wasClosed {
				if err == io.EOF {
					// EOF might indicate server closed due to error
					// Try to get the trailer metadata which might contain error info
					fmt.Printf("[YANDEX STT] EOF received after %d messages\n", messageCount)

					// Try to get trailer metadata for error details
					if trailer := stream.Trailer(); len(trailer) > 0 {
						fmt.Printf("[YANDEX STT] Trailer metadata: %v\n", trailer)
					}

					c.Close()
					return
				}

				// Log the actual error for debugging - include full error details
				errMsg := fmt.Sprintf("STT read error after %d messages: %v (type: %T)", messageCount, err, err)
				fmt.Printf("[YANDEX STT] %s\n", errMsg)

				select {
				case c.errCh <- fmt.Errorf(errMsg):
				default:
				}
				c.Close()
			}
			return
		}

		if resp != nil {
			// Process the response
			result := c.parseResponse(resp)
			if result != nil {
				// Log transcript at trace level
				if result.Text != "" {
					c.logger.Debug("Yandex STT result",
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
		}
	}
}

// parseResponse converts Yandex response to STTResult
func (c *yandexSTTClient) parseResponse(resp *stt.StreamingResponse) *models.STTResult {
	result := &models.STTResult{
		Metadata: make(map[string]any),
	}

	// Handle different event types
	switch event := resp.Event.(type) {
	case *stt.StreamingResponse_Partial:
		// Partial results
		if event.Partial != nil && len(event.Partial.Alternatives) > 0 {
			alt := event.Partial.Alternatives[0]
			result.Text = alt.Text
			result.IsFinal = false
			result.Confidence = alt.Confidence
			result.StartTime = float64(alt.StartTimeMs) / 1000.0
			result.EndTime = float64(alt.EndTimeMs) / 1000.0
			result.Words = c.parseWords(alt.Words)
		}

	case *stt.StreamingResponse_Final:
		// Final results
		if event.Final != nil && len(event.Final.Alternatives) > 0 {
			alt := event.Final.Alternatives[0]
			result.Text = alt.Text
			result.IsFinal = true
			result.Confidence = alt.Confidence
			result.StartTime = float64(alt.StartTimeMs) / 1000.0
			result.EndTime = float64(alt.EndTimeMs) / 1000.0
			result.Words = c.parseWords(alt.Words)
		}

	case *stt.StreamingResponse_EouUpdate:
		// End of utterance
		result.Metadata["eou"] = true
		result.Metadata["eou_time_ms"] = event.EouUpdate.TimeMs
		return nil // Don't send EOU as a result

	case *stt.StreamingResponse_FinalRefinement:
		// Final refinement (normalized text)
		if event.FinalRefinement != nil && event.FinalRefinement.GetNormalizedText() != nil {
			normalized := event.FinalRefinement.GetNormalizedText()
			if len(normalized.Alternatives) > 0 {
				alt := normalized.Alternatives[0]
				result.Text = alt.Text
				result.IsFinal = true
				result.Confidence = alt.Confidence
				result.StartTime = float64(alt.StartTimeMs) / 1000.0
				result.EndTime = float64(alt.EndTimeMs) / 1000.0
				result.Words = c.parseWords(alt.Words)
				result.Metadata["normalized"] = true
			}
		}

	case *stt.StreamingResponse_StatusCode:
		// Status messages
		result.Metadata["status"] = event.StatusCode.Message
		return nil // Don't send status as a result

	default:
		return nil
	}

	return result
}

// normalizeLanguageCode converts language codes to Yandex-supported format
// Yandex supports: de-DE, en-US, es-ES, fi-FI, fr-FR, he-IL, it-IT, kk-KZ, nl-NL, pl-PL, pt-PT, pt-BR, ru-RU, sv-SE, tr-TR, uz-UZ
func (c *yandexSTTClient) normalizeLanguageCode(lang string) string {
	// Map of common language codes to Yandex supported codes
	langMap := map[string]string{
		// English variants
		"en":    "en-US",
		"en-US": "en-US",
		"en-GB": "en-US", // Fallback to US English
		"en-AU": "en-US",
		"en-CA": "en-US",
		"en-NZ": "en-US",
		"en-IN": "en-US",
		"en-IE": "en-US",

		// German variants
		"de":    "de-DE",
		"de-DE": "de-DE",
		"de-AT": "de-DE",
		"de-CH": "de-DE",

		// Spanish variants
		"es":    "es-ES",
		"es-ES": "es-ES",
		"es-MX": "es-ES",
		"es-AR": "es-ES",

		// French variants
		"fr":    "fr-FR",
		"fr-FR": "fr-FR",
		"fr-CA": "fr-FR",
		"fr-BE": "fr-FR",
		"fr-CH": "fr-FR",

		// Portuguese variants
		"pt":    "pt-PT",
		"pt-PT": "pt-PT",
		"pt-BR": "pt-BR",

		// Russian
		"ru":    "ru-RU",
		"ru-RU": "ru-RU",

		// Other supported languages
		"fi":    "fi-FI",
		"fi-FI": "fi-FI",
		"he":    "he-IL",
		"he-IL": "he-IL",
		"it":    "it-IT",
		"it-IT": "it-IT",
		"kk":    "kk-KZ",
		"kk-KZ": "kk-KZ",
		"nl":    "nl-NL",
		"nl-NL": "nl-NL",
		"pl":    "pl-PL",
		"pl-PL": "pl-PL",
		"sv":    "sv-SE",
		"sv-SE": "sv-SE",
		"tr":    "tr-TR",
		"tr-TR": "tr-TR",
		"uz":    "uz-UZ",
		"uz-UZ": "uz-UZ",
	}

	if normalized, ok := langMap[lang]; ok {
		return normalized
	}

	// If not found, default to en-US
	fmt.Printf("[YANDEX STT] Unknown language code '%s', defaulting to en-US\n", lang)
	return "en-US"
}

// parseWords converts Yandex words to WordInfo
func (c *yandexSTTClient) parseWords(words []*stt.Word) []models.WordInfo {
	if len(words) == 0 {
		return nil
	}

	result := make([]models.WordInfo, len(words))
	for i, word := range words {
		result[i] = models.WordInfo{
			Word:       word.Text,
			StartTime:  float64(word.StartTimeMs) / 1000.0,
			EndTime:    float64(word.EndTimeMs) / 1000.0,
			Confidence: 1.0, // Yandex doesn't provide per-word confidence
		}
	}

	return result
}
