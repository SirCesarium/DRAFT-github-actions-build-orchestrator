# Configuration Reference (refinery.toml)

The `refinery.toml` file is the central authority for Refinery's orchestration. It is strictly validated before any execution.

## Global Settings

- **`refinery_version`** (string, required): The version of the Refinery binary to use in CI. Supports semantic aliases (`latest`, `1`, `2.0`, etc.).
- **`output_dir`** (string): Directory where built artifacts and packages will be stored. Defaults to `"dist"`.

## [project]
Metadata describing the overall project.

- **`name`** (string, required): Technical name of the project.
- **`lang`** (string, required): Programming language (`"rust"` or `"go"`).
- **`description`** (string): Short project description for package metadata.
- **`author`** (string): Project maintainer or author.
- **`license`** (string): SPDX license identifier (e.g., `"MIT"`, `"Apache-2.0"`).
- **`homepage`** (string): URL to the project website or repository.

## Global Build Steps

Refinery supports global build steps that can be executed as part of the CI/CD pipeline. These are defined using the `[[pre_build]]` and `[[post_build]]` arrays.

- **`id`** (string): Unique identifier for the step.
- **`action`** (string): Name of the action to run (e.g., `"smoke-test"`). Maps to `./.github/actions/<action>` or a full GitHub Action path.
- **`command`** (list of strings): Shell commands to execute if no action is specified.
- **`os`** (list of strings): Filter to run step only on specific operating systems (e.g., `["linux", "windows"]`).
- **`with`** (map): Input parameters for the action.
- **`once`** (boolean): If `true`, the step runs only once in the global setup/teardown phase (e.g., for global environment setup or final publishing) instead of for every matrix target.

## [build_refinery]

Configuration for projects that need to build the Refinery binary itself from source during CI (primarily used for Refinery development).

- **`enabled`** (boolean): Enables building Refinery from source.
- **`source`** (string): Path to Refinery source code (e.g., `"./cmd/refinery"`).

## [artifacts.\<name\>]
Defines a specific build unit. The `<name>` key must match the component name defined in your language manifest (e.g., the `name` field in `Cargo.toml` for `[[bin]]` or `[lib]`). Refinery uses this name to locate the compiled files.

- **`type`** (string, required): Either `"bin"` (executable) or `"lib"` (library).
- **`source`** (string, required): For Rust: Cargo package name (from `[package] name`). For Go: path to the entry point (e.g., `"."`).
- **`bin_name`** (string, optional): **Rust only**. The binary name as defined in `[[bin]] name` in `Cargo.toml`. Use this when the binary name differs from the artifact name.
- **`lib_name`** (string, optional): **Rust only**. The library name as defined in `[lib] name` in `Cargo.toml`. Use this when the library name differs from the package name.
- **`library_types`** (list of strings): Only for `type = "lib"`. Formats like `"cdylib"`, `"staticlib"`, `"rlib"`.
- **`packages`** (list of strings): Distribution formats: `"deb"`, `"rpm"`, `"msi"`, `"tar.gz"`, `"zip"`.
- **`headers`** (boolean): If `true`, includes `.h` and `.hpp` files in archive packages.

### [artifacts.\<name\>.targets.\<id\>]
Platform-specific build configurations. The `<id>` is an arbitrary, unique string used to identify the target block (e.g., `"linux-x64"`, `"legacy"`, or `"desktop"`).

- **`os`** (string): Target operating system (`"linux"`, `"windows"`, `"darwin"`, `"wasm"`, `"wasi"`). If omitted, Refinery defaults the OS to the value of `<id>`.
- **`archs`** (list of strings, required): Target architectures (e.g., `["x86_64", "aarch64"]`).
- **`abis`** (list of strings): Target ABIs (e.g., `["gnu", "musl", "msvc"]`).
- **`runner`** (string): Custom GitHub runner image for this target (optional).
- **`packages`** (list of strings): System packages to install before building.
- **`lang_opts`** (map): Language-specific options (see below).

## Language Options (`lang_opts`)

