# Architectures and ABIs

Refinery provides a high-level abstraction over the complex world of cross-compilation target triples and binary interfaces.

## Supported Operating Systems

| OS Key | Description | Target Triple Base |
| :--- | :--- | :--- |
| `linux` | Standard Linux distributions | `unknown-linux` |
| `windows` | Microsoft Windows | `pc-windows` |
| `darwin` | Apple macOS | `apple-darwin` |
| `wasm` | WebAssembly (Browser/Generic) | `wasm32-unknown-unknown` |
| `wasi` | WebAssembly (System Interface) | `wasm32-wasip1` |

## Architectures

| Arch Key | Description |
| :--- | :--- |
| `x86_64` | 64-bit Intel/AMD |
| `aarch64` | 64-bit ARM (Apple Silicon, Raspberry Pi 4+, Graviton) |
| `i686` | 32-bit Intel/AMD (Windows only) |
| `wasm32` | 32-bit WebAssembly |

## ABIs (Application Binary Interfaces)

The ABI determines how a program interacts with the operating system at a low level.

### GNU (`gnu`)
Utilizes the GNU C Library (`glibc`). This is the standard for most Linux desktop and server distributions (Ubuntu, Fedora, Debian). It is also available for Windows via the MinGW toolchain.

### MUSL (`musl`)
Utilizes the `musl` libc implementation. It is used to create **fully static binaries** on Linux. A binary compiled with the `musl` ABI has no external dependencies and can run on any Linux distribution (including ultra-lightweight ones like Alpine Linux) regardless of the installed `glibc` version.

### MSVC (`msvc`)
The native Application Binary Interface for Windows. It utilizes the Microsoft Visual C++ runtime. This is the recommended ABI for professional Windows software distribution.

## Target Triple Resolution

Refinery automatically constructs the correct compiler target triple by combining the configuration fields. For example, a Rust build with:
- `os = "linux"`
- `arch = "aarch64"`
- `abi = "musl"`

Results in the target triple: `aarch64-unknown-linux-musl`.

## Intelligent Linker Inference

For Linux cross-compilation, Refinery automatically selects the appropriate linker if one is not manually specified in `lang_opts`:

- **aarch64:** `aarch64-linux-gnu-gcc`
- **i686:** `i686-linux-gnu-gcc`

Refinery ensures that these linkers are only applied to their respective architectures, preventing architecture-mismatch errors during the linking phase.
