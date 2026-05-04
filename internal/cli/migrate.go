package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SirCesarium/refinery/internal/app"
	"github.com/SirCesarium/refinery/internal/config"
	"github.com/SirCesarium/refinery/internal/engine/rust"
	"github.com/SirCesarium/refinery/internal/pipeline"
	"github.com/SirCesarium/refinery/internal/ui"
	"github.com/spf13/cobra"
)

var dryRun bool

var supportedArchCmd = &cobra.Command{
	Use:   "supported-archs [os]",
	Short: "Print supported architectures for an OS using the project engine",
	Args:  cobra.MaximumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		cfg, err := config.Load("refinery.toml")
		if err != nil {
			ui.Fatal(err, "Could not load 'refinery.toml'. Run 'refinery init' first.")
		}

		engineRegistry := app.NewDefaultEngineRegistry()
		eng := engineRegistry.Get(cfg.Project.Lang)
		if eng == nil {
			ui.Fatal(nil, "Unsupported language: "+cfg.Project.Lang)
		}

		osFilter := ""
		if len(args) > 0 {
			osFilter = args[0]
		}

		allOS := []string{"linux", "windows", "darwin", "wasi"}
		sort.Strings(allOS)

		ui.Section("Supported Architectures")
		for _, osName := range allOS {
			if osFilter != "" && osName != osFilter {
				continue
			}
			archs := eng.GetSupportedArchs(osName)
			if len(archs) == 0 {
				continue
			}
			sort.Strings(archs)
			ui.Info("%-*s %s", 12, osName+":", strings.Join(archs, ", "))
		}
	},
}

var migrateCmd = &cobra.Command{
	Use:   "migrate [provider]",
	Short: "Generate CI/CD workflows with validation",
	Args:  cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		ui.Section("Migration")

		providerName := args[0]

		cfg, err := config.Load("refinery.toml")
		if err != nil {
			ui.Fatal(err, "Could not load 'refinery.toml'. Run 'refinery init' first.")
		}

		engineRegistry := app.NewDefaultEngineRegistry()
		eng := engineRegistry.Get(cfg.Project.Lang)
		if eng == nil {
			ui.Fatal(nil, "Unsupported language: "+cfg.Project.Lang)
		}

		if err := eng.Validate(cfg); err != nil {
			ui.Fatal(err, "Configuration validation failed. Fix refinery.toml before migrating.")
		}

		if cfg.Project.Lang == "rust" {
			if err := validateRustEngine(eng, cfg); err != nil {
				ui.Fatal(err, "Rust validation failed. Check Cargo.toml matches refinery.toml")
			}
		}

		providerRegistry, err := app.NewDefaultProviderRegistry()
		if err != nil {
			ui.Fatal(err, "Failed to initialize provider registry.")
		}

		provider := providerRegistry.Get(providerName)
		if provider == nil {
			ui.Fatal(nil, "Unsupported provider: "+providerName+". Supported: github")
		}

		ui.Info("Generating workflow for %s...", providerName)
		gen := pipeline.NewGenerator(provider, eng)
		data, err := gen.Generate(cfg)
		if err != nil {
			ui.Fatal(err, "CI workflow generation failed. Check your artifact configuration.")
		}

		if dryRun {
			ui.Info("Dry run - generated workflow:")
			fmt.Println(string(data))
			ui.Success("Dry run completed. No files were written.")
			return
		}

		outputPath := gen.Filename()
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			ui.Fatal(err, "Failed to create directory structure for: "+outputPath)
		}

		if existing, err := os.ReadFile(outputPath); err == nil && string(existing) == string(data) {
			ui.Info("Workflow unchanged, skipping write.")
		} else {
			if err := os.WriteFile(outputPath, data, 0644); err != nil {
				ui.Fatal(err, "Failed to write workflow file: "+outputPath)
			}
			ui.Success("Workflow generated: %s", outputPath)
		}
		ui.Info("Review the workflow file before pushing to avoid CI failures.")
	},
}

func init() {
	migrateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview generated workflow without writing")
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(supportedArchCmd)
}

func validateRustEngine(eng interface{}, cfg *config.Config) error {
	if rustEngine, ok := eng.(*rust.RustEngine); ok {
		ui.Info("Running Rust-specific validation...")
		if cfg == nil {
			return fmt.Errorf("config is nil, cannot run Rust-specific validation")
		}
		return rustEngine.ValidateRustSpecific(cfg)
	}
	return nil
}
