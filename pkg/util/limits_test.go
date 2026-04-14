package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimits_GetTimeout(t *testing.T) {
	tests := map[string]struct {
		limits   *Limits
		expected time.Duration
		isSet    bool
		hasErr   bool
	}{
		"nil Limits": {
			limits:   nil,
			expected: 0,
			isSet:    false,
		},
		"empty timeout": {
			limits:   &Limits{Timeout: ""},
			expected: 0,
			isSet:    false,
		},
		"valid minutes": {
			limits:   &Limits{Timeout: "15m"},
			expected: 15 * time.Minute,
			isSet:    true,
		},
		"valid seconds": {
			limits:   &Limits{Timeout: "30s"},
			expected: 30 * time.Second,
			isSet:    true,
		},
		"valid compound duration": {
			limits:   &Limits{Timeout: "1h30m"},
			expected: 90 * time.Minute,
			isSet:    true,
		},
		"valid milliseconds": {
			limits:   &Limits{Timeout: "500ms"},
			expected: 500 * time.Millisecond,
			isSet:    true,
		},
		"invalid duration": {
			limits: &Limits{Timeout: "abc"},
			hasErr: true,
		},
		"number without unit": {
			limits: &Limits{Timeout: "30"},
			hasErr: true,
		},
		"negative duration": {
			limits: &Limits{Timeout: "-30s"},
			hasErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			d, ok, err := tc.limits.GetTimeout()
			if tc.hasErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid timeout")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, d)
			assert.Equal(t, tc.isSet, ok)
		})
	}
}

func TestLimits_GetCleanupTimeout(t *testing.T) {
	tests := map[string]struct {
		limits   *Limits
		expected time.Duration
		isSet    bool
		hasErr   bool
	}{
		"nil Limits": {
			limits:   nil,
			expected: 0,
			isSet:    false,
		},
		"empty cleanupTimeout": {
			limits:   &Limits{CleanupTimeout: ""},
			expected: 0,
			isSet:    false,
		},
		"valid minutes": {
			limits:   &Limits{CleanupTimeout: "5m"},
			expected: 5 * time.Minute,
			isSet:    true,
		},
		"valid seconds": {
			limits:   &Limits{CleanupTimeout: "30s"},
			expected: 30 * time.Second,
			isSet:    true,
		},
		"valid compound duration": {
			limits:   &Limits{CleanupTimeout: "2m30s"},
			expected: 2*time.Minute + 30*time.Second,
			isSet:    true,
		},
		"invalid duration": {
			limits: &Limits{CleanupTimeout: "abc"},
			hasErr: true,
		},
		"number without unit": {
			limits: &Limits{CleanupTimeout: "30"},
			hasErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			d, ok, err := tc.limits.GetCleanupTimeout()
			if tc.hasErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid cleanupTimeout")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, d)
			assert.Equal(t, tc.isSet, ok)
		})
	}
}
