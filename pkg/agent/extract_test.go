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

func TestExtractOutputSteps(t *testing.T) {
	title := "Read File"

	tt := map[string]struct {
		updates  []acp.SessionUpdate
		expected []agent.OutputStep
	}{
		"nil updates": {
			updates:  nil,
			expected: nil,
		},
		"empty updates": {
			updates:  []acp.SessionUpdate{},
			expected: nil,
		},
		"consecutive thinking chunks consolidated": {
			updates: []acp.SessionUpdate{
				acp.UpdateAgentThoughtText("Let me "),
				acp.UpdateAgentThoughtText("think about this."),
			},
			expected: []agent.OutputStep{
				{Type: "thinking", Content: "Let me think about this."},
			},
		},
		"consecutive message chunks consolidated": {
			updates: []acp.SessionUpdate{
				acp.UpdateAgentMessageText("Hello "),
				acp.UpdateAgentMessageText("world!"),
			},
			expected: []agent.OutputStep{
				{Type: "message", Content: "Hello world!"},
			},
		},
		"interleaved thinking, tool_call, message produces 3 steps": {
			updates: []acp.SessionUpdate{
				acp.UpdateAgentThoughtText("thinking..."),
				{
					ToolCall: &acp.SessionUpdateToolCall{
						ToolCallId: "tc-1",
						Title:      "Read File",
						Kind:       "read",
						Status:     "running",
					},
				},
				acp.UpdateAgentMessageText("done"),
			},
			expected: []agent.OutputStep{
				{Type: "thinking", Content: "thinking..."},
				{Type: "tool_call", ToolCall: &agent.ToolCallSummary{
					Title:  "Read File",
					Kind:   "read",
					Status: "running",
				}},
				{Type: "message", Content: "done"},
			},
		},
		"tool call dedup merges ToolCallUpdate": {
			updates: []acp.SessionUpdate{
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
			},
			expected: []agent.OutputStep{
				{Type: "tool_call", ToolCall: &agent.ToolCallSummary{
					Title:     "Read File",
					RawInput:  map[string]any{"path": "/tmp/test"},
					RawOutput: map[string]any{"content": "hello"},
				}},
			},
		},
		"non-consecutive thinking produces separate steps": {
			updates: []acp.SessionUpdate{
				acp.UpdateAgentThoughtText("first thought"),
				acp.UpdateAgentMessageText("message"),
				acp.UpdateAgentThoughtText("second thought"),
			},
			expected: []agent.OutputStep{
				{Type: "thinking", Content: "first thought"},
				{Type: "message", Content: "message"},
				{Type: "thinking", Content: "second thought"},
			},
		},
		"ToolCallUpdate without prior ToolCall creates step": {
			updates: []acp.SessionUpdate{
				{
					ToolCallUpdate: &acp.SessionToolCallUpdate{
						ToolCallId: "tc-1",
						Title:      &title,
						RawOutput:  "result data",
					},
				},
			},
			expected: []agent.OutputStep{
				{Type: "tool_call", ToolCall: &agent.ToolCallSummary{
					Title:     "Read File",
					RawOutput: "result data",
				}},
			},
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			result := agent.ExtractOutputSteps(tc.updates)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFinalMessageFromSteps(t *testing.T) {
	tt := map[string]struct {
		steps    []agent.OutputStep
		expected string
	}{
		"nil steps": {
			steps:    nil,
			expected: "",
		},
		"empty steps": {
			steps:    []agent.OutputStep{},
			expected: "",
		},
		"single message step": {
			steps: []agent.OutputStep{
				{Type: "message", Content: "hello"},
			},
			expected: "hello",
		},
		"multiple message steps returns last": {
			steps: []agent.OutputStep{
				{Type: "message", Content: "hello "},
				{Type: "message", Content: "world"},
			},
			expected: "world",
		},
		"ignores non-message steps": {
			steps: []agent.OutputStep{
				{Type: "thinking", Content: "thinking..."},
				{Type: "message", Content: "result"},
				{Type: "tool_call", ToolCall: &agent.ToolCallSummary{Title: "Read"}},
			},
			expected: "result",
		},
		"mixed steps returns last message": {
			steps: []agent.OutputStep{
				{Type: "thinking", Content: "let me think"},
				{Type: "tool_call", ToolCall: &agent.ToolCallSummary{Title: "Read"}},
				{Type: "message", Content: "first "},
				{Type: "tool_call", ToolCall: &agent.ToolCallSummary{Title: "Write"}},
				{Type: "message", Content: "second"},
			},
			expected: "second",
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, agent.FinalMessageFromSteps(tc.steps))
		})
	}
}
