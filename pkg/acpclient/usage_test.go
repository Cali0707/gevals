package acpclient

import (
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToInt64(t *testing.T) {
	tt := map[string]struct {
		input    any
		expected int64
		ok       bool
	}{
		"float64":        {float64(42), 42, true},
		"float32":        {float32(42), 42, true},
		"int":            {int(42), 42, true},
		"int64":          {int64(42), 42, true},
		"int32":          {int32(42), 42, true},
		"string int":     {"42", 42, true},
		"string float":   {"42.5", 42, true},
		"nil":            {nil, 0, false},
		"invalid string": {"not-a-number", 0, false},
		"bool":           {true, 0, false},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			got, ok := toInt64(tc.input)
			assert.Equal(t, tc.ok, ok)
			if ok {
				assert.Equal(t, tc.expected, got)
			}
		})
	}
}

func int64Ptr(v int64) *int64 { return &v }

func TestExtractUsageFromMeta(t *testing.T) {
	tt := map[string]struct {
		updates            []acp.SessionUpdate
		expectNil          bool
		expectedInput      int64
		expectedOutput     int64
		expectedTotal      int64
		expectedThought    *int64
		expectedCachedRead *int64
		expectedCachedWrite *int64
	}{
		"nil updates": {
			updates:   nil,
			expectNil: true,
		},
		"empty updates": {
			updates:   []acp.SessionUpdate{},
			expectNil: true,
		},
		"nil meta": {
			updates: []acp.SessionUpdate{
				{
					AgentMessageChunk: &acp.SessionUpdateAgentMessageChunk{
						Content: acp.TextBlock("hello"),
						Meta:    nil,
					},
				},
			},
			expectNil: true,
		},
		"no usage key in meta": {
			updates: []acp.SessionUpdate{
				{
					AgentMessageChunk: &acp.SessionUpdateAgentMessageChunk{
						Content: acp.TextBlock("hello"),
						Meta:    map[string]any{"other": "data"},
					},
				},
			},
			expectNil: true,
		},
		"zero token counts": {
			updates: []acp.SessionUpdate{
				{
					AgentMessageChunk: &acp.SessionUpdateAgentMessageChunk{
						Content: acp.TextBlock("hello"),
						Meta: map[string]any{
							"usage": map[string]any{
								"input_tokens":  float64(0),
								"output_tokens": float64(0),
								"total_tokens":  float64(0),
							},
						},
					},
				},
			},
			expectNil: true,
		},
		"from AgentMessageChunk": {
			updates: []acp.SessionUpdate{
				{
					AgentMessageChunk: &acp.SessionUpdateAgentMessageChunk{
						Content: acp.TextBlock("hello"),
						Meta: map[string]any{
							"usage": map[string]any{
								"input_tokens":  float64(100),
								"output_tokens": float64(50),
								"total_tokens":  float64(150),
							},
						},
					},
				},
			},
			expectedInput:  100,
			expectedOutput: 50,
			expectedTotal:  150,
		},
		"from AgentThoughtChunk": {
			updates: []acp.SessionUpdate{
				{
					AgentThoughtChunk: &acp.SessionUpdateAgentThoughtChunk{
						Content: acp.TextBlock("thinking"),
						Meta: map[string]any{
							"usage": map[string]any{
								"input_tokens":  float64(200),
								"output_tokens": float64(100),
								"total_tokens":  float64(300),
							},
						},
					},
				},
			},
			expectedInput:  200,
			expectedOutput: 100,
			expectedTotal:  300,
		},
		"from ToolCall": {
			updates: []acp.SessionUpdate{
				{
					ToolCall: &acp.SessionUpdateToolCall{
						ToolCallId: "tc-1",
						Title:      "Read File",
						Meta: map[string]any{
							"usage": map[string]any{
								"input_tokens":  float64(50),
								"output_tokens": float64(25),
								"total_tokens":  float64(75),
							},
						},
					},
				},
			},
			expectedInput:  50,
			expectedOutput: 25,
			expectedTotal:  75,
		},
		"returns last usage (cumulative)": {
			updates: []acp.SessionUpdate{
				{
					AgentMessageChunk: &acp.SessionUpdateAgentMessageChunk{
						Content: acp.TextBlock("chunk1"),
						Meta: map[string]any{
							"usage": map[string]any{
								"input_tokens":  float64(10),
								"output_tokens": float64(5),
								"total_tokens":  float64(15),
							},
						},
					},
				},
				{
					AgentMessageChunk: &acp.SessionUpdateAgentMessageChunk{
						Content: acp.TextBlock("chunk2"),
						Meta: map[string]any{
							"usage": map[string]any{
								"input_tokens":  float64(100),
								"output_tokens": float64(50),
								"total_tokens":  float64(150),
							},
						},
					},
				},
			},
			expectedInput:  100,
			expectedOutput: 50,
			expectedTotal:  150,
		},
		"optional fields": {
			updates: []acp.SessionUpdate{
				{
					AgentMessageChunk: &acp.SessionUpdateAgentMessageChunk{
						Content: acp.TextBlock("hello"),
						Meta: map[string]any{
							"usage": map[string]any{
								"input_tokens":        float64(100),
								"output_tokens":       float64(50),
								"total_tokens":        float64(150),
								"thought_tokens":      float64(30),
								"cached_read_tokens":  float64(20),
								"cached_write_tokens": float64(10),
							},
						},
					},
				},
			},
			expectedInput:       100,
			expectedOutput:      50,
			expectedTotal:       150,
			expectedThought:     int64Ptr(30),
			expectedCachedRead:  int64Ptr(20),
			expectedCachedWrite: int64Ptr(10),
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			result := ExtractUsageFromMeta(tc.updates)

			if tc.expectNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			assert.Equal(t, tc.expectedInput, result.InputTokens)
			assert.Equal(t, tc.expectedOutput, result.OutputTokens)
			assert.Equal(t, tc.expectedTotal, result.TotalTokens)
			assert.Equal(t, tc.expectedThought, result.ThoughtTokens)
			assert.Equal(t, tc.expectedCachedRead, result.CachedReadTokens)
			assert.Equal(t, tc.expectedCachedWrite, result.CachedWriteTokens)
		})
	}
}

func TestExtractUsageFromPromptResponse(t *testing.T) {
	tt := map[string]struct {
		resp           acp.PromptResponse
		expectNil      bool
		expectedInput  int64
		expectedOutput int64
		expectedTotal  int64
	}{
		"nil meta": {
			resp:      acp.PromptResponse{Meta: nil},
			expectNil: true,
		},
		"with usage": {
			resp: acp.PromptResponse{
				Meta: map[string]any{
					"usage": map[string]any{
						"input_tokens":  float64(500),
						"output_tokens": float64(200),
						"total_tokens":  float64(700),
					},
				},
			},
			expectedInput:  500,
			expectedOutput: 200,
			expectedTotal:  700,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			result := ExtractUsageFromPromptResponse(tc.resp)

			if tc.expectNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			assert.Equal(t, tc.expectedInput, result.InputTokens)
			assert.Equal(t, tc.expectedOutput, result.OutputTokens)
			assert.Equal(t, tc.expectedTotal, result.TotalTokens)
		})
	}
}
