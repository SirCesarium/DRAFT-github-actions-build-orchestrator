package rust

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/SirCesarium/refinery/internal/config"
	"github.com/SirCesarium/refinery/internal/engine"
	"github.com/SirCesarium/refinery/internal/ui"
)

// getValidPackages returns packages valid for the given OS
func getValidPackages(packages []string, osName string) []string {
	osPackages := map[string][]string{
		"linux":   {"tar.gz", "deb", "rpm"},
		"windows": {"zip", "msi"},
		"darwin":  {"tar.gz"},
	}

	valid := osPackages[osName]
	if len(valid) == 0 {
		return nil
	}

	// Intersect requested packages with valid ones
	var result []string
	for _, p := range packages {
		for _, v := range valid {
			if p == v {
				result = append(result, p)
				break
			}
		}
	}
	return result
}

// build orchestrates the full build process: setup, compile, and move artifacts.
func (e *RustEngine) build(cfg *config.Config, art *config.ArtifactConfig, opts engine.BuildOptions) error {
	bestMatch := e.getBestMatch(art, opts.OS, opts.Arch, opts.ABI)
	if bestMatch == nil {
		return fmt.Errorf("no matching target found for %s-%s-%s", opts.OS, opts.Arch, opts.ABI)
	}

	targetTriple := e.resolveTarget(*bestMatch, opts.Arch, opts.ABI)
	manifest, err := e.loadManifest()
	if err != nil {
		return err
	}

	profile := e.getProfile(*bestMatch)
	if err := e.setupEnvironment(art, opts.OS, opts.Arch, opts.ABI, targetTriple); err != nil {
		return err
	}

	version := opts.Version
	if version == "" || version == "0.0.0" {
		version = manifest.Package.Version
	}

	binaryName, _ := e.resolveBinaryInfo(cfg, art, opts, manifest, targetTriple, profile, version)
	if err := e.runHooks(art, opts, binaryName, "PreBuild"); err != nil {
		return err
	}

	if err := e.addTarget(targetTriple); err != nil {
		return err
	}

	if err := e.runCargoBuild(art, opts.ArtifactName, opts.OS, opts.Arch, opts.ABI, targetTriple, profile); err != nil {
		return err
	}

	// Filter packages by OS
	validPackages := getValidPackages(art.Packages, opts.OS)

	for _, format := range validPackages {
		if err := e.pkg(cfg, art, opts.ArtifactName, opts.OS, opts.Arch, opts.ABI, version, format); err != nil {
			ui.Warn("Failed to create package %s: %v", format, err)
		}
	}

	return e.runHooks(art, opts, binaryName, "PostBuild")
}

// getProfile extracts the build profile from target config.
func (e *RustEngine) getProfile(tCfg config.TargetConfig) string {
	profile := "release"
	if p, ok := tCfg.LangOpts["profile"].(string); ok {
		profile = p
	}
	return profile
}

// resolveBinaryInfo returns the binary name and full path based on naming config.
func (e *RustEngine) resolveBinaryInfo(cfg *config.Config, art *config.ArtifactConfig, opts engine.BuildOptions, _ *cargoManifest, _, _, version string) (string, string) {
	ext := e.getBinaryExt(art, opts.OS, opts.ABI)
	binaryName := cfg.Naming.Resolve(cfg.Naming.Binary, opts.ArtifactName, opts.OS, opts.Arch, version, opts.ABI, ext)
	return binaryName, filepath.Join(cfg.OutputDir, binaryName)
}

// getBinaryExt determines the file extension for the binary based on OS and type.
func (e *RustEngine) getBinaryExt(art *config.ArtifactConfig, osName, abi string) string {
	ext := ""
	if art.Type == "bin" {
		ext, _ = e.getExtAndPrefix(osName, abi, art.Type, "bin")
	} else if len(art.LibraryTypes) > 0 {
		ext, _ = e.getExtAndPrefix(osName, abi, art.Type, art.LibraryTypes[0])
	} else {
		ext, _ = e.getExtAndPrefix(osName, abi, art.Type, "cdylib")
	}
	return ext
}

