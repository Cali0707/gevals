package openaiagent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"time"

	acp "github.com/coder/acp-go-sdk"
	"github.com/openai/openai-go/v2"
)

// acpAgent wraps an AIAgent to provide ACP protocol support.
type acpAgent struct {
	agent    *AIAgent
	conn     *acp.AgentSideConnection
	sessions map[acp.SessionId]*acpSession
	mu       sync.Mutex
}

type acpSession struct {
	ctx           context.Context
	sessionCancel context.CancelFunc
	promptCancel  context.CancelFunc
	promptGen     uint64
	mcpClients    []*McpClient
}

// cleanup cancels any active context and closes all MCP clients.
func (s *acpSession) cleanup() {
	if s.promptCancel != nil {
		s.promptCancel()
	}
	if s.sessionCancel != nil {
		s.sessionCancel()
	}
	for _, c := range s.mcpClients {
		c.Close()
	}
}

var _ acp.Agent = (*acpAgent)(nil)

// RunACP runs the agent as an ACP server using the provided I/O streams.
// It blocks until the connection is closed or the context is cancelled.
func RunACP(ctx context.Context, agent *AIAgent, in io.Reader, out io.Writer) error {
	acpAgent := newACPAgent(agent)
	conn := acp.NewAgentSideConnection(acpAgent, out, in)
	acpAgent.SetAgentConnection(conn)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-conn.Done():
		return nil
	}
}

// newACPAgent creates a new ACP agent wrapping the given AIAgent.
func newACPAgent(agent *AIAgent) *acpAgent {
	return &acpAgent{
		agent:    agent,
		sessions: make(map[acp.SessionId]*acpSession),
	}
}

// SetAgentConnection implements acp.AgentConnAware.
func (a *acpAgent) SetAgentConnection(conn *acp.AgentSideConnection) {
	a.conn = conn
}

// Initialize implements acp.Agent.
func (a *acpAgent) Initialize(ctx context.Context, params acp.InitializeRequest) (acp.InitializeResponse, error) {
	return acp.InitializeResponse{
		ProtocolVersion: acp.ProtocolVersionNumber,
		AgentCapabilities: acp.AgentCapabilities{
			LoadSession: false,
			McpCapabilities: acp.McpCapabilities{
				Http: true,
			},
		},
	}, nil
}

// NewSession implements acp.Agent.
func (a *acpAgent) NewSession(ctx context.Context, params acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	sessionID := acp.SessionId(randomID("sess"))

	// Create a cancellable context for this session
	sessionCtx, sessionCancel := context.WithCancel(context.Background())

	// Connect to MCP servers provided in the request
	var mcpClients []*McpClient
	for _, srv := range params.McpServers {
		if srv.Http == nil {
			continue
		}
		client, err := NewMcpClient(ctx, srv.Http.Url)
		if err != nil {
			// Close any clients we've already created
			for _, c := range mcpClients {
				c.Close()
			}
			sessionCancel()
			return acp.NewSessionResponse{}, fmt.Errorf("failed to create MCP client for %s: %w", srv.Http.Name, err)
		}
		if err := client.LoadTools(ctx); err != nil {
			client.Close()
			for _, c := range mcpClients {
				c.Close()
			}
			sessionCancel()
			return acp.NewSessionResponse{}, fmt.Errorf("failed to load tools from MCP server %s: %w", srv.Http.Name, err)
		}
		mcpClients = append(mcpClients, client)
	}

	a.mu.Lock()
	a.sessions[sessionID] = &acpSession{
		ctx:           sessionCtx,
		sessionCancel: sessionCancel,
		mcpClients:    mcpClients,
	}
	a.mu.Unlock()

	return acp.NewSessionResponse{SessionId: sessionID}, nil
}

// Authenticate implements acp.Agent.
func (a *acpAgent) Authenticate(ctx context.Context, params acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	return acp.AuthenticateResponse{}, nil
}

// Cancel implements acp.Agent.
func (a *acpAgent) Cancel(ctx context.Context, params acp.CancelNotification) error {
	a.mu.Lock()
	s := a.sessions[params.SessionId]
	if s != nil {
		delete(a.sessions, params.SessionId)
	}
	a.mu.Unlock()

	if s != nil {
		s.cleanup()
	}
	return nil
}

// SetSessionMode implements acp.Agent.
func (a *acpAgent) SetSessionMode(ctx context.Context, params acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	return acp.SetSessionModeResponse{}, nil
}

