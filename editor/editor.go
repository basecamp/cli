// Package editor provides $EDITOR integration for composing content.
package editor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Open launches $EDITOR with initialContent and returns the edited text.
// Falls back to vi if $EDITOR is not set.
// Returns an error if the editor exits non-zero or the result is empty.
func Open(initialContent string) (string, error) {
	editorCmd := strings.TrimSpace(os.Getenv("EDITOR"))
	if editorCmd == "" {
		editorCmd = "vi"
	}

	tmp, err := os.CreateTemp("", "cli-edit-*.md")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if initialContent != "" {
		_, writeErr := tmp.WriteString(initialContent)
		if writeErr != nil {
			_ = tmp.Close()
			return "", fmt.Errorf("writing initial content: %w", writeErr)
		}
	}
	closeErr := tmp.Close()
	if closeErr != nil {
		return "", fmt.Errorf("closing temp file: %w", closeErr)
	}

	// Use sh -c to handle quoted arguments and paths with spaces in $EDITOR
	cmd := exec.Command("sh", "-c", editorCmd+` "$1"`, "_", tmp.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	runErr := cmd.Run()
	if runErr != nil {
		return "", fmt.Errorf("editor exited with error: %w", runErr)
	}

	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		return "", fmt.Errorf("reading edited file: %w", err)
	}

	result := strings.TrimSpace(string(data))
	if result == "" {
		return "", fmt.Errorf("empty content — aborting")
	}

	return result, nil
}
