// Package github implements the CIProvider interface for GitHub Actions workflows.
package github

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SirCesarium/refinery/internal/config"
	"github.com/SirCesarium/refinery/internal/engine"
	"gopkg.in/yaml.v3"
)

// OrderedMapAny is a map[string]any that marshals to YAML with sorted keys.
type OrderedMapAny map[string]any

func (m OrderedMapAny) MarshalYAML() (interface{}, error) {
	if m == nil || len(m) == 0 {
		return nil, nil
	}
	node := &yaml.Node{
		Kind: yaml.MappingNode,
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		data, err := yaml.Marshal(m[k])
		if err != nil {
			return nil, err
		}
		var valNode yaml.Node
		if err := yaml.Unmarshal(data, &valNode); err != nil {
			return nil, err
		}
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: k},
			valNode.Content[0],
		)
	}
	return node, nil
}

// OrderedMapString is a map[string]string that marshals to YAML with sorted keys.
type OrderedMapString map[string]string

func (m OrderedMapString) MarshalYAML() (interface{}, error) {
	if m == nil || len(m) == 0 {
		return nil, nil
	}
	node := &yaml.Node{
		Kind: yaml.MappingNode,
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: k},
			&yaml.Node{Kind: yaml.ScalarNode, Value: m[k]},
		)
	}
	return node, nil
}

// Permissions represents GitHub Actions permissions with fixed key order.
type Permissions struct {
	Contents string `yaml:"contents"`
	Packages string `yaml:"packages"`
}

// ReleaseConfig represents the release trigger configuration.
type ReleaseConfig struct {
	Types []string `yaml:"types"`
}

// On represents the workflow trigger configuration.
type On struct {
	Push    *Event         `yaml:"push,omitempty"`
	Release *ReleaseConfig `yaml:"release,omitempty"`
}

// Event represents a push event trigger.
type Event struct {
	Tags []string `yaml:"tags,omitempty"`
}

// MatrixEntry represents a single entry in the matrix include array.
type MatrixEntry struct {
	Artifact string `yaml:"artifact"`
	OS       string `yaml:"os"`
	Arch     string `yaml:"arch"`
	RunsOn   string `yaml:"runs_on"`
	ABI      string `yaml:"abi,omitempty"`
}

// Matrix represents the strategy matrix.
type Matrix struct {
	FailFast bool          `yaml:"fail-fast,omitempty"`
	Include  []MatrixEntry `yaml:"include"`
}

// Strategy defines the build matrix strategy.
type Strategy struct {
	FailFast bool    `yaml:"fail-fast,omitempty"`
	Matrix   *Matrix `yaml:"matrix,omitempty"`
}

// Job defines a single job in the workflow.
type Job struct {
	Name     string    `yaml:"name,omitempty"`
	RunsOn   string    `yaml:"runs-on"`
	Needs    []string  `yaml:"needs,omitempty"`
	If       string    `yaml:"if,omitempty"`
	Strategy *Strategy `yaml:"strategy,omitempty"`
	Steps    []Step    `yaml:"steps"`
}

// Jobs represents the workflow jobs with fixed key order (setup, build, teardown, release).
type Jobs struct {
	Setup    *Job `yaml:"setup,omitempty"`
	Build    Job  `yaml:"build"`
	Teardown *Job `yaml:"teardown,omitempty"`
	Release  Job  `yaml:"release"`
}

// Workflow represents a GitHub Actions workflow YAML structure.
type Workflow struct {
	Name        string      `yaml:"name"`
	On          On          `yaml:"on"`
	Permissions Permissions `yaml:"permissions,omitempty"`
	Jobs        Jobs        `yaml:"jobs"`
}

// Step represents a single step within a job.
type Step struct {
	Name  string           `yaml:"name,omitempty"`
	If    string           `yaml:"if,omitempty"`
	Uses  string           `yaml:"uses,omitempty"`
	With  OrderedMapAny    `yaml:"with,omitempty"`
	Run   string           `yaml:"run,omitempty"`
	Env   OrderedMapString `yaml:"env,omitempty"`
	Shell string           `yaml:"shell,omitempty"`
}

type GithubProvider struct {
	filename string
}

// NewProvider creates a new GitHub Actions workflow provider.
func NewProvider(filename string) (*GithubProvider, error) {
	if filename == "" {
		return nil, fmt.Errorf("workflow filename cannot be empty")
	}
	return &GithubProvider{filename: filename}, nil
}

