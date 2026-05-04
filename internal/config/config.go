// Package config handles loading and validation of Refinery's TOML configuration.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/viper"
)

// Config is the main configuration structure for Refinery.
// It holds project settings, artifacts, build steps, and naming conventions.
type Config struct {
	RefineryVersion string                     `toml:"refinery_version" mapstructure:"refinery_version"`
	Project         Project                    `toml:"project" mapstructure:"project"`
	OutputDir       string                     `toml:"output_dir" mapstructure:"output_dir"`
	PreBuild        []BuildStep                `toml:"pre_build,omitempty" mapstructure:"pre_build"`
	PostBuild       []BuildStep                `toml:"post_build,omitempty" mapstructure:"post_build"`
	BuildRefinery   *BuildRefineryConfig       `toml:"build_refinery,omitempty" mapstructure:"build_refinery"`
	Artifacts       map[string]*ArtifactConfig `toml:"artifacts" mapstructure:"artifacts"`
	Naming          NamingConfig               `toml:"naming,omitempty" mapstructure:"naming"`
}

// BuildRefineryConfig defines how to build refinery itself.
type BuildRefineryConfig struct {
	Enabled bool   `toml:"enabled" mapstructure:"enabled"`
	Source  string `toml:"source,omitempty" mapstructure:"source"`
	Version string `toml:"version,omitempty" mapstructure:"version"`
}

type BuildStep struct {
	ID      string         `toml:"id,omitempty" mapstructure:"id"`
	Command []string       `toml:"command,omitempty" mapstructure:"command"`
	Action  string         `toml:"action,omitempty" mapstructure:"action"`
	OS      []string       `toml:"os,omitempty" mapstructure:"os"`
	With    map[string]any `toml:"with,omitempty" mapstructure:"with"`
	Once    bool           `toml:"once,omitempty" mapstructure:"once"`
}

// GetCIRequirements returns CI requirements from pre/post build steps.
func (c *Config) GetCIRequirements() []string {
	reqs := []string{}
	for _, step := range c.PreBuild {
		if step.Action != "" {
			reqs = append(reqs, "action:"+step.Action)
		}
	}
	for _, step := range c.PostBuild {
		if step.Action != "" {
			reqs = append(reqs, "action:"+step.Action)
		}
	}
	return reqs
}

type Project struct {
	Name string `toml:"name" mapstructure:"name"`
	Lang string `toml:"lang" mapstructure:"lang"`
}

type ArtifactConfig struct {
	Type         string                  `toml:"type" mapstructure:"type"`
	Source       string                  `toml:"source" mapstructure:"source"`
	LibraryTypes []string                `toml:"library_types,omitempty" mapstructure:"library_types"`
	Packages     []string                `toml:"packages,omitempty" mapstructure:"packages"`
	Headers      bool                    `toml:"headers,omitempty" mapstructure:"headers"`
	Targets      map[string]TargetConfig `toml:"targets" mapstructure:"targets"`
	Hooks        Hooks                   `toml:"hooks,omitempty" mapstructure:"hooks"`
}

type TargetConfig struct {
	OS       string         `toml:"os" mapstructure:"os"`
	Runner   string         `toml:"runner,omitempty" mapstructure:"runner"`
	Archs    []string       `toml:"archs" mapstructure:"archs"`
	ABIs     []string       `toml:"abis,omitempty" mapstructure:"abis"`
	LangOpts map[string]any `toml:"lang_opts,omitempty" mapstructure:"lang_opts"`
}

type Hooks struct {
	PreBuild  []string `toml:"pre_build,omitempty" mapstructure:"pre_build"`
	PostBuild []string `toml:"post_build,omitempty" mapstructure:"post_build"`
}

func (h Hooks) ResolveAll(artifact, osName, arch, version, abi, binaryPath string) Hooks {
	resolve := func(scripts []string) []string {
		resolved := make([]string, len(scripts))
		r := strings.NewReplacer(
			"{artifact}", artifact,
			"{os}", osName,
			"{arch}", arch,
			"{version}", version,
			"{abi}", abi,
			"{binary}", binaryPath,
		)
		for i, s := range scripts {
			resolved[i] = r.Replace(s)
		}
		return resolved
	}

	return Hooks{
		PreBuild:  resolve(h.PreBuild),
		PostBuild: resolve(h.PostBuild),
	}
}

type NamingConfig struct {
	Binary  string `toml:"binary,omitempty" mapstructure:"binary"`
	Package string `toml:"package,omitempty" mapstructure:"package"`
}

