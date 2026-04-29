package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	sdkmcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/pedronauck/agh/internal/tools"
)

// HostedProjectionHandler receives projection snapshots from the daemon stream.
type HostedProjectionHandler func(HostedProjectionResponse) error

// HostedProxyClient is the UDS client surface consumed by the hosted stdio proxy.
type HostedProxyClient interface {
	BindHostedMCP(ctx context.Context, req HostedBindRequest) (HostedBindResponse, error)
	HostedMCPProjection(ctx context.Context, bindID string) (HostedProjectionResponse, error)
	StreamHostedMCPProjection(
		ctx context.Context,
		bindID string,
		lastDigest string,
		handler HostedProjectionHandler,
	) error
	CallHostedMCP(ctx context.Context, req HostedCallRequest) (HostedCallResponse, error)
	ReleaseHostedMCP(ctx context.Context, req HostedReleaseRequest) error
}

// HostedProxyOptions configures one `agh tool mcp` stdio proxy process.
type HostedProxyOptions struct {
	SessionID string
	Nonce     string
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	Version   string
}

// RunHostedProxy binds to the daemon and serves hosted AGH tools over MCP stdio.
func RunHostedProxy(ctx context.Context, client HostedProxyClient, opts HostedProxyOptions) error {
	if ctx == nil {
		return errors.New("mcp: proxy context is required")
	}
	if client == nil {
		return errors.New("mcp: proxy client is required")
	}
	sessionID := strings.TrimSpace(opts.SessionID)
	if sessionID == "" {
		return ErrHostedSessionRequired
	}
	nonce := strings.TrimSpace(opts.Nonce)
	if nonce == "" {
		return ErrHostedNonceRequired
	}
	stdin := opts.Stdin
	if stdin == nil {
		return errors.New("mcp: proxy stdin is required")
	}
	stdout := opts.Stdout
	if stdout == nil {
		return errors.New("mcp: proxy stdout is required")
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = io.Discard
	}
	version := strings.TrimSpace(opts.Version)
	if version == "" {
		version = "0.0.0"
	}

	bind, err := client.BindHostedMCP(ctx, HostedBindRequest{SessionID: sessionID, Nonce: nonce})
	if err != nil {
		return err
	}
	defer func() {
		releaseErr := client.ReleaseHostedMCP(context.Background(), HostedReleaseRequest{BindID: bind.BindID})
		if releaseErr != nil {
			log.New(stderr, "", 0).Printf("hosted MCP release failed: %v", releaseErr)
		}
	}()

	mcpServer := server.NewMCPServer(HostedServerName, version, server.WithToolCapabilities(true))
	applyHostedTools(mcpServer, client, bind.BindID, bind.Tools)

	proxyCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var streamWG sync.WaitGroup
	streamWG.Go(func() {
		streamHostedProjection(proxyCtx, client, mcpServer, bind)
	})

	stdio := server.NewStdioServer(mcpServer)
	stdio.SetErrorLogger(log.New(stderr, "", 0))
	err = stdio.Listen(proxyCtx, stdin, stdout)
	cancel()
	streamWG.Wait()
	return err
}

func streamHostedProjection(
	ctx context.Context,
	client HostedProxyClient,
	mcpServer *server.MCPServer,
	initial HostedBindResponse,
) {
	lastDigest := strings.TrimSpace(initial.Digest)
	err := client.StreamHostedMCPProjection(
		ctx,
		initial.BindID,
		lastDigest,
		func(snapshot HostedProjectionResponse) error {
			if strings.TrimSpace(snapshot.Digest) == "" || snapshot.Digest == lastDigest {
				return nil
			}
			lastDigest = snapshot.Digest
			applyHostedTools(mcpServer, client, initial.BindID, snapshot.Tools)
			return nil
		},
	)
	if err != nil && !errors.Is(err, context.Canceled) {
		return
	}
}

