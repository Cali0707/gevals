package agent

import (
	"testing"

	"github.com/mcpchecker/mcpchecker/pkg/mcpproxy"
	"github.com/mcpchecker/mcpchecker/pkg/tokens"
	"github.com/stretchr/testify/assert"
)

func TestRecalculateAggregates(t *testing.T) {
	tt := map[string]struct {
		estimate      tokens.Estimate
		history       *mcpproxy.CallHistory
		expectedInput int64
		expectedOut   int64
		expectedTotal int64
	}{
		"no-op when source is actual": {
			estimate: tokens.Estimate{
				InputTokens:  999,
				OutputTokens: 888,
				TotalTokens:  1887,
				PromptTokens: 100,
				Source:       tokens.SourceActual,
			},
			expectedInput: 999,
			expectedOut:   888,
			expectedTotal: 1887,
		},
		"empty turns falls back to simple sum": {
			estimate: tokens.Estimate{
				PromptTokens:          50,
				MessageTokens:         20,
				ThinkingTokens:        10,
				ToolInputTokens:       5,
				ToolOutputTokens:      30,
				McpSchemaTokens:       15,
				ResourceInputTokens:   3,
				ResourceOutputTokens:  7,
				PromptGetInputTokens:  2,
				PromptGetOutputTokens: 8,
			},
			// Input = prompt(50) + toolOutput(30) + mcpSchema(15) + resourceOutput(7) + promptGetOutput(8) = 110
			expectedInput: 110,
			// Output = message(20) + thinking(10) + toolInput(5) + resourceInput(3) + promptGetInput(2) = 40
			expectedOut:   40,
			expectedTotal: 150,
		},
		"nil turns falls back to simple sum": {
			estimate: tokens.Estimate{
				PromptTokens:    100,
				MessageTokens:   50,
				McpSchemaTokens: 20,
			},
			// Input = 100 + 20 = 120
			expectedInput: 120,
			// Output = 50
			expectedOut:   50,
			expectedTotal: 170,
		},
		"single turn no tool calls": {
			estimate: tokens.Estimate{
				PromptTokens:    50,
				McpSchemaTokens: 10,
				Turns: []tokens.TurnTokens{
					{OutputTokens: 30, NumToolCalls: 0},
				},
			},
			// Input = context(50+10) sent once
			expectedInput: 60,
			// Output = turn output(30)
			expectedOut:   30,
			expectedTotal: 90,
		},
		"cumulative with two turns and one tool call": {
			estimate: tokens.Estimate{
				PromptTokens:    100,
				McpSchemaTokens: 40,
				Turns: []tokens.TurnTokens{
					{OutputTokens: 20, NumToolCalls: 1}, // thinking+msg for turn 0
					{OutputTokens: 50, NumToolCalls: 0}, // final response
				},
			},
			history: &mcpproxy.CallHistory{
				ToolCalls: []*mcpproxy.ToolCall{
					{Tokens: mcpproxy.NewTokenCount(10, 80)}, // input=call params, output=result
				},
			},
			// Turn 0: context=140, cumInput+=140, turnOutput=20+10=30, turnResults=80, context→140+30+80=250
			// Turn 1: context=250, cumInput+=250, turnOutput=50
			expectedInput: 390,
			// Turn 0: cumOutput+=30, Turn 1: cumOutput+=50
			expectedOut:   80,
			expectedTotal: 470,
		},
		"parallel tool calls grouped in one turn": {
			estimate: tokens.Estimate{
				PromptTokens:    100,
				McpSchemaTokens: 0,
				Turns: []tokens.TurnTokens{
					{OutputTokens: 10, NumToolCalls: 2}, // two parallel tool calls
					{OutputTokens: 20, NumToolCalls: 0}, // final response
				},
			},
			history: &mcpproxy.CallHistory{
				ToolCalls: []*mcpproxy.ToolCall{
					{Tokens: mcpproxy.NewTokenCount(5, 50)},
					{Tokens: mcpproxy.NewTokenCount(5, 50)},
				},
			},
			// Turn 0: context=100, cumInput+=100, turnOutput=10+5+5=20, turnResults=50+50=100, context→100+20+100=220
			// Turn 1: context=220, cumInput+=220, turnOutput=20
			expectedInput: 320,
			expectedOut:   40,
			expectedTotal: 360,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			tc.estimate.RecalculateAggregates(tc.history)

			assert.Equal(t, tc.expectedInput, tc.estimate.InputTokens)
			assert.Equal(t, tc.expectedOut, tc.estimate.OutputTokens)
			assert.Equal(t, tc.expectedTotal, tc.estimate.TotalTokens)
		})
	}
}

