package rust

import (
	"testing"

	"github.com/SirCesarium/refinery/internal/config"
)

func TestGetLinkerFromConfig(t *testing.T) {
	e := &RustEngine{}

	art := &config.ArtifactConfig{
		Targets: map[string]config.TargetConfig{
			"linux-x86_64": {
				OS:    "linux",
				Archs: []string{"x86_64"},
				LangOpts: map[string]any{
					"linker": "my-linker",
				},
			},
		},
	}

	linker := e.getLinkerFromConfig(art, "linux", "x86_64")
	if linker != "my-linker" {
		t.Errorf("expected 'my-linker', got '%s'", linker)
	}

	art2 := &config.ArtifactConfig{}
	linker = e.getLinkerFromConfig(art2, "linux", "x86_64")
	if linker != "" {
		t.Errorf("expected empty string, got '%s'", linker)
	}

	linker = e.getLinkerFromConfig(nil, "linux", "x86_64")
	if linker != "" {
		t.Errorf("expected empty string for nil config, got '%s'", linker)
	}
}

func TestRunHook(t *testing.T) {
	e := &RustEngine{}

	if err := e.runHook("echo hello"); err != nil {
		t.Errorf("runHook with echo failed: %v", err)
	}

	if err := e.runHook("exit 1"); err == nil {
		t.Error("expected error from failing hook")
	}

	if err := e.runHook(""); err != nil {
		t.Errorf("expected no error for empty hook, got: %v", err)
	}
}