func (p *GithubProvider) Name() string {
	return "github"
}

func (p *GithubProvider) Filename() string {
	return filepath.Join(".github", "workflows", fmt.Sprintf("%s.yml", p.filename))
}

// Generate creates a GitHub Actions workflow YAML for the given config and engine.
func (p *GithubProvider) Generate(cfg *config.Config, eng engine.BuildEngine) ([]byte, error) {
	if err := eng.Validate(cfg); err != nil {
		return nil, err
	}

	include := p.buildMatrix(cfg)
	setup, build, teardown := p.getSplitSteps(eng, cfg)
	jobs := p.assembleJobs(include, setup, build, teardown)

	wf := Workflow{
		Name: "Refinery Build",
		On: On{
			Push:    &Event{Tags: []string{"v*"}},
			Release: &ReleaseConfig{Types: []string{"created"}},
		},
		Permissions: Permissions{
			Contents: "write",
			Packages: "write",
		},
		Jobs: jobs,
	}

	return yaml.Marshal(wf)
}

func (p *GithubProvider) getSplitSteps(eng engine.BuildEngine, cfg *config.Config) (setup, build, teardown []Step) {
	buildRefinery := cfg.BuildRefinery != nil && cfg.BuildRefinery.Enabled

	// 1. Setup Stage (Global Pre-Build)
	setup = append(setup, Step{Name: "Checkout", Uses: ActionCheckout})
	setup = p.addCIRequirementSteps(setup, eng, cfg)

	if buildRefinery {
		setup = append(setup, Step{
			Name:  "Build Refinery from Source",
			Run:   "go build -o ./refinery-local ./cmd/refinery",
			Shell: "bash",
		})
	}

	for _, step := range cfg.PreBuild {
		if step.Once {
			setup = append(setup, p.createGithubStep(step, "Pre-Build (Global)"))
		}
	}

	// 2. Build Stage (Matrix)
	build = append(build, Step{Name: "Checkout", Uses: ActionCheckout})
	build = p.addCIRequirementSteps(build, eng, cfg)

	if buildRefinery {
		build = append(build, Step{
			Name:  "Build Refinery from Source",
			Run:   "go build -o ./refinery-local ./cmd/refinery",
			Shell: "bash",
		})
	}

	for _, step := range cfg.PreBuild {
		if !step.Once {
			build = append(build, p.createGithubStep(step, "Pre-Build"))
		}
	}

	build = append(build, p.getBuildArtifactStep(cfg)...)

	for _, step := range cfg.PostBuild {
		if !step.Once {
			build = append(build, p.createGithubStep(step, "Post-Build"))
		}
	}

	// 3. Teardown Stage (Global Post-Build)
	hasGlobalPost := false
	for _, s := range cfg.PostBuild {
		if s.Once {
			hasGlobalPost = true
			break
		}
	}

	if hasGlobalPost {
		teardown = append(teardown, Step{Name: "Checkout", Uses: ActionCheckout})
		teardown = append(teardown, Step{
			Name: "Download All Artifacts",
			Uses: ActionDownloadArtifact,
			With: OrderedMapAny{"merge-multiple": true, "path": "./artifacts"},
		})

		for _, step := range cfg.PostBuild {
			if step.Once {
				teardown = append(teardown, p.createGithubStep(step, "Post-Build (Global)"))
			}
		}
	}

	return
}

