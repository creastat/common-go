package models

import (
	"time"
)

// TTSConfig represents TTS configuration
type TTSConfig struct {
	Enabled    bool           `json:"enabled"`
	Voice      string         `json:"voice,omitempty"`
	Language   string         `json:"language,omitempty"`
	Model      string         `json:"model,omitempty"`
	SampleRate int            `json:"sample_rate,omitempty"`
	Encoding   string         `json:"encoding,omitempty"`
	Speed      float64        `json:"speed,omitempty"`
	Volume     float64        `json:"volume,omitempty"`
	Pitch      float64        `json:"pitch,omitempty"`
	Options    map[string]any `json:"options,omitempty"`
}

// STTConfig represents STT configuration
type STTConfig struct {
	Language           string         `json:"language,omitempty"`
	Model              string         `json:"model,omitempty"`
	SampleRate         int            `json:"sample_rate,omitempty"`
	Encoding           string         `json:"encoding,omitempty"`
	Channels           int            `json:"channels,omitempty"`
	InterimResults     bool           `json:"interim_results,omitempty"`
	PunctuationEnabled bool           `json:"punctuation_enabled,omitempty"`
	Options            map[string]any `json:"options,omitempty"`
}

// STTResult represents a speech-to-text result
type STTResult struct {
	Text       string         `json:"text"`
	Confidence float64        `json:"confidence"`
	IsFinal    bool           `json:"is_final"`
	Language   string         `json:"language,omitempty"`
	Duration   float64        `json:"duration,omitempty"`
	Timestamp  time.Time      `json:"timestamp"`
	StartTime  float64        `json:"start_time,omitempty"`
	EndTime    float64        `json:"end_time,omitempty"`
	Words      []WordInfo     `json:"words,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// WordInfo represents information about a single word in STT result
type WordInfo struct {
	Word       string  `json:"word"`
	StartTime  float64 `json:"start_time"`
	EndTime    float64 `json:"end_time"`
	Confidence float64 `json:"confidence"`
}

// Voice represents a TTS voice
type Voice struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Language    string   `json:"language"`
	Gender      string   `json:"gender,omitempty"`
	Description string   `json:"description,omitempty"`
	SampleRate  int      `json:"sample_rate,omitempty"`
	Styles      []string `json:"styles,omitempty"`
}
