package rust

import (
	"fmt"
	"slices"

	"github.com/SirCesarium/refinery/internal/config"
	"github.com/SirCesarium/refinery/internal/ui"
)

// resolveTarget converts OS/arch/ABI to a Rust target triple.
func (e *RustEngine) resolveTarget(opts config.TargetConfig, arch, abi string) string {
	switch opts.OS {
	case "darwin":
		if arch == "arm64" {
			arch = "aarch64"
		}
		return fmt.Sprintf("%s-apple-darwin", arch)
	case "windows":
		if abi == "" {
			abi = "msvc"
		}
		return fmt.Sprintf("%s-pc-windows-%s", arch, abi)
	case "wasm":
		return "wasm32-unknown-unknown"
	case "wasi":
		return "wasm32-wasip1"
	default:
		triple := fmt.Sprintf("%s-unknown-%s", arch, opts.OS)
		if abi != "" {
			triple = fmt.Sprintf("%s-%s", triple, abi)
		}
		return triple
	}
}

// getBestMatch finds the target config matching OS, arch, and ABI.
func (e *RustEngine) getBestMatch(art *config.ArtifactConfig, osName, arch, abi string) *config.TargetConfig {
	var bestMatch *config.TargetConfig
	for _, tCfg := range art.Targets {
		if tCfg.OS == osName && e.sliceContains(tCfg.Archs, arch) {
			if abi != "" && e.sliceContains(tCfg.ABIs, abi) {
				targetCopy := tCfg
				bestMatch = &targetCopy
				break
			}
			if abi == "" || (len(tCfg.ABIs) == 1 && tCfg.ABIs[0] == "") || len(tCfg.ABIs) == 0 {
				targetCopy := tCfg
				bestMatch = &targetCopy
				if abi == "" {
					break
				}
			}
		}
	}
	return bestMatch
}

// getExtAndPrefix returns file extension and prefix based on OS and artifact type.
func (e *RustEngine) getExtAndPrefix(osName, abi, artType, format string) (string, string) {
	var ext, prefix string
	if artType == "lib" {
		prefix = "lib"
		switch osName {
		case "windows":
			if abi == "msvc" || abi == "" {
				prefix = ""
			} else {
				prefix = "lib"
			}
			if format == "staticlib" {
				if abi == "gnu" {
					ext = "a"
				} else {
					ext = "lib"
				}
			} else {
				ext = "dll"
			}
		case "darwin":
			if format == "staticlib" {
				ext = "a"
			} else {
				ext = "dylib"
			}
		case "wasm", "wasi":
			if format == "staticlib" {
				prefix = "lib"
				ext = "a"
			} else {
				prefix = ""
				ext = "wasm"
			}
		default:
			if format == "staticlib" {
				ext = "a"
			} else {
				ext = "so"
			}
		}
	} else {
		switch osName {
		case "windows":
			ext = "exe"
		case "wasm", "wasi":
			ext = "wasm"
		}
	}
	return ext, prefix
}

func (e *RustEngine) sliceContains(slice []string, val string) bool {
	return slices.Contains(slice, val)
}

func (e *RustEngine) validateTriple(osName, arch, abi string) error {
	switch osName {
	case "darwin":
		if arch != "x86_64" && arch != "aarch64" {
			return fmt.Errorf("unsupported architecture for macOS: %s", arch)
		}
	case "windows":
		if arch == "aarch64" && abi == "gnu" {
			return fmt.Errorf("windows-aarch64-gnu is a Tier 3 target (use msvc)")
		}
		if arch != "x86_64" && arch != "i686" && arch != "aarch64" {
			return fmt.Errorf("unsupported architecture for Windows: %s", arch)
		}
	case "wasm", "wasi":
		if arch != "wasm32" {
			return fmt.Errorf("unsupported architecture for web: %s", arch)
		}
	case "linux":
		// Skip musl on aarch64 - uncommon and requires special cross-compiler setup
		if arch == "aarch64" && abi == "musl" {
			ui.Warn("skipping linux-aarch64-musl: musl on aarch64 is uncommon and not supported")
			return nil
		}
	}
	return nil
}
