package llmjudge

import (
	"fmt"

	"github.com/mcpchecker/mcpchecker/pkg/agent"
)

const (
	EvaluationModeExact    = "EXACT"
	EvaluationModeContains = "CONTAINS"
)

type LLMJudgeEvalConfig struct {
	Env      *LLMJudgeEnvConfig `json:"env,omitempty"`
	AgentRef *agent.AgentRef    `json:"ref,omitempty"`
}

type LLMJudgeEnvConfig struct {
	BaseUrlKey   string `json:"baseUrlKey"`
	ApiKeyKey    string `json:"apiKeyKey"`
	ModelNameKey string `json:"modelNameKey"`
}

type LLMJudgeStepConfig struct {
	Contains string `json:"contains,omitempty"`
	Exact    string `json:"exact,omitempty"`
}

func (cfg *LLMJudgeStepConfig) EvaluationMode() string {
	if cfg.Exact != "" {
		return EvaluationModeExact
	}

	return EvaluationModeContains
}

func (cfg *LLMJudgeStepConfig) ReferenceAnswer() string {
	if cfg.Exact != "" {
		return cfg.Exact
	}

	return cfg.Contains
}

func (cfg *LLMJudgeStepConfig) Validate() error {
	if cfg.Exact == "" && cfg.Contains == "" {
		return fmt.Errorf("one of contains or exact must be specified")
	}

	if cfg.Exact != "" && cfg.Contains != "" {
		return fmt.Errorf("only one of contains or exact can be specified, not both")
	}

	return nil
}
