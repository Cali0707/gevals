package agent

import (
	"github.com/coder/acp-go-sdk"
	"github.com/mcpchecker/mcpchecker/pkg/tokens"
)

// acpResult is a shared AgentResult implementation for ACP-based runners.
type acpResult struct {
	updates     []acp.SessionUpdate
	prompt      string
	actualUsage *tokens.Usage
}

var _ AgentResult = &acpResult{}

func (res *acpResult) GetOutput() []OutputStep {
	return ExtractOutputSteps(res.updates)
}

func (res *acpResult) getFinalMessage() string {
	return ExtractFinalMessage(res.updates)
}

func (res *acpResult) GetToolCalls() []ToolCallSummary {
	return ExtractToolCalls(res.updates)
}

func (res *acpResult) getThinking() string {
	return ExtractThinking(res.updates)
}

func (res *acpResult) GetRawUpdates() any {
	return res.updates
}

func (res *acpResult) GetTokenEstimate() tokens.Estimate {
	estimate := tokens.ComputeEstimate(
		res.prompt,
		res.getFinalMessage(),
		res.getThinking(),
		toolCallSummaryToToolCallData(res.GetToolCalls()),
	)
	estimate.Source = tokens.SourceEstimated
	estimate.Turns = ExtractTurns(res.updates)

	if res.actualUsage != nil {
		estimate.Source = tokens.SourceActual
		estimate.Actual = res.actualUsage
		estimate.InputTokens = res.actualUsage.InputTokens
		estimate.OutputTokens = res.actualUsage.OutputTokens
		estimate.TotalTokens = res.actualUsage.TotalTokens
	}

	return estimate
}