func applyHostedTools(
	mcpServer *server.MCPServer,
	client HostedProxyClient,
	bindID string,
	views []tools.ToolView,
) {
	if mcpServer == nil {
		return
	}
	toolsForServer := make([]server.ServerTool, 0, len(views))
	for i := range views {
		view := views[i]
		tool := sdkmcp.Tool{
			Name:            view.Descriptor.ID.String(),
			Description:     hostedToolDescription(view.Descriptor),
			RawInputSchema:  cloneRaw(view.Descriptor.InputSchema),
			RawOutputSchema: cloneRaw(view.Descriptor.OutputSchema),
		}
		toolsForServer = append(toolsForServer, server.ServerTool{
			Tool: tool,
			Handler: func(ctx context.Context, req sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
				return callHostedTool(ctx, client, bindID, req)
			},
		})
	}
	mcpServer.SetTools(toolsForServer...)
}

func callHostedTool(
	ctx context.Context,
	client HostedProxyClient,
	bindID string,
	req sdkmcp.CallToolRequest,
) (*sdkmcp.CallToolResult, error) {
	rawInput, err := rawArguments(req.GetRawArguments())
	if err != nil {
		return sdkmcp.NewToolResultError(err.Error()), nil
	}
	response, err := client.CallHostedMCP(ctx, HostedCallRequest{
		BindID:     bindID,
		ToolName:   req.Params.Name,
		ToolCallID: hostedToolCallID(req),
		Input:      rawInput,
	})
	if err != nil {
		return sdkmcp.NewToolResultError(hostedToolErrorMessage(err)), nil
	}
	return hostedToolResult(response.Result)
}

func hostedToolDescription(descriptor tools.Descriptor) string {
	if title := strings.TrimSpace(descriptor.DisplayTitle); title != "" {
		if description := strings.TrimSpace(descriptor.Description); description != "" {
			return title + "\n\n" + description
		}
		return title
	}
	return strings.TrimSpace(descriptor.Description)
}

func rawArguments(args any) (json.RawMessage, error) {
	if args == nil {
		return json.RawMessage(`{}`), nil
	}
	if raw, ok := args.(json.RawMessage); ok {
		return cloneRaw(raw), nil
	}
	payload, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("mcp: marshal hosted MCP arguments: %w", err)
	}
	if len(payload) == 0 || string(payload) == "null" {
		return json.RawMessage(`{}`), nil
	}
	return json.RawMessage(payload), nil
}

func hostedToolCallID(req sdkmcp.CallToolRequest) string {
	if req.Params.Meta == nil {
		return ""
	}
	if value, ok := req.Params.Meta.AdditionalFields["toolCallId"]; ok {
		if text, ok := value.(string); ok {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func hostedToolResult(result tools.ToolResult) (*sdkmcp.CallToolResult, error) {
	if len(result.Structured) > 0 {
		var structured any
		if err := json.Unmarshal(result.Structured, &structured); err == nil {
			return sdkmcp.NewToolResultStructured(structured, hostedResultFallback(result)), nil
		}
	}
	if len(result.Content) == 0 {
		return sdkmcp.NewToolResultText(hostedResultFallback(result)), nil
	}
	content := make([]sdkmcp.Content, 0, len(result.Content))
	for _, block := range result.Content {
		switch strings.TrimSpace(block.Type) {
		case "text":
			content = append(content, sdkmcp.NewTextContent(block.Text))
		default:
			if len(block.Data) > 0 {
				content = append(content, sdkmcp.NewTextContent(string(block.Data)))
			} else if strings.TrimSpace(block.Text) != "" {
				content = append(content, sdkmcp.NewTextContent(block.Text))
			}
		}
	}
	if len(content) == 0 {
		return sdkmcp.NewToolResultText(hostedResultFallback(result)), nil
	}
	return &sdkmcp.CallToolResult{Content: content}, nil
}

func hostedResultFallback(result tools.ToolResult) string {
	if preview := strings.TrimSpace(result.Preview); preview != "" {
		return preview
	}
	if len(result.Structured) > 0 {
		return string(result.Structured)
	}
	if len(result.Content) > 0 {
		parts := make([]string, 0, len(result.Content))
		for _, block := range result.Content {
			if text := strings.TrimSpace(block.Text); text != "" {
				parts = append(parts, text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	return "{}"
}

func hostedToolErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	var toolErr *tools.ToolError
	if errors.As(err, &toolErr) {
		if len(toolErr.ReasonCodes) > 0 {
			return string(toolErr.ReasonCodes[0]) + ": " + toolErr.Error()
		}
		if toolErr.Code != "" {
			return string(toolErr.Code) + ": " + toolErr.Error()
		}
	}
	return err.Error()
}
