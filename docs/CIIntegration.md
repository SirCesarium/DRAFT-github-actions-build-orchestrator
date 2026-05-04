# CI/CD Integration

Refinery is designed to minimize the complexity and overhead of maintaining multi-platform CI pipelines.

## GitHub Actions

Generate a complete workflow by running:

```bash
refinery migrate github
```

### Key Features of Generated Workflows

1.  **Pre-compiled Binaries**: Instead of compiling Refinery from source in every job, the workflow uses the `SirCesarium/refinery` action to download optimized, pre-built binaries. This significantly reduces total build time.
2.  **Semantic Versioning**: The workflow respects the `refinery_version` field in `refinery.toml`. You can pin to a major version (e.g., `"2"`) or use `"latest"` to always receive the newest updates and security patches.
3.  **Automatic Dependency Management**:
    - **Linkers**: If the build engine detects a cross-compilation target (like ARM on Linux), the CI will automatically install the required cross-compiler toolchain via `apt-get`.
    - **System Tools**: If you request a `.deb` package, the CI will automatically install `cargo-deb` or its equivalent.
    - **ABIs**: For `musl` targets, the CI installs `musl-tools` automatically.
4.  **Build Matrix**: Refinery automatically calculates the optimal build matrix based on your artifact configurations, ensuring every target is built in a clean, isolated environment.
5.  **Artifact Uploads**: Every built artifact and package is automatically uploaded as a GitHub Action artifact, organized by target architecture.
6.  **Automated Releases**: If the pipeline is triggered by a version tag (e.g., `v1.2.3`), a GitHub Release is automatically created containing all the generated packages.

## Environment Isolation

Each build task in the matrix is strictly isolated. Environment variables (like `CARGO_TARGET_<TRIPLE>_LINKER` or `MACOSX_DEPLOYMENT_TARGET`) are scoped specifically to the architecture and ABI being processed, preventing configuration "bleeding" between different jobs.
