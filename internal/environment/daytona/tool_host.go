package daytona

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/environment"
)

var _ environment.ToolHost = (*daytonaToolHost)(nil)

type daytonaToolHost struct {
	sandbox        sandbox
	transport      transport
	sandboxInfo    sandboxInfo
	root           string
	permission     config.PermissionMode
	terminalsMu    sync.Mutex
	nextTerminal   int
	terminals      map[string]*remoteTerminal
	outputMaxBytes int
}

func newDaytonaToolHost(
	sandbox sandbox,
	transport transport,
	info sandboxInfo,
	root string,
	permission config.PermissionMode,
) (*daytonaToolHost, error) {
	if sandbox == nil {
		return nil, errors.New("environment/daytona: tool host sandbox is required")
	}
	if transport == nil {
		return nil, errors.New("environment/daytona: tool host transport is required")
	}
	if permission == "" {
		permission = config.PermissionModeApproveReads
	}
	if err := permission.Validate("permissions.mode"); err != nil {
		return nil, err
	}
	return &daytonaToolHost{
		sandbox:        sandbox,
		transport:      transport,
		sandboxInfo:    info,
		root:           root,
		permission:     permission,
		terminals:      make(map[string]*remoteTerminal),
		outputMaxBytes: 1 << 20,
	}, nil
}

