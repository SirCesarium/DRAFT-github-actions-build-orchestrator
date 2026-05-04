// Package rust implements the BuildEngine interface for Rust projects.
package rust

import (
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/SirCesarium/refinery/internal/config"
	"github.com/SirCesarium/refinery/internal/engine"
	"github.com/SirCesarium/refinery/internal/ui"
)

type RustEngine struct{}

func (e *RustEngine) ID() string {
	return "rust"
}

// Prepare implements the BuildEngine interface for Rust.
// It checks for required toolchain availability.
func (e *RustEngine) Prepare(_ *config.Config) error {
	if _, err := exec.LookPath("rustup"); err != nil {
		return fmt.Errorf("rustup not found in PATH")
	}
	return nil
}

// Validate checks if refinery artifacts match Cargo.toml definitions.
// It verifies binary and library names, and validates target triples.
func (e *RustEngine) Validate(cfg *config.Config) error {
	manifest, err := e.loadManifest()
	if err != nil {
		return err
	}

	// Check each configured artifact against Cargo.toml.
	for name, art := range cfg.Artifacts {
		if !e.isArtifactDefinedInManifest(name, art.Type, manifest) {
			return fmt.Errorf("artifact mismatch: '%s' (type %s) defined in refinery.toml not found in Cargo.toml", name, art.Type)
		}

		// Validate each target configuration (OS, arch, ABI).
		for tID, tCfg := range art.Targets {
			for _, arch := range tCfg.Archs {
				abis := tCfg.ABIs
				if len(abis) == 0 {
					abis = []string{""}
				}
				for _, abi := range abis {
					if err := e.validateTriple(tCfg.OS, arch, abi); err != nil {
						return fmt.Errorf("invalid target config '%s' in artifact '%s': %w", tID, name, err)
					}
				}
			}
		}
	}
	return nil
}

// isArtifactDefinedInManifest checks if an artifact exists in Cargo.toml.
// For Rust, we check if the package exists and if the binary/lib is defined.
func (e *RustEngine) isArtifactDefinedInManifest(name, artifactType string, manifest *cargoManifest) bool {
	// The package itself must exist
	if manifest.Package.Name == "" {
		return false
	}

	if artifactType == "lib" {
		return e.isLibDefined(name, manifest)
	}
	return e.isBinDefined(name, manifest)
}

// isLibDefined checks if a library artifact is defined in Cargo.toml.
func (e *RustEngine) isLibDefined(name string, manifest *cargoManifest) bool {
	// Check if [lib] section exists
	if manifest.Lib == nil {
		return false
	}
	// The lib name in Cargo.toml should match the artifact name
	// Cargo allows underscores, refinery might use hyphens (we convert)
	libName := manifest.Lib.Name
	if libName == "" {
		libName = manifest.Package.Name
	}
	return strings.ReplaceAll(libName, "-", "_") == strings.ReplaceAll(name, "-", "_")
}

// isBinDefined checks if a binary artifact is defined in Cargo.toml.
func (e *RustEngine) isBinDefined(name string, manifest *cargoManifest) bool {
	// Check if there's a [[bin]] section with matching name
	for _, b := range manifest.Bin {
		if b.Name == name {
			return true
		}
	}
	// If no explicit bin sections, the package name is used as the binary name
	return len(manifest.Bin) == 0 && manifest.Package.Name == name
}

// GetCIRequirements returns necessary tools/packages for CI based on config.
func (e *RustEngine) GetCIRequirements(cfg *config.Config) []string {
	reqs := []string{"rust"}
	for _, art := range cfg.Artifacts {
		for _, pkg := range art.Packages {
			switch pkg {
			case "deb":
				reqs = append(reqs, "pkg:cargo-deb")
			case "rpm":
				reqs = append(reqs, "pkg:cargo-generate-rpm")
			case "msi":
				reqs = append(reqs, "pkg:cargo-wix")
			}
		}
	}
	return reqs
}

// GetSupportedArchs returns supported architectures for a given OS in Rust.
func (e *RustEngine) GetSupportedArchs(os string) []string {
	switch os {
	case "linux":
		return []string{"x86_64", "i686", "aarch64"}
	case "windows":
		return []string{"x86_64", "i686", "aarch64"}
	case "darwin":
		return []string{"x86_64", "aarch64"}
	case "wasi":
		return []string{"wasm32"}
	default:
		return []string{}
	}
}

// addTargetRequirements adds CI requirements based on target configuration.
func (e *RustEngine) addTargetRequirements(reqs []string, art *config.ArtifactConfig) []string {
	for _, tCfg := range art.Targets {
		if tCfg.OS == "linux" && e.sliceContains(tCfg.Archs, "aarch64") {
			reqs = append(reqs, "cross-linker:linux-aarch64")
		}
		if e.sliceContains(tCfg.ABIs, "musl") {
			reqs = append(reqs, "pkg:musl-tools")
		}
	}
	return reqs
}

// addPackageRequirements adds CI requirements based on package formats.
func (e *RustEngine) addPackageRequirements(reqs []string, art *config.ArtifactConfig) []string {
	formats := append([]string{}, art.Packages...)
	for _, p := range e.uniqueFormats(formats) {
		switch p {
		case "deb":
			reqs = append(reqs, "pkg:cargo-deb")
		case "rpm":
			reqs = append(reqs, "pkg:cargo-generate-rpm")
		case "msi":
			reqs = append(reqs, "pkg:cargo-wix")
		}
	}
	return reqs
}

func (e *RustEngine) uniqueFormats(values []string) []string {
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

// Build implements the BuildEngine interface for Rust.
func (e *RustEngine) Build(cfg *config.Config, art *config.ArtifactConfig, opts engine.BuildOptions) error {
	return e.build(cfg, art, opts)
}

// Package implements the BuildEngine interface for Rust.
func (e *RustEngine) Package(cfg *config.Config, art *config.ArtifactConfig, opts engine.BuildOptions, format string) error {
	return e.pkg(cfg, art, opts.ArtifactName, opts.OS, opts.Arch, opts.ABI, opts.Version, format)
}

// ValidateRustSpecific runs Rust-specific validation.
func (e *RustEngine) ValidateRustSpecific(cfg *config.Config) error {
	manifest, err := e.loadManifest()
	if err != nil {
		return err
	}

	for name, art := range cfg.Artifacts {
		if art.Type == "bin" {
			found := false
			for _, b := range manifest.Bin {
				if b.Name == name {
					found = true
					break
				}
			}

			if !found && manifest.Package.Name != name {
				ui.Warn("Artifact '%s' (bin) not found in Cargo.toml bin section", name)
			}
		} else if art.Type == "lib" {
			if manifest.Lib == nil || (manifest.Lib.Name != name && name != strings.ReplaceAll(manifest.Package.Name, "-", "_")) {
				ui.Warn("Artifact '%s' (lib) may not match Cargo.toml lib section", name)
			}
		}

		for tName, tCfg := range art.Targets {
			if tCfg.OS == "darwin" {
				if slices.Contains(tCfg.Archs, "arm64") {
					return fmt.Errorf("target '%s': use 'aarch64' instead of 'arm64' for Rust Darwin targets", tName)
				}
			}
		}
	}
	return nil
}
