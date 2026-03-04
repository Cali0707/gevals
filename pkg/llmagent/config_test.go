package llmagent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseModel(t *testing.T) {
	tests := map[string]struct {
		model          string
		expectProvider string
		expectModelID  string
		expectErr      bool
		errContains    string
	}{
		"valid openai model": {
			model:          "openai:gpt-4",
			expectProvider: "openai",
			expectModelID:  "gpt-4",
		},
		"valid anthropic model": {
			model:          "anthropic:claude-sonnet-4-20250514",
			expectProvider: "anthropic",
			expectModelID:  "claude-sonnet-4-20250514",
		},
		"valid gemini model": {
			model:          "gemini:gemini-2.5-pro",
			expectProvider: "gemini",
			expectModelID:  "gemini-2.5-pro",
		},
		"model with multiple colons uses first as separator": {
			model:          "openai:ft:gpt-4:my-org",
			expectProvider: "openai",
			expectModelID:  "ft:gpt-4:my-org",
		},
		"empty string": {
			model:       "",
			expectErr:   true,
			errContains: "provider:model-id",
		},
		"no colon separator": {
			model:       "gpt-4",
			expectErr:   true,
			errContains: "provider:model-id",
		},
		"empty provider": {
			model:       ":gpt-4",
			expectErr:   true,
			errContains: "provider:model-id",
		},
		"empty model id": {
			model:       "openai:",
			expectErr:   true,
			errContains: "provider:model-id",
		},
		"only colon": {
			model:       ":",
			expectErr:   true,
			errContains: "provider:model-id",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := Config{Model: tc.model}
			provider, modelID, err := cfg.ParseModel()

			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectProvider, provider)
			assert.Equal(t, tc.expectModelID, modelID)
		})
	}
}
