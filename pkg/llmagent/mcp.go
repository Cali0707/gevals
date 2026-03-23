package llmagent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/mcpchecker/mcpchecker/pkg/mcpclient"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type McpClient interface {
	GetTools() []mcpsdk.Tool
	CallTool(ctx context.Context, name string, arguments map[string]any) (string, error)
	Close() error
}

type mcpClient struct {
	session atomic.Pointer[mcpsdk.ClientSession]
	mu      sync.RWMutex
	tools   []mcpsdk.Tool
}

func NewMcpClient(ctx context.Context, serverURL string, headers http.Header) (McpClient, error) {
	mc := &mcpClient{}

	client := mcpsdk.NewClient(&mcpsdk.Implementation{
		Name:    "mcpchecker-agent",
		Version: "1.0.0",
	}, &mcpsdk.ClientOptions{
		ToolListChangedHandler: func(ctx context.Context, tlcr *mcpsdk.ToolListChangedRequest) {
			_ = mc.reloadTools(ctx)
		},
	})

	transport := &mcpsdk.StreamableClientTransport{
		Endpoint: serverURL,
	}

	if len(headers) > 0 {
		transport.HTTPClient = &http.Client{
			Transport: mcpclient.NewHeaderRoundTripper(headers, nil),
		}
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}

	mc.session.Store(session)
	err = mc.reloadTools(ctx)
	return mc, err
}

func (c *mcpClient) GetTools() []mcpsdk.Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]mcpsdk.Tool, len(c.tools))
	copy(out, c.tools)

	return out
}

func (c *mcpClient) CallTool(ctx context.Context, name string, arguments map[string]any) (string, error) {
	session := c.session.Load()
	if session == nil {
		return "", fmt.Errorf("mcp client session not initialized")
	}
	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      name,
		Arguments: arguments,
	})
	if err != nil {
		return "", fmt.Errorf("failed to call tool %s: %w", name, err)
	}

	// TODO: we should probably only marshal either the content or the structured content (not everything)
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal tool result: %w", err)
	}

	return string(resultBytes), nil
}

func (c *mcpClient) Close() error {
	session := c.session.Load()
	if session == nil {
		return nil
	}
	return session.Close()
}

func (c *mcpClient) reloadTools(ctx context.Context) error {
	session := c.session.Load()
	if session == nil {
		return nil
	}
	result, err := session.ListTools(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	tools := make([]mcpsdk.Tool, 0, len(result.Tools))
	for _, tool := range result.Tools {
		if tool != nil {
			tools = append(tools, *tool)
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.tools = tools

	return nil
}
