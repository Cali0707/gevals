package llmjudge

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/mcpchecker/mcpchecker/pkg/mcpclient"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	judgeRequestHeader    = "X-Judge-Request-Id"
	judgeServerName       = "llm-judge"
	judgeServerVersion    = "1.0.0"
	submitJudgementTool   = "submit_judgement"
	jsonSchemaTypeObject  = "object"
	jsonSchemaTypeString  = "string"
	jsonSchemaTypeBoolean = "boolean"
)

var submitJudgementSchema = jsonschema.Schema{
	Type: jsonSchemaTypeObject,
	Properties: map[string]*jsonschema.Schema{
		"passed": &jsonschema.Schema{
			Type:        jsonSchemaTypeBoolean,
			Description: "Binary result: true for pass, false for fail",
		},
		"reason": &jsonschema.Schema{
			Type:        jsonSchemaTypeString,
			Description: "A detailed explanation for the score, referencing the evaluation criterion and the text",
		},
		"failureCategory": &jsonschema.Schema{
			Type:        jsonSchemaTypeString,
			Description: "If passed is false, specify the reason. Use 'n/a' if passing",
			Enum:        []any{"semantic_mismatch", "missing_information", "contains_extra_info", "n/a"},
		},
	},
	Required: []string{"passed", "reason", "failureCategory"},
}

// judgeServer is a long-running MCP HTTP server exposing the submit_judgement tool.
// it supports concurrent evaluations.
type judgeServer struct {
	url   string
	ready chan struct{}
	done  chan error

	requests sync.Map
}

func newJudgeServer() *judgeServer {
	return &judgeServer{
		ready: make(chan struct{}),
		done:  make(chan error, 1),
	}
}

// Run starts the MCP server and blocks until ctx is cancelled
func (s *judgeServer) Run(ctx context.Context) error {
	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    judgeServerName,
			Version: judgeServerVersion,
		},
		&mcp.ServerOptions{
			Capabilities: &mcp.ServerCapabilities{
				Tools: &mcp.ToolCapabilities{ListChanged: true},
			},
		},
	)

	tool := s.GetSubmitJudgementTool()

	mcpServer.AddTool(tool, s.submitJudgement)

	handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{Stateless: true})

	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		listenErr := fmt.Errorf("failed to listen: %w", err)
		s.done <- listenErr
		return listenErr
	}

	s.url = fmt.Sprintf("http://%s/mcp", listener.Addr().String())
	close(s.ready)

	httpServer := &http.Server{
		Handler: mux,
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	var runErr error
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			runErr = fmt.Errorf("server shutdown failed: %w", err)
		}
	case runErr = <-serverErr:
	}

	s.done <- runErr
	return runErr
}

func (s *judgeServer) WaitReady(ctx context.Context) error {
	select {
	case <-s.ready:
		return nil
	case err := <-s.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *judgeServer) RegisterRequest(id string) <-chan *LLMJudgeResult {
	ch := make(chan *LLMJudgeResult, 1)
	s.requests.Store(id, ch)
	return ch
}

func (s *judgeServer) DeregisterRequest(id string) {
	s.requests.Delete(id)
}

func (s *judgeServer) GetConfig(requestID string) *mcpclient.ServerConfig {
	return &mcpclient.ServerConfig{
		Type: mcpclient.TransportTypeHttp,
		URL:  s.url,
		Headers: map[string]string{
			judgeRequestHeader: requestID,
		},
	}
}

func (s *judgeServer) GetSubmitJudgementTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        submitJudgementTool,
		Title:       "Submit Judgement",
		Description: "Submit the judgement result for evaluation",
		InputSchema: submitJudgementSchema,
	}
}

func (s *judgeServer) submitJudgement(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	extra := req.GetExtra()

	var requestID string
	if extra != nil && extra.Header != nil {
		requestID = extra.Header.Get(judgeRequestHeader)
	}
	if requestID == "" {
		return nil, fmt.Errorf("missing %s header", judgeRequestHeader)
	}

	result := &LLMJudgeResult{}
	if err := json.Unmarshal(req.Params.Arguments, result); err != nil {
		return nil, fmt.Errorf("failed to parse judgement result: %w", err)
	}

	ch, ok := s.requests.Load(requestID)
	if !ok {
		return nil, fmt.Errorf("no registered request for ID %q", requestID)
	}

	select {
	case ch.(chan *LLMJudgeResult) <- result:
	default:
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "judgement submitted succesfully"},
		},
	}, nil
}
