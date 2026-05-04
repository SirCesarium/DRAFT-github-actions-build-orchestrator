package pipeline

import (
	"testing"

	"github.com/SirCesarium/refinery/internal/config"
	"github.com/SirCesarium/refinery/internal/engine"
)

// Mock CIProvider for testing.
type mockCIProvider struct {
	name        string
	generateErr error
	filename    string
}

func (m *mockCIProvider) Name() string {
	return m.name
}
func (m *mockCIProvider) Generate(_ *config.Config, _ engine.BuildEngine) ([]byte, error) {
	if m.generateErr != nil {
		return nil, m.generateErr
	}
	return []byte("test-output"), nil
}
func (m *mockCIProvider) Filename() string {
	return m.filename
}

// TestNewRegistry checks registry initialization.
func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if len(r.providers) != 0 {
		t.Errorf("expected empty providers map, got %d", len(r.providers))
	}
}

// TestRegisterAndGet checks provider registration and retrieval.
func TestRegisterAndGet(t *testing.T) {
	r := NewRegistry()

	p := &mockCIProvider{name: "test-provider", filename: "test.yml"}
	r.Register(p)

	// Retrieve the provider
	retrieved := r.Get("test-provider")
	if retrieved == nil {
		t.Fatal("expected non-nil provider")
	}
	if retrieved.Name() != "test-provider" {
		t.Errorf("expected 'test-provider', got '%s'", retrieved.Name())
	}
}

// TestGetNonExistent checks nil for non-existent provider.
func TestGetNonExistent(t *testing.T) {
	r := NewRegistry()
	p := r.Get("non-existent")
	if p != nil {
		t.Error("expected nil for non-existent provider")
	}
}

// TestNewGenerator checks generator initialization.
func TestNewGenerator(t *testing.T) {
	p := &mockCIProvider{name: "test"}
	eng := &mockEngine{}

	g := NewGenerator(p, eng)
	if g == nil {
		t.Fatal("expected non-nil generator")
	}
	if g.provider == nil {
		t.Error("expected non-nil provider in generator")
	}
	if g.engine == nil {
		t.Error("expected non-nil engine in generator")
	}
}

// TestGeneratorGenerate checks Generate method.
func TestGeneratorGenerate(t *testing.T) {
	p := &mockCIProvider{name: "test", filename: "test.yml"}
	eng := &mockEngine{}
	cfg := &config.Config{}

	g := NewGenerator(p, eng)
	data, err := g.Generate(cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if string(data) != "test-output" {
		t.Errorf("expected 'test-output', got '%s'", string(data))
	}
}

// TestGeneratorFilename checks Filename method.
func TestGeneratorFilename(t *testing.T) {
	p := &mockCIProvider{name: "test", filename: "workflows/test.yml"}
	g := NewGenerator(p, nil)

	filename := g.Filename()
	if filename != "workflows/test.yml" {
		t.Errorf("expected 'workflows/test.yml', got '%s'", filename)
	}
}

// mockEngine implements engine.BuildEngine for testing.
type mockEngine struct{}

func (m *mockEngine) ID() string                      { return "mock" }
func (m *mockEngine) Prepare(_ *config.Config) error  { return nil }
func (m *mockEngine) Validate(_ *config.Config) error { return nil }
func (m *mockEngine) Build(_ *config.Config, _ *config.ArtifactConfig, _ engine.BuildOptions) error {
	return nil
}
func (m *mockEngine) GetCIRequirements(_ *config.Config) []string { return nil }
func (m *mockEngine) Package(_ *config.Config, _ *config.ArtifactConfig, _ engine.BuildOptions, _ string) error {
	return nil
}
func (m *mockEngine) GetSupportedArchs(_ string) []string {
	return []string{"amd64", "386", "arm64"}
}
