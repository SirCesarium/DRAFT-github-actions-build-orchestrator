package github

import (
	"strings"
	"testing"

	"github.com/SirCesarium/refinery/internal/config"
)

// TestGetRunsOn tests OS to runner label mapping.
func TestGetRunsOn(t *testing.T) {
	p := &Provider{}

	if label := p.getRunsOn("linux"); label != "ubuntu-latest" {
		t.Errorf("expected 'ubuntu-latest', got '%s'", label)
	}
	if label := p.getRunsOn("windows"); label != "windows-latest" {
		t.Errorf("expected 'windows-latest', got '%s'", label)
	}
	if label := p.getRunsOn("darwin"); label != "macos-latest" {
		t.Errorf("expected 'macos-latest', got '%s'", label)
	}
}

// TestSortedArtifactNames tests artifact name sorting.
func TestSortedArtifactNames(t *testing.T) {
	p := &Provider{}
	cfg := &config.Config{
		Artifacts: map[string]*config.ArtifactConfig{
			"zebra": {},
			"apple": {},
			"mango": {},
		},
	}

	names := p.sortedArtifactNames(cfg)
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	if names[0] != "apple" || names[1] != "mango" || names[2] != "zebra" {
		t.Errorf("expected sorted names [apple, mango, zebra], got %v", names)
	}
}

// TestAssembleJobs tests job assembly logic.
func TestAssembleJobs(t *testing.T) {
	p := &Provider{}
	include := []MatrixEntry{
		{Artifact: "test", OS: "linux", Arch: "x86_64", RunsOn: "ubuntu-latest"},
	}
	setupSteps := []Step{
		{Name: "Checkout", Uses: "actions/checkout@v6"},
		{Name: "Setup", Run: "echo setup"},
	}
	buildSteps := []Step{
		{Name: "Test Step", Run: "echo test"},
	}
	teardownSteps := []Step{
		{Name: "Checkout", Uses: "actions/checkout@v6"},
		{Name: "Download", Uses: "actions/download-artifact@v8"},
		{Name: "Teardown", Run: "echo teardown"},
	}

	jobs := p.assembleJobs(include, setupSteps, buildSteps, teardownSteps)

	// Check that expected jobs exist (fixed order struct)
	if jobs.Setup == nil {
		t.Error("expected 'setup' job")
	}
	if jobs.Build.Name == "" {
		t.Error("expected 'build' job")
	}
	if jobs.Teardown == nil {
		t.Error("expected 'teardown' job")
	}
	if len(jobs.Build.Needs) != 1 || jobs.Build.Needs[0] != "setup" {
		t.Error("expected 'build' job to need 'setup'")
	}
	if len(jobs.Teardown.Needs) != 1 || jobs.Teardown.Needs[0] != "build" {
		t.Error("expected 'teardown' job to need 'build'")
	}
}

// TestBuildMatrix tests matrix generation.
func TestBuildMatrix(t *testing.T) {
	p := &Provider{}
	cfg := &config.Config{
		Artifacts: map[string]*config.ArtifactConfig{
			"test-art": {
				Targets: map[string]config.TargetConfig{
					"linux-x64": {OS: "linux", Archs: []string{"x86_64"}},
				},
			},
		},
	}

	include := p.buildMatrix(cfg)
	if len(include) != 1 {
		t.Fatalf("expected 1 matrix entry, got %d", len(include))
	}
	if include[0].OS != "linux" {
		t.Errorf("expected os 'linux', got '%v'", include[0].OS)
	}
}

// TestNewProvider tests provider initialization.
func TestNewProvider(t *testing.T) {
	p, err := NewProvider(".github/workflows/build.yml")
	if err != nil {
		t.Fatalf("NewProvider returned error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.Name() != "github" {
		t.Errorf("expected 'github', got '%s'", p.Name())
	}
}

// TestProviderName tests Name method.
func TestProviderName(t *testing.T) {
	p, _ := NewProvider(".github/workflows/build.yml")
	if p.Name() != "github" {
		t.Errorf("expected 'github', got '%s'", p.Name())
	}
}

// TestProviderFilename tests Filename method.
func TestProviderFilename(t *testing.T) {
	filename := "custom"
	p, _ := NewProvider(filename)
	expected := ".github/workflows/" + filename + ".yml"
	if p.Filename() != expected {
		t.Errorf("expected '%s', got '%s'", expected, p.Filename())
	}
}

func TestProviderGenerate(t *testing.T) {
	p, err := NewProvider("build")
	if err != nil {
		t.Fatalf("NewProvider returned error: %v", err)
	}

	cfg := &config.Config{
		Project: config.Project{
			Name: "test-project",
			Lang: "rust",
		},
	}

	mockEng := &mockBuildEngineForGithub{
		requirements: []string{"rust"},
	}

	if _, err := p.Generate(cfg, mockEng); err != nil {
		t.Errorf("Generate failed: %v", err)
	}
}

func TestGetSplitSteps(t *testing.T) {
	p, err := NewProvider("build")
	if err != nil {
		t.Fatalf("NewProvider returned error: %v", err)
	}

	mockEng := &mockBuildEngineForGithub{
		requirements: []string{"rust"},
	}
	cfg := &config.Config{
		Project: config.Project{
			Name: "test-project",
			Lang: "rust",
		},
		BuildRefinery: &config.BuildRefineryConfig{Enabled: true},
		PreBuild: []config.BuildStep{
			{Action: "smoke-test", Once: true},
			{Action: "other-test", Once: false},
		},
		PostBuild: []config.BuildStep{
			{Action: "cleanup", Once: true},
		},
	}

	setup, build, teardown := p.getSplitSteps(mockEng, cfg)

	// Check setup steps
	foundGlobal := false
	foundBuildRef := false
	for _, s := range setup {
		if strings.Contains(s.Name, "Global") {
			foundGlobal = true
		}
		if s.Name == "Build Refinery from Source" {
			foundBuildRef = true
		}
	}
	if !foundGlobal {
		t.Error("expected global pre-build step in setup")
	}
	if !foundBuildRef {
		t.Error("expected refinery build step in setup")
	}

	// Check build steps
	foundLocal := false
	for _, s := range build {
		if s.Name == "Pre-Build: other-test" {
			foundLocal = true
		}
	}
	if !foundLocal {
		t.Error("expected matrix pre-build step in build")
	}

	// Check teardown steps
	if len(teardown) <= 2 { // Checkout + Download
		t.Error("expected teardown steps to be populated")
	}
}

func TestGetBuildArtifactSteps(t *testing.T) {
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
				Type: "binary",
				Targets: map[string]config.TargetConfig{
					"linux-x64": {
						OS:    "linux",
						Archs: []string{"x86_64"},
					},
				},
			},
		},
	}

	steps := p.getBuildArtifactStep(cfg)
	if len(steps) == 0 {
		t.Error("expected non-empty steps")
	}
}
