package rust

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/SirCesarium/refinery/internal/config"
)

// pkg dispatches packaging to the correct format handler (deb, rpm, msi, etc.).
func (e *RustEngine) pkg(cfg *config.Config, art *config.ArtifactConfig, artifactName, osName, arch, abi, version, format string) error {
	manifest := e.loadManifestOrFatal()
	if (version == "" || version == "0.0.0") && manifest != nil {
		version = manifest.Package.Version
	}

	switch format {
	case "deb", "rpm", "msi":
		return e.handleSystemPackage(cfg, art, artifactName, osName, arch, abi, format)
	case "tar.gz", "targz":
		return e.createTarGz(cfg, art, artifactName, osName, arch, abi, version, manifest)
	case "zip":
		return e.createZip(cfg, art, artifactName, osName, arch, abi, version, manifest)
	default:
		return fmt.Errorf("unsupported package format: %s", format)
	}
}

// handleSystemPackage processes deb, rpm, or msi packages.
func (e *RustEngine) handleSystemPackage(_ *config.Config, art *config.ArtifactConfig, _ string, osName, arch, abi, format string) error {
	if err := e.validateSystemPackage(osName, abi, arch, format); err != nil {
		return err
	}

	bestMatch := e.getBestMatch(art, osName, arch, abi)
	if bestMatch == nil {
		return fmt.Errorf("no matching target found for %s-%s-%s", osName, arch, abi)
	}
	target := e.resolveTarget(*bestMatch, arch, abi)

	command := e.getPackageCommand(format)
	cmd := exec.Command("cargo", append([]string{command}, "--target", target)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// validateSystemPackage checks OS, ABI, arch, and tool availability.
func (e *RustEngine) validateSystemPackage(osName, abi, arch, format string) error {
	if format == "msi" && osName != "windows" {
		return fmt.Errorf("%s packaging is only supported for windows", format)
	}

	if (format == "deb" || format == "rpm") && osName != "linux" {
		return fmt.Errorf("%s packaging is only supported for linux", format)
	}

	if osName != "linux" || abi == "musl" || arch == "wasm32" {
		return fmt.Errorf("%s packaging is only supported for linux (non-musl, non-wasm)", format)
	}

	tool := map[string]string{
		"deb": "cargo-deb",
		"rpm": "cargo-generate-rpm",
		"msi": "candle",
	}[format]

	if tool == "candle" && osName != "windows" {
		return fmt.Errorf("msi packaging is only supported for windows")
	}

	if _, err := exec.LookPath(tool); err != nil {
		return fmt.Errorf("%s not found. Install it with: cargo install %s", tool, tool)
	}
	return nil
}

// getPackageCommand returns the cargo subcommand for a given format.
func (e *RustEngine) getPackageCommand(format string) string {
	switch format {
	case "deb":
		return "deb"
	case "rpm":
		return "generate-rpm"
	case "msi":
		return "wix"
	default:
		return ""
	}
}

// loadManifestOrFatal loads manifest or returns nil (caller should handle error).
func (e *RustEngine) loadManifestOrFatal() *cargoManifest {
	manifest, err := e.loadManifest()
	if err != nil {
		return nil
	}
	return manifest
}

// createTarGz creates a .tar.gz package from built artifacts.
func (e *RustEngine) createTarGz(cfg *config.Config, art *config.ArtifactConfig, artifactName, osName, arch, abi, version string, manifest *cargoManifest) error {
	packageName := cfg.Naming.Resolve(cfg.Naming.Package, artifactName, osName, arch, version, abi, "tar.gz")
	outPath := filepath.Join(cfg.OutputDir, packageName)

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	return e.archiveArtifactFiles(tw, nil, cfg, art, artifactName, osName, arch, abi, version, manifest)
}

// createTarGz creates a .tar.gz package from built artifacts.

func (e *RustEngine) createZip(cfg *config.Config, art *config.ArtifactConfig, artifactName, osName, arch, abi, version string, manifest *cargoManifest) error {
	packageName := cfg.Naming.Resolve(cfg.Naming.Package, artifactName, osName, arch, version, abi, "zip")
	outPath := filepath.Join(cfg.OutputDir, packageName)

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	return e.archiveArtifactFiles(nil, zw, cfg, art, artifactName, osName, arch, abi, version, manifest)
}
