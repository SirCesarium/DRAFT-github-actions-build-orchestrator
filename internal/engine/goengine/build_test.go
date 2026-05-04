package goengine

import (
	"testing"

	"github.com/SirCesarium/refinery/internal/config"
	"github.com/SirCesarium/refinery/internal/engine"
)

func TestGetExtAndPrefix(t *testing.T) {
	e := &GoEngine{}

	if ext, _ := e.getExtAndPrefix("linux", "bin"); ext != "" {
		t.Errorf("expected empty string for bin on linux, got '%s'", ext)
	}

	if ext, _ := e.getExtAndPrefix("windows", "bin"); ext != "exe" {
		t.Errorf("expected 'exe' for bin on windows, got '%s'", ext)
	}

	if ext, prefix := e.getExtAndPrefix("linux", "lib"); ext != "so" {
		t.Errorf("expected 'so' for lib on linux, got '%s'", ext)
	} else if prefix != "lib" {
		t.Errorf("expected 'lib' prefix for lib on linux, got '%s'", prefix)
	}

	if ext, _ := e.getExtAndPrefix("darwin", "lib"); ext != "dylib" {
		t.Errorf("expected 'dylib' for lib on darwin, got '%s'", ext)
	}
}

func TestResolveBinaryInfo(t *testing.T) {
	e := &GoEngine{}

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
		Arch:         "amd64",
	}

	binaryName, _ := e.resolveBinaryInfo(cfg, art, opts, "0.0.0")
	if binaryName == "" {
		t.Error("expected non-empty binary name")
	}
}

func TestRunHooks(t *testing.T) {
	e := &GoEngine{}

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
		Arch:         "amd64",
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

func TestGetBuildTags(t *testing.T) {
	e := &GoEngine{}

	tCfg := &config.TargetConfig{}
	if tags := e.getBuildTags(tCfg); tags != "" {
		t.Errorf("expected empty tags, got '%s'", tags)
	}

	tCfg = &config.TargetConfig{
		LangOpts: map[string]any{"tags": "netgo,osusergo"},
	}
	if tags := e.getBuildTags(tCfg); tags != "netgo,osusergo" {
		t.Errorf("expected 'netgo,osusergo', got '%s'", tags)
	}

	tCfg = &config.TargetConfig{
		LangOpts: map[string]any{"tags": []any{"netgo", "osusergo"}},
	}
	tags := e.getBuildTags(tCfg)
	if tags != "netgo,osusergo" {
		t.Errorf("expected 'netgo,osusergo', got '%s'", tags)
	}
}

func TestGetLdFlags(t *testing.T) {
	e := &GoEngine{}

	tCfg := &config.TargetConfig{}
	if flags := e.getLdFlags(tCfg); flags != "" {
		t.Errorf("expected empty ldflags, got '%s'", flags)
	}

	tCfg = &config.TargetConfig{
		LangOpts: map[string]any{"ldflags": "-s -w"},
	}
	if flags := e.getLdFlags(tCfg); flags != "-s -w" {
		t.Errorf("expected '-s -w', got '%s'", flags)
	}
}

func TestGetBestMatch(t *testing.T) {
	e := &GoEngine{}

	art := &config.ArtifactConfig{
		Targets: map[string]config.TargetConfig{
			"linux":   {OS: "linux", Archs: []string{"amd64", "arm64"}},
			"windows": {OS: "windows", Archs: []string{"amd64"}},
		},
	}

	match := e.getBestMatch(art, "linux", "amd64", "")
	if match == nil {
		t.Fatal("expected match for linux/amd64")
	}
	if match.OS != "linux" {
		t.Errorf("expected OS to be 'linux', got '%s'", match.OS)
	}

	match = e.getBestMatch(art, "windows", "amd64", "")
	if match == nil {
		t.Fatal("expected match for windows/amd64")
	}

	match = e.getBestMatch(art, "darwin", "amd64", "")
	if match != nil {
		t.Error("expected no match for darwin/amd64")
	}
}
