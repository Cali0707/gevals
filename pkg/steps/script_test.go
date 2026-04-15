package steps

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScriptStepConfig_Validate(t *testing.T) {
	tt := map[string]struct {
		config    *ScriptStepConfig
		expectErr bool
	}{
		"valid file config": {
			config: &ScriptStepConfig{
				File: "./script.sh",
			},
			expectErr: false,
		},
		"valid inline config": {
			config: &ScriptStepConfig{
				Inline: "echo hello",
			},
			expectErr: false,
		},
		"invalid: both file and inline set": {
			config: &ScriptStepConfig{
				File:   "./script.sh",
				Inline: "echo hello",
			},
			expectErr: true,
		},
		"invalid: neither file nor inline set": {
			config:    &ScriptStepConfig{},
			expectErr: true,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestNewScriptStep(t *testing.T) {
	tt := map[string]struct {
		config          *ScriptStepConfig
		expectedTimeout string
		expectErr       bool
	}{
		"default timeout": {
			config: &ScriptStepConfig{
				Inline: "echo hello",
			},
			expectedTimeout: "5m0s",
			expectErr:       false,
		},
		"custom timeout": {
			config: &ScriptStepConfig{
				Inline:  "echo hello",
				Timeout: "30s",
			},
			expectedTimeout: "30s",
			expectErr:       false,
		},
		"invalid timeout": {
			config: &ScriptStepConfig{
				Inline:  "echo hello",
				Timeout: "invalid",
			},
			expectErr: true,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			step, err := NewScriptStep(tc.config)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedTimeout, step.Timeout.String())
		})
	}
}

func TestScriptStep_Execute(t *testing.T) {
	tt := map[string]struct {
		config    *ScriptStepConfig
		input     *StepInput
		expected  *StepOutput
		expectErr bool
	}{
		"inline script succeeds": {
			config: &ScriptStepConfig{
				Inline: "echo hello",
			},
			input: &StepInput{},
			expected: &StepOutput{
				Success: true,
				Message: "hello\n",
			},
			expectErr: false,
		},
		"inline script with shebang": {
			config: &ScriptStepConfig{
				Inline: "#!/bin/sh\necho shebang",
			},
			input: &StepInput{},
			expected: &StepOutput{
				Success: true,
				Message: "shebang\n",
			},
			expectErr: false,
		},
		"inline script uses env vars from config": {
			config: &ScriptStepConfig{
				Inline: "echo $TEST_VAR",
				Env:    map[string]string{"TEST_VAR": "from_config"},
			},
			input: &StepInput{},
			expected: &StepOutput{
				Success: true,
				Message: "from_config\n",
			},
			expectErr: false,
		},
		"env resolves step output template": {
			config: &ScriptStepConfig{
				Inline: "echo $MY_VAR",
				Env:    map[string]string{"MY_VAR": "{steps.setup.value}"},
			},
			input: &StepInput{
				StepOutputs: map[string]map[string]string{
					"setup": {"value": "resolved_output"},
				},
			},
			expected: &StepOutput{
				Success: true,
				Message: "resolved_output\n",
			},
			expectErr: false,
		},
		"env resolves random template": {
			config: &ScriptStepConfig{
				Inline: "echo $MY_ID",
				Env:    map[string]string{"MY_ID": "{random.id}"},
			},
			input: &StepInput{
				Random: NewRandomResolver(),
			},
			expected: &StepOutput{
				Success: true,
			},
			expectErr: false,
		},
		"inline script fails": {
			config: &ScriptStepConfig{
				Inline: "exit 1",
			},
			input:     &StepInput{},
			expectErr: true,
		},
		"inline script fails with continueOnError": {
			config: &ScriptStepConfig{
				Inline:          "exit 1",
				ContinueOnError: true,
			},
			input: &StepInput{},
			expected: &StepOutput{
				Success: false,
			},
			expectErr: false,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			step, err := NewScriptStep(tc.config)
			require.NoError(t, err)

			got, err := step.Execute(context.Background(), tc.input)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected.Success, got.Success)
			if tc.expected.Message != "" {
				assert.Equal(t, tc.expected.Message, got.Message)
			}
		})
	}
}

func TestScriptStep_Execute_File(t *testing.T) {
	// Create a temporary directory for test scripts
	tmpDir, err := os.MkdirTemp("", "script-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a test script
	scriptPath := filepath.Join(tmpDir, "test.sh")
	err = os.WriteFile(scriptPath, []byte("#!/bin/sh\necho file_script"), 0755)
	require.NoError(t, err)

	tt := map[string]struct {
		config    *ScriptStepConfig
		input     *StepInput
		expected  *StepOutput
		expectErr bool
	}{
		"file script succeeds": {
			config: &ScriptStepConfig{
				File: scriptPath,
			},
			input: &StepInput{},
			expected: &StepOutput{
				Success: true,
				Message: "file_script\n",
			},
			expectErr: false,
		},
		"file script with workdir": {
			config: &ScriptStepConfig{
				File: "test.sh",
			},
			input: &StepInput{
				Workdir: tmpDir,
			},
			expected: &StepOutput{
				Success: true,
				Message: "file_script\n",
			},
			expectErr: false,
		},
		"file script not found": {
			config: &ScriptStepConfig{
				File: "/nonexistent/script.sh",
			},
			input:     &StepInput{},
			expectErr: true,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			step, err := NewScriptStep(tc.config)
			require.NoError(t, err)

			got, err := step.Execute(context.Background(), tc.input)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected.Success, got.Success)
			if tc.expected.Message != "" {
				assert.Equal(t, tc.expected.Message, got.Message)
			}
		})
	}
}