// Validate checks required fields and supported formats in the configuration.
func (c *Config) Validate() error {
	if c.RefineryVersion == "" {
		return fmt.Errorf("refinery_version is required")
	}

	if c.Project.Name == "" || c.Project.Lang == "" {
		return fmt.Errorf("project name and lang are required")
	}

	if c.OutputDir == "" {
		c.OutputDir = "dist"
	}

	if c.Naming.Binary == "" || c.Naming.Package == "" {
		return fmt.Errorf("naming binary and package are required")
	}

	if len(c.Artifacts) == 0 {
		return fmt.Errorf("at least one artifact must be defined")
	}

	for name, art := range c.Artifacts {
		if art.Type != "bin" && art.Type != "lib" {
			return fmt.Errorf("artifact %s type must be 'bin' or 'lib'", name)
		}

		if art.Source == "" {
			return fmt.Errorf("artifact %s source is required", name)
		}

		if len(art.Targets) == 0 {
			return fmt.Errorf("artifact %s must have at least one target", name)
		}

		supportedFormats := map[string]bool{
			"deb": true, "rpm": true, "msi": true,
			"tar.gz": true, "targz": true, "zip": true,
		}

		for _, pkg := range art.Packages {
			if !supportedFormats[pkg] {
				return fmt.Errorf("artifact %s has unsupported package format: %s", name, pkg)
			}
		}

		for tID, tCfg := range art.Targets {
			if tCfg.OS == "" {
				return fmt.Errorf("target %s in artifact %s must have an 'os' field", tID, name)
			}
			if len(tCfg.Archs) == 0 {
				return fmt.Errorf("target %s in artifact %s must have at least one arch", tID, name)
			}
		}
	}
	return nil
}

func (n NamingConfig) Resolve(template, artifact, osName, arch, version, abi, ext string) string {
	if template == "" {
		return ""
	}

	formattedAbi := abi
	if formattedAbi != "" && !strings.HasPrefix(formattedAbi, "-") {
		formattedAbi = "-" + formattedAbi
	}

	var formattedExt string
	if ext != "" {
		formattedExt = strings.TrimPrefix(ext, ".")
	}

	r := strings.NewReplacer(
		"{artifact}", artifact,
		"{os}", osName,
		"{arch}", arch,
		"{version}", version,
		"{abi}", formattedAbi,
		"{ext}", formattedExt,
	)

	result := r.Replace(template)
	result = strings.Trim(result, "-")

	if formattedExt != "" {
		if !strings.HasSuffix(result, "."+formattedExt) {
			result = strings.TrimSuffix(result, ".")
			result = result + "." + formattedExt
		}
	}

	return result
}

func Load(path string) (*Config, error) {
	v := viper.New()

	v.SetDefault("output_dir", "dist")
	v.SetDefault("naming.binary", "{artifact}-{os}-{arch}{abi}")
	v.SetDefault("naming.package", "{artifact}-{version}-{os}-{arch}{abi}.{ext}")

	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		}
	}

	cfg := &Config{}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	if cfg.Naming.Binary == "" {
		cfg.Naming.Binary = v.GetString("naming.binary")
	}

	if cfg.Naming.Package == "" {
		cfg.Naming.Package = v.GetString("naming.package")
	}

	if cfg.OutputDir == "" {
		cfg.OutputDir = v.GetString("output_dir")
	}

	for _, art := range cfg.Artifacts {
		for tName, tCfg := range art.Targets {
			if tCfg.OS == "" {
				tCfg.OS = tName
			}
			if len(tCfg.ABIs) == 0 {
				tCfg.ABIs = []string{""}
			}
			art.Targets[tName] = tCfg
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func Default(name string) *Config {
	return &Config{
		RefineryVersion: "latest",
		Project: Project{
			Name: name,
			Lang: "rust",
		},
		OutputDir: "dist",
		Artifacts: make(map[string]*ArtifactConfig),
		Naming: NamingConfig{
			Binary:  "{artifact}-{os}-{arch}{abi}",
			Package: "{artifact}-{version}-{os}-{arch}{abi}.{ext}",
		},
	}
}

func (c *Config) RemoveRedundantFields() {
	c.Project = Project{
		Name: c.Project.Name,
		Lang: c.Project.Lang,
	}
}

func (c *Config) Write(path string) error {
	if len(c.Artifacts) == 0 {
		c.Artifacts = nil
	}

	for _, art := range c.Artifacts {
		if len(art.Targets) == 0 {
			art.Targets = nil
		}
	}

	data, err := toml.Marshal(c)

	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