// buildMatrix creates the matrix include array from config artifacts and targets.
func (p *GithubProvider) buildMatrix(cfg *config.Config) []MatrixEntry {
	var include []MatrixEntry
	uniqueMatrix := make(map[string]bool)

	for _, aName := range p.sortedArtifactNames(cfg) {
		art := cfg.Artifacts[aName]
		// Sort OS keys from the map to ensure deterministic iteration
		osList := make([]string, 0, len(art.Targets))
		for os := range art.Targets {
			osList = append(osList, os)
		}
		sort.Strings(osList)

		for _, osKey := range osList {
			tCfg := art.Targets[osKey]
			runsOn := p.getRunsOn(tCfg.OS)
			// Sort architectures
			archs := make([]string, len(tCfg.Archs))
			copy(archs, tCfg.Archs)
			sort.Strings(archs)
			for _, arch := range archs {
				abis := tCfg.ABIs
				if len(abis) == 0 {
					abis = []string{""}
				}
				// Sort ABIs
				sortedABIs := make([]string, len(abis))
				copy(sortedABIs, abis)
				sort.Strings(sortedABIs)
				for _, abi := range sortedABIs {
					key := fmt.Sprintf("%s-%s-%s-%s", aName, osKey, arch, abi)
					if uniqueMatrix[key] {
						continue
					}
					uniqueMatrix[key] = true
					entry := MatrixEntry{
						Artifact: aName,
						OS:       tCfg.OS,
						Arch:     arch,
						RunsOn:   runsOn,
					}
					if abi != "" {
						entry.ABI = abi
					}
					include = append(include, entry)
				}
			}
		}
	}
	// Sort include slice deterministically by artifact, os, arch, abi
	sort.Slice(include, func(i, j int) bool {
		if include[i].Artifact != include[j].Artifact {
			return include[i].Artifact < include[j].Artifact
		}
		if include[i].OS != include[j].OS {
			return include[i].OS < include[j].OS
		}
		if include[i].Arch != include[j].Arch {
			return include[i].Arch < include[j].Arch
		}
		return include[i].ABI < include[j].ABI
	})
	return include
}

