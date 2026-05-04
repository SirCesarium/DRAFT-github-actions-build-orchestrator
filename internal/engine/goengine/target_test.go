package goengine

import (
	"os"
	"testing"

	"github.com/SirCesarium/refinery/internal/config"
	"github.com/SirCesarium/refinery/internal/engine"
)

func TestValidateTarget(t *testing.T) {
	e := &GoEngine{}

	if err := e.validateTarget("linux", "amd64", ""); err != nil {
		t.Errorf("expected linux/amd64 to be valid, got: %v", err)
	}

	if err := e.validateTarget("windows", "amd64", ""); err != nil {
		t.Errorf("expected windows/amd64 to be valid, got: %v", err)
	}

	if err := e.validateTarget("darwin", "arm64", ""); err != nil {
		t.Errorf("expected darwin/arm64 to be valid, got: %v", err)
	}

	if err := e.validateTarget("linux", "amd64", "gnu"); err == nil {
		t.Error("expected error for Go with ABI specified")
	}

	if err := e.validateTarget("invalid_os", "amd64", ""); err == nil {
		t.Error("expected error for invalid OS")
	}

	if err := e.validateTarget("linux", "invalid_arch", ""); err == nil {
		t.Error("expected error for invalid arch")
	}
}

func TestValidateTargetDarwinArm(t *testing.T) {
	e := &GoEngine{}

	if err := e.validateTarget("darwin", "arm", ""); err == nil {
		t.Error("expected error for darwin/arm (should be arm64)")
	}
}

func TestGetBestMatchForTarget(t *testing.T) {
	e := &GoEngine{}

	art := &config.ArtifactConfig{
		Targets: map[string]config.TargetConfig{
			"linux-amd64": {OS: "linux", Archs: []string{"amd64"}},
			"linux-arm64": {OS: "linux", Archs: []string{"arm64"}},
		},
	}

	match := e.getBestMatch(art, "linux", "amd64", "")
	if match == nil {
		t.Fatal("expected match for linux/amd64")
	}

	match = e.getBestMatch(art, "linux", "arm64", "")
	if match == nil {
		t.Fatal("expected match for linux/arm64")
	}

	match = e.getBestMatch(art, "windows", "amd64", "")
	if match != nil {
		t.Error("expected no match for windows/amd64")
	}
}

func TestGetExtAndPrefixForOS(t *testing.T) {
	e := &GoEngine{}

	if ext, _ := e.getExtAndPrefix("linux", "bin"); ext != "" {
		t.Errorf("expected empty ext for linux bin, got '%s'", ext)
	}

	if ext, _ := e.getExtAndPrefix("windows", "bin"); ext != "exe" {
		t.Errorf("expected 'exe' for windows bin, got '%s'", ext)
	}

	if ext, _ := e.getExtAndPrefix("darwin", "bin"); ext != "" {
		t.Errorf("expected empty ext for darwin bin, got '%s'", ext)
	}

	if ext, prefix := e.getExtAndPrefix("linux", "lib"); ext != "so" {
		t.Errorf("expected 'so' for linux lib, got '%s'", ext)
	} else if prefix != "lib" {
		t.Errorf("expected 'lib' prefix for lib, got '%s'", prefix)
	}
}

func TestBuildWithGoTags(t *testing.T) {
	if _, err := os.Stat("/usr/local/go/bin/go"); os.IsNotExist(err) {
		if _, err := os.Stat("/usr/bin/go"); os.IsNotExist(err) {
			t.Skip("go not installed")
		}
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

	goModContent := `module test-tags

go 1.21
`
	if err := os.WriteFile("go.mod", []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	mainGo := `package main

import "fmt"

func main() {
	fmt.Println("Hello with tags!")
}
`
	if err := os.WriteFile("main.go", []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll("dist", 0755); err != nil {
		t.Fatal(err)
	}

	e := &GoEngine{}
	cfg := &config.Config{
		Project:   config.Project{Name: "test-tags", Lang: "go"},
		OutputDir: "dist",
		Artifacts: map[string]*config.ArtifactConfig{
			"test-tags": {
				Type:   "bin",
				Source: ".",
				Targets: map[string]config.TargetConfig{
					"linux-x64": {
						OS:    "linux",
						Archs: []string{"amd64"},
						LangOpts: map[string]any{
							"tags": "netgo",
						},
					},
				},
			},
		},
	}

	if err := e.Build(cfg, cfg.Artifacts["test-tags"], engine.BuildOptions{
		ArtifactName: "test-tags",
		OS:           "linux",
		Arch:         "amd64",
	}); err != nil {
		t.Errorf("Build with tags failed: %v", err)
	}
}
