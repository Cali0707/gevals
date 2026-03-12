package agent_test

import (
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/mcpchecker/mcpchecker/pkg/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractToolCalls(t *testing.T) {
	title := "Read File"

	tt := map[string]struct {
		updates        []acp.SessionUpdate
		expectedCount  int
		expectedTitles []string
	}{
		"nil updates": {
			updates:       nil,
			expectedCount: 0,
		},
		"empty updates": {
			updates:       []acp.SessionUpdate{},
			expectedCount: 0,
		},
		"single tool call": {
			updates: []acp.SessionUpdate{
				{
					ToolCall: &acp.SessionUpdateToolCall{
						ToolCallId: "tc-1",
						Title:      "Read File",
						Kind:       "read",
						Status:     "completed",
						RawInput:   map[string]any{"path": "/tmp/test"},
						RawOutput:  map[string]any{"content": "hello"},
					},
				},
			},
			expectedCount:  1,
			expectedTitles: []string{"Read File"},
		},
		"merges ToolCall with ToolCallUpdate": {
			updates: []acp.SessionUpdate{
				{
					ToolCall: &acp.SessionUpdateToolCall{
						ToolCallId: "tc-1",
						Title:      "Read File",
						Kind:       "read",
						Status:     "running",
						RawInput:   map[string]any{"path": "/tmp/test"},
					},
				},
				{
					ToolCallUpdate: &acp.SessionToolCallUpdate{
						ToolCallId: "tc-1",
						Title:      &title,
						RawOutput:  map[string]any{"content": "hello"},
					},
				},
			},
			expectedCount:  1,
			expectedTitles: []string{"Read File"},
		},
		"preserves order": {
			updates: []acp.SessionUpdate{
				{ToolCall: &acp.SessionUpdateToolCall{ToolCallId: "tc-1", Title: "First"}},
				{ToolCall: &acp.SessionUpdateToolCall{ToolCallId: "tc-2", Title: "Second"}},
				{ToolCall: &acp.SessionUpdateToolCall{ToolCallId: "tc-3", Title: "Third"}},
			},
			expectedCount:  3,
			expectedTitles: []string{"First", "Second", "Third"},
		},
		"deduplicates same ToolCallId": {
			updates: []acp.SessionUpdate{
				{ToolCall: &acp.SessionUpdateToolCall{ToolCallId: "tc-1", Title: "Read File"}},
				{ToolCall: &acp.SessionUpdateToolCall{ToolCallId: "tc-1", Title: "Read File Again"}},
			},
			expectedCount: 1,
		},
		"ToolCallUpdate without prior ToolCall": {
			updates: []acp.SessionUpdate{
				{
					ToolCallUpdate: &acp.SessionToolCallUpdate{
						ToolCallId: "tc-1",
						Title:      &title,
						RawOutput:  "result data",
					},
				},
			},
			expectedCount:  1,
			expectedTitles: []string{"Read File"},
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			result := agent.ExtractToolCalls(tc.updates)

			require.Len(t, result, tc.expectedCount)
			for i, expectedTitle := range tc.expectedTitles {
				assert.Equal(t, expectedTitle, result[i].Title)
			}
		})
	}
}

func TestExtractToolCalls_FieldMapping(t *testing.T) {
	// Verify all ToolCallSummary fields are populated from a SessionUpdateToolCall.
	updates := []acp.SessionUpdate{
		{
			ToolCall: &acp.SessionUpdateToolCall{
				ToolCallId: "tc-1",
				Title:      "Read File",
				Kind:       "read",
				Status:     "completed",
				RawInput:   map[string]any{"path": "/tmp/test"},
				RawOutput:  map[string]any{"content": "hello"},
			},
		},
	}

	result := agent.ExtractToolCalls(updates)

	require.Len(t, result, 1)
	assert.Equal(t, "Read File", result[0].Title)
	assert.Equal(t, "read", result[0].Kind)
	assert.Equal(t, "completed", result[0].Status)
	assert.Equal(t, map[string]any{"path": "/tmp/test"}, result[0].RawInput)
	assert.Equal(t, map[string]any{"content": "hello"}, result[0].RawOutput)
}

func TestExtractToolCalls_MergedFieldMapping(t *testing.T) {
	// Verify RawOutput from a ToolCallUpdate is merged onto the original ToolCall.
	title := "Read File"
	updates := []acp.SessionUpdate{
		{
			ToolCall: &acp.SessionUpdateToolCall{
				ToolCallId: "tc-1",
				Title:      "Read File",
				RawInput:   map[string]any{"path": "/tmp/test"},
			},
		},
		{
			ToolCallUpdate: &acp.SessionToolCallUpdate{
				ToolCallId: "tc-1",
				Title:      &title,
				RawOutput:  map[string]any{"content": "hello"},
			},
		},
	}

	result := agent.ExtractToolCalls(updates)

	require.Len(t, result, 1)
	assert.Equal(t, map[string]any{"path": "/tmp/test"}, result[0].RawInput)
	assert.Equal(t, map[string]any{"content": "hello"}, result[0].RawOutput)
}

func TestExtractFinalMessage(t *testing.T) {
	tt := map[string]struct {
		updates  []acp.SessionUpdate
		expected string
	}{
		"nil updates": {
			updates:  nil,
			expected: "",
		},
		"empty updates": {
			updates:  []acp.SessionUpdate{},
			expected: "",
		},
		"concatenates chunks": {
			updates: []acp.SessionUpdate{
				acp.UpdateAgentMessageText("Hello "),
				acp.UpdateAgentMessageText("world!"),
			},
			expected: "Hello world!",
		},
		"ignores non-message updates": {
			updates: []acp.SessionUpdate{
				acp.UpdateAgentMessageText("Hello"),
				acp.UpdateAgentThoughtText("thinking..."),
				acp.UpdateAgentMessageText(" world"),
			},
			expected: "Hello world",
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, agent.ExtractFinalMessage(tc.updates))
		})
	}
}

func TestExtractThinking(t *testing.T) {
	tt := map[string]struct {
		updates  []acp.SessionUpdate
		expected string
	}{
		"nil updates": {
			updates:  nil,
			expected: "",
		},
		"empty updates": {
			updates:  []acp.SessionUpdate{},
			expected: "",
		},
		"concatenates chunks": {
			updates: []acp.SessionUpdate{
				acp.UpdateAgentThoughtText("Let me "),
				acp.UpdateAgentThoughtText("think about this."),
			},
			expected: "Let me think about this.",
		},
		"ignores non-thought updates": {
			updates: []acp.SessionUpdate{
				acp.UpdateAgentThoughtText("thinking"),
				acp.UpdateAgentMessageText("message"),
				acp.UpdateAgentThoughtText(" more"),
			},
			expected: "thinking more",
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, agent.ExtractThinking(tc.updates))
		})
	}
}
