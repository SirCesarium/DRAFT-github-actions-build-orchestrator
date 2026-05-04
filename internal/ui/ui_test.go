package ui

import (
	"os"
	"testing"
)

func TestSuccess(_ *testing.T) {
	Success("test %s", "message")
}

func TestInfo(_ *testing.T) {
	Info("test %s", "info")
}

func TestWarn(_ *testing.T) {
	Warn("test %s", "warning")
}

func TestError(_ *testing.T) {
	Error(os.ErrNotExist, "help message")
	Error(nil, "help message")
}

func TestFatal(_ *testing.T) {
	_ = Fatal
}

func TestSection(_ *testing.T) {
	Section("Test Section")
}

func TestConstants(t *testing.T) {
	if Reset == "" {
		t.Error("expected Reset to be non-empty")
	}
	if Red == "" {
		t.Error("expected Red to be non-empty")
	}
	if Green == "" {
		t.Error("expected Green to be non-empty")
	}
}
