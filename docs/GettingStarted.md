# Getting Started

This guide will walk you through the initial setup of Refinery and your first build and deployment cycle.

## Installation

### Binary Download
The most efficient way to use Refinery is by downloading a pre-compiled binary for your system from the [Releases](https://github.com/SirCesarium/refinery/releases/latest) page.

### From Source
If you have the Go toolchain installed (v1.26.2 or higher):

```bash
go install github.com/SirCesarium/refinery/cmd/refinery@latest
```

## 1. Initializing a Project

To start, run the initialization command in your project root:

```bash
refinery init
```

This creates a `refinery.toml` file with a baseline configuration.

## 2. Configuring Artifacts

Edit `refinery.toml` to define your project structure. Refinery automatically detects your project language during `init`, but you can always change it.

### For a Rust Project
Typical for a CLI or system library:

```toml
refinery_version = "latest"
output_dir = "dist"

[project]
name = "my-rust-app"
lang = "rust"

[artifacts.cli]
type = "bin"
source = "src/main.rs"
packages = ["tar.gz", "deb"]

[artifacts.cli.targets.linux]
os = "linux"
archs = ["x86_64", "aarch64"]
abis = ["gnu", "musl"]
```

### For a Go Project
Typical for a web server or backend tool:

```toml
refinery_version = "latest"
output_dir = "dist"

[project]
name = "my-go-app"
lang = "go"

[artifacts.server]
type = "bin"
source = "."
packages = ["tar.gz"]

[artifacts.server.targets.linux]
os = "linux"
archs = ["amd64", "arm64"]
```

## 3. Local Build and Packaging

To build a specific artifact for a target architecture locally:

```bash
# For the Rust example:
refinery build --artifact cli --os linux --arch x86_64 --abi gnu

# For the Go example:
refinery build --artifact server --os linux --arch amd64
```

Refinery will:
1. Validate the configuration against your manifest (`Cargo.toml` or `go.mod`).
2. Orchestrate the compiler using the correct target environment.
3. Package the resulting binary into the specified format (e.g., `.tar.gz`) in the `dist/` directory.

## 4. Automating CI/CD

Generate a GitHub Actions workflow to automate multi-platform builds:

```bash
refinery migrate github --dry-run
```

This command outputs a preview of `.github/workflows/refinery-build.yml`. Remove `--dry-run` to write the file. The workflow is pre-configured with a build matrix and automatic binary delivery.
