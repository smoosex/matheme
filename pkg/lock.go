package pkg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// AcquireLock serializes theme switches. The returned function releases the lock.
func AcquireLock() (func(), error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	dir := filepath.Join(homeDir, ".config", "matheme")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.OpenFile(filepath.Join(dir, ".switch.lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("another matheme switch is already running")
		}
		return nil, fmt.Errorf("failed to lock %s: %w", file.Name(), err)
	}

	return func() {
		syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		file.Close()
	}, nil
}
