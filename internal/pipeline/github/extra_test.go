package github

import (
	"testing"

	"github.com/SirCesarium/refinery/internal/config"
	"github.com/SirCesarium/refinery/internal/engine"
)

func TestNewProviderFunction(t *testing.T) {
	p, err := NewProvider("build")
	if err != nil {
		t.Fatalf("NewProvider returned error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.Filename() != ".github/workflows/build.yml" {
		t.Errorf("expected '.github/workflows/build.yml', got '%s'", p.Filename())
	}

	p, err = NewProvider("")
	if err == nil {
		t.Error("expected error for empty filename")
	}
	if p != nil {
		t.Error("expected nil provider for empty filename")
	}
}

func TestGenerateFunction(t *testing.T) {
	mockEng := &mockBuildEngineForGithub{
		requirements: []string{"rust", "cargo-deb"},
	}

	p, err := NewProvider("build")
	if err != nil {
		t.Fatalf("NewProvider returned error: %v", err)
	}
	cfg := &config.Config{
		Project: config.Project{
			Name: "test-project",
			Lang: "rust",
		},
		Artifacts: map[string]*config.ArtifactConfig{
			"test": {
				Type: "bin",
				Targets: map[string]config.TargetConfig{
					"linux-x64": {
						OS:    "linux",
						Archs: []string{"x86_64"},
					},
				},
				Source: "main.go",
			},
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Generate panicked: %v", r)
		}
	}()

	if _, err := p.Generate(cfg, mockEng); err != nil {
		t.Errorf("Generate failed: %v", err)
	}
}

type mockBuildEngineForGithub struct {
	requirements []string
}

func (m *mockBuildEngineForGithub) ID() string                      { return "mock" }
func (m *mockBuildEngineForGithub) Prepare(_ *config.Config) error  { return nil }
func (m *mockBuildEngineForGithub) Validate(_ *config.Config) error { return nil }
func (m *mockBuildEngineForGithub) Build(_ *config.Config, _ *config.ArtifactConfig, _ engine.BuildOptions) error {
	return nil
}
func (m *mockBuildEngineForGithub) GetCIRequirements(_ *config.Config) []string {
	return m.requirements
}
func (m *mockBuildEngineForGithub) Package(_ *config.Config, _ *config.ArtifactConfig, _ engine.BuildOptions, _ string) error {
	return nil
}
func (m *mockBuildEngineForGithub) GetSupportedArchs(_ string) []string {
	return []string{"x86_64", "i686", "aarch64"}
}

func TestGetSplitStepsFunction(t *testing.T) {
	mockEng := &mockBuildEngineForGithub{
		requirements: []string{"rust", "cargo-deb"},
	}
	cfg := &config.Config{
		Project: config.Project{
			Name: "test-project",
			Lang: "rust",
		},
		BuildRefinery: &config.BuildRefineryConfig{Enabled: true},
		PostBuild: []config.BuildStep{
			{Action: "cleanup", Once: true},
		},
	}

	p, err := NewProvider("build")
	if err != nil {
		t.Fatalf("NewProvider returned error: %v", err)
	}
	setup, build, teardown := p.getSplitSteps(mockEng, cfg)
	if len(setup) == 0 || len(build) == 0 || len(teardown) == 0 {
		t.Error("expected non-empty setup, build, and teardown steps")
	}
}

func TestGetBuildArtifactStepsFunction(t *testing.T) {
	cfg := &config.Config{
		Project: config.Project{
			Name: "test-project",
			Lang: "rust",
		},
		Artifacts: map[string]*config.ArtifactConfig{
			"test": {
				Type: "bin",
				Targets: map[string]config.TargetConfig{
					"linux-x64": {
						OS:    "linux",
						Archs: []string{"x86_64"},
					},
				},
				Source: "main.go",
			},
		},
	}

	p, err := NewProvider("build")
	if err != nil {
		t.Fatalf("NewProvider returned error: %v", err)
	}
	steps := p.getBuildArtifactStep(cfg)
	if len(steps) == 0 {
		t.Error("expected non-empty steps")
	}
}

func TestCreateGithubStepResolution(t *testing.T) {
	p := &Provider{}
	eng := &mockBuildEngineForGithub{}
	cfg := &config.Config{
		PreBuild: []config.BuildStep{
			{Action: "smoke-test", With: map[string]any{"key": "value"}, Once: true},
			{Action: "custom/action@v1", OS: []string{"linux"}},
			{Action: "my-workflow.yml", OS: []string{"windows", "linux"}, Once: true},
		},
	}

	setup, build, _ := p.getSplitSteps(eng, cfg)

	// Test local action resolution and Once flag
	foundGlobal := false
	for _, s := range setup {
		if s.Uses == "./.github/actions/smoke-test" {
			foundGlobal = true
			break
		}
	}
	if !foundGlobal {
		t.Error("expected './.github/actions/smoke-test' in setup steps")
	}

	// Test OS filtering
	foundOS := false
	for _, s := range build {
		if s.Uses == "custom/action@v1" && s.If == "runner.os == 'Linux'" {
			foundOS = true
			break
		}
	}
	if !foundOS {
		t.Error("expected 'custom/action@v1' with OS filter in build steps")
	}

	// Test combined OS and Once
	foundCombined := false
	for _, s := range setup {
		if s.Uses == "my-workflow.yml" {
			foundCombined = true
			expectedIf := "runner.os == 'Windows' || runner.os == 'Linux'"
			if s.If != expectedIf {
				t.Errorf("expected if '%s', got '%s'", expectedIf, s.If)
			}
			break
		}
	}
	if !foundCombined {
		t.Error("expected 'my-workflow.yml' in setup steps")
	}
}
