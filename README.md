# [UNMANTAINED] github-actions-build-orchestrator

[![CI](https://github.com/SirCesarium/refinery/actions/workflows/ci.yml/badge.svg)](https://github.com/SirCesarium/refinery/actions/workflows/ci.yml)
[![Build](https://github.com/SirCesarium/refinery/actions/workflows/refinery-build.yml/badge.svg)](https://github.com/SirCesarium/refinery/actions/workflows/refinery-build.yml)
[![Smoke Tests](https://github.com/SirCesarium/refinery/actions/workflows/smoke-tests.yml/badge.svg)](https://github.com/SirCesarium/refinery/actions/workflows/smoke-tests.yml)
[![Latest Release](https://img.shields.io/github/v/release/SirCesarium/refinery?label=release)](https://github.com/SirCesarium/refinery/releases/latest)
[![Wiki](https://img.shields.io/badge/docs-wiki-blue?logo=github)](https://github.com/SirCesarium/refinery-rs/wiki)

**Multi-ecosystem artifact orchestrator and CI/CD pipeline generator.**

---

`refinery` is a build orchestrator designed to decouple compilation logic from CI/CD providers. It manages the lifecycle of an artifact: from configuration validation and cross-compilation to multi-format packaging and automated workflow generation.

## Features

- **Strategy-Based Architecture**
  - Modular engines for languages (Rust, Go) and CI providers (GitHub Actions).
  - Registry-based system for extending supported ecosystems.

- **Multi-Format Packaging**
  - Support for distribution formats: `.deb`, `.rpm`, `.msi`, `.tar.gz`, and `.zip`.
  - Maps internal build types to system-native extensions.

- **Fail-Fast Validation**
  - TOML schema verification for `refinery.toml`.
  - Manifest cross-validation (e.g., against `Cargo.toml` or `go.mod`) before execution.

- **Cross-Compilation Management**
  - Automatic linker selection for Linux ARM and x86 targets.
  - Support for WebAssembly targets (`wasm32-unknown-unknown` and `wasm32-wasip1`).

- **Build Hooks**
  - Execution of `pre_build` and `post_build` system commands.
  - Variable substitution for hook commands.

- **CI Automation**
  - Generates GitHub Actions workflows using composite actions.
  - Optimized binary delivery to avoid compilation overhead in CI.

### Workflow

1. **Validation:** Verifies configuration and project manifest integrity.
2. **Orchestration:** Invokes the appropriate compiler for the specified target triple.
3. **Packaging:** Groups binaries into archives or system-native installers.
4. **CI Generation:** Outputs the necessary YAML for automated pipelines.

## Architecture & Ecosystems

### Why an Orchestrator?

Refinery abstracts underlying build tools. Instead of writing complex CI scripts per project, you define **what** to build and **where** to distribute. Refinery handles the **how** by orchestrating compilers (like Cargo) and system packagers.

### ABIs: GNU vs. MUSL vs. MSVC

Refinery allows targeting specific ABIs for cross-platform compatibility:

- **MSVC (Windows):** The standard for Windows development using the Microsoft Visual C++ runtime.
- **GNU (Linux/Windows):** Uses the GNU C Library (glibc). Standard for most desktop/server Linux.
- **MUSL (Linux):** A lightweight C library. Essential for creating **fully static binaries** that run on any Linux distribution (like Alpine) without external dependencies.

### WebAssembly & WASI

- **WASM (`wasm32-unknown-unknown`):** For running code in web browsers.
- **WASI (`wasm32-wasip1`):** A system interface for running WebAssembly outside the browser (servers, edge), providing controlled access to system resources.

### SDK Bundling (Archives for Libraries)

When a `lib` artifact specifies `zip` or `tar.gz`, Refinery creates an **SDK Bundle**:

1. **Multi-format inclusion:** Automatically groups all built types (e.g., both `.so` and `.a`) into the same archive.
2. **Automatic Headers:** If `headers = true`, Refinery scans and includes all `.h` and `.hpp` files.
3. **Distribution-Ready:** Provides everything a developer needs to link against your library in a single download.

## Installation

### From Source (Go)

```bash
go install github.com/SirCesarium/refinery/cmd/refinery@latest
```

### Direct Download

Download the binary for your system from the [Releases](https://github.com/SirCesarium/refinery/releases/latest) page.

## Command Reference

| Command            | Description           | Details                                                  |
| :----------------- | :-------------------- | :------------------------------------------------------- |
| `init`             | Initialize project    | Creates a template `refinery.toml`.                      |
| `build`            | Compile & Package     | Executes building and packaging for a target.            |
| `migrate`          | Generate CI workflow  | Validates and generates CI pipelines (GitHub, etc).      |
| `supported-archs` | List support          | Prints supported architectures for an OS via the engine. |

**`init`:** Creates a `refinery.toml` configuration file for the project.

**`migrate`:** Validates `refinery.toml` against the engine (e.g., Rust/Cargo.toml match) before writing workflows. Supports `--dry-run`.

**`build` options:**

- `--artifact`: Artifact name from configuration.
- `--os`: Target operating system.
- `--arch`: Target architecture.
- `--abi`: Target ABI (optional).
- `--version`: Explicit version for the build (defaults to manifest version).

## Usage Examples

### 1. Initialize

```bash
refinery init my-project
```

### 2. Generate CI

```bash
refinery migrate github
# Or: refinery migrate github --dry-run
```

### 3. Local Build

```bash
# Example: Build a specific binary for Linux ARM64
refinery build --artifact my-app --os linux --arch aarch64 --abi musl --version 1.2.0
```
## Configuration Examples (`refinery.toml`)

### Rust Project
```toml
refinery_version = "latest"

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

### Go Project
```toml
refinery_version = "latest"

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

## Naming Templates
```toml
[naming]
binary = "{artifact}-{os}-{arch}{abi}"
package = "{artifact}-{version}-{os}-{arch}{abi}.{ext}"
```


## Technical Details

- **Language:** Written in Go.
- **Packaging:** Archive generation (`tar.gz`, `zip`) uses Go standard library.
- **Environment:** Scoped environment variables for linker and target isolation.
