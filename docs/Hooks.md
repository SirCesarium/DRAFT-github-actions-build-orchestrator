# Build Hooks

Refinery allows the execution of custom system commands at both the global and artifact levels through the `pre_build` and `post_build` configuration. This is essential for tasks like code generation, asset processing, or post-compilation cleanup.

## Artifact Hooks

Hooks defined at the artifact level run for every build of that specific artifact:

```toml
[artifacts.my_app.hooks]
pre_build = ["./scripts/generate_version.sh", "echo 'Starting build...'"]
post_build = ["cp {binary} ./backups/"]
```

## Global Hooks

Global hooks run at the start or end of the entire pipeline. These are especially useful in CI/CD environments.

```toml
[[pre_build]]
id = "setup-env"
command = ["./scripts/setup.sh"]
once = true

[[post_build]]
id = "upload-logs"
action = "upload-artifacts"
with = { name = "logs", path = "logs/" }
once = true
```

### Hook Types
- **`pre_build`**: Executed before the compilation phase.
- **`post_build`**: Executed after the compilation phase.

## Execution Logic

1.  **Global Preparation**: Global `pre_build` hooks with `once = true` are executed.
2.  **Environment Preparation**: Refinery sets up all target-specific variables (linkers, paths).
3.  **Local Pre-Build Hooks**: Global `pre_build` (without `once`) and artifact-level `pre_build` hooks are executed sequentially. If any hook returns a non-zero exit code, the build is aborted (Fail-Fast).
4.  **Compilation**: The underlying compiler is invoked.
5.  **Artifact Retrieval**: Files are moved to the distribution directory.
6.  **Local Post-Build Hooks**: Artifact-level `post_build` and global `post_build` (without `once`) hooks are executed.
7.  **Global Teardown**: Global `post_build` hooks with `once = true` are executed.

## Variable Substitution

Commands defined in hooks can use dynamic placeholders that Refinery resolves at runtime based on the current build target:

- **`{artifact}`**: The name of the artifact.
- **`{os}`**: Target operating system.
- **`{arch}`**: Target architecture.
- **`{abi}`**: Target ABI.
- **`{version}`**: Project version.
- **`{binary}`**: The full path to the compiled binary before it is moved to the final output directory.

### Example: Version Header Generation

```toml
[artifacts.core.hooks]
pre_build = ["sed -i 's/VERSION/%v/g' version.h" ]
```
(Note: substitution occurs in the Go engine before the command is sent to the shell).
