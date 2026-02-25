package agent

import (
	"testing"

	"github.com/mcpchecker/mcpchecker/pkg/mcpproxy"
	"github.com/stretchr/testify/assert"
)

func TestComputeTokenEstimate_NilRawInputOutput(t *testing.T) {
	// Regression: when RawInput/RawOutput are nil (e.g. ACP agent didn't send them),
	// json.Marshal(nil) produces "null" which tokenizes to 1 token. This bogus count
	// then prevents MergeCallHistory from using the real MCP proxy data.
	toolCalls := []ToolCallSummary{
		{Title: "tool1", RawInput: nil, RawOutput: nil},
		{Title: "tool2", RawInput: nil, RawOutput: nil},
		{Title: "tool3", RawInput: nil, RawOutput: nil},
	}

	estimate := ComputeTokenEstimate("", "", "", toolCalls)

	assert.Equal(t, int64(0), estimate.ToolInputTokens, "nil RawInput should contribute 0 tokens, not 1 per tool call")
	assert.Equal(t, int64(0), estimate.ToolOutputTokens, "nil RawOutput should contribute 0 tokens, not 1 per tool call")
}

func TestComputeTokenEstimate_WithRawInputOutput(t *testing.T) {
	toolCalls := []ToolCallSummary{
		{
			Title:     "tool1",
			RawInput:  map[string]any{"query": "hello world"},
			RawOutput: map[string]any{"result": "some response"},
		},
	}

	estimate := ComputeTokenEstimate("", "", "", toolCalls)

	assert.Greater(t, estimate.ToolInputTokens, int64(1), "real RawInput should produce more than 1 token")
	assert.Greater(t, estimate.ToolOutputTokens, int64(1), "real RawOutput should produce more than 1 token")
}

func TestComputeTokenEstimate_MixedNilAndReal(t *testing.T) {
	toolCalls := []ToolCallSummary{
		{Title: "tool1", RawInput: nil, RawOutput: nil},
		{Title: "tool2", RawInput: map[string]any{"query": "test"}, RawOutput: map[string]any{"result": "ok"}},
	}

	estimate := ComputeTokenEstimate("", "", "", toolCalls)

	// Only the second tool call (with real data) should contribute tokens.
	// The nil tool call should add 0, not 1.
	assert.Greater(t, estimate.ToolInputTokens, int64(0))
	assert.Greater(t, estimate.ToolOutputTokens, int64(0))
}

func TestMergeCallHistory_NilRawInputFallsThrough(t *testing.T) {
	// This is the key end-to-end scenario: ACP agent sends tool calls without
	// rawInput/rawOutput, so ComputeTokenEstimate leaves ToolInputTokens=0.
	// MergeCallHistory should then use the MCP proxy's real counts.
	toolCalls := []ToolCallSummary{
		{Title: "tool1", RawInput: nil, RawOutput: nil},
	}
	estimate := ComputeTokenEstimate("", "", "", toolCalls)

	history := &mcpproxy.CallHistory{
		ToolCalls: []*mcpproxy.ToolCall{
			{Tokens: mcpproxy.NewTokenCount(100, 200)},
		},
	}

	estimate.MergeCallHistory(history)

	assert.Equal(t, int64(100), estimate.ToolInputTokens, "should use MCP proxy input tokens when ACP didn't provide rawInput")
	assert.Equal(t, int64(200), estimate.ToolOutputTokens, "should use MCP proxy output tokens when ACP didn't provide rawOutput")
}

func TestMergeCallHistory_RealRawInputSkipsMerge(t *testing.T) {
	// When ACP provides real rawInput/rawOutput, MergeCallHistory should not
	// overwrite them with MCP proxy data (to avoid double-counting).
	toolCalls := []ToolCallSummary{
		{
			Title:     "tool1",
			RawInput:  map[string]any{"query": "hello world this is a test"},
			RawOutput: map[string]any{"result": "some response data"},
		},
	}
	estimate := ComputeTokenEstimate("", "", "", toolCalls)
	originalInput := estimate.ToolInputTokens
	originalOutput := estimate.ToolOutputTokens

	history := &mcpproxy.CallHistory{
		ToolCalls: []*mcpproxy.ToolCall{
			{Tokens: mcpproxy.NewTokenCount(999, 999)},
		},
	}

	estimate.MergeCallHistory(history)

	assert.Equal(t, originalInput, estimate.ToolInputTokens, "should preserve ACP-derived input tokens")
	assert.Equal(t, originalOutput, estimate.ToolOutputTokens, "should preserve ACP-derived output tokens")
}
