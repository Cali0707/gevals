package llmjudge

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

const (
	openaiSeed = 0 // allows for consistent eval results
)

var (
	submitJudgementFunction = openai.FunctionDefinitionParam{
		Name:        "submit_judgement",
		Description: openai.String(""),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"passed": map[string]any{
					"type":        "boolean",
					"description": "Binary result: true for pass, false for fail",
				},
				"reason": map[string]any{
					"type":        "string",
					"description": "A detailed explanation for the score, referencing the evaluation criterion and the text",
				},
				"failureCategory": map[string]any{
					"type":        "string",
					"description": "If passed is false, specify the reason. Use 'n/a' if passing",
					"enum": []string{
						"semantic_mismatch",
						"missing_information",
						"contains_extra_info",
						"n/a",
					},
				},
			},
			"required": []string{"passed", "reason", "failureCategory"},
		},
	}
)

type LLMJudge interface {
	EvaluateText(ctx context.Context, judgeConfig *LLMJudgeTaskConfig, prompt, output string) (*LLMJudgeResult, error)
}

type LLMJudgeResult struct {
	Passed          bool   `json:"passed"`
	Reason          string `json:"reason"`
	FailureCategory string `json:"failureCategory"`
}

type llmJudge struct {
	client openai.Client
	model  string
}

type noopLLMJudge struct{}

func (n *noopLLMJudge) EvaluateText(ctx context.Context, judgeConfig *LLMJudgeTaskConfig, prompt, output string) (*LLMJudgeResult, error) {
	return &LLMJudgeResult{
		Passed:          true,
		Reason:          "noop judge always passes",
		FailureCategory: "n/a",
	}, nil
}

func NewLLMJudge(cfg *LLMJudgeEvalConfig) (LLMJudge, error) {
	if cfg == nil {
		return &noopLLMJudge{}, nil
	}
	if cfg.Env == nil {
		return nil, fmt.Errorf("llm judge env config is required to create an llm judge")
	}
	baseUrl := cfg.BaseUrl()
	apiKey := cfg.ApiKey()
	model := cfg.ModelName()

	var missingVars []string
	if baseUrl == "" {
		missingVars = append(missingVars, fmt.Sprintf("%s (base URL)", cfg.Env.BaseUrlKey))
	}
	if apiKey == "" {
		missingVars = append(missingVars, fmt.Sprintf("%s (API key)", cfg.Env.ApiKeyKey))
	}
	if model == "" {
		missingVars = append(missingVars, fmt.Sprintf("%s (model name)", cfg.Env.ModelNameKey))
	}

	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables for LLM judge: %v", missingVars)
	}

	client := openai.NewClient(
		option.WithBaseURL(baseUrl),
		option.WithAPIKey(apiKey),
	)

	return &llmJudge{
		client: client,
		model:  model,
	}, nil
}

func (j *llmJudge) EvaluateText(ctx context.Context, judgeConfig *LLMJudgeTaskConfig, prompt, output string) (*LLMJudgeResult, error) {
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

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
		Tools: []openai.ChatCompletionToolUnionParam{
			{
				OfFunction: &openai.ChatCompletionFunctionToolParam{
					Function: submitJudgementFunction,
				},
			},
		},
		ToolChoice: openai.ToolChoiceOptionFunctionToolChoice(openai.ChatCompletionNamedToolChoiceFunctionParam{Name: submitJudgementFunction.Name}),
		Seed:       openai.Int(openaiSeed),
		Model:      j.model,
	}

	completion, err := j.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to call llm judge: %w", err)
	}

	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices returned from LLM")
	}

	toolCalls := completion.Choices[0].Message.ToolCalls

	if len(toolCalls) != 1 {
		return nil, fmt.Errorf("failed to call the correct number of tools, expected 1 call, got %d", len(toolCalls))
	}

	toolCall := toolCalls[0]

	if toolCall.Function.Name != submitJudgementFunction.Name {
		return nil, fmt.Errorf("llm judge failed to call '%s' tool, called '%s' instead", submitJudgementFunction.Name, toolCall.Function.Name)
	}

	result := &LLMJudgeResult{}

	err = json.Unmarshal([]byte(toolCall.Function.Arguments), result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall '%s' tool call arguments: %w", submitJudgementFunction.Name, err)
	}

	return result, nil
}