### Rust Engines
- **`profile`**: Cargo profile (`"release"`, `"dev"`, or custom).
- **`features`**: List or comma-separated string of Cargo features.
- **`tags`**: Alias for `features`.
- **`all-features`**: Boolean to enable all features.
- **`no-default-features`**: Boolean to disable default features.
- **`linker`**: Manual override for the linker executable.
- **`deployment_target`**: (macOS) Minimum OS version (default `"11.0"`).

### Go Engine
- **`tags`**: List or comma-separated string of Go build tags (e.g., `["netgo", "osusergo"]`).
- **`ldflags`**: String of linker flags (e.g., `"-s -w"`).

## [naming]
Templates for output filenames. If not specified, the system applies standard defaults.

- **`binary`**: Format for raw binaries (default: `"{artifact}-{os}-{arch}{abi}"`).
- **`package`**: Format for distribution packages (default: `"{artifact}-{version}-{os}-{arch}{abi}.{ext}"`).

### Template Variables
- **`{artifact}`**: The technical name of the artifact.
- **`{os}`**: Target operating system (e.g., `linux`, `windows`).
- **`{arch}`**: Target architecture (e.g., `x86_64`, `aarch64`).
- **`{version}`**: Project version extracted automatically from the language manifest (e.g., `Cargo.toml`).
- **`{abi}`**: Target ABI. If present, it is automatically prefixed with a hyphen (e.g., `-musl`).
- **`{ext}`**: The appropriate file extension for the format and OS (e.g., `exe`, `deb`, `tar.gz`).

## Validation Rules
Refinery performs strict validation before any execution:
1. **Uniqueness**: Artifact names must be unique keys within the `artifacts` map.
2. **Mandatory Fields**: `project.name`, `project.lang`, and `os` in targets are non-optional.
3. **Target Integrity**: Each target must define at least one architecture in `archs`.
4. **Type Consistency**: `library_types` is only validated and used when `type = "lib"`.

## Complete Example: Build Everything

This example demonstrates a comprehensive configuration using multiple engines, global hooks, and platform-specific options.

```toml
refinery_version = "latest"
output_dir = "dist"

[project]
name = "refinery-orchestrator"
lang = "rust" # Root project is Rust
author = "Refinery Team"
license = "MIT"

# Build refinery itself to use it for later steps
[build_refinery]
enabled = true
source = "./cmd/refinery"

# Global CI steps
[[pre_build]]
id = "lint"
command = ["cargo clippy -- -D warnings"]
os = ["linux"]
once = true

[[pre_build]]
action = "smoke-test"
with = { refinery_bin = "./refinery-local" }
once = true

# Define a CLI binary for all major platforms
[artifacts.refinery-cli]
type = "bin"
source = "refinery-cli"  # Cargo package name
bin_name = "refinery-cli"  # [[bin]] name in Cargo.toml
packages = ["tar.gz", "zip", "deb", "rpm", "msi"]

[artifacts.refinery-cli.targets.linux]
os = "linux"
archs = ["x86_64", "aarch64", "i686"]
abis = ["gnu", "musl"]
lang_opts = { profile = "release", features = ["full"] }

[artifacts.refinery-cli.targets.windows]
os = "windows"
archs = ["x86_64", "i686"]
abis = ["msvc", "gnu"]

[artifacts.refinery-cli.targets.darwin]
os = "darwin"
archs = ["x86_64", "aarch64"]

[artifacts.refinery-cli.targets.web]
os = "wasi"
archs = ["wasm32"]

# Define a cross-platform library
[artifacts.refinery-core]
type = "lib"
source = "refinery-core"  # Cargo package name
lib_name = "refinery_core"  # [lib] name in Cargo.toml
library_types = ["cdylib", "staticlib"]
packages = ["zip"]
headers = true

[artifacts.refinery-core.targets.linux]
os = "linux"
archs = ["x86_64"]
abis = ["gnu"]

# Custom naming templates
[naming]
binary = "{artifact}-{os}-{arch}{abi}"
package = "{artifact}-{version}-{os}-{arch}{abi}.{ext}"
```
