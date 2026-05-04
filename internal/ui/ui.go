// Package ui provides helper functions for formatted terminal output.
package ui

import (
	"fmt"
	"os"
)

// ANSI escape codes for terminal formatting.
const (
	Reset  = "\033[0m"  // Reset all attributes
	Red    = "\033[31m" // Red text
	Green  = "\033[32m" // Green text
	Yellow = "\033[33m" // Yellow text
	Blue   = "\033[34m" // Blue text
	Purple = "\033[35m" // Purple text
	Cyan   = "\033[36m" // Cyan text
	Gray   = "\033[37m" // Gray text
	White  = "\033[97m" // White text
	Bold   = "\033[1m"  // Bold text
)

// Success prints a success message with a green checkmark.
func Success(format string, a ...any) {
	fmt.Printf("%s%s✔%s %s\n", Green, Bold, Reset, fmt.Sprintf(format, a...))
}

// Info prints an informational message in blue.
func Info(format string, a ...any) {
	fmt.Printf("%s%sℹ%s %s\n", Blue, Bold, Reset, fmt.Sprintf(format, a...))
}

// Warn prints a warning message in yellow.
func Warn(format string, a ...any) {
	fmt.Printf("%s%s⚠%s %s\n", Yellow, Bold, Reset, fmt.Sprintf(format, a...))
}

// Error prints an error and a suggestion to stderr.
func Error(err error, help string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s%s✖ Error:%s %v\n", Red, Bold, Reset, err)
	}
	if help != "" {
		fmt.Fprintf(os.Stderr, "%s%s💡 Suggestion:%s %s\n", Cyan, Bold, Reset, help)
	}
}

// Fatal prints error and exits the program.
func Fatal(err error, help string) {
	Error(err, help)
	os.Exit(1)
}

// Section prints a section header in purple.
func Section(name string) {
	fmt.Printf("\n%s%s===> %s%s\n", Purple, Bold, name, Reset)
}
