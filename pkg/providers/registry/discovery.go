package registry

import (
	"context"
	"fmt"
	"sync"

	"github.com/creastat/common-go/pkg/interfaces"
	"github.com/creastat/common-go/pkg/models"
	"github.com/creastat/common-go/pkg/types"
)

// ProviderPlugin defines the interface for provider plugins
type ProviderPlugin interface {
	// Name returns the unique identifier for this plugin
	Name() string

	// Version returns the version of the plugin
	Version() string

	// Capabilities returns the capabilities this plugin provides
	Capabilities() []types.Capability

	// Initialize initializes the plugin and returns a Provider instance
	Initialize(ctx context.Context, config models.ProviderConfig) (interfaces.Provider, error)

	// Metadata returns additional metadata about the plugin
	Metadata() map[string]any
}

// PluginRegistry manages provider plugin registration and discovery
type PluginRegistry interface {
	// RegisterPlugin registers a provider plugin
	RegisterPlugin(plugin ProviderPlugin) error

	// GetPlugin retrieves a plugin by name
	GetPlugin(name string) (ProviderPlugin, error)

	// ListPlugins returns all registered plugins
	ListPlugins() []ProviderPlugin

	// UnregisterPlugin removes a plugin from the registry
	UnregisterPlugin(name string) error

	// DiscoverAndRegister discovers plugins and registers them with the provider registry
	DiscoverAndRegister(ctx context.Context, configs map[string]models.ProviderConfig, providerRegistry ProviderRegistry) error
}