// Prompt implements acp.Agent.
func (a *acpAgent) Prompt(ctx context.Context, params acp.PromptRequest) (acp.PromptResponse, error) {
	a.mu.Lock()
	s, ok := a.sessions[params.SessionId]
	a.mu.Unlock()

	if !ok {
		return acp.PromptResponse{}, fmt.Errorf("session %s not found", params.SessionId)
	}

	// Cancel any previous turn
	a.mu.Lock()
	if s.promptCancel != nil {
		cancelPrev := s.promptCancel
		a.mu.Unlock()
		cancelPrev()
	} else {
		a.mu.Unlock()
	}

	// Create a prompt context derived from the session context
	promptCtx, promptCancel := context.WithCancel(s.ctx)

	a.mu.Lock()
	s.promptGen++
	myGen := s.promptGen
	s.promptCancel = promptCancel
	a.mu.Unlock()

	opts := a.buildRunOpts(params.Prompt, params.SessionId, s.mcpClients)
	_, err := a.agent.runTask(promptCtx, opts)

	// Only clear cancel if it's still ours (another Prompt may have started)
	a.mu.Lock()
	if s.promptGen == myGen {
		s.promptCancel = nil
	}
	a.mu.Unlock()

	if err != nil {
		if promptCtx.Err() != nil {
			return acp.PromptResponse{StopReason: acp.StopReasonCancelled}, nil
		}
		return acp.PromptResponse{}, err
	}

	return acp.PromptResponse{StopReason: acp.StopReasonEndTurn}, nil
}

// buildRunOpts constructs runOpts with ACP-aware handlers.
func (a *acpAgent) buildRunOpts(promptParts []acp.ContentBlock, sessionID acp.SessionId, mcpClients []*McpClient) runOpts {
	var prompt string
	for _, p := range promptParts {
		if p.Text != nil {
			prompt += p.Text.Text
		}
	}

	return runOpts{
		prompt:              prompt,
		mcpClients:          mcpClients,
		onNewMessage:        a.onNewMessageHandler(sessionID),
		onNewToolCall:       a.onNewToolCallHandler(sessionID),
		toolCallAllowed:     a.toolCallAllowedHandler(sessionID),
		onToolCallCompleted: a.onToolCallCompletedHandler(sessionID),
	}
}

// onNewMessageHandler returns a handler that streams agent messages to the client.
func (a *acpAgent) onNewMessageHandler(sessionID acp.SessionId) func(ctx context.Context, msg openai.ChatCompletionMessage) error {
	return func(ctx context.Context, msg openai.ChatCompletionMessage) error {
		return a.conn.SessionUpdate(ctx, acp.SessionNotification{
			SessionId: sessionID,
			Update:    acp.UpdateAgentMessageText(msg.Content),
		})
	}
}

// onNewToolCallHandler returns a handler that notifies the client of new tool calls.
func (a *acpAgent) onNewToolCallHandler(sessionID acp.SessionId) func(ctx context.Context, name string, args map[string]any) (string, error) {
	return func(ctx context.Context, name string, args map[string]any) (string, error) {
		id := randomID("tool")

		err := a.conn.SessionUpdate(ctx, acp.SessionNotification{
			SessionId: sessionID,
			Update: acp.StartToolCall(
				acp.ToolCallId(id),
				name,
				acp.WithStartStatus(acp.ToolCallStatusPending),
				acp.WithStartRawInput(args),
			),
		})
		if err != nil {
			return "", err
		}

		return id, nil
	}
}

// toolCallAllowedHandler returns a handler that requests permission for tool calls.
func (a *acpAgent) toolCallAllowedHandler(sessionID acp.SessionId) func(ctx context.Context, id string, args map[string]any) (bool, error) {
	return func(ctx context.Context, id string, args map[string]any) (bool, error) {
		resp, err := a.conn.RequestPermission(ctx, acp.RequestPermissionRequest{
			SessionId: sessionID,
			ToolCall: acp.RequestPermissionToolCall{
				ToolCallId: acp.ToolCallId(id),
				RawInput:   args,
			},
			Options: []acp.PermissionOption{
				{Kind: acp.PermissionOptionKindAllowOnce, Name: "Allow", OptionId: acp.PermissionOptionId("allow")},
				{Kind: acp.PermissionOptionKindRejectOnce, Name: "Reject", OptionId: acp.PermissionOptionId("reject")},
			},
		})
		if err != nil {
			return false, err
		}

		if resp.Outcome.Cancelled != nil || resp.Outcome.Selected == nil {
			return false, nil
		}

		return resp.Outcome.Selected.OptionId == "allow", nil
	}
}

// onToolCallCompletedHandler returns a handler that notifies the client of tool call completion.
func (a *acpAgent) onToolCallCompletedHandler(sessionID acp.SessionId) func(ctx context.Context, id string, output string) error {
	return func(ctx context.Context, id string, output string) error {
		return a.conn.SessionUpdate(ctx, acp.SessionNotification{
			SessionId: sessionID,
			Update: acp.UpdateToolCall(
				acp.ToolCallId(id),
				acp.WithUpdateStatus(acp.ToolCallStatusCompleted),
				acp.WithUpdateRawOutput(output),
			),
		})
	}
}

func randomID(prefix string) string {
	var b [12]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}
