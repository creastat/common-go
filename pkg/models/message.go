package models

import (
	"encoding/json"
	"time"
)

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeText        MessageType = "text"
	MessageTypeAudio       MessageType = "audio"
	MessageTypeEndSpeech   MessageType = "end-speech"
	MessageTypeStartSpeech MessageType = "start-speech"
	MessageTypeBreak       MessageType = "break"
	MessageTypeControl     MessageType = "control"
	MessageTypeError       MessageType = "error"
)

// Message represents a message in the system
type Message struct {
	ID        string         `json:"id"`
	Type      MessageType    `json:"type"`
	SessionID string         `json:"session_id"`
	Payload   any            `json:"payload"`
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// ProviderSelection specifies provider and model for a capability
type ProviderSelection struct {
	Provider string         `json:"provider"`
	Model    string         `json:"model,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
}

// SessionProviderConfig allows per-session provider configuration
type SessionProviderConfig struct {
	Chat      *ProviderSelection `json:"chat,omitempty"`
	Embedding *ProviderSelection `json:"embedding,omitempty"`
	STT       *ProviderSelection `json:"stt,omitempty"`
	TTS       *ProviderSelection `json:"tts,omitempty"`
}

// TextMessagePayload represents the payload for text messages
type TextMessagePayload struct {
	Content  string             `json:"content"`
	Role     string             `json:"role,omitempty"` // user, assistant, system
	Context  map[string]any     `json:"context,omitempty"`
	Provider *ProviderSelection `json:"provider,omitempty"`
}

// AudioMessagePayload represents the payload for audio messages
type AudioMessagePayload struct {
	Data     []byte             `json:"data,omitempty"`
	Format   string             `json:"format"` // wav, mp3, opus, etc.
	Duration float64            `json:"duration,omitempty"`
	Context  map[string]any     `json:"context,omitempty"`
	Provider *ProviderSelection `json:"provider,omitempty"`
}

// ControlMessagePayload represents the payload for control messages
type ControlMessagePayload struct {
	Action string         `json:"action"` // start, stop, pause, resume, configure
	Params map[string]any `json:"params,omitempty"`
}

// BreakMessagePayload represents the payload for break signal messages
// Break signals are sent when the user interrupts the bot mid-response
type BreakMessagePayload struct {
	// Empty struct - break signal carries no additional data
	// The presence of the message itself is the signal
}

// ErrorMessagePayload represents the payload for error messages
type ErrorMessagePayload struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	Retryable bool           `json:"retryable"`
	Timestamp time.Time      `json:"timestamp"`
}

// STTRequest represents a speech-to-text request
type STTRequest struct {
	Audio    []byte         `json:"audio"`
	Format   string         `json:"format"`
	Language string         `json:"language,omitempty"`
	Model    string         `json:"model,omitempty"`
	Context  map[string]any `json:"context,omitempty"`
}

// STTResponse represents a speech-to-text response
type STTResponse struct {
	Text       string    `json:"text"`
	Confidence float64   `json:"confidence,omitempty"`
	Language   string    `json:"language,omitempty"`
	Duration   float64   `json:"duration,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// TTSRequest represents a text-to-speech request
type TTSRequest struct {
	Text     string         `json:"text"`
	Voice    string         `json:"voice,omitempty"`
	Language string         `json:"language,omitempty"`
	Model    string         `json:"model,omitempty"`
	Speed    float64        `json:"speed,omitempty"`
	Context  map[string]any `json:"context,omitempty"`
}

// TTSResponse represents a text-to-speech response
type TTSResponse struct {
	Audio     []byte    `json:"audio"`
	Format    string    `json:"format"`
	Duration  float64   `json:"duration,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// EmbeddingRequest represents an embedding generation request
type EmbeddingRequest struct {
	Text       string         `json:"text"`
	Model      string         `json:"model,omitempty"`
	Dimensions int            `json:"dimensions,omitempty"`
	Context    map[string]any `json:"context,omitempty"`
}

// EmbeddingResponse represents an embedding generation response
type EmbeddingResponse struct {
	Embedding  []float32 `json:"embedding"`
	Dimensions int       `json:"dimensions"`
	Model      string    `json:"model"`
	Timestamp  time.Time `json:"timestamp"`
}

// NewMessage creates a new message with the given parameters
func NewMessage(msgType MessageType, sessionID string, payload any) *Message {
	return &Message{
		ID:        generateMessageID(),
		Type:      msgType,
		SessionID: sessionID,
		Payload:   payload,
		Timestamp: time.Now(),
		Metadata:  make(map[string]any),
	}
}

// UnmarshalPayload unmarshals the message payload into the target type
func (m *Message) UnmarshalPayload(target any) error {
	data, err := json.Marshal(m.Payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// WithMetadata adds metadata to the message
func (m *Message) WithMetadata(key string, value any) *Message {
	if m.Metadata == nil {
		m.Metadata = make(map[string]any)
	}
	m.Metadata[key] = value
	return m
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	// Simple implementation - in production, use UUID or similar
	return "msg-" + time.Now().Format("20060102150405.000000")
}