func TestMergeCallHistory(t *testing.T) {
	tt := map[string]struct {
		estimate                tokens.Estimate
		history                 *mcpproxy.CallHistory
		expectedToolInput       int64
		expectedToolOutput      int64
		expectedPromptGetInput  int64
		expectedPromptGetOutput int64
		expectedResourceInput   int64
		expectedResourceOutput  int64
	}{
		"nil history is no-op": {
			estimate:          tokens.Estimate{ToolInputTokens: 10},
			history:           nil,
			expectedToolInput: 10,
		},
		"merges prompt and resource fields": {
			estimate: tokens.Estimate{},
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
			estimate: tokens.Estimate{},
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
			estimate: tokens.Estimate{ToolInputTokens: 0, ToolOutputTokens: 0},
			history: &mcpproxy.CallHistory{
				ToolCalls: []*mcpproxy.ToolCall{
					{Tokens: mcpproxy.NewTokenCount(100, 200)},
				},
			},
			expectedToolInput:  100,
			expectedToolOutput: 200,
		},
		"preserves estimate tool tokens when non-zero": {
			estimate: tokens.Estimate{ToolInputTokens: 50, ToolOutputTokens: 60},
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
		toolCalls []ToolCallSummary

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
			expectNoError:        true,
			expectZeroToolInput:  true,
			expectZeroToolOutput: true,
		},
		"prompt and message produce tokens": {
			prompt:                "What is the meaning of life?",
			message:               "The answer is 42.",
			thinking:              "Let me think about this question.",
			expectPromptNonZero:   true,
			expectMessageNonZero:  true,
			expectThinkingNonZero: true,
			expectZeroToolInput:   true,
			expectZeroToolOutput:  true,
			expectNoError:         true,
		},
		"tool calls produce tokens": {
			toolCalls: []ToolCallSummary{
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
			toolCalls: []ToolCallSummary{
				{Title: "tool1", RawInput: nil, RawOutput: nil},
			},
			expectZeroToolInput:  true,
			expectZeroToolOutput: true,
			expectNoError:        true,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			estimate := tokens.ComputeEstimate(
				tc.prompt,
				tc.message,
				tc.thinking,
				toolCallSummaryToToolCallData(tc.toolCalls),
			)

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
	// End-to-end: compute estimate, merge call history, recalculate with cumulative.
	estimate := tokens.ComputeEstimate("test prompt", "test message", "", nil)

	history := &mcpproxy.CallHistory{
		ToolCalls: []*mcpproxy.ToolCall{
			{Tokens: mcpproxy.NewTokenCount(50, 100)},
		},
		ResourceReads: []*mcpproxy.ResourceRead{
			{Tokens: mcpproxy.NewTokenCount(10, 20)},
		},
	}

	// Simulate two turns: one with a tool call, one final response.
	// Split message tokens roughly between the two turns.
	estimate.Turns = []tokens.TurnTokens{
		{OutputTokens: estimate.MessageTokens / 2, NumToolCalls: 1},
		{OutputTokens: estimate.MessageTokens - estimate.MessageTokens/2, NumToolCalls: 0},
	}

	estimate.McpSchemaTokens = 30
	estimate.MergeCallHistory(history)
	estimate.RecalculateAggregates(history)

	// With cumulative calculation, aggregates should be greater than a simple
	// sum of breakdowns due to context replay amplification.
	assert.Equal(t, estimate.InputTokens+estimate.OutputTokens, estimate.TotalTokens)
	assert.Greater(t, estimate.TotalTokens, int64(0))
	// The prompt+schema should appear in both turns' input (amplified).
	assert.Greater(t, estimate.InputTokens, estimate.PromptTokens+estimate.McpSchemaTokens)
}
