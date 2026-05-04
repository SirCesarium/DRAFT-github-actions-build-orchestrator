package goengine

import (
	"fmt"
	"os"
	"strings"
)

type goModManifest struct {
	Module struct {
		Path string `toml:"path"`
	} `toml:"module"`
	Go        string `toml:"go"`
	Toolchain string `toml:"toolchain"`
}

// loadManifest reads and parses the go.mod file into a struct.
func (e *GoEngine) loadManifest() (*goModManifest, error) {
	goModBytes, err := os.ReadFile("go.mod")
	if err != nil {
		return nil, fmt.Errorf("could not read go.mod: %w", err)
	}

	content := string(goModBytes)
	manifest := &goModManifest{}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			manifest.Module.Path = strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
		if strings.HasPrefix(line, "go ") {
			manifest.Go = strings.TrimSpace(strings.TrimPrefix(line, "go"))
		}
		if strings.HasPrefix(line, "toolchain ") {
			manifest.Toolchain = strings.TrimSpace(strings.TrimPrefix(line, "toolchain"))
		}
	}

	return manifest, nil
}