// sortedArtifactNames returns artifact names in sorted order.
func (p *GithubProvider) sortedArtifactNames(cfg *config.Config) []string {
	var names []string
	for name := range cfg.Artifacts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// getRunsOn maps OS to GitHub Actions runner labels.
func (p *GithubProvider) getRunsOn(osName string) string {
	switch osName {
	case "windows":
		return "windows-latest"
	case "darwin":
		return "macos-latest"
	default:
		return "ubuntu-latest"
	}
}

// addCIRequirementSteps adds steps based on engine requirements.
func (p *GithubProvider) addCIRequirementSteps(steps []Step, eng engine.BuildEngine, cfg *config.Config) []Step {
	for _, req := range eng.GetCIRequirements(cfg) {
		switch req {
		case "go":
			steps = append(steps, Step{
				Name: "Setup Go",
				Uses: ActionSetupGo,
				With: OrderedMapAny{"go-version": "1.26.2", "cache": true},
			})
		case "pkg:go-bin-tools":
			steps = append(steps, Step{
				Name:  "Install Go Bin Tools",
				If:    "runner.os == 'Linux'",
				Run:   "go install github.com/mh-cbon/go-bin-deb@latest && go install github.com/mh-cbon/go-bin-rpm@latest",
				Shell: "bash",
			})
		case "rust":
			steps = append(steps, Step{
				Name: "Setup Rust",
				Uses: ActionRustToolchain,
				With: OrderedMapAny{"cache": true},
			})
		case "cross-linker:linux-aarch64":
			steps = append(steps, Step{
				Name:  "Install ARM Linker",
				If:    "runner.os == 'Linux'",
				Run:   "sudo apt-get update && sudo apt-get install -y gcc-aarch64-linux-gnu",
				Shell: "bash",
			})
		case "pkg:musl-tools":
			steps = append(steps, Step{
				Name:  "Install Musl Tools",
				If:    "runner.os == 'Linux'",
				Run:   "sudo apt-get update && sudo apt-get install -y musl-tools",
				Shell: "bash",
			})
		case "pkg:cargo-deb":
			steps = append(steps, Step{
				Name:  "Install cargo-deb",
				If:    "runner.os == 'Linux'",
				Run:   "cargo install cargo-deb",
				Shell: "bash",
			})
		case "pkg:cargo-generate-rpm":
			steps = append(steps, Step{
				Name:  "Install cargo-generate-rpm",
				If:    "runner.os == 'Linux'",
				Run:   "cargo install cargo-generate-rpm",
				Shell: "bash",
			})
		case "pkg:cargo-wix":
			steps = append(steps, Step{
				Name:  "Install cargo-wix",
				If:    "runner.os == 'Windows'",
				Run:   "cargo install cargo-wix",
				Shell: "bash",
			})
		}
	}
	return steps
}

// getBuildArtifactStep returns the artifact build step.
func (p *GithubProvider) getBuildArtifactStep(cfg *config.Config) []Step {
	if cfg.BuildRefinery != nil && cfg.BuildRefinery.Enabled {
		return []Step{{
			Name:  "Build Artifact using Local Refinery",
			Run:   "./refinery-local build --artifact ${{ matrix.artifact }} --os ${{ matrix.os }} --arch ${{ matrix.arch }}${{ matrix.abi != '' && format(' --abi {0}', matrix.abi) || '' }} --version ${{ github.ref_name }}",
			Shell: "bash",
		}}
	}
	return []Step{{
		Name: "Build Artifact",
		Uses: ActionRefinery,
		With: OrderedMapAny{
			"abi":      "${{ matrix.abi }}",
			"arch":     "${{ matrix.arch }}",
			"artifact": "${{ matrix.artifact }}",
			"os":       "${{ matrix.os }}",
			"version":  "${{ github.ref_name }}",
		},
	}}
}

func (p *GithubProvider) createGithubStep(step config.BuildStep, prefix string) Step {
	// Resolve action name to full action path if needed
	action := step.Action
	if action != "" && !strings.Contains(action, "/") && !strings.HasSuffix(action, ".yml") {
		action = fmt.Sprintf("./.github/actions/%s", action)
	}

	// Use action name as ID if not provided
	id := step.ID
	if id == "" && step.Action != "" {
		parts := strings.Split(step.Action, "/")
		name := parts[len(parts)-1]
		name = strings.TrimSuffix(name, ".yml")
		id = name
	}

	ghStep := Step{Name: fmt.Sprintf("%s: %s", prefix, id)}
	if action != "" {
		ghStep.Uses = action
		ghStep.With = step.With
	} else if len(step.Command) > 0 {
		ghStep.Run = strings.Join(step.Command, "\n")
		ghStep.Shell = "bash"
	}
	if len(step.OS) > 0 {
		var conditions []string
		for _, osName := range step.OS {
			switch strings.ToLower(osName) {
			case "linux":
				conditions = append(conditions, "runner.os == 'Linux'")
			case "windows":
				conditions = append(conditions, "runner.os == 'Windows'")
			case "darwin", "macos":
				conditions = append(conditions, "runner.os == 'macOS'")
			}
		}
		if len(conditions) > 0 {
			ghStep.If = strings.Join(conditions, " || ")
		}
	}
	return ghStep
}

// assembleJobs creates the jobs with fixed order for the workflow.
func (p *GithubProvider) assembleJobs(include []MatrixEntry, setup, build, teardown []Step) Jobs {
	jobs := Jobs{}
	hasSetup := len(setup) > 1
	hasTeardown := len(teardown) > 0

	if hasSetup {
		jobs.Setup = &Job{Name: "Setup and Global Checks", RunsOn: "ubuntu-latest", Steps: setup}
	}

	buildJob := Job{
		Name:     "Build ${{ matrix.artifact }} (${{ matrix.os }}-${{ matrix.arch }})",
		RunsOn:   "${{ matrix.runs_on }}",
		Strategy: &Strategy{FailFast: true, Matrix: &Matrix{Include: include}},
		Steps:    build,
	}
	if hasSetup {
		buildJob.Needs = []string{"setup"}
	}
	jobs.Build = buildJob

	if hasTeardown {
		jobs.Teardown = &Job{
			Name:   "Global Post-Build Tasks",
			Needs:  []string{"build"},
			RunsOn: "ubuntu-latest",
			Steps:  teardown,
		}
	}

	releaseNeeds := []string{"build"}
	if hasTeardown {
		releaseNeeds = append(releaseNeeds, "teardown")
	}

	jobs.Release = Job{
		Name:   "Release Artifacts",
		Needs:  releaseNeeds,
		RunsOn: "ubuntu-latest",
		If:     "startsWith(github.ref, 'refs/tags/')",
		Steps: []Step{
			{
				Name: "Download Artifacts",
				Uses: ActionDownloadArtifact,
				With: OrderedMapAny{"merge-multiple": true, "path": "./artifacts"},
			},
			{
				Name:  "List Artifacts",
				Run:   "find ./artifacts -type f | sort",
				Shell: "bash",
			},
			{
				Name: "Publish Release",
				Uses: ActionGHRelease,
				With: OrderedMapAny{"fail_on_unmatched_files": true, "files": "./artifacts/**/*"},
				Env:  OrderedMapString{"GITHUB_TOKEN": "${{ secrets.GITHUB_TOKEN }}"},
			},
		},
	}
	return jobs
}
