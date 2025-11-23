package factory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/providers/registry"
	"github.com/creastat/common-go/pkg/types"
)

// ProviderFactory creates and manages provider service instances
type ProviderFactory interface {
	// CreateChatService creates a chat service for the specified provider
	CreateChatService(ctx context.Context, providerName string) (interfaces.ChatService, error)

	// CreateEmbeddingService creates an embedding service for the specified provider
	CreateEmbeddingService(ctx context.Context, providerName string) (interfaces.EmbeddingService, error)

	// CreateSTTService creates a speech-to-text service for the specified provider
	CreateSTTService(ctx context.Context, providerName string) (interfaces.STTService, error)

	// CreateTTSService creates a text-to-speech service for the specified provider
	CreateTTSService(ctx context.Context, providerName string) (interfaces.TTSService, error)

	// ClearCache clears the provider instance cache
	ClearCache()

	// ClearCacheForProvider clears cache for a specific provider
	ClearCacheForProvider(providerName string)
}

// Configuration defines the interface for configuration needed by the factory
type Configuration interface {
	GetFallbackProvider(capability string) string
}

// providerFactory is the concrete implementation of ProviderFactory
type providerFactory struct {
	registry registry.ProviderRegistry
	config   Configuration

	// Cache for provider instances to avoid redundant initialization
	cache   map[string]any
	cacheMu sync.RWMutex

	// Initialization tracking to prevent concurrent initialization
	initLocks   map[string]*sync.Mutex
	initLocksMu sync.Mutex
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(registry registry.ProviderRegistry, cfg Configuration) ProviderFactory {
	return &providerFactory{
		registry:  registry,
		config:    cfg,
		cache:     make(map[string]any),
		initLocks: make(map[string]*sync.Mutex),
	}
}

// CreateChatService creates a chat service for the specified provider
func (f *providerFactory) CreateChatService(ctx context.Context, providerName string) (interfaces.ChatService, error) {
	cacheKey := fmt.Sprintf("chat:%s", providerName)

	// Check cache first
	if service := f.getCached(cacheKey); service != nil {
		if chatService, ok := service.(interfaces.ChatService); ok {
			return chatService, nil
		}
	}

	// Get provider from registry
	provider, err := f.registry.Get(providerName, types.CapabilityChat)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat provider %s: %w", providerName, err)
	}

	// Type assert to ChatService
	chatService, ok := provider.(interfaces.ChatService)
	if !ok {
		return nil, fmt.Errorf("provider %s does not implement ChatService interface", providerName)
	}

	// Cache the service
	f.setCached(cacheKey, chatService)

	return chatService, nil
}

// CreateEmbeddingService creates an embedding service for the specified provider
func (f *providerFactory) CreateEmbeddingService(ctx context.Context, providerName string) (interfaces.EmbeddingService, error) {
	cacheKey := fmt.Sprintf("embedding:%s", providerName)

	// Check cache first
	if service := f.getCached(cacheKey); service != nil {
		if embeddingService, ok := service.(interfaces.EmbeddingService); ok {
			return embeddingService, nil
		}
	}

	// Get provider from registry
	provider, err := f.registry.Get(providerName, types.CapabilityEmbedding)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding provider %s: %w", providerName, err)
	}

	// Type assert to EmbeddingService
	embeddingService, ok := provider.(interfaces.EmbeddingService)
	if !ok {
		return nil, fmt.Errorf("provider %s does not implement EmbeddingService interface", providerName)
	}

	// Cache the service
	f.setCached(cacheKey, embeddingService)

	return embeddingService, nil
}

// CreateSTTService creates a speech-to-text service for the specified provider
func (f *providerFactory) CreateSTTService(ctx context.Context, providerName string) (interfaces.STTService, error) {
	cacheKey := fmt.Sprintf("stt:%s", providerName)

	// Check cache first
	if service := f.getCached(cacheKey); service != nil {
		if sttService, ok := service.(interfaces.STTService); ok {
			return sttService, nil
		}
	}

	// Get provider from registry
	provider, err := f.registry.Get(providerName, types.CapabilitySTT)
	if err != nil {
		return nil, fmt.Errorf("failed to get STT provider %s: %w", providerName, err)
	}

	// Type assert to SpeechToTextService
	sttService, ok := provider.(interfaces.STTService)
	if !ok {
		return nil, fmt.Errorf("provider %s does not implement SpeechToTextService interface", providerName)
	}

	// Cache the service
	f.setCached(cacheKey, sttService)

	return sttService, nil
}

// CreateTTSService creates a text-to-speech service for the specified provider
func (f *providerFactory) CreateTTSService(ctx context.Context, providerName string) (interfaces.TTSService, error) {
	cacheKey := fmt.Sprintf("tts:%s", providerName)

	// Check cache first
	if service := f.getCached(cacheKey); service != nil {
		if ttsService, ok := service.(interfaces.TTSService); ok {
			return ttsService, nil
		}
	}

	// Get provider from registry
	provider, err := f.registry.Get(providerName, types.CapabilityTTS)
	if err != nil {
		return nil, fmt.Errorf("failed to get TTS provider %s: %w", providerName, err)
	}

	// Type assert to TextToSpeechService
	ttsService, ok := provider.(interfaces.TTSService)
	if !ok {
		return nil, fmt.Errorf("provider %s does not implement TextToSpeechService interface", providerName)
	}

	// Cache the service
	f.setCached(cacheKey, ttsService)

	return ttsService, nil
}

