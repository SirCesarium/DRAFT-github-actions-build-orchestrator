package goengine

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/SirCesarium/refinery/internal/config"
	"github.com/SirCesarium/refinery/internal/engine"
)

// build orchestrates the full build process: setup environment, compile, and move artifacts.
func (e *GoEngine) build(cfg *config.Config, art *config.ArtifactConfig, opts engine.BuildOptions) error {
	bestMatch := e.getBestMatch(art, opts.OS, opts.Arch, opts.ABI)
	if bestMatch == nil {
		return fmt.Errorf("no matching target found for %s-%s-%s", opts.OS, opts.Arch, opts.ABI)
	}

	binaryName, binaryPath := e.resolveBinaryInfo(cfg, art, opts, opts.Version)
	_ = binaryPath

	if err := e.runHooks(art, opts, binaryName, "PreBuild"); err != nil {
		return err
	}

	if err := e.runGoBuild(cfg, art, opts); err != nil {
		return err
	}

	if err := e.moveArtifacts(cfg, art, opts); err != nil {
		return err
	}

	return e.runHooks(art, opts, binaryName, "PostBuild")
}

// resolveBinaryInfo returns the binary name and full path based on naming config.
func (e *GoEngine) resolveBinaryInfo(cfg *config.Config, art *config.ArtifactConfig, opts engine.BuildOptions, version string) (string, string) {
	ext, _ := e.getExtAndPrefix(opts.OS, art.Type)
	binaryName := cfg.Naming.Resolve(cfg.Naming.Binary, opts.ArtifactName, opts.OS, opts.Arch, version, opts.ABI, ext)
	return binaryName, filepath.Join(cfg.OutputDir, binaryName)
}

// runGoBuild executes 'go build' with the appropriate GOOS and GOARCH environment variables.
func (e *GoEngine) runGoBuild(cfg *config.Config, art *config.ArtifactConfig, opts engine.BuildOptions) error {
	outputName, _ := e.resolveBinaryInfo(cfg, art, opts, opts.Version)

	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return err
	}

	outputPath := filepath.Join(cfg.OutputDir, outputName)

	args := []string{"build"}

	if tags := e.getBuildTags(e.getBestMatch(art, opts.OS, opts.Arch, opts.ABI)); tags != "" {
		args = append(args, "-tags", tags)
	}

	if ldflags := e.getLdFlags(e.getBestMatch(art, opts.OS, opts.Arch, opts.ABI)); ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}

	args = append(args, "-o", outputPath, art.Source)

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	cmd.Env = append(cmd.Env, fmt.Sprintf("GOOS=%s", opts.OS))
	cmd.Env = append(cmd.Env, fmt.Sprintf("GOARCH=%s", opts.Arch))

	if opts.OS == "windows" && opts.Arch == "arm64" {
		cmd.Env = append(cmd.Env, "GOARM=7")
	}

	return cmd.Run()
}

// runHooks executes either pre-build or post-build hooks.
func (e *GoEngine) runHooks(art *config.ArtifactConfig, opts engine.BuildOptions, binaryPath string, hookType string) error {
	resolvedHooks := art.Hooks.ResolveAll(opts.ArtifactName, opts.OS, opts.Arch, opts.Version, opts.ABI, binaryPath)

	var hooks []string
	if hookType == "PreBuild" {
		hooks = resolvedHooks.PreBuild
	} else {
		hooks = resolvedHooks.PostBuild
	}

	for _, hook := range hooks {
		if err := e.runHook(hook); err != nil {
			return fmt.Errorf("%s hook failed: %w", strings.ToLower(hookType), err)
		}
	}
	return nil
}

// runHook executes a shell command hook.
func (e *GoEngine) runHook(hook string) error {
	parts := strings.Fields(hook)
	if len(parts) == 0 {
		return nil
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// moveArtifacts copies built files from the output directory.
func (e *GoEngine) moveArtifacts(cfg *config.Config, art *config.ArtifactConfig, _ engine.BuildOptions) error {
	if art.Headers {
		headers, err := filepath.Glob("*.go")
		if err != nil {
			return fmt.Errorf("failed to search for .go headers: %w", err)
		}

		for _, h := range headers {
			dest := filepath.Join(cfg.OutputDir, h)
			if err := copyFile(h, dest); err != nil {
				return fmt.Errorf("failed to copy header %s to %s: %w", h, dest, err)
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	stat, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, stat.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
