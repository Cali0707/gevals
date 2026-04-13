package util

import (
	"fmt"
	"time"
)

// Limits configures timeout constraints for task execution.
type Limits struct {
	Timeout        string `json:"timeout,omitempty"`
	CleanupTimeout string `json:"cleanupTimeout,omitempty"`
}

// GetTimeout parses the Timeout field as a time.Duration.
// Returns (duration, true, nil) if set and valid, or (0, false, nil) if not set.
// Returns an error if the string is set but cannot be parsed.
func (l *Limits) GetTimeout() (time.Duration, bool, error) {
	if l == nil || l.Timeout == "" {
		return 0, false, nil
	}

	d, err := time.ParseDuration(l.Timeout)
	if err != nil {
		return 0, false, fmt.Errorf("invalid timeout %q: %w", l.Timeout, err)
	}

	if d <= 0 {
		return 0, false, fmt.Errorf("invalid timeout: timeout duration must be > 0, got %q", l.Timeout)
	}

	return d, true, nil
}

// GetCleanupTimeout parses the CleanupTimeout field as a time.Duration.
// Returns (duration, true, nil) if set and valid, or (0, false, nil) if not set.
// Returns an error if the string is set but cannot be parsed.
func (l *Limits) GetCleanupTimeout() (time.Duration, bool, error) {
	if l == nil || l.CleanupTimeout == "" {
		return 0, false, nil
	}

	d, err := time.ParseDuration(l.CleanupTimeout)
	if err != nil {
		return 0, false, fmt.Errorf("invalid cleanupTimeout %q: %w", l.CleanupTimeout, err)
	}

	if d <= 0 {
		return 0, false, fmt.Errorf("invalid timeout: timeout duration must be > 0, got %q", l.Timeout)
	}

	return d, true, nil
}
