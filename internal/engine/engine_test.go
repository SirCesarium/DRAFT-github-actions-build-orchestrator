package engine

import (
	"testing"

	"github.com/SirCesarium/refinery/internal/config"
)

// TestNewRegistry checks that registry is properly initialized.
func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if len(r.engines) != 0 {
		t.Errorf("expected empty registry, got %d engines", len(r.engines))
	}
}

// TestRegisterAndGet checks engine registration and retrieval.
func TestRegisterAndGet(t *testing.T) {
	r := NewRegistry()

	// Create a mock engine
	eng := &mockEngine{id: "test-engine"}
	r.Register(eng)

	// Retrieve the engine
	retrieved := r.Get("test-engine")
	if retrieved == nil {
		t.Fatal("expected non-nil engine")
	}
	if retrieved.ID() != "test-engine" {
		t.Errorf("expected 'test-engine', got '%s'", retrieved.ID())
	}
}

// TestGetNonExistent checks nil for non-existent engine.
func TestGetNonExistent(t *testing.T) {
	r := NewRegistry()
	eng := r.Get("non-existent")
	if eng != nil {
		t.Error("expected nil for non-existent engine")
	}
}

// TestGetAfterRegister checks engine retrieval after registration.
func TestGetAfterRegister(t *testing.T) {
	r := NewRegistry()

	p := &mockEngine{id: "test"}
	r.Register(p)

	retrieved := r.Get("test")
	if retrieved == nil {
		t.Fatal("expected non-nil engine")
	}
	if retrieved.ID() != "test" {
		t.Errorf("expected 'test', got '%s'", retrieved.ID())
	}
}

// mockEngine implements engine.BuildEngine for testing.
type mockEngine struct {
	id string
}

func (m *mockEngine) ID() string                      { return m.id }
func (m *mockEngine) Prepare(_ *config.Config) error  { return nil }
func (m *mockEngine) Validate(_ *config.Config) error { return nil }
func (m *mockEngine) Build(_ *config.Config, _ *config.ArtifactConfig, _ BuildOptions) error {
	return nil
}
func (m *mockEngine) GetCIRequirements(_ *config.Config) []string { return nil }
func (m *mockEngine) Package(_ *config.Config, _ *config.ArtifactConfig, _ BuildOptions, _ string) error {
	return nil
}
func (m *mockEngine) GetSupportedArchs(_ string) []string {
	return []string{"amd64", "386", "arm64"}
}
