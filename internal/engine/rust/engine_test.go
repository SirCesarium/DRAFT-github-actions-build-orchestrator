package rust

import (
	"os"
	"testing"

	"github.com/SirCesarium/refinery/internal/config"
	"github.com/SirCesarium/refinery/internal/engine"
)

func TestRustEngineValidate(t *testing.T) {
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
name = "my-rust-app"
version = "0.1.0"

[[bin]]
name = "my-bin"
path = "src/main.rs"
`
	if err := os.WriteFile("Cargo.toml", []byte(cargoContent), 0644); err != nil {
		t.Fatal(err)
	}

	e := &RustEngine{}
	cfg := &config.Config{
		Project: config.Project{Name: "my-rust-app", Lang: "rust"},
		Artifacts: map[string]*config.ArtifactConfig{
			"my-bin": {Type: "bin"},
		},
	}

	if err := e.Validate(cfg); err != nil {
		t.Errorf("expected validation to pass, got: %v", err)
	}

	cfg.Artifacts["non-existent"] = &config.ArtifactConfig{Type: "bin"}
	if err := e.Validate(cfg); err == nil {
		t.Error("expected validation to fail for non-existent artifact")
	}
}

func TestIsLibDefined(t *testing.T) {
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
name = "my-package"
version = "0.1.0"

[lib]
name = "my-lib"
`
	if err := os.WriteFile("Cargo.toml", []byte(cargoContent), 0644); err != nil {
		t.Fatal(err)
	}

	e := &RustEngine{}
	m, err := e.loadManifest()
	if err != nil {
		t.Fatal(err)
	}

	if !e.isLibDefined("my-lib", m) {
		t.Error("expected my-lib to be defined")
	}
	if e.isLibDefined("my_package", m) {
		t.Error("my_package should not match when lib name is explicitly set")
	}
	if e.isLibDefined("wrong-lib", m) {
		t.Error("expected wrong-lib to not be defined")
	}
}

func TestIsBinDefined(t *testing.T) {
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
name = "my-package"
version = "0.1.0"

[[bin]]
name = "bin1"

[[bin]]
name = "bin2"
`
	if err := os.WriteFile("Cargo.toml", []byte(cargoContent), 0644); err != nil {
		t.Fatal(err)
	}

	e := &RustEngine{}
	m, err := e.loadManifest()
	if err != nil {
		t.Fatal(err)
	}

	if !e.isBinDefined("bin1", m) {
		t.Error("expected bin1 to be defined")
	}
	if e.isBinDefined("my-package", m) {
		t.Error("my-package should not match when bin names are explicitly defined")
	}
	if e.isBinDefined("wrong-bin", m) {
		t.Error("expected wrong-bin to not be defined")
	}
}

func TestAddTargetRequirements(t *testing.T) {
	e := &RustEngine{}

	reqs := []string{"rust"}
	art := &config.ArtifactConfig{
		Targets: map[string]config.TargetConfig{
			"linux-arm": {OS: "linux", Archs: []string{"aarch64"}},
		},
	}
	reqs = e.addTargetRequirements(reqs, art)

	found := false
	for _, r := range reqs {
		if r == "cross-linker:linux-aarch64" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected cross-linker:linux-aarch64 requirement")
	}
}

func TestAddPackageRequirements(t *testing.T) {
	e := &RustEngine{}

	reqs := []string{}
	art := &config.ArtifactConfig{
		Packages: []string{"deb"},
	}
	reqs = e.addPackageRequirements(reqs, art)

	found := false
	for _, r := range reqs {
		if r == "pkg:cargo-deb" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected pkg:cargo-deb requirement")
	}
}

func TestID(t *testing.T) {
	e := &RustEngine{}
	if e.ID() != "rust" {
		t.Errorf("expected ID to be 'rust', got %s", e.ID())
	}
}

func TestPrepare(t *testing.T) {
	e := &RustEngine{}
	if err := e.Prepare(nil); err != nil {
		t.Errorf("expected no error from Prepare, got: %v", err)
	}
}

func TestIsArtifactDefinedInManifest(t *testing.T) {
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
version = "0.1.0"
edition = "2024"

[[bin]]
name = "my-bin"
`
	if err := os.WriteFile("Cargo.toml", []byte(cargoContent), 0644); err != nil {
		t.Fatal(err)
	}

	e := &RustEngine{}
	m, err := e.loadManifest()
	if err != nil {
		t.Fatal(err)
	}

	if !e.isArtifactDefinedInManifest("my-bin", "bin", m) {
		t.Error("expected my-bin to be defined as bin")
	}
	if e.isArtifactDefinedInManifest("non-existent", "bin", m) {
		t.Error("expected non-existent to not be defined as bin")
	}

	cargoContent = `
[package]
name = "test-pkg"
version = "0.1.0"
edition = "2024"

[lib]
name = "my-lib"
`
	if err := os.WriteFile("Cargo.toml", []byte(cargoContent), 0644); err != nil {
		t.Fatal(err)
	}

	m, err = e.loadManifest()
	if err != nil {
		t.Fatal(err)
	}

	if !e.isArtifactDefinedInManifest("my-lib", "lib", m) {
		t.Error("expected my-lib to be defined as lib")
	}
}

