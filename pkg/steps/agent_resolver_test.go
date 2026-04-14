package steps

import (
	"testing"
)

func TestAgentResolver(t *testing.T) {
	tests := []struct {
		name      string
		agent     *AgentContext
		field     string
		want      string
		expectErr bool
	}{
		{
			name:  "resolve output",
			agent: &AgentContext{Prompt: "do something", Output: "I did it"},
			field: "output",
			want:  "I did it",
		},
		{
			name:  "resolve prompt",
			agent: &AgentContext{Prompt: "do something", Output: "I did it"},
			field: "prompt",
			want:  "do something",
		},
		{
			name:      "nil agent context",
			agent:     nil,
			field:     "output",
			expectErr: true,
		},
		{
			name:      "unknown field",
			agent:     &AgentContext{Prompt: "p", Output: "o"},
			field:     "unknown",
			expectErr: true,
		},
		{
			name:  "empty output returns empty string",
			agent: &AgentContext{Prompt: "p", Output: ""},
			field: "output",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewAgentResolver(tt.agent)
			got, err := resolver.Resolve(tt.field)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Resolve(%q) expected error, got nil", tt.field)
				}
				return
			}
			if err != nil {
				t.Errorf("Resolve(%q) unexpected error = %v", tt.field, err)
				return
			}
			if got != tt.want {
				t.Errorf("Resolve(%q) = %q, want %q", tt.field, got, tt.want)
			}
		})
	}
}
