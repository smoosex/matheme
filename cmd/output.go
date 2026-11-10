package cmd

import (
	"encoding/json"
	"fmt"
	"os"
)

// exitError carries the process exit code through cobra so that every deferred cleanup in the
// command still runs before the process leaves.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func exitWith(code int, format string, args ...any) error {
	return &exitError{code: code, err: fmt.Errorf(format, args...)}
}

// silentExit stops the command without printing anything else; the caller already reported.
func silentExit(code int) error {
	return &exitError{code: code}
}

func emitJSON(payload any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		return fmt.Errorf("failed to write JSON output: %w", err)
	}
	return nil
}

// reportError prints a JSON payload for machine consumers, then fails the command.
func reportError(payload any, format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	if jsonOut {
		if err := emitJSON(payload); err != nil {
			return exitWith(1, "%s (also: %v)", message, err)
		}
	}
	return exitWith(1, "%s", message)
}
