// Package github defines constants for GitHub Actions used in workflow generation.
package github

// GitHub Actions used in workflow generation.
const (
	ActionCheckout         = "actions/checkout@v6"
	ActionSetupGo          = "actions/setup-go@v6"
	ActionUploadArtifact   = "actions/upload-artifact@v7"
	ActionDownloadArtifact = "actions/download-artifact@v8"
	ActionGHRelease        = "softprops/action-gh-release@v3"
	ActionRustToolchain    = "actions-rust-lang/setup-rust-toolchain@v1"
)