func TestGetCIRequirements(t *testing.T) {
	e := &RustEngine{}
	cfg := &config.Config{
		Artifacts: map[string]*config.ArtifactConfig{
			"test": {
				Type:     "bin",
				Packages: []string{"deb", "rpm"},
				Targets: map[string]config.TargetConfig{
					"linux-arm": {OS: "linux", Archs: []string{"aarch64"}},
				},
			},
		},
	}

	reqs := e.GetCIRequirements(cfg)
	if len(reqs) == 0 {
		t.Error("expected CI requirements")
	}
}

func TestBuild(t *testing.T) {
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

	cargoContent := `
[package]
name = "test"
version = "0.1.0"
edition = "2024"

[[bin]]
name = "test"
`
	if err := os.WriteFile("Cargo.toml", []byte(cargoContent), 0644); err != nil {
		t.Fatal(err)
	}

	e := &RustEngine{}
	cfg := &config.Config{
		Project: config.Project{Name: "test", Lang: "rust"},
		Artifacts: map[string]*config.ArtifactConfig{
			"test": {
				Type: "bin",
				Targets: map[string]config.TargetConfig{
					"linux-x64": {OS: "linux", Archs: []string{"x86_64"}},
				},
			},
		},
	}

	if err := e.Build(cfg, cfg.Artifacts["test"], engine.BuildOptions{
		ArtifactName: "test",
		OS:           "linux",
		Arch:         "x86_64",
	}); err != nil {
		t.Errorf("Build returned error: %v", err)
	}
}

func TestPackage(t *testing.T) {
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
name = "test"
version = "0.1.0"
edition = "2024"

[[bin]]
name = "test"
`
	if err := os.WriteFile("Cargo.toml", []byte(cargoContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll("target/x86_64-unknown-linux-gnu/release", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("target/x86_64-unknown-linux-gnu/release/test", []byte("binary"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll("dist", 0755); err != nil {
		t.Fatal(err)
	}

	t.Skip("Package test requires complex setup with cargo tools")
}

func TestValidateRustSpecific(t *testing.T) {
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
name = "test-project"
version = "0.1.0"
edition = "2024"

[[bin]]
name = "test-bin"
`
	if err := os.WriteFile("Cargo.toml", []byte(cargoContent), 0644); err != nil {
		t.Fatal(err)
	}

	e := &RustEngine{}
	cfg := &config.Config{
		Project: config.Project{Name: "test-project", Lang: "rust"},
		Artifacts: map[string]*config.ArtifactConfig{
			"test-bin": {Type: "bin"},
		},
	}

	if err := e.ValidateRustSpecific(cfg); err != nil {
		t.Errorf("ValidateRustSpecific failed: %v", err)
	}
}

func TestUniqueFormats(t *testing.T) {
	e := &RustEngine{}
	formats := []string{"deb", "rpm", "deb", "tar.gz", "rpm"}
	unique := e.uniqueFormats(formats)
	if len(unique) != 3 {
		t.Errorf("expected 3 unique formats, got %d", len(unique))
	}
}

func TestSliceContains(t *testing.T) {
	e := &RustEngine{}
	if !e.sliceContains([]string{"a", "b", "c"}, "b") {
		t.Error("expected 'b' to be found")
	}
	if e.sliceContains([]string{"a", "b", "c"}, "d") {
		t.Error("expected 'd' to not be found")
	}
}

func TestValidateWithInvalidTarget(t *testing.T) {
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
name = "my-rust-app"
version = "0.1.0"
edition = "2024"

[[bin]]
name = "my-bin"
path = "src/main.rs"
`
	if err := os.WriteFile("Cargo.toml", []byte(cargoContent), 0644); err != nil {
		t.Fatal(err)
	}

	e := &RustEngine{}
	cfg := &config.Config{
		Project: config.Project{Name: "my-rust-app", Lang: "rust"},
		Artifacts: map[string]*config.ArtifactConfig{
			"my-bin":       {Type: "bin"},
			"non-existent": {Type: "bin"},
		},
	}

	if err := e.Validate(cfg); err == nil {
		t.Error("expected validation to fail for non-existent artifact")
	}
}

func TestValidateRustSpecificWithArm64(t *testing.T) {
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
name = "test-project"
version = "0.1.0"
edition = "2024"

[[bin]]
name = "test-bin"
`
	if err := os.WriteFile("Cargo.toml", []byte(cargoContent), 0644); err != nil {
		t.Fatal(err)
	}

	e := &RustEngine{}
	cfg := &config.Config{
		Project: config.Project{Name: "test-project", Lang: "rust"},
		Artifacts: map[string]*config.ArtifactConfig{
			"test-bin": {
				Type: "bin",
				Targets: map[string]config.TargetConfig{
					"darwin-arm": {OS: "darwin", Archs: []string{"arm64"}},
				},
			},
		},
	}

	if err := e.ValidateRustSpecific(cfg); err == nil {
		t.Error("expected error for arm64 (should be aarch64)")
	}
}
