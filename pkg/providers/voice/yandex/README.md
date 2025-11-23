# Yandex SpeechKit Provider

This provider implements both speech-to-text (STT) and text-to-speech (TTS) capabilities using Yandex SpeechKit v3 API.

## Features

### Speech-to-Text (STT)
- Real-time streaming speech recognition
- Support for multiple languages (Russian, English, Turkish, Kazakh, Uzbek)
- Word-level timestamps
- Text normalization and punctuation
- End-of-utterance detection
- Multiple recognition models

### Text-to-Speech (TTS)
- Real-time streaming synthesis
- Multiple voices (Russian, English, Kazakh, Uzbek)
- Speed, pitch, and volume control
- Emotional and neutral voices
- High-quality audio output (22050 Hz)

## Configuration

Add the following to your `config.yaml`:

```yaml
providers:
  stt:
    primary: yandex  # or keep deepgram as primary
    config:
      yandex:
        api_key: ${YANDEX_API_KEY}
        model: general
        timeout: 30s
        options:
          folder_id: ${YANDEX_FOLDER_ID}
          punctuate: true
          normalization: true
  
  tts:
    primary: yandex  # or keep minimax as primary
    config:
      yandex:
        api_key: ${YANDEX_API_KEY}
        timeout: 30s
        options:
          folder_id: ${YANDEX_FOLDER_ID}
          default_voices:
            ru: alena
            en: john
          sample_rate: 22050
          speed: 1.0
          volume: 0.7
```

Set environment variables in `.env`:

```bash
YANDEX_API_KEY=your_api_key_here
YANDEX_FOLDER_ID=your_folder_id_here
```

## Getting API Credentials

1. Go to [Yandex Cloud Console](https://console.cloud.yandex.com/)
2. Create or select a folder
3. Enable SpeechKit API
4. Create an API key or use IAM token
5. Copy your folder ID from the console

## Available Models

- `general` - General-purpose speech recognition (default)
- `general:rc` - Release candidate with latest improvements
- `general:deprecated` - Older version (deprecated)
- `deferred-general` - For asynchronous file recognition

## Supported Languages

### STT Languages
- `ru-RU` - Russian (default)
- `en-US` - English
- `tr-TR` - Turkish
- `kk-KZ` - Kazakh
- `uz-UZ` - Uzbek

### TTS Voices

**Russian (ru-RU)**
- `alena` - Female, neutral tone (default)
- `filipp` - Male, neutral tone
- `ermil` - Male, emotional tone
- `jane` - Female, emotional tone
- `omazh` - Female, calm tone
- `zahar` - Male, calm tone

**English (en-US)**
- `john` - Male voice

**Kazakh (kk-KK)**
- `amira` - Female voice
- `madi` - Male voice

**Uzbek (uz-UZ)**
- `nigora` - Female voice

## Audio Format Requirements

- Sample rate: 8000 Hz (default) or 16000 Hz
- Encoding: LINEAR16_PCM (raw PCM)
- Channels: 1 (mono)

## Usage Examples

### Speech-to-Text (STT)

```go
// Create STT service
sttService, err := factory.CreateSTTService(ctx, "yandex")
if err != nil {
    log.Fatal(err)
}

// Configure recognition
config := interfaces.STTConfig{
    Model:              "general",
    Language:           "ru-RU",
    SampleRate:         8000,
    Encoding:           "linear16",
    Channels:           1,
    InterimResults:     true,
    PunctuationEnabled: true,
}

// Create streaming client
client, err := sttService.NewClient(ctx, config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Send audio data
go func() {
    for audioChunk := range audioStream {
        if err := client.Send(ctx, audioChunk); err != nil {
            log.Printf("Send error: %v", err)
            return
        }
    }
}()

// Receive results
for {
    result, err := client.Receive(ctx)
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Printf("Receive error: %v", err)
        break
    }
    
    if result.IsFinal {
        fmt.Printf("Final: %s\n", result.Text)
    } else {
        fmt.Printf("Partial: %s\n", result.Text)
    }
}
```

### Text-to-Speech (TTS)

```go
// Create TTS service
ttsService, err := factory.CreateTTSService(ctx, "yandex")
if err != nil {
    log.Fatal(err)
}

// Configure synthesis
config := interfaces.TTSConfig{
    Voice:      "alena",
    Language:   "ru-RU",
    SampleRate: 22050,
    Encoding:   "linear16",
    Speed:      1.0,
    Volume:     0.7,
    Pitch:      0,
}

// Non-streaming synthesis
audioData, err := ttsService.Synthesize(ctx, "Привет, как дела?", config)
if err != nil {
    log.Fatal(err)
}

// Save or play audio
ioutil.WriteFile("output.pcm", audioData, 0644)

// Streaming synthesis
client, err := ttsService.NewClient(ctx, config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Send text
if err := client.Send(ctx, "Привет, мир!"); err != nil {
    log.Printf("Send error: %v", err)
}

// Receive audio chunks
for {
    audioChunk, err := client.Receive(ctx)
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Printf("Receive error: %v", err)
        break
    }
    
    // Play or process audio chunk
    playAudio(audioChunk)
}
```

## Advanced Options

### Text Normalization

```yaml
options:
  normalization: true
  profanity_filter: false
  literature_text: false
```

### End-of-Utterance Detection

```yaml
options:
  eou_sensitivity: default  # or "high"
  max_pause_between_words_ms: 1000
```

## API Documentation

- [Yandex SpeechKit Documentation](https://cloud.yandex.com/docs/speechkit/)
- [API Reference](https://cloud.yandex.com/docs/speechkit/stt/api/streaming-api)
- [gRPC API v3](https://cloud.yandex.com/docs/speechkit/stt-v3/api-ref/grpc/)

## Notes

- The provider uses gRPC for streaming communication
- API key authentication is used (add `Api-Key` header)
- The `folder_id` is required for all requests
- Real-time recognition requires audio to be sent in chunks
- For production, consider using proper proto-generated types instead of stubs

## Implementation Details

This implementation includes:

1. **plugin.go** - Provider registration and lifecycle management
2. **stt.go** - STT service implementation with streaming support
3. **proto_types.go** - Stub types for Yandex SpeechKit v3 API

For production use, generate proper types from proto files:

```bash
protoc --go_out=. --go-grpc_out=. \
  yandex/cloud/ai/stt/v3/*.proto
```

## Troubleshooting

### Connection Issues

- Verify API key is correct
- Check folder_id is valid
- Ensure network access to `stt.api.cloud.yandex.net:443`

### Recognition Quality

- Use appropriate sample rate (8000 Hz for telephony, 16000 Hz for general)
- Ensure audio is in LINEAR16_PCM format
- Select correct language code
- Enable text normalization for better results

### Rate Limits

Yandex SpeechKit has rate limits:
- Check your quota in Yandex Cloud Console
- Implement retry logic with exponential backoff
- Consider using multiple API keys for high-volume applications
