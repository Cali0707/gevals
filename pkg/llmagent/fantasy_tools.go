package llmagent

import (
	"context"
	"encoding/json"
	"fmt"

	"charm.land/fantasy"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolsFromServers adapts all of the tools available in the mcp clients to fantasy.AgentTool
func ToolsFromMcpClients(clients []McpClient, interceptor toolInterceptor) []fantasy.AgentTool {
	var tools []fantasy.AgentTool
	for _, client := range clients {
		for _, tool := range client.GetTools() {
			tools = append(tools, &mcpTool{
				client:      client,
				tool:        tool,
				interceptor: interceptor,
			})
		}
	}

	return tools
}

type mcpTool struct {
	client      McpClient
	tool        mcpsdk.Tool
	interceptor toolInterceptor
}

var _ fantasy.AgentTool = &mcpTool{}

// toolInterceptor is called before a tool is executed, and determines whether the call is allowed to proceed
type toolInterceptor func(ctx context.Context, call fantasy.ToolCall) (bool, error)

func (t *mcpTool) Info() fantasy.ToolInfo {
	info := fantasy.ToolInfo{
		Name:        t.tool.Name,
		Description: t.tool.Description,
	}

	if t.tool.InputSchema != nil {
		if params, ok := t.tool.InputSchema.(map[string]any); ok {
			if _, hasProps := params["properties"]; !hasProps {
				params["properties"] = map[string]any{}
			}
			info.Parameters = params
		}
	} else {
		info.Parameters = map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	return info
}

func (t *mcpTool) Run(ctx context.Context, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	if t.interceptor != nil {
		allowed, err := t.interceptor(ctx, call)
		if err != nil {
			return fantasy.ToolResponse{}, err
		}
		if !allowed {
			return fantasy.NewTextErrorResponse("Tool call was rejected by user"), nil
		}
	}

	var args map[string]any
	if call.Input != "" {
		if err := json.Unmarshal([]byte(call.Input), &args); err != nil {
			return fantasy.ToolResponse{}, fmt.Errorf("failed to parse tool arguments: %w", err)
		}
	}

	result, err := t.client.CallTool(ctx, call.Name, args)
	if err != nil {
		return fantasy.NewTextErrorResponse(fmt.Sprintf("Error calling tool: %v", err)), nil
	}

	return fantasy.NewTextResponse(result), nil
}

func (t *mcpTool) ProviderOptions() fantasy.ProviderOptions     { return nil }
func (t *mcpTool) SetProviderOptions(_ fantasy.ProviderOptions) {}