// pluginRegistry is the concrete implementation of PluginRegistry
type pluginRegistry struct {
	mu      sync.RWMutex
	plugins map[string]ProviderPlugin
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry() PluginRegistry {
	return &pluginRegistry{
		plugins: make(map[string]ProviderPlugin),
	}
}

// RegisterPlugin registers a provider plugin
func (pr *pluginRegistry) RegisterPlugin(plugin ProviderPlugin) error {
	if plugin == nil {
		return fmt.Errorf("plugin cannot be nil")
	}

	name := plugin.Name()
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	capabilities := plugin.Capabilities()
	if len(capabilities) == 0 {
		return fmt.Errorf("plugin %s must support at least one capability", name)
	}

	pr.mu.Lock()
	defer pr.mu.Unlock()

	// Check if plugin already exists
	if _, exists := pr.plugins[name]; exists {
		return fmt.Errorf("plugin %s is already registered", name)
	}

	pr.plugins[name] = plugin
	return nil
}

// GetPlugin retrieves a plugin by name
func (pr *pluginRegistry) GetPlugin(name string) (ProviderPlugin, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	plugin, exists := pr.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin, nil
}

// ListPlugins returns all registered plugins
func (pr *pluginRegistry) ListPlugins() []ProviderPlugin {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	plugins := make([]ProviderPlugin, 0, len(pr.plugins))
	for _, plugin := range pr.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// UnregisterPlugin removes a plugin from the registry
func (pr *pluginRegistry) UnregisterPlugin(name string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if _, exists := pr.plugins[name]; !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	delete(pr.plugins, name)
	return nil
}

// DiscoverAndRegister discovers plugins and registers them with the provider registry
func (pr *pluginRegistry) DiscoverAndRegister(
	ctx context.Context,
	configs map[string]models.ProviderConfig,
	providerRegistry ProviderRegistry,
) error {
	pr.mu.RLock()
	pluginsCopy := make(map[string]ProviderPlugin, len(pr.plugins))
	for name, plugin := range pr.plugins {
		pluginsCopy[name] = plugin
	}
	pr.mu.RUnlock()

	var errors []error
	var mu sync.Mutex

	// Process each plugin
	for name, plugin := range pluginsCopy {
		// Check if we have configuration for this plugin
		config, hasConfig := configs[name]
		if !hasConfig {
			// Skip plugins without configuration
			continue
		}

		// Initialize the plugin
		provider, err := plugin.Initialize(ctx, config)
		if err != nil {
			mu.Lock()
			errors = append(errors, fmt.Errorf("failed to initialize plugin %s: %w", name, err))
			mu.Unlock()
			continue
		}

		// Register the provider
		if err := providerRegistry.Register(provider); err != nil {
			mu.Lock()
			errors = append(errors, fmt.Errorf("failed to register provider %s: %w", name, err))
			mu.Unlock()
			// Close the provider since registration failed
			_ = provider.Close()
			continue
		}
	}

	// Return combined errors if any
	if len(errors) > 0 {
		return fmt.Errorf("plugin discovery encountered %d error(s): %v", len(errors), errors)
	}

	return nil
}

// ProviderDiscovery provides utilities for discovering and loading providers
type ProviderDiscovery struct {
	pluginRegistry   PluginRegistry
	providerRegistry ProviderRegistry
}

// NewProviderDiscovery creates a new provider discovery instance
func NewProviderDiscovery(pluginRegistry PluginRegistry, providerRegistry ProviderRegistry) *ProviderDiscovery {
	return &ProviderDiscovery{
		pluginRegistry:   pluginRegistry,
		providerRegistry: providerRegistry,
	}
}

// LoadProviders loads and initializes all configured providers
func (pd *ProviderDiscovery) LoadProviders(ctx context.Context, configs map[string]models.ProviderConfig) error {
	return pd.pluginRegistry.DiscoverAndRegister(ctx, configs, pd.providerRegistry)
}

// ReloadProvider reloads a specific provider with new configuration
func (pd *ProviderDiscovery) ReloadProvider(ctx context.Context, name string, config models.ProviderConfig) error {
	// Get the plugin
	plugin, err := pd.pluginRegistry.GetPlugin(name)
	if err != nil {
		return fmt.Errorf("plugin not found: %w", err)
	}

	// Unregister the existing provider if it exists
	_ = pd.providerRegistry.Unregister(name)

	// Initialize the plugin with new config
	provider, err := plugin.Initialize(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to initialize provider: %w", err)
	}

	// Register the new provider
	if err := pd.providerRegistry.Register(provider); err != nil {
		_ = provider.Close()
		return fmt.Errorf("failed to register provider: %w", err)
	}

	return nil
}

// GetProviderMetadata returns metadata about all available plugins
func (pd *ProviderDiscovery) GetProviderMetadata() []PluginMetadata {
	plugins := pd.pluginRegistry.ListPlugins()
	metadata := make([]PluginMetadata, 0, len(plugins))

	for _, plugin := range plugins {
		meta := PluginMetadata{
			Name:         plugin.Name(),
			Version:      plugin.Version(),
			Capabilities: plugin.Capabilities(),
			Metadata:     plugin.Metadata(),
		}

		// Check if provider is registered and get its info
		if info, err := pd.providerRegistry.GetProviderInfo(plugin.Name()); err == nil {
			meta.Available = info.Available
			meta.HealthStatus = string(info.HealthStatus)
		}

		metadata = append(metadata, meta)
	}

	return metadata
}

// PluginMetadata represents metadata about a provider plugin
type PluginMetadata struct {
	Name         string             `json:"name"`
	Version      string             `json:"version"`
	Capabilities []types.Capability `json:"capabilities"`
	Available    bool               `json:"available"`
	HealthStatus string             `json:"health_status"`
	Metadata     map[string]any     `json:"metadata,omitempty"`
}

// DiscoveryConfig contains configuration for provider discovery
type DiscoveryConfig struct {
	// AutoDiscover enables automatic discovery of providers
	AutoDiscover bool

	// ProviderConfigs maps provider names to their configurations
	ProviderConfigs map[string]models.ProviderConfig

	// EnabledProviders lists which providers should be loaded (empty = all)
	EnabledProviders []string

	// DisabledProviders lists which providers should not be loaded
	DisabledProviders []string
}

// ShouldLoadProvider checks if a provider should be loaded based on config
func (dc *DiscoveryConfig) ShouldLoadProvider(name string) bool {
	// Check if explicitly disabled
	for _, disabled := range dc.DisabledProviders {
		if disabled == name {
			return false
		}
	}

	// If enabled list is empty, load all (except disabled)
	if len(dc.EnabledProviders) == 0 {
		return true
	}

	// Check if explicitly enabled
	for _, enabled := range dc.EnabledProviders {
		if enabled == name {
			return true
		}
	}

	return false
}

// FilterConfigs returns only the configurations for providers that should be loaded
func (dc *DiscoveryConfig) FilterConfigs() map[string]models.ProviderConfig {
	if dc.ProviderConfigs == nil {
		return make(map[string]models.ProviderConfig)
	}

	filtered := make(map[string]models.ProviderConfig)
	for name, config := range dc.ProviderConfigs {
		if dc.ShouldLoadProvider(name) {
			filtered[name] = config
		}
	}

	return filtered
}
