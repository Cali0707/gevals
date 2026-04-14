package steps

import "fmt"

// AgentResolver resolves {agent.output} and {agent.prompt} template variables.
// It returns an error if the agent context has not been set yet (i.e. the step
// is running before the agent phase).
type AgentResolver struct {
	agent *AgentContext
}

// NewAgentResolver creates a resolver for agent template variables.
// agent may be nil; Resolve will return an error in that case.
func NewAgentResolver(agent *AgentContext) *AgentResolver {
	return &AgentResolver{agent: agent}
}

// Resolve returns the value for an agent template variable.
// Supported fields: "output" and "prompt".
func (r *AgentResolver) Resolve(fieldName string) (string, error) {
	if r.agent == nil {
		return "", fmt.Errorf("agent context is not available: agent has not run yet")
	}

	switch fieldName {
	case "output":
		return r.agent.Output, nil
	case "prompt":
		return r.agent.Prompt, nil
	default:
		return "", fmt.Errorf("unknown agent field %q: supported fields are \"output\" and \"prompt\"", fieldName)
	}
}
