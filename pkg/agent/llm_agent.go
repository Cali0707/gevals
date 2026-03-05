package agent

import "fmt"

// LLMAgent is a builtin agent type that uses the llmagent package with ACP protocol.
type LLMAgent struct{}

func (a *LLMAgent) Name() string {
	return "llm-agent"
}

func (a *LLMAgent) Description() string {
	return "LLM agent using ACP protocol (supports openai, anthropic, gemini, and more)"
}

func (a *LLMAgent) RequiresModel() bool {
	return true
}

func (a *LLMAgent) ValidateEnvironment() error {
	return nil
}

func (a *LLMAgent) GetDefaults(model string) (*AgentSpec, error) {
	if model == "" {
		return nil, fmt.Errorf("model is required for llm-agent (e.g. 'openai:gpt-4o')")
	}

	return &AgentSpec{
		Metadata: AgentMetadata{
			Name: fmt.Sprintf("llm-agent-%s", model),
		},
		Builtin: &BuiltinRef{
			Type:  "llm-agent",
			Model: model,
		},
	}, nil
}