// runHooks executes either pre-build or post-build hooks.
func (e *RustEngine) runHooks(art *config.ArtifactConfig, opts engine.BuildOptions, binaryPath, hookType string) error {
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
func (e *RustEngine) runHook(hook string) error {
	parts := strings.Fields(hook)
	if len(parts) == 0 {
		return nil
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// setupEnvironment sets up environment variables for the build.
func (e *RustEngine) setupEnvironment(art *config.ArtifactConfig, osName, arch, abi, target string) error {
	if err := e.setupMacOSDeployment(); err != nil {
		return err
	}

	if err := e.setupLinker(art, osName, arch, target); err != nil {
		return err
	}

	// For musl targets, set CC_<target> to use the correct compiler
	if abi == "musl" {
		ccVar := fmt.Sprintf("CC_%s", strings.ReplaceAll(strings.ToUpper(target), "-", "_"))
		if os.Getenv(ccVar) == "" {
			// Try to find an appropriate compiler
			compiler := "musl-gcc"
			if arch == "aarch64" {
				// On aarch64, we might be able to use gcc directly or need a specific cross-compiler
				// Check if we're running on aarch64
				if runtime.GOARCH == "arm64" {
					compiler = "gcc"
				} else {
					// For cross-compilation, use the aarch64-linux-gnu-gcc with proper musl support
					// or fallback to aarch64-linux-musl-gcc if available
					compiler = "aarch64-linux-gnu-gcc"
				}
			}
			if err := os.Setenv(ccVar, compiler); err != nil {
				return fmt.Errorf("failed to set %s: %w", ccVar, err)
			}
		}
	}

	return nil
}

// setupLinker configures the linker for the target if needed.
func (e *RustEngine) setupLinker(art *config.ArtifactConfig, osName, arch, target string) error {
	linker := e.getLinkerFromConfig(art, osName, arch)
	if linker == "" {
		return nil
	}

	isArmLinker := strings.Contains(linker, "aarch64") || strings.Contains(linker, "arm")
	isArmTarget := strings.Contains(arch, "aarch64") || strings.Contains(arch, "arm")
	isX64Linker := strings.Contains(linker, "x86_64") || strings.Contains(linker, "x64")
	isX64Target := strings.Contains(arch, "x86_64") || strings.Contains(arch, "x64")

	shouldApply := (!isArmLinker || isArmTarget) && (!isX64Linker || isX64Target)

	if shouldApply {
		envKey := fmt.Sprintf("CARGO_TARGET_%s_LINKER",
			strings.ReplaceAll(strings.ReplaceAll(strings.ToUpper(target), "-", "_"), ".", "_"))
		if err := os.Setenv(envKey, linker); err != nil {
			return fmt.Errorf("failed to set linker env %s: %w", envKey, err)
		}
	}
	return nil
}

// getLinkerFromConfig returns the linker specified in target config.
func (e *RustEngine) getLinkerFromConfig(art *config.ArtifactConfig, osName, arch string) string {
	if art == nil {
		return ""
	}

	for _, tCfg := range art.Targets {
		if tCfg.OS == osName {
			for _, a := range tCfg.Archs {
				if a == arch {
					if linker, ok := tCfg.LangOpts["linker"].(string); ok {
						return linker
					}
				}
			}
		}
	}
	return ""
}

// setupMacOSDeployment sets a default deployment target for macOS.
func (e *RustEngine) setupMacOSDeployment() error {
	if runtime.GOOS == "darwin" && os.Getenv("MACOSX_DEPLOYMENT_TARGET") == "" {
		ui.Warn("MACOSX_DEPLOYMENT_TARGET not set, defaulting to 10.7")
		if err := os.Setenv("MACOSX_DEPLOYMENT_TARGET", "10.7"); err != nil {
			return fmt.Errorf("failed to set MACOSX_DEPLOYMENT_TARGET: %w", err)
		}
	}
	return nil
}

// addTarget adds the compilation target to rustup.
func (e *RustEngine) addTarget(target string) error {
	cmd := exec.Command("rustup", "target", "add", target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		ui.Warn("Failed to add target %s: %v (may already exist)", target, err)
	}
	return nil
}

// runCargoBuild executes 'cargo build' with the appropriate flags.
func (e *RustEngine) runCargoBuild(art *config.ArtifactConfig, artifactName, _, _, _, target, profile string) error {
	args := []string{"build", "--target", target}

	if profile != "debug" && profile != "dev" {
		args = append(args, "--release")
	}

	// Use -p to specify the package (from Cargo.toml [package] name)
	// The source field should contain the Cargo package name
	pkgName := art.Source
	if pkgName == "" {
		pkgName = artifactName
	}
	args = append(args, "-p", pkgName)

	// Specify what to build: --bin for binaries, --lib for libraries
	switch art.Type {
	case "bin":
		// For bin artifacts, we need to specify which binary to build
		// The artifact name typically matches the binary name in [[bin]] section
		args = append(args, "--bin", artifactName)
	case "lib":
		args = append(args, "--lib")
	}

	cmd := exec.Command("cargo", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// moveArtifacts copies built files from cargo target dir to the output directory.
func (e *RustEngine) moveArtifacts(cfg *config.Config, art *config.ArtifactConfig, artifactName, osName, arch, abi, target, version, profile string, _ *cargoManifest) error {
	var buildTypes []string
	if art.Type == "bin" {
		buildTypes = []string{"bin"}
	} else {
		buildTypes = art.LibraryTypes
		if len(buildTypes) == 0 {
			buildTypes = []string{"cdylib"}
		}
	}

	movedCount := 0
	cargoProfileDir := profile
	if profile == "debug" || profile == "dev" {
		cargoProfileDir = "debug"
	}

	// Get the actual binary/lib name from Cargo.toml
	// For bin artifacts, use bin_name field (or artifact name as fallback)
	// For lib artifacts, use lib_name field (or source field as fallback)
	cargoName := artifactName
	if art.Type == "bin" && art.BinName != "" {
		cargoName = art.BinName
	} else if art.Type == "lib" && art.LibName != "" {
		cargoName = art.LibName
	}

	for _, bt := range buildTypes {
		ext, prefix := e.getExtAndPrefix(osName, abi, art.Type, bt)
		finalName := cfg.Naming.Resolve(cfg.Naming.Binary, artifactName, osName, arch, version, abi, ext)

		// Cargo outputs files with the actual crate name, not the refinery naming scheme
		// Binary: target/{target}/{profile}/{name} (with .exe for Windows)
		// Library: target/{target}/{profile}/lib{name}.{ext}
		realSrcName := cargoName
		if prefix != "" && !strings.HasPrefix(realSrcName, prefix) {
			realSrcName = prefix + realSrcName
		}
		if ext != "" {
			realSrcName += "." + ext
		}

		srcPath := filepath.Join("target", target, cargoProfileDir, realSrcName)

		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			ui.Warn("Expected artifact not found at %s (build type: %s). Skipping...", srcPath, bt)
			continue
		}

		if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
			return err
		}

		destPath := filepath.Join(cfg.OutputDir, finalName)
		if err := moveFile(srcPath, destPath); err != nil {
			return err
		}
		movedCount++
	}

	if art.Headers {
		headers, err := filepath.Glob("*.h")
		if err != nil {
			return fmt.Errorf("failed to search for .h headers: %w", err)
		}

		headers2, err := filepath.Glob("*.hpp")
		if err != nil {
			return fmt.Errorf("failed to search for .hpp headers: %w", err)
		}

		headers = append(headers, headers2...)

		for _, h := range headers {
			dest := filepath.Join(cfg.OutputDir, h)
			if err := copyFile(h, dest); err != nil {
				return fmt.Errorf("failed to copy header %s to %s: %w", h, dest, err)
			}
		}
	}

	if movedCount == 0 {
		return fmt.Errorf("no artifacts found for %s in target %s (searched for build types: %v)", artifactName, target, buildTypes)
	}
	return nil
}

func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	return copyFile(src, dst)
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
