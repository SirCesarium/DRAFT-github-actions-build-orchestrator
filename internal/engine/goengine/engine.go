// Package goengine implements the BuildEngine interface for Go projects.
package goengine

import (
	"fmt"
	"os/exec"

	"github.com/SirCesarium/refinery/internal/config"
	"github.com/SirCesarium/refinery/internal/engine"
	"github.com/SirCesarium/refinery/internal/ui"
)

type GoEngine struct{}

func (e *GoEngine) ID() string {
	return "go"
}

// Prepare verifies that the Go toolchain is available.
func (e *GoEngine) Prepare(_ *config.Config) error {
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("go not found in PATH")
	}
	return nil
}

// Validate checks if refinery artifacts match go.mod definitions
// and validates target configurations.
func (e *GoEngine) Validate(cfg *config.Config) error {
	manifest, err := e.loadManifest()
	if err != nil {
		return err
	}

	for name, art := range cfg.Artifacts {
		if art.Type == "lib" {
			ui.Warn("Go libraries are typically distributed as source or via pkg.go.dev, not as binary artifacts")
		}

		if art.Source == "" {
			return fmt.Errorf("artifact %s requires source (module path)", name)
		}

		for tID, tCfg := range art.Targets {
			for _, arch := range tCfg.Archs {
				abis := tCfg.ABIs
				if len(abis) == 0 {
					abis = []string{""}
				}
				for _, abi := range abis {
					if err := e.validateTarget(tCfg.OS, arch, abi); err != nil {
						return fmt.Errorf("invalid target config '%s' in artifact '%s': %w", tID, name, err)
					}
				}
			}
		}
	}
	_ = manifest
	return nil
}

// Build compiles the Go project for the specified target.
func (e *GoEngine) Build(cfg *config.Config, art *config.ArtifactConfig, opts engine.BuildOptions) error {
	return e.build(cfg, art, opts)
}

// Package creates distribution packages for the built artifact.
func (e *GoEngine) Package(cfg *config.Config, art *config.ArtifactConfig, opts engine.BuildOptions, format string) error {
	return e.pkg(cfg, art, opts.ArtifactName, opts.OS, opts.Arch, opts.ABI, opts.Version, format)
}

// GetCIRequirements returns necessary tools for CI based on config.
func (e *GoEngine) GetCIRequirements(cfg *config.Config) []string {
	reqs := []string{"go"}
	for _, art := range cfg.Artifacts {
		for _, pkg := range art.Packages {
			switch pkg {
			case "deb", "rpm":
				reqs = append(reqs, "pkg:go-bin-tools")
			}
		}
	}
	return reqs
}

// GetSupportedArchs returns supported architectures for a given OS in Go.
func (e *GoEngine) GetSupportedArchs(os string) []string {
	switch os {
	case "linux":
		return []string{"amd64", "386", "arm64", "arm", "ppc64", "ppc64le", "mips", "mipsle", "mips64", "mips64le", "s390x", "sparc64", "riscv64"}
	case "windows":
		return []string{"amd64", "386", "arm64", "arm"}
	case "darwin":
		return []string{"amd64", "arm64"}
	case "wasi":
		return []string{"wasm"}
	default:
		return []string{}
	}
}

// addPackageRequirements adds CI requirements based on package formats.
func (e *GoEngine) addPackageRequirements(reqs []string, art *config.ArtifactConfig) []string {
	for _, p := range e.uniqueFormats(art.Packages) {
		switch p {
		case "deb", "rpm":
			reqs = append(reqs, "pkg:go-bin-tools")
		}
	}
	return reqs
}

func (e *GoEngine) uniqueFormats(values []string) []string {
	seen := map[string]bool{}
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	return unique
}

// validateTarget checks if the OS/arch/ABI combination is valid for Go.
func (e *GoEngine) validateTarget(osName, arch, abi string) error {
	if abi != "" {
		return fmt.Errorf("go does not use ABI specification (got: %s)", abi)
	}

	validGOOS := map[string]bool{
		"linux": true, "windows": true, "darwin": true,
		"freebsd": true, "openbsd": true, "netbsd": true,
		"dragonfly": true, "solaris": true, "aix": true,
		"android": true, "illumos": true, "js": true, "wasi": true,
	}
	validGOARCH := map[string]bool{
		"386": true, "amd64": true, "arm": true, "arm64": true,
		"ppc64": true, "ppc64le": true, "mips": true, "mipsle": true,
		"mips64": true, "mips64le": true, "s390x": true, "sparc64": true,
		"wasm": true, "riscv64": true,
	}

	if !validGOOS[osName] {
		return fmt.Errorf("unsupported GOOS: %s", osName)
	}
	if !validGOARCH[arch] {
		return fmt.Errorf("unsupported GOARCH: %s", arch)
	}

	if osName == "darwin" && arch == "arm" {
		return fmt.Errorf("use 'arm64' instead of 'arm' for Darwin targets")
	}

	return nil
}
