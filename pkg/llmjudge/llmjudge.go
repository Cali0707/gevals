package llmjudge

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/mcpchecker/mcpchecker/pkg/agent"
	"github.com/mcpchecker/mcpchecker/pkg/tokens"
)

type LLMJudge interface {
	EvaluateText(ctx context.Context, judgeConfig *LLMJudgeStepConfig, prompt, output string) (*LLMJudgeResult, error)
	ModelName() string
	Close() error
}

type LLMJudgeResult struct {
	Passed          bool          `json:"passed"`
	Reason          string        `json:"reason"`
	FailureCategory string        `json:"failureCategory"`
	Usage           *tokens.Usage `json:"usage,omitempty"`
}

type llmJudge struct {
	runner agent.Runner
	name   string
	server *judgeServer
	cancel context.CancelFunc
}

type noopLLMJudge struct{}

func (n *noopLLMJudge) EvaluateText(ctx context.Context, judgeConfig *LLMJudgeStepConfig, prompt, output string) (*LLMJudgeResult, error) {
	return &LLMJudgeResult{
		Passed:          true,
		Reason:          "noop judge always passes",
		FailureCategory: "n/a",
	}, nil
}

func (n *noopLLMJudge) ModelName() string {
	return "noop"
}

func (n *noopLLMJudge) Close() error {
	return nil
}

func NewLLMJudge(cfg *LLMJudgeEvalConfig) (LLMJudge, error) {
	if cfg == nil {
		return &noopLLMJudge{}, nil
	}

	ref := cfg.AgentRef

	// Deprecated: translate env config to agent ref
	if ref == nil && cfg.Env != nil {
		var err error
		ref, err = translateEnvToAgentRef(cfg.Env)
		if err != nil {
			return nil, err
		}
	}

	if ref == nil {
		return nil, fmt.Errorf("llm judge requires either an agent ref or env config")
	}

	// Resolve agent ref to spec, then to runner
	spec, err := agent.ResolveAgentRef(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve judge agent ref: %w", err)
	}

	runner, err := agent.NewRunnerForSpec(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to create judge agent runner: %w", err)
	}

	// Start the judge MCP server
	server := newJudgeServer()
	serverCtx, cancel := context.WithCancel(context.Background())

	go func() {
		_ = server.Run(serverCtx)
	}()

	if err := server.WaitReady(serverCtx); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start judge server: %w", err)
	}

	return &llmJudge{
		runner: runner,
		name:   runner.AgentName(),
		server: server,
		cancel: cancel,
	}, nil
}

func (j *llmJudge) EvaluateText(ctx context.Context, judgeConfig *LLMJudgeStepConfig, prompt, output string) (*LLMJudgeResult, error) {
	systemPrompt, err := BuildSystemPrompt(SystemPromptData{
		EvaluationMode:  judgeConfig.EvaluationMode(),
		ReferenceAnswer: judgeConfig.ReferenceAnswer(),
	})
	if err != nil {
		return nil, err
	}

	userPrompt, err := BuildUserPrompt(UserPromptData{
		UserPrompt:    prompt,
		ModelResponse: output,
	})
	if err != nil {
		return nil, err
	}

	combinedPrompt := systemPrompt + "\n\n" + userPrompt

	requestID := uuid.New().String()
	resultCh := j.server.RegisterRequest(requestID)
	defer j.server.DeregisterRequest(requestID)

	manager := &judgeServerManager{server: j.server, requestID: requestID}
	judgeRunner := j.runner.WithMcpServerInfo(manager)

	result, err := judgeRunner.RunTask(ctx, combinedPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to run judge agent: %w", err)
	}

	estimate := result.GetTokenEstimate()

	select {
	case res := <-resultCh:
		res.Usage = estimate.ToUsage()
		return res, nil
	default:
		return nil, fmt.Errorf("judge agent completed without calling submit_judgement tool")
	}
}

func (j *llmJudge) ModelName() string {
	return j.name
}

func (j *llmJudge) Close() error {
	j.cancel()
	return nil
}

// translateEnvToAgentRef converts deprecated LLMJudgeEnvConfig to an agent.AgentRef.
// It reads the environment variables specified in the config, sets the provider-specific
// env vars that llmagent expects, and returns an AgentRef for builtin.llm-agent.
func translateEnvToAgentRef(env *LLMJudgeEnvConfig) (*agent.AgentRef, error) {
	baseUrl := os.Getenv(env.BaseUrlKey)
	apiKey := os.Getenv(env.ApiKeyKey)
	model := os.Getenv(env.ModelNameKey)

	var missingVars []string
	if baseUrl == "" {
		missingVars = append(missingVars, fmt.Sprintf("%s (base URL)", env.BaseUrlKey))
	}
	if apiKey == "" {
		missingVars = append(missingVars, fmt.Sprintf("%s (API key)", env.ApiKeyKey))
	}
	if model == "" {
		missingVars = append(missingVars, fmt.Sprintf("%s (model name)", env.ModelNameKey))
	}

	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables for LLM judge: %v", missingVars)
	}

	// Set provider env vars so the llmagent picks them up
	setIfEmpty := func(envVar, value string) {
		if os.Getenv(envVar) == "" {
			os.Setenv(envVar, value)
		}
	}
	setIfEmpty("OPENAI_BASE_URL", baseUrl)
	setIfEmpty("OPENAI_API_KEY", apiKey)

	// Prepend "openai:" if bare model name
	providerModel := model
	if !strings.Contains(model, ":") {
		providerModel = "openai:" + model
	}

	fmt.Fprintf(os.Stderr, "WARNING: LLM judge env config is deprecated. "+
		"Use agent ref instead: ref: {type: builtin.llm-agent, model: %s}\n", providerModel)

	return &agent.AgentRef{
		Type:  "builtin.llm-agent",
		Model: providerModel,
	}, nil
}
