// Package pipeline generates CI/CD workflows using a provider and engine.
package pipeline

import (
	"github.com/SirCesarium/refinery/internal/config"
	"github.com/SirCesarium/refinery/internal/engine"
)

// CIProvider defines the interface for CI workflow generators.
type CIProvider interface {
	Name() string
	Generate(cfg *config.Config, eng engine.BuildEngine) ([]byte, error)
	Filename() string
}

// ProviderRegistry stores available CI providers.
type ProviderRegistry struct {
	providers map[string]CIProvider
}

// NewRegistry creates and returns a new ProviderRegistry.
func NewRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]CIProvider),
	}
}

// Register adds a CI provider to the registry.
func (r *ProviderRegistry) Register(p CIProvider) {
	r.providers[p.Name()] = p
}

// Get retrieves a CI provider by name.
func (r *ProviderRegistry) Get(name string) CIProvider {
	return r.providers[name]
}

// Generator orchestrates workflow generation using a provider and engine.
type Generator struct {
	provider CIProvider
	engine   engine.BuildEngine
}

// NewGenerator creates a new Generator with the given provider and engine.
func NewGenerator(p CIProvider, e engine.BuildEngine) *Generator {
	return &Generator{provider: p, engine: e}
}

// Generate creates the workflow file using the configured provider.
func (g *Generator) Generate(cfg *config.Config) ([]byte, error) {
	return g.provider.Generate(cfg, g.engine)
}

// Filename returns the target workflow filename from the provider.
func (g *Generator) Filename() string {
	return g.provider.Filename()
}