// ClearCache clears the entire provider instance cache
func (f *providerFactory) ClearCache() {
	f.cacheMu.Lock()
	defer f.cacheMu.Unlock()

	f.cache = make(map[string]any)
}

// ClearCacheForProvider clears cache entries for a specific provider
func (f *providerFactory) ClearCacheForProvider(providerName string) {
	f.cacheMu.Lock()
	defer f.cacheMu.Unlock()

	// Remove all cache entries for this provider
	keysToDelete := []string{
		fmt.Sprintf("chat:%s", providerName),
		fmt.Sprintf("embedding:%s", providerName),
		fmt.Sprintf("stt:%s", providerName),
		fmt.Sprintf("tts:%s", providerName),
	}

	for _, key := range keysToDelete {
		delete(f.cache, key)
	}
}

// getCached retrieves a cached service instance
func (f *providerFactory) getCached(key string) any {
	f.cacheMu.RLock()
	defer f.cacheMu.RUnlock()

	return f.cache[key]
}

// setCached stores a service instance in the cache
func (f *providerFactory) setCached(key string, service any) {
	f.cacheMu.Lock()
	defer f.cacheMu.Unlock()

	f.cache[key] = service
}

// getInitLock gets or creates a mutex for provider initialization
func (f *providerFactory) getInitLock(providerName string) *sync.Mutex {
	f.initLocksMu.Lock()
	defer f.initLocksMu.Unlock()

	if lock, exists := f.initLocks[providerName]; exists {
		return lock
	}

	lock := &sync.Mutex{}
	f.initLocks[providerName] = lock
	return lock
}

// ProviderFactoryWithFallback wraps a ProviderFactory with fallback support
type ProviderFactoryWithFallback struct {
	factory ProviderFactory
	config  Configuration
}

// NewProviderFactoryWithFallback creates a factory with automatic fallback support
func NewProviderFactoryWithFallback(factory ProviderFactory, cfg Configuration) *ProviderFactoryWithFallback {
	return &ProviderFactoryWithFallback{
		factory: factory,
		config:  cfg,
	}
}

// CreateChatService creates a chat service with fallback support
func (f *ProviderFactoryWithFallback) CreateChatService(ctx context.Context, providerName string) (interfaces.ChatService, error) {
	service, err := f.factory.CreateChatService(ctx, providerName)
	if err == nil {
		return service, nil
	}

	// Try fallback provider if configured
	fallback := f.config.GetFallbackProvider("chat")
	if fallback != "" && fallback != providerName {
		return f.factory.CreateChatService(ctx, fallback)
	}

	return nil, err
}

// CreateEmbeddingService creates an embedding service with fallback support
func (f *ProviderFactoryWithFallback) CreateEmbeddingService(ctx context.Context, providerName string) (interfaces.EmbeddingService, error) {
	service, err := f.factory.CreateEmbeddingService(ctx, providerName)
	if err == nil {
		return service, nil
	}

	// Try fallback provider if configured
	fallback := f.config.GetFallbackProvider("embedding")
	if fallback != "" && fallback != providerName {
		return f.factory.CreateEmbeddingService(ctx, fallback)
	}

	return nil, err
}

// CreateSTTService creates an STT service with fallback support
func (f *ProviderFactoryWithFallback) CreateSTTService(ctx context.Context, providerName string) (interfaces.STTService, error) {
	service, err := f.factory.CreateSTTService(ctx, providerName)
	if err == nil {
		return service, nil
	}

	// Try fallback provider if configured
	fallback := f.config.GetFallbackProvider("stt")
	if fallback != "" && fallback != providerName {
		return f.factory.CreateSTTService(ctx, fallback)
	}

	return nil, err
}

// CreateTTSService creates a TTS service with fallback support
func (f *ProviderFactoryWithFallback) CreateTTSService(ctx context.Context, providerName string) (interfaces.TTSService, error) {
	service, err := f.factory.CreateTTSService(ctx, providerName)
	if err == nil {
		return service, nil
	}

	// Try fallback provider if configured
	fallback := f.config.GetFallbackProvider("tts")
	if fallback != "" && fallback != providerName {
		return f.factory.CreateTTSService(ctx, fallback)
	}

	return nil, err
}

// ClearCache clears the cache
func (f *ProviderFactoryWithFallback) ClearCache() {
	f.factory.ClearCache()
}

// ClearCacheForProvider clears cache for a specific provider
func (f *ProviderFactoryWithFallback) ClearCacheForProvider(providerName string) {
	f.factory.ClearCacheForProvider(providerName)
}

// ProviderInitializationError represents an error during provider initialization
type ProviderInitializationError struct {
	ProviderName string
	Capability   types.Capability
	Cause        error
	Timestamp    time.Time
}

// Error implements the error interface
func (e *ProviderInitializationError) Error() string {
	return fmt.Sprintf("failed to initialize provider %s for capability %s: %v",
		e.ProviderName, e.Capability, e.Cause)
}

// Unwrap returns the underlying error
func (e *ProviderInitializationError) Unwrap() error {
	return e.Cause
}

// NewProviderInitializationError creates a new provider initialization error
func NewProviderInitializationError(providerName string, capability types.Capability, cause error) *ProviderInitializationError {
	return &ProviderInitializationError{
		ProviderName: providerName,
		Capability:   capability,
		Cause:        cause,
		Timestamp:    time.Now(),
	}
}
