package rust

import (
	"os"
	"testing"

	"github.com/SirCesarium/refinery/internal/config"
	"github.com/SirCesarium/refinery/internal/engine"
)

func TestGetProfile(t *testing.T) {
	e := &RustEngine{}

	tCfg := config.TargetConfig{}
	if profile := e.getProfile(tCfg); profile != "release" {
		t.Errorf("expected 'release', got '%s'", profile)
	}

	tCfg.LangOpts = map[string]any{"profile": "debug"}
	if profile := e.getProfile(tCfg); profile != "debug" {
		t.Errorf("expected 'debug', got '%s'", profile)
	}

	tCfg.LangOpts = map[string]any{"profile": "custom"}
	if profile := e.getProfile(tCfg); profile != "custom" {
		t.Errorf("expected 'custom', got '%s'", profile)
	}
}

func TestGetBinaryExt(t *testing.T) {
	e := &RustEngine{}

	art := &config.ArtifactConfig{Type: "bin"}
	ext := e.getBinaryExt(art, "linux", "")
	if ext != "" {
		t.Errorf("expected empty string for bin on linux, got '%s'", ext)
	}

	art = &config.ArtifactConfig{Type: "lib", LibraryTypes: []string{"cdylib"}}
	ext = e.getBinaryExt(art, "linux", "")
	if ext != "so" {
		t.Errorf("expected 'so' for lib on linux, got '%s'", ext)
	}

	art = &config.ArtifactConfig{Type: "lib", LibraryTypes: []string{"staticlib"}}
	ext = e.getBinaryExt(art, "linux", "")
	if ext != "a" {
		t.Errorf("expected 'a' for staticlib on linux, got '%s'", ext)
	}

	art = &config.ArtifactConfig{Type: "lib", LibraryTypes: []string{"cdylib"}}
	ext = e.getBinaryExt(art, "windows", "")
	if ext != "dll" {
		t.Errorf("expected 'dll' for cdylib on windows, got '%s'", ext)
	}
}

func TestResolveBinaryInfo(t *testing.T) {
	e := &RustEngine{}

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

	cargoContent := `
[package]
name = "test-pkg"
version = "1.0.0"
`
	if err := os.WriteFile("Cargo.toml", []byte(cargoContent), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := e.loadManifest()
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Naming: config.NamingConfig{
			Binary:  "{artifact}-{os}-{arch}{abi}",
			Package: "{artifact}-{version}-{os}-{arch}{abi}.{ext}",
		},
		OutputDir: "dist",
	}

	art := &config.ArtifactConfig{Type: "bin"}
	opts := engine.BuildOptions{
		ArtifactName: "test-app",
		OS:           "linux",
		Arch:         "x86_64",
	}

	binaryName, binaryPath := e.resolveBinaryInfo(cfg, art, opts, m, "x86_64-unknown-linux-gnu", "release", "1.0.0")
	if binaryName == "" {
		t.Error("expected non-empty binary name")
	}
	if binaryPath == "" {
		t.Error("expected non-empty binary path")
	}
}

func TestRunHooks(t *testing.T) {
	e := &RustEngine{}

	art := &config.ArtifactConfig{
		Type: "bin",
		Hooks: config.Hooks{
			PreBuild:  []string{"echo pre1", "echo pre2"},
			PostBuild: []string{"echo post1"},
		},
	}

	opts := engine.BuildOptions{
		ArtifactName: "test",
		OS:           "linux",
		Arch:         "x86_64",
	}

	if err := e.runHooks(art, opts, "/bin/test", "PreBuild"); err != nil {
		t.Errorf("PreBuild hooks failed: %v", err)
	}

	if err := e.runHooks(art, opts, "/bin/test", "PostBuild"); err != nil {
		t.Errorf("PostBuild hooks failed: %v", err)
	}

	art.Hooks.PreBuild = []string{"exit 1"}
	if err := e.runHooks(art, opts, "/bin/test", "PreBuild"); err == nil {
		t.Error("expected error from failing hook")
	}
}

func TestMoveArtifacts(t *testing.T) {
	e := &RustEngine{}

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

	if err := os.MkdirAll("target/x86_64-unknown-linux-gnu/release", 0755); err != nil {
		t.Fatal(err)
	}

	// Create file with the name that moveArtifacts will look for
	// The naming template is "{artifact}-{os}-{arch}{abi}"
	expectedFileName := "test-app-linux-x86_64"
	if err := os.WriteFile("target/x86_64-unknown-linux-gnu/release/"+expectedFileName, []byte("binary"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		OutputDir: "dist",
		Naming: config.NamingConfig{
			Binary: "{artifact}-{os}-{arch}{abi}",
		},
	}

	art := &config.ArtifactConfig{Type: "bin"}
	manifest := &cargoManifest{}
	manifest.Package.Name = "test-app"
	manifest.Package.Version = "1.0.0"

	if err := e.moveArtifacts(cfg, art, "test-app", "linux", "x86_64", "", "x86_64-unknown-linux-gnu", "1.0.0", "release", manifest); err != nil {
		t.Errorf("moveArtifacts failed: %v", err)
	}

	if _, err := os.Stat("dist/test-app-linux-x86_64"); os.IsNotExist(err) {
		t.Error("expected artifact to be moved to dist")
	}
}
