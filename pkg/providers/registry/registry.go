package registry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"
	"github.com/creastat/common-go/pkg/types"
)

// ProviderRegistry manages provider registration and retrieval
type ProviderRegistry interface {
	// Register registers a provider with the registry
	Register(provider interfaces.Provider) error

	// Get retrieves a provider by name and capability
	Get(name string, capability types.Capability) (interfaces.Provider, error)

	// List returns all providers that support a given capability
	List(capability types.Capability) []interfaces.Provider

	// Unregister removes a provider from the registry
	Unregister(name string) error

	// GetProviderInfo returns metadata about a provider
	GetProviderInfo(name string) (*models.ProviderInfo, error)

	// ListAll returns all registered providers
	ListAll() []interfaces.Provider

	// HealthCheck performs health checks on all providers
	HealthCheck(ctx context.Context) map[string]error

	// GetAvailableProviders returns all healthy providers for a capability
	GetAvailableProviders(capability types.Capability) []interfaces.Provider
}

// providerRegistry is the concrete implementation of ProviderRegistry
type providerRegistry struct {
	mu sync.RWMutex

	// providers stores all registered providers by name
	providers map[string]interfaces.Provider

	// capabilityIndex maps capabilities to provider names for fast lookups
	capabilityIndex map[types.Capability][]string

	// providerInfo stores metadata about each provider
	providerInfo map[string]*models.ProviderInfo

	// healthStatus tracks the health status of each provider
	healthStatus map[string]models.HealthStatus

	// lastHealthCheck tracks when each provider was last checked
	lastHealthCheck map[string]time.Time
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() ProviderRegistry {
	return &providerRegistry{
		providers:       make(map[string]interfaces.Provider),
		capabilityIndex: make(map[types.Capability][]string),
		providerInfo:    make(map[string]*models.ProviderInfo),
		healthStatus:    make(map[string]models.HealthStatus),
		lastHealthCheck: make(map[string]time.Time),
	}
}

// Register registers a provider with the registry
func (r *providerRegistry) Register(provider interfaces.Provider) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	name := provider.Name()
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	capabilities := provider.Capabilities()
	if len(capabilities) == 0 {
		return fmt.Errorf("provider %s must support at least one capability", name)
	}

	// Validate capabilities
	if err := r.validateCapabilities(capabilities); err != nil {
		return fmt.Errorf("invalid capabilities for provider %s: %w", name, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if provider already exists
	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %s is already registered", name)
	}

	// Register the provider
	r.providers[name] = provider

	// Update capability index
	for _, capability := range capabilities {
		r.capabilityIndex[capability] = append(r.capabilityIndex[capability], name)
	}

	// Initialize provider info
	info := models.NewProviderInfo(name, models.ProviderType(name), convertCapabilities(capabilities))
	info.Available = true
	info.HealthStatus = models.HealthStatusUnknown
	r.providerInfo[name] = info

	// Initialize health status
	r.healthStatus[name] = models.HealthStatusUnknown
	r.lastHealthCheck[name] = time.Time{}

	return nil
}

// Get retrieves a provider by name and capability
func (r *providerRegistry) Get(name string, capability types.Capability) (interfaces.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	// Verify the provider supports the requested capability
	if !r.hasCapability(provider, capability) {
		return nil, fmt.Errorf("provider %s does not support capability %s", name, capability)
	}

	// Check if provider is healthy
	if status, ok := r.healthStatus[name]; ok {
		if status == models.HealthStatusUnhealthy {
			return nil, fmt.Errorf("provider %s is currently unhealthy", name)
		}
	}

	return provider, nil
}

// List returns all providers that support a given capability
func (r *providerRegistry) List(capability types.Capability) []interfaces.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providerNames, exists := r.capabilityIndex[capability]
	if !exists {
		return []interfaces.Provider{}
	}

	providers := make([]interfaces.Provider, 0, len(providerNames))
	for _, name := range providerNames {
		if provider, ok := r.providers[name]; ok {
			providers = append(providers, provider)
		}
	}

	return providers
}

