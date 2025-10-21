package eval

import (
	"fmt"
	"os"

	"sigs.k8s.io/yaml"

	"github.com/genmcp/gevals/pkg/util"
)

const (
	KindEval = "Eval"
)

type EvalSpec struct {
	Metadata EvalMetadata `json:"metadata"`
	Config   EvalConfig   `json:"config"`
}

type EvalMetadata struct {
	Name string `json:"name"`
}

type EvalConfig struct {
	// Agent and MCP configuration
	AgentFile     string `json:"agentFile"`
	McpConfigFile string `json:"mcpConfigFile"`

	// Advanced mode: different assertion sets
	TaskSets []TaskSet `json:"taskSets,omitempty"`
}

type TaskSet struct {
	// Exactly one of Glob or Path must be set
	Glob string `json:"glob,omitempty"`
	Path string `json:"path,omitempty"`

	Assertions *TaskAssertions `json:"assertions,omitempty"`
}

// TODO: add a custom Verify script for another form of assertion
type TaskAssertions struct {
	// Tool assertions
	ToolsUsed    []ToolAssertion `json:"toolsUsed,omitempty"`
	RequireAny   []ToolAssertion `json:"requireAny,omitempty"`
	ToolsNotUsed []ToolAssertion `json:"toolsNotUsed,omitempty"`
	MinToolCalls *int            `json:"minToolCalls,omitempty"`
	MaxToolCalls *int            `json:"maxToolCalls,omitempty"`

	// Resource assertions
	ResourcesRead    []ResourceAssertion `json:"resourcesRead,omitempty"`
	ResourcesNotRead []ResourceAssertion `json:"resourcesNotReady,omitempty"`

	// Prompt assertions
	PromptsUsed    []PromptAssertion `json:"promptsUsed,omitempty"`
	PromptsNotUsed []PromptAssertion `json:"prompteNotUsed,omitempty"`

	// Order assertions
	CallOrder []CallOrderAssertion `json:"callOrder,omitempty"`

	// Efficiency assertions
	NoDuplicateCalls bool `json:"noDuplicateCalls,omitempty"`
}

type ToolAssertion struct {
	Server string `json:"server"`

	// Exactly one of Tool or ToolPattern should be set
	// If neither is set, matches any tool from the server
	Tool        string `json:"tool,omitempty"`
	ToolPattern string `json:"toolPattern,omitempty"` // regex pattern
}

type ResourceAssertion struct {
	Server string `json:"server"`

	// Exactly one of URI or URIPattern should be set
	// If neither is set, matches any resource from the server
	URI        string `json:"uri,omitempty"`
	URIPattern string `json:"uriPattern,omitempty"` // regex pattern
}

type PromptAssertion struct {
	Server string `json:"server"`

	// Exactly one of Prompt or PromptPattern should be set
	// If neither is set, matches any prompt from the server
	Prompt        string `json:"prompt,omitempty"`
	PromptPattern string `json:"promptPattern,omitempty"`
}

type CallOrderAssertion struct {
	Type   string `json:"type"` // "tool", "resource", "prompt"
	Server string `json:"server"`
	Name   string `json:"name"`
}

func (e *EvalSpec) UnmarshalJSON(data []byte) error {
	return util.UnmarshalWithKind(data, e, KindEval)
}

func Read(data []byte) (*EvalSpec, error) {
	spec := &EvalSpec{}

	err := yaml.Unmarshal(data, spec)
	if err != nil {
		return nil, err
	}

	return spec, nil
}

func FromFile(path string) (*EvalSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file '%s' for evalspec: %w", path, err)
	}

	return Read(data)
}
