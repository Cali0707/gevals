# LLM Judge Verification

Instead of verifying task results with scripts, you can use an LLM judge to semantically evaluate the agent's response. This is useful when:

- The expected output format may vary but the meaning should be consistent
- You want to check if the response contains specific information
- The agent provides text responses rather than performing observable actions

In the v1alpha2 task format, LLM judge steps can be combined with other verification steps in the same task.

## Setup

Configure the LLM judge in your `eval.yaml` using an agent ref:

```yaml
config:
  llmJudge:
    ref:
      type: builtin.llm-agent
      model: "openai:gpt-4o"
```

The `ref` field accepts any valid agent ref. The `type` can be:

- `builtin.llm-agent` — uses a built-in LLM agent. Requires a `model` in `provider:model-id` format (e.g., `openai:gpt-4o`, `anthropic:claude-sonnet-4-20250514`).
- `builtin.claude-code` — uses Claude Code as the judge agent.
- `file` — uses a custom agent configuration file specified by `path`.

Set the appropriate environment variables for your provider before running. For example, for OpenAI:

```bash
export OPENAI_API_KEY="sk-..."
```

### Deprecated: env-based config

The previous `env`-based configuration is still supported but deprecated. If you are using it, you will see a warning at runtime suggesting migration to the agent ref format.

```yaml
# Deprecated — use ref instead
config:
  llmJudge:
    env:
      baseUrlKey: JUDGE_BASE_URL
      apiKeyKey: JUDGE_API_KEY
      modelNameKey: JUDGE_MODEL_NAME
```

## Evaluation Modes

### Contains

Checks whether the agent's response semantically contains the expected information. Extra, non-contradictory information is acceptable. Format and phrasing differences are ignored.

Use this when you want to confirm the response includes specific facts without requiring an exact match.

### Exact

Checks whether the agent's response is semantically equivalent to a reference answer. Simple rephrasing is acceptable (e.g., "Paris is the capital" vs "The capital is Paris"), but adding or omitting information will fail.

Use this when you need precise semantic equivalence.

## Usage in Tasks (v1alpha2)

In the v1alpha2 format, `llmJudge` is a step type in the verify phase. You can use it alongside other verification steps:

```yaml
kind: Task
apiVersion: mcpchecker/v1alpha2
metadata:
  name: "check-image-version"
  difficulty: easy

spec:
  verify:
    # Script-based check
    - script:
        file: ./verify-pod-running.sh

    # LLM judge check (in the same task)
    - llmJudge:
        contains: "mysql:8.0.36"

  prompt:
    inline: What container image is the web-server pod running?
```

Using `exact` mode:

```yaml
spec:
  verify:
    - llmJudge:
        exact: "The pod web-server is running in namespace test-ns"
```

## Usage in Tasks (v1alpha1 / Legacy)

In the legacy format, LLM judge verification replaces script-based verification -- you cannot use both in the same task:

```yaml
kind: Task
metadata:
  name: "check-image-version"
  difficulty: easy
steps:
  verify:
    contains: "mysql:8.0.36"
  prompt:
    inline: What container image is the web-server pod running?
```

## Implementation Details

The LLM judge runs as an agent via the agent framework. An internal MCP server exposes a `submit_judgement` tool that the judge agent calls to return its structured verdict (passed, reason, failure category). Both evaluation modes use the same approach — the difference is in the system prompt given to the judge. See [`pkg/llmjudge/prompts.go`](../../pkg/llmjudge/prompts.go) for the prompt templates.
