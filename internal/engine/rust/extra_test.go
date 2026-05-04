package rust

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SirCesarium/refinery/internal/config"
	"github.com/SirCesarium/refinery/internal/engine"
)

func TestBuildFunction(t *testing.T) {
	if _, err := os.Stat("/usr/bin/cargo"); os.IsNotExist(err) {
		t.Skip("cargo not installed")
	}

	tmpDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	}()

	cargoContent := `[package]
name = "test-build"
version = "0.1.0"
edition = "2024"
`
	if err := os.WriteFile("Cargo.toml", []byte(cargoContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll("src", 0755); err != nil {
		t.Fatal(err)
	}
	mainRs := `fn main() {
	println!("Hello, world!");
}
`
	if err := os.WriteFile("src/main.rs", []byte(mainRs), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Project: config.Project{
			Name: "test-build",
			Lang: "rust",
		},
		Artifacts: map[string]*config.ArtifactConfig{
			"test-build": {
				Type:   "bin",
				Source: "src/main.rs",
			},
		},
		OutputDir: filepath.Join(tmpDir, "output"),
	}
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		t.Fatal(err)
	}

	e := &RustEngine{}
	art := cfg.Artifacts["test-build"]
	opts := engine.BuildOptions{
		ArtifactName: "test-build",
		OS:           "linux",
		Arch:         "x86_64",
	}

	if err := e.build(cfg, art, opts); err != nil {
		t.Errorf("build failed: %v", err)
	}

	expectedBinary := filepath.Join(tmpDir, "target", "release", "test-build")
	if _, err := os.Stat(expectedBinary); os.IsNotExist(err) {
		t.Errorf("expected binary to be created at %s", expectedBinary)
	}
}

func TestRunHookFunction(t *testing.T) {
	e := &RustEngine{}

	if err := e.runHook("echo hello"); err != nil {
		t.Errorf("runHook with echo failed: %v", err)
	}

	if err := e.runHook("exit 1"); err == nil {
		t.Error("expected error from failing hook")
	}
}