func (h *daytonaToolHost) ReadTextFile(ctx context.Context, requestPath string) (string, error) {
	if err := h.Authorize(environment.PermissionOperationReadTextFile); err != nil {
		return "", err
	}
	resolved, err := h.ResolvePath(requestPath)
	if err != nil {
		return "", err
	}
	content, err := h.sandbox.ReadFile(ctx, resolved)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (h *daytonaToolHost) WriteTextFile(ctx context.Context, requestPath string, content string) error {
	if err := h.Authorize(environment.PermissionOperationWriteTextFile); err != nil {
		return err
	}
	resolved, err := h.ResolvePath(requestPath)
	if err != nil {
		return err
	}
	if err := h.sandbox.WriteFile(ctx, resolved, []byte(content)); err != nil {
		return err
	}
	return nil
}

func (h *daytonaToolHost) ResolvePath(requestPath string) (string, error) {
	target := strings.TrimSpace(requestPath)
	if target == "" {
		return "", errors.New("environment/daytona: request path is required")
	}
	if !path.IsAbs(target) {
		target = path.Join(h.root, target)
	}
	cleaned := path.Clean(target)
	if !isWithinRemoteRoot(h.root, cleaned) {
		return "", fmt.Errorf("environment/daytona: path %q escapes runtime root %q", requestPath, h.root)
	}
	return cleaned, nil
}

func (h *daytonaToolHost) Authorize(op environment.PermissionOperation) error {
	if h.isAllowed(op) {
		return nil
	}
	return fmt.Errorf("environment/daytona: %s blocked by %s", op, h.permission)
}

func (h *daytonaToolHost) isAllowed(op environment.PermissionOperation) bool {
	switch h.permission {
	case config.PermissionModeApproveAll:
		return true
	case config.PermissionModeApproveReads:
		return op == environment.PermissionOperationReadTextFile
	case config.PermissionModeDenyAll:
		return false
	default:
		return false
	}
}

func (h *daytonaToolHost) PermissionDecision(
	req acpsdk.RequestPermissionRequest,
) (environment.PermissionDecision, bool) {
	for _, location := range req.ToolCall.Locations {
		if _, err := h.ResolvePath(location.Path); err != nil {
			return environment.PermissionDecisionRejectOnce, false
		}
	}
	switch h.permission {
	case config.PermissionModeApproveAll:
		return environment.PermissionDecisionAllowOnce, false
	case config.PermissionModeApproveReads:
		if req.ToolCall.Kind != nil && *req.ToolCall.Kind == acpsdk.ToolKindRead {
			return environment.PermissionDecisionAllowOnce, false
		}
		return environment.PermissionDecisionPending, true
	case config.PermissionModeDenyAll:
		return environment.PermissionDecisionPending, true
	default:
		return environment.PermissionDecisionRejectOnce, false
	}
}

func (h *daytonaToolHost) CreateTerminal(
	ctx context.Context,
	req acpsdk.CreateTerminalRequest,
) (acpsdk.CreateTerminalResponse, error) {
	if err := h.Authorize(environment.PermissionOperationCreateTerminal); err != nil {
		return acpsdk.CreateTerminalResponse{}, err
	}
	command := remoteTerminalCommand(h.root, req)
	session, err := h.transport.Dial(ctx, h.sandboxInfo, command)
	if err != nil {
		return acpsdk.CreateTerminalResponse{}, fmt.Errorf("environment/daytona: create terminal: %w", err)
	}
	terminal := &remoteTerminal{
		session: session,
		done:    make(chan struct{}),
	}
	limit := h.outputMaxBytes
	if req.OutputByteLimit != nil && *req.OutputByteLimit > 0 {
		limit = *req.OutputByteLimit
	}
	h.terminalsMu.Lock()
	h.nextTerminal++
	id := fmt.Sprintf("daytona-%d", h.nextTerminal)
	h.terminals[id] = terminal
	h.terminalsMu.Unlock()

	go terminal.capture(limit)
	return acpsdk.CreateTerminalResponse{TerminalId: id}, nil
}

func (h *daytonaToolHost) KillTerminal(id string) error {
	terminal, err := h.lookupTerminal(id)
	if err != nil {
		return err
	}
	return terminal.session.Close()
}

func (h *daytonaToolHost) TerminalOutput(id string) (string, error) {
	terminal, err := h.lookupTerminal(id)
	if err != nil {
		return "", err
	}
	terminal.mu.Lock()
	defer terminal.mu.Unlock()
	return terminal.output.String(), nil
}

func (h *daytonaToolHost) WaitForTerminalExit(ctx context.Context, id string) (int, error) {
	terminal, err := h.lookupTerminal(id)
	if err != nil {
		return 0, err
	}
	select {
	case <-terminal.done:
	case <-ctx.Done():
		return 0, fmt.Errorf("environment/daytona: wait terminal %q: %w", id, ctx.Err())
	}
	return terminal.exitCode, terminal.err
}

func (h *daytonaToolHost) ReleaseTerminal(id string) error {
	h.terminalsMu.Lock()
	terminal, ok := h.terminals[id]
	if ok {
		delete(h.terminals, id)
	}
	h.terminalsMu.Unlock()
	if !ok {
		return fmt.Errorf("environment/daytona: terminal %q not found", id)
	}
	return terminal.session.Close()
}

func (h *daytonaToolHost) lookupTerminal(id string) (*remoteTerminal, error) {
	h.terminalsMu.Lock()
	defer h.terminalsMu.Unlock()
	terminal, ok := h.terminals[id]
	if !ok {
		return nil, fmt.Errorf("environment/daytona: terminal %q not found", id)
	}
	return terminal, nil
}

type remoteTerminal struct {
	session  transportSession
	mu       sync.Mutex
	output   bytes.Buffer
	done     chan struct{}
	exitCode int
	err      error
}

func (t *remoteTerminal) capture(limit int) {
	_, readErr := ioCopyLimit(&t.output, t.session, limit, &t.mu)
	waitErr := t.session.Wait()
	stderr := t.session.Stderr()
	t.mu.Lock()
	if stderr != "" {
		appendLimited(&t.output, []byte(stderr), limit)
	}
	t.mu.Unlock()
	if readErr != nil && !errors.Is(readErr, context.Canceled) {
		t.err = readErr
	}
	if waitErr != nil {
		t.exitCode = 1
		if t.err == nil {
			t.err = waitErr
		}
	}
	close(t.done)
}

func ioCopyLimit(dst *bytes.Buffer, src transportSession, limit int, mu *sync.Mutex) (int64, error) {
	buf := make([]byte, 32*1024)
	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			mu.Lock()
			appendLimited(dst, buf[:n], limit)
			mu.Unlock()
			total += int64(n)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return total, nil
			}
			return total, err
		}
	}
}

func appendLimited(dst *bytes.Buffer, data []byte, limit int) {
	if limit <= 0 {
		dst.Write(data)
		return
	}
	dst.Write(data)
	if dst.Len() <= limit {
		return
	}
	trim := dst.Len() - limit
	remaining := append([]byte(nil), dst.Bytes()[trim:]...)
	dst.Reset()
	dst.Write(remaining)
}

func isWithinRemoteRoot(root string, target string) bool {
	cleanRoot := path.Clean(root)
	cleanTarget := path.Clean(target)
	if cleanRoot == cleanTarget {
		return true
	}
	return strings.HasPrefix(cleanTarget, cleanRoot+"/")
}
