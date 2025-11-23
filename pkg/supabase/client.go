package supabase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/creastat/common-go/pkg/types"
)

// Client implements the SupabaseService interface using HTTP REST API
type Client struct {
	url        string
	apiKey     string
	httpClient *http.Client
	cache      *sourceCache
	cacheTTL   time.Duration
	logger     types.Logger
}

// ClientConfig holds configuration for the Supabase client
type ClientConfig struct {
	URL      string
	APIKey   string
	CacheTTL time.Duration // Default: 5 minutes
	Timeout  time.Duration // HTTP client timeout
	Logger   types.Logger
}

// sourceCache provides thread-safe caching for source configurations
type sourceCache struct {
	mu      sync.RWMutex
	byToken map[string]*cacheEntry
	byID    map[string]*cacheEntry
}

type cacheEntry struct {
	source    *types.SourceConfig
	expiresAt time.Time
}

// NewClient creates a new Supabase client
func NewClient(config ClientConfig) (*Client, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("supabase URL is required")
	}
	if config.APIKey == "" {
		return nil, fmt.Errorf("supabase API key is required")
	}

	// Set defaults
	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	logger := config.Logger
	if logger == nil {
		logger = &types.NoOpLogger{}
	}

	return &Client{
		url:    strings.TrimSuffix(config.URL, "/"),
		apiKey: config.APIKey,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		cache: &sourceCache{
			byToken: make(map[string]*cacheEntry),
			byID:    make(map[string]*cacheEntry),
		},
		cacheTTL: config.CacheTTL,
		logger:   logger,
	}, nil
}

// ValidateToken validates a site token and returns the associated source configuration
func (c *Client) ValidateToken(ctx context.Context, publicToken string) (*types.SourceConfig, error) {
	// Check cache first
	if source := c.getFromCache("token", publicToken); source != nil {
		return source, nil
	}

	// Query Supabase sources table
	url := fmt.Sprintf("%s/rest/v1/sources?public_token=eq.%s&select=*", c.url, publicToken)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.logger.Debug("Querying Supabase for token", "url", url, "public_token", publicToken)

	// Set required headers
	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query supabase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Supabase token validation failed", "status", resp.StatusCode, "url", url)
		return nil, fmt.Errorf("token validation failed: status %d", resp.StatusCode)
	}

	// Parse response
	var results []supabaseSource
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("source not found")
	}

	// Convert to domain model
	source := c.convertToDomain(&results[0])

	// Validate source is enabled
	if !source.IsEnabled() {
		return nil, fmt.Errorf("source is disabled")
	}

	// Cache the result by both token and ID
	c.addToCache(source)

	return source, nil
}

// GetSourceByID retrieves source configuration by source ID
func (c *Client) GetSourceByID(ctx context.Context, sourceID string) (*types.SourceConfig, error) {
	// Check cache first
	if source := c.getFromCache("id", sourceID); source != nil {
		return source, nil
	}

	// Query Supabase sources table
	url := fmt.Sprintf("%s/rest/v1/sources?id=eq.%s&select=*", c.url, sourceID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.logger.Debug("Querying Supabase for source ID", "url", url, "source_id", sourceID)

	// Set required headers
	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query supabase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Supabase source query failed", "status", resp.StatusCode, "url", url)
		return nil, fmt.Errorf("source query failed: status %d", resp.StatusCode)
	}

	// Parse response
	var results []supabaseSource
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("source not found")
	}

	// Convert to domain model
	source := c.convertToDomain(&results[0])

	// Cache the result by both token and ID
	c.addToCache(source)

	return source, nil
}

// SearchDocuments performs vector similarity search against documents for a source
func (c *Client) SearchDocuments(ctx context.Context, req types.SearchRequest) ([]types.SearchResult, error) {
	// Prepare RPC parameters
	params := map[string]any{
		"p_source_id":     req.SourceID,
		"query_embedding": req.QueryEmbedding,
		"match_threshold": req.Threshold,
		"match_count":     req.MaxResults,
	}

	rpcURL := fmt.Sprintf("%s/rest/v1/rpc/search_documents_by_source", c.url)
	rpcReq, err := http.NewRequestWithContext(ctx, "POST", rpcURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set body
	jsonBody, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}
	rpcReq.Body = io.NopCloser(bytes.NewReader(jsonBody))

	// Set headers
	rpcReq.Header.Set("apikey", c.apiKey)
	rpcReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	rpcReq.Header.Set("Content-Type", "application/json")
	rpcReq.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute RPC: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read body for error details
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("Supabase RPC failed", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("RPC failed: status %d", resp.StatusCode)
	}

	// Parse response
	// The RPC returns a table: id, content_chunk, metadata, document_id, similarity, created_at
	type rpcResult struct {
		ID           string         `json:"id"`
		ContentChunk string         `json:"content_chunk"`
		Metadata     map[string]any `json:"metadata"`
		DocumentID   string         `json:"document_id"`
		Similarity   float64        `json:"similarity"`
		CreatedAt    time.Time      `json:"created_at"`
	}

	var results []rpcResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to domain model
	searchResults := make([]types.SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = types.SearchResult{
			Content:    r.ContentChunk,
			Similarity: r.Similarity,
			Metadata:   r.Metadata,
		}
	}

	return searchResults, nil
}

// getFromCache retrieves a source from cache by token or ID
func (c *Client) getFromCache(keyType, key string) *types.SourceConfig {
	c.cache.mu.RLock()
	defer c.cache.mu.RUnlock()

	var entry *cacheEntry
	switch keyType {
	case "token":
		entry = c.cache.byToken[key]
	case "id":
		entry = c.cache.byID[key]
	default:
		return nil
	}

	if entry == nil {
		return nil
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		return nil
	}

	return entry.source
}

// addToCache adds a source to cache by both token and ID
func (c *Client) addToCache(source *types.SourceConfig) {
	c.cache.mu.Lock()
	defer c.cache.mu.Unlock()

	entry := &cacheEntry{
		source:    source,
		expiresAt: time.Now().Add(c.cacheTTL),
	}

	// Cache by token
	if source.PublicToken != "" {
		c.cache.byToken[source.PublicToken] = entry
	}

	// Cache by ID
	if source.ID != "" {
		c.cache.byID[source.ID] = entry
	}
}

// ClearCache clears all cached source configurations
func (c *Client) ClearCache() {
	c.cache.mu.Lock()
	defer c.cache.mu.Unlock()

	c.cache.byToken = make(map[string]*cacheEntry)
	c.cache.byID = make(map[string]*cacheEntry)
}

// supabaseSource represents the raw source data from Supabase
type supabaseSource struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	PublicToken    string         `json:"public_token"`
	AllowedOrigins []string       `json:"allowed_origins"`
	Strategy       string         `json:"strategy"`
	Content        string         `json:"content"`
	SystemPrompt   string         `json:"system_prompt"`
	RateLimit      int            `json:"rate_limit"`
	Enabled        *bool          `json:"enabled"` // Pointer to handle null
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// convertToDomain converts a Supabase source to domain model
func (c *Client) convertToDomain(s *supabaseSource) *types.SourceConfig {
	enabled := true
	if s.Enabled != nil {
		enabled = *s.Enabled
	}

	return &types.SourceConfig{
		ID:             s.ID,
		Name:           s.Name,
		PublicToken:    s.PublicToken,
		AllowedOrigins: s.AllowedOrigins,
		Strategy:       s.Strategy,
		Content:        s.Content,
		SystemPrompt:   s.SystemPrompt,
		RateLimit:      s.RateLimit,
		Enabled:        enabled,
		Metadata:       s.Metadata,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}
