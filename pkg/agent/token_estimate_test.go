package agent_test

import (
	"testing"

	"github.com/mcpchecker/mcpchecker/pkg/agent"
	"github.com/mcpchecker/mcpchecker/pkg/mcpproxy"
	"github.com/stretchr/testify/assert"
)

func TestRecalculateAggregates(t *testing.T) {
	tt := map[string]struct {
		estimate      agent.TokenEstimate
		expectedInput int64
		expectedOut   int64
		expectedTotal int64
	}{
		"sums all component fields": {
			estimate: agent.TokenEstimate{
				PromptTokens:          100,
				MessageTokens:         50,
				ThinkingTokens:        30,
				ToolInputTokens:       20,
				ToolOutputTokens:      80,
				McpSchemaTokens:       40,
				ResourceInputTokens:   10,
				ResourceOutputTokens:  15,
				PromptGetInputTokens:  5,
				PromptGetOutputTokens: 25,
				Source:                 agent.TokenSourceEstimated,
			},
			// Input = Prompt(100) + ToolOutput(80) + McpSchema(40) + ResourceOutput(15) + PromptGetOutput(25)
			expectedInput: 260,
			// Output = Message(50) + Thinking(30) + ToolInput(20) + ResourceInput(10) + PromptGetInput(5)
			expectedOut:   115,
			expectedTotal: 375,
		},
		"no-op when source is actual": {
			estimate: agent.TokenEstimate{
				InputTokens:  999,
				OutputTokens: 888,
				TotalTokens:  1887,
				PromptTokens: 100,
				Source:        agent.TokenSourceActual,
			},
			expectedInput: 999,
			expectedOut:   888,
			expectedTotal: 1887,
		},
		"recalculates when source is empty": {
			estimate: agent.TokenEstimate{
				PromptTokens:  50,
				MessageTokens: 30,
			},
			expectedInput: 50,
			expectedOut:   30,
			expectedTotal: 80,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			tc.estimate.RecalculateAggregates()

			assert.Equal(t, tc.expectedInput, tc.estimate.InputTokens)
			assert.Equal(t, tc.expectedOut, tc.estimate.OutputTokens)
			assert.Equal(t, tc.expectedTotal, tc.estimate.TotalTokens)
		})
	}
}

func TestMergeCallHistory(t *testing.T) {
	tt := map[string]struct {
		estimate                   agent.TokenEstimate
		history                    *mcpproxy.CallHistory
		expectedToolInput          int64
		expectedToolOutput         int64
		expectedPromptGetInput     int64
		expectedPromptGetOutput    int64
		expectedResourceInput      int64
		expectedResourceOutput     int64
	}{
		"nil history is no-op": {
			estimate:          agent.TokenEstimate{ToolInputTokens: 10},
			history:           nil,
			expectedToolInput: 10,
		},
		"merges prompt and resource fields": {
			estimate: agent.TokenEstimate{},
			history: &mcpproxy.CallHistory{
				PromptGets: []*mcpproxy.PromptGet{
					{Tokens: mcpproxy.NewTokenCount(10, 20)},
					{Tokens: mcpproxy.NewTokenCount(5, 15)},
				},
				ResourceReads: []*mcpproxy.ResourceRead{
					{Tokens: mcpproxy.NewTokenCount(30, 40)},
				},
			},
			expectedPromptGetInput:  15,
			expectedPromptGetOutput: 35,
			expectedResourceInput:   30,
			expectedResourceOutput:  40,
		},
		"skips nil token records": {
			estimate: agent.TokenEstimate{},
			history: &mcpproxy.CallHistory{
				PromptGets: []*mcpproxy.PromptGet{
					{Tokens: nil},
					{Tokens: mcpproxy.NewTokenCount(10, 20)},
				},
				ResourceReads: []*mcpproxy.ResourceRead{
					{Tokens: nil},
				},
				ToolCalls: []*mcpproxy.ToolCall{
					{Tokens: nil},
				},
			},
			expectedPromptGetInput:  10,
			expectedPromptGetOutput: 20,
			expectedResourceInput:   0,
			expectedToolInput:       0,
		},
		"uses proxy tool tokens when estimate has zero": {
			estimate: agent.TokenEstimate{ToolInputTokens: 0, ToolOutputTokens: 0},
			history: &mcpproxy.CallHistory{
				ToolCalls: []*mcpproxy.ToolCall{
					{Tokens: mcpproxy.NewTokenCount(100, 200)},
				},
			},
			expectedToolInput:  100,
			expectedToolOutput: 200,
		},
		"preserves estimate tool tokens when non-zero": {
			estimate: agent.TokenEstimate{ToolInputTokens: 50, ToolOutputTokens: 60},
			history: &mcpproxy.CallHistory{
				ToolCalls: []*mcpproxy.ToolCall{
					{Tokens: mcpproxy.NewTokenCount(999, 999)},
				},
			},
			expectedToolInput:  50,
			expectedToolOutput: 60,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			tc.estimate.MergeCallHistory(tc.history)

			assert.Equal(t, tc.expectedToolInput, tc.estimate.ToolInputTokens)
			assert.Equal(t, tc.expectedToolOutput, tc.estimate.ToolOutputTokens)
			assert.Equal(t, tc.expectedPromptGetInput, tc.estimate.PromptGetInputTokens)
			assert.Equal(t, tc.expectedPromptGetOutput, tc.estimate.PromptGetOutputTokens)
			assert.Equal(t, tc.expectedResourceInput, tc.estimate.ResourceInputTokens)
			assert.Equal(t, tc.expectedResourceOutput, tc.estimate.ResourceOutputTokens)
		})
	}
}

