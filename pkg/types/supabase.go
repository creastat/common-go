package types

import (
	"context"
	"time"
)

// SupabaseService provides integration with Supabase for source management and document search
type SupabaseService interface {
	// ValidateToken validates a site token and returns the associated source configuration
	ValidateToken(ctx context.Context, publicToken string) (*SourceConfig, error)

	// GetSourceByID retrieves source configuration by source ID
	GetSourceByID(ctx context.Context, sourceID string) (*SourceConfig, error)

	// SearchDocuments performs vector similarity search against documents for a source
	SearchDocuments(ctx context.Context, req SearchRequest) ([]SearchResult, error)
}

// SourceConfig represents the configuration for a source from the Supabase sources table
type SourceConfig struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	PublicToken    string                 `json:"public_token"`
	AllowedOrigins []string               `json:"allowed_origins"`
	Strategy       string                 `json:"strategy"` // "none", "vector", "fulltext"
	Content        string                 `json:"content"`  // Static content for "none" strategy
	SystemPrompt   string                 `json:"system_prompt"`
	RateLimit      int                    `json:"rate_limit"` // requests per minute
	Enabled        bool                   `json:"enabled"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// SearchRequest represents a request to search documents by vector similarity
type SearchRequest struct {
	SourceID       string    // Source ID to filter documents
	QueryEmbedding []float32 // Query embedding vector
	MaxResults     int       // Maximum number of results to return
	Threshold      float64   // Minimum similarity threshold (0.0-1.0)
}

// SearchResult represents a single document search result
type SearchResult struct {
	ID         string                 `json:"id"`
	Content    string                 `json:"content"`
	Similarity float64                `json:"similarity"`
	Metadata   map[string]any `json:"metadata"`
	DocumentID string                 `json:"document_id"`
	CreatedAt  time.Time              `json:"created_at"`
}

// IsOriginAllowed checks if the given origin is allowed for this source
func (s *SourceConfig) IsOriginAllowed(origin string) bool {
	// If no origins are specified, deny all
	if len(s.AllowedOrigins) == 0 {
		return false
	}

	// Check for wildcard
	for _, allowed := range s.AllowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
	}

	return false
}

// IsEnabled checks if the source is enabled
func (s *SourceConfig) IsEnabled() bool {
	return s.Enabled
}

// GetStrategy returns the content strategy for this source
func (s *SourceConfig) GetStrategy() string {
	if s.Strategy == "" {
		return "vector" // Default strategy
	}
	return s.Strategy
}

// GetSystemPrompt returns the system prompt, or a default if not set
func (s *SourceConfig) GetSystemPrompt() string {
	if s.SystemPrompt == "" {
		return "You are a helpful AI assistant. Use the provided context to answer accurately. If no context is provided, use your general knowledge. Keep responses conversational and helpful."
	}
	return s.SystemPrompt
}

// GetRateLimit returns the rate limit in requests per minute
func (s *SourceConfig) GetRateLimit() int {
	if s.RateLimit <= 0 {
		return 60 // Default: 60 requests per minute
	}
	return s.RateLimit
}
