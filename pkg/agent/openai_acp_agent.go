package agent

import (
	"fmt"
	"os"
)

type OpenAIACPAgent struct{}

func (a *OpenAIACPAgent) Name() string {
	return "openai-acp"
}

func (a *OpenAIACPAgent) Description() string {
	return "OpenAI-compatible agent using ACP protocol with in-memory transport"
}

func (a *OpenAIACPAgent) RequiresModel() bool {
	return true
}

func (a *OpenAIACPAgent) ValidateEnvironment() error {
	// No external binary required - we use the openaiagent package directly
	return nil
}

func (a *OpenAIACPAgent) GetDefaults(model string) (*AgentSpec, error) {
	if model == "" {
		return nil, fmt.Errorf("model is required for openai-acp")
	}

	baseURL := os.Getenv("MODEL_BASE_URL")
	apiKey := os.Getenv("MODEL_KEY")

	if baseURL == "" || apiKey == "" {
		return nil, fmt.Errorf("environment variables MODEL_BASE_URL and MODEL_KEY must be set")
	}

	useVirtualHome := false
	return &AgentSpec{
		Metadata: AgentMetadata{
			Name: fmt.Sprintf("openai-acp-%s", model),
		},
		Builtin: &BuiltinRef{
			Type:    "openai-acp",
			Model:   model,
			BaseURL: baseURL,
			APIKey:  apiKey,
		},
		Commands: AgentCommands{
			UseVirtualHome:       &useVirtualHome,
			ArgTemplateMcpServer: "{{ .URL }}",
			RunPrompt:            "",
		},
	}, nil
}
