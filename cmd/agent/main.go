package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mcpchecker/mcpchecker/pkg/llmagent"
	"github.com/spf13/cobra"
)

var (
	model        string
	systemPrompt string
)

var rootCmd = &cobra.Command{
	Use:   "agent-cli",
	Short: "An ACP agent that uses an LLM provider to handle tasks",
	Long: `agent-cli runs as an ACP (Agent Communication Protocol) agent, reading from stdin
and writing to stdout. It supports multiple LLM providers via the "provider:model-id" format.`,
	Example: `  # Run with OpenAI
  agent-cli --model openai:gpt-4o

  # Run with Anthropic
  agent-cli --model anthropic:claude-sonnet-4-20250514

  # Run with a system prompt
  agent-cli --model openai:gpt-4o --system "You are a helpful assistant"`,
	RunE: runAgent,
}

func init() {
	rootCmd.Flags().StringVar(&model, "model", os.Getenv("MODEL"), "Model in provider:model-id format (e.g. openai:gpt-4o) (env: MODEL)")
	rootCmd.Flags().StringVar(&systemPrompt, "system", os.Getenv("SYSTEM_PROMPT"), "System prompt for the agent (env: SYSTEM_PROMPT)")
}

func runAgent(cmd *cobra.Command, args []string) error {
	if model == "" {
		return fmt.Errorf("model is required via --model flag or MODEL environment variable (e.g. openai:gpt-4o)")
	}

	ctx := context.Background()

	agent, err := llmagent.New(ctx, llmagent.Config{
		Model:        model,
		SystemPrompt: systemPrompt,
	})
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	return agent.RunACP(ctx, os.Stdin, os.Stdout)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