// Unregister removes a provider from the registry
func (r *providerRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	provider, exists := r.providers[name]
	if !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	// Close the provider
	if err := provider.Close(); err != nil {
		return fmt.Errorf("failed to close provider %s: %w", name, err)
	}

	// Remove from capability index
	capabilities := provider.Capabilities()
	for _, capability := range capabilities {
		r.removeFromCapabilityIndex(capability, name)
	}

	// Remove from maps
	delete(r.providers, name)
	delete(r.providerInfo, name)
	delete(r.healthStatus, name)
	delete(r.lastHealthCheck, name)

	return nil
}

// GetProviderInfo returns metadata about a provider
func (r *providerRegistry) GetProviderInfo(name string) (*models.ProviderInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, exists := r.providerInfo[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	// Return a copy to prevent external modifications
	infoCopy := *info
	return &infoCopy, nil
}

// ListAll returns all registered providers
func (r *providerRegistry) ListAll() []interfaces.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]interfaces.Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}

	return providers
}

// HealthCheck performs health checks on all providers
func (r *providerRegistry) HealthCheck(ctx context.Context) map[string]error {
	r.mu.RLock()
	providersCopy := make(map[string]interfaces.Provider, len(r.providers))
	for name, provider := range r.providers {
		providersCopy[name] = provider
	}
	r.mu.RUnlock()

	results := make(map[string]error)
	var wg sync.WaitGroup
	var resultsMu sync.Mutex

	for name, provider := range providersCopy {
		wg.Add(1)
		go func(n string, p interfaces.Provider) {
			defer wg.Done()

			err := p.HealthCheck(ctx)

			resultsMu.Lock()
			results[n] = err
			resultsMu.Unlock()

			// Update health status
			r.mu.Lock()
			r.lastHealthCheck[n] = time.Now()
			if err != nil {
				r.healthStatus[n] = models.HealthStatusUnhealthy
				if info, ok := r.providerInfo[n]; ok {
					info.UpdateHealthStatus(models.HealthStatusUnhealthy)
					info.Available = false
				}
			} else {
				r.healthStatus[n] = models.HealthStatusHealthy
				if info, ok := r.providerInfo[n]; ok {
					info.UpdateHealthStatus(models.HealthStatusHealthy)
					info.Available = true
				}
			}
			r.mu.Unlock()
		}(name, provider)
	}

	wg.Wait()
	return results
}

// GetAvailableProviders returns all healthy providers for a capability
func (r *providerRegistry) GetAvailableProviders(capability types.Capability) []interfaces.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providerNames, exists := r.capabilityIndex[capability]
	if !exists {
		return []interfaces.Provider{}
	}

	providers := make([]interfaces.Provider, 0)
	for _, name := range providerNames {
		provider, ok := r.providers[name]
		if !ok {
			continue
		}

		// Only include healthy providers
		status, hasStatus := r.healthStatus[name]
		if !hasStatus || status == models.HealthStatusUnhealthy {
			continue
		}

		providers = append(providers, provider)
	}

	return providers
}

// validateCapabilities validates that all capabilities are valid
func (r *providerRegistry) validateCapabilities(capabilities []types.Capability) error {
	validCapabilities := map[types.Capability]bool{
		types.CapabilityChat:      true,
		types.CapabilityEmbedding: true,
		types.CapabilitySTT:       true,
		types.CapabilityTTS:       true,
	}

	for _, capability := range capabilities {
		if !validCapabilities[capability] {
			return fmt.Errorf("invalid capability: %s", capability)
		}
	}

	return nil
}

// hasCapability checks if a provider has a specific capability
func (r *providerRegistry) hasCapability(provider interfaces.Provider, capability types.Capability) bool {
	capabilities := provider.Capabilities()
	for _, cap := range capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// removeFromCapabilityIndex removes a provider from the capability index
func (r *providerRegistry) removeFromCapabilityIndex(capability types.Capability, name string) {
	names, exists := r.capabilityIndex[capability]
	if !exists {
		return
	}

	// Find and remove the provider name
	for i, n := range names {
		if n == name {
			r.capabilityIndex[capability] = append(names[:i], names[i+1:]...)
			break
		}
	}

	// Clean up empty capability entries
	if len(r.capabilityIndex[capability]) == 0 {
		delete(r.capabilityIndex, capability)
	}
}

// convertCapabilities converts interface capabilities to model capabilities
func convertCapabilities(caps []types.Capability) []models.Capability {
	result := make([]models.Capability, len(caps))
	for i, cap := range caps {
		result[i] = models.Capability(cap)
	}
	return result
}
