package supabase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Job represents an ingestion job in Supabase
type Job struct {
	ID             uuid.UUID `json:"id,omitempty"`
	SourceID       uuid.UUID `json:"source_id"`
	Status         string    `json:"status"`
	JobType        string    `json:"job_type"`
	ResourceURL    string    `json:"resource_url"`
	PagesProcessed int       `json:"pages_processed"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

// Document represents a document in Supabase
type Document struct {
	ID        uuid.UUID      `json:"id,omitempty"`
	SourceID  uuid.UUID      `json:"source_id"`
	URL       string         `json:"url"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata"`
	Hash      string         `json:"hash"`
	CreatedAt time.Time      `json:"created_at,omitempty"`
	UpdatedAt time.Time      `json:"updated_at,omitempty"`
}

// Embedding represents an embedding in Supabase
type Embedding struct {
	ID         uuid.UUID `json:"id,omitempty"`
	DocumentID uuid.UUID `json:"document_id"`
	Vector     []float32 `json:"vector"`
	Chunk      string    `json:"chunk"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
}

// CreateJob creates a new ingestion job
func (c *Client) CreateJob(ctx context.Context, job *Job) error {
	url := fmt.Sprintf("%s/rest/v1/ingestion_jobs", c.url)

	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("create job failed: status %d", resp.StatusCode)
	}

	// Parse response to get the created job with ID
	var results []Job
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(results) > 0 {
		*job = results[0]
	}

	return nil
}

// UpdateJob updates an existing job
func (c *Client) UpdateJob(ctx context.Context, job *Job) error {
	url := fmt.Sprintf("%s/rest/v1/ingestion_jobs?id=eq.%s", c.url, job.ID.String())

	payload, err := json.Marshal(map[string]any{
		"status":          job.Status,
		"pages_processed": job.PagesProcessed,
		"error_message":   job.ErrorMessage,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update job failed: status %d", resp.StatusCode)
	}

	// Parse response to get updated job
	var results []Job
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(results) > 0 {
		*job = results[0]
	}

	return nil
}

// GetJob retrieves a job by ID
func (c *Client) GetJob(ctx context.Context, id uuid.UUID) (*Job, error) {
	url := fmt.Sprintf("%s/rest/v1/ingestion_jobs?id=eq.%s", c.url, id.String())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get job failed: status %d", resp.StatusCode)
	}

	var results []Job
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("job not found")
	}

	return &results[0], nil
}

// UpsertDocument creates or updates a document
func (c *Client) UpsertDocument(ctx context.Context, doc *Document) (uuid.UUID, error) {
	url := fmt.Sprintf("%s/rest/v1/documents", c.url)

	payload, err := json.Marshal(doc)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to marshal document: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation,resolution=merge-duplicates")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to upsert document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return uuid.Nil, fmt.Errorf("upsert document failed: status %d", resp.StatusCode)
	}

	var results []Document
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return uuid.Nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(results) == 0 {
		return uuid.Nil, fmt.Errorf("no document returned")
	}

	return results[0].ID, nil
}

// BatchInsertEmbeddings inserts multiple embeddings
func (c *Client) BatchInsertEmbeddings(ctx context.Context, embeddings []Embedding) error {
	if len(embeddings) == 0 {
		return nil
	}

	url := fmt.Sprintf("%s/rest/v1/embeddings", c.url)

	payload, err := json.Marshal(embeddings)
	if err != nil {
		return fmt.Errorf("failed to marshal embeddings: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to insert embeddings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("insert embeddings failed: status %d", resp.StatusCode)
	}

	return nil
}