func TestComputeTokenEstimate(t *testing.T) {
	tt := map[string]struct {
		prompt    string
		message   string
		thinking  string
		toolCalls []agent.ToolCallSummary

		expectPromptNonZero    bool
		expectMessageNonZero   bool
		expectThinkingNonZero  bool
		expectToolInputNonZero bool
		expectToolOutNonZero   bool
		expectZeroToolInput    bool
		expectZeroToolOutput   bool
		expectNoError          bool
	}{
		"empty inputs": {
			expectNoError:      true,
			expectZeroToolInput:  true,
			expectZeroToolOutput: true,
		},
		"prompt and message produce tokens": {
			prompt:               "What is the meaning of life?",
			message:              "The answer is 42.",
			thinking:             "Let me think about this question.",
			expectPromptNonZero:  true,
			expectMessageNonZero: true,
			expectThinkingNonZero: true,
			expectZeroToolInput:  true,
			expectZeroToolOutput: true,
			expectNoError:        true,
		},
		"tool calls produce tokens": {
			toolCalls: []agent.ToolCallSummary{
				{
					Title:     "tool1",
					RawInput:  map[string]any{"query": "hello world"},
					RawOutput: map[string]any{"result": "some response"},
				},
			},
			expectToolInputNonZero: true,
			expectToolOutNonZero:   true,
			expectNoError:          true,
		},
		"nil tool call input/output produce zero": {
			toolCalls: []agent.ToolCallSummary{
				{Title: "tool1", RawInput: nil, RawOutput: nil},
			},
			expectZeroToolInput:  true,
			expectZeroToolOutput: true,
			expectNoError:        true,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			estimate := agent.ComputeTokenEstimate(tc.prompt, tc.message, tc.thinking, tc.toolCalls)

			if tc.expectPromptNonZero {
				assert.Greater(t, estimate.PromptTokens, int64(0))
			}
			if tc.expectMessageNonZero {
				assert.Greater(t, estimate.MessageTokens, int64(0))
			}
			if tc.expectThinkingNonZero {
				assert.Greater(t, estimate.ThinkingTokens, int64(0))
			}
			if tc.expectToolInputNonZero {
				assert.Greater(t, estimate.ToolInputTokens, int64(0))
			}
			if tc.expectToolOutNonZero {
				assert.Greater(t, estimate.ToolOutputTokens, int64(0))
			}
			if tc.expectZeroToolInput {
				assert.Equal(t, int64(0), estimate.ToolInputTokens)
			}
			if tc.expectZeroToolOutput {
				assert.Equal(t, int64(0), estimate.ToolOutputTokens)
			}
			if tc.expectNoError {
				assert.Empty(t, estimate.Error)
			}
		})
	}
}

func TestRecalculateAggregates_AfterMerge(t *testing.T) {
	// End-to-end: compute estimate, merge call history, recalculate, verify identity holds.
	estimate := agent.ComputeTokenEstimate("test prompt", "test message", "", nil)

	history := &mcpproxy.CallHistory{
		ToolCalls: []*mcpproxy.ToolCall{
			{Tokens: mcpproxy.NewTokenCount(50, 100)},
		},
		ResourceReads: []*mcpproxy.ResourceRead{
			{Tokens: mcpproxy.NewTokenCount(10, 20)},
		},
	}

	estimate.McpSchemaTokens = 30
	estimate.MergeCallHistory(history)
	estimate.RecalculateAggregates()

	assert.Equal(t,
		estimate.PromptTokens+estimate.ToolOutputTokens+estimate.McpSchemaTokens+
			estimate.ResourceOutputTokens+estimate.PromptGetOutputTokens,
		estimate.InputTokens,
	)
	assert.Equal(t,
		estimate.MessageTokens+estimate.ThinkingTokens+estimate.ToolInputTokens+
			estimate.ResourceInputTokens+estimate.PromptGetInputTokens,
		estimate.OutputTokens,
	)
	assert.Equal(t, estimate.InputTokens+estimate.OutputTokens, estimate.TotalTokens)
	assert.Greater(t, estimate.TotalTokens, int64(0))
}
