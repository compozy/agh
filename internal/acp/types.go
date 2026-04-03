// Package acp provides the AGH ACP client implementation.
package acp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	aghconfig "github.com/pedronauck/agh/internal/config"
)

const (
	// EventTypeUserMessage is emitted when the agent echoes a user message chunk.
	EventTypeUserMessage = "user_message"
	// EventTypeAgentMessage is emitted for assistant message chunks.
	EventTypeAgentMessage = "agent_message"
	// EventTypeThought is emitted for assistant thought chunks.
	EventTypeThought = "thought"
	// EventTypeToolCall is emitted when a tool call starts or is updated in-flight.
	EventTypeToolCall = "tool_call"
	// EventTypeToolResult is emitted when a tool call finishes.
	EventTypeToolResult = "tool_result"
	// EventTypePlan is emitted for plan updates.
	EventTypePlan = "plan"
	// EventTypePermission is emitted when the daemon applies a permission decision.
	EventTypePermission = "permission"
	// EventTypeUsage is emitted when unstable usage metadata is reported.
	EventTypeUsage = "usage"
	// EventTypeSystem is emitted for system-level ACP updates.
	EventTypeSystem = "system"
	// EventTypeDone is emitted when a prompt turn finishes.
	EventTypeDone = "done"
	// EventTypeError is emitted when prompt processing fails.
	EventTypeError = "error"
)

// StartOpts defines how to launch and initialize an ACP agent process.
type StartOpts struct {
	AgentName       string
	Command         string
	Cwd             string
	Env             []string
	MCPServers      []aghconfig.MCPServer
	Permissions     aghconfig.PermissionMode
	ResumeSessionID string
}

// Validate ensures the start options are minimally usable.
func (o StartOpts) Validate() error {
	switch {
	case strings.TrimSpace(o.AgentName) == "":
		return errors.New("acp: agent name is required")
	case strings.TrimSpace(o.Command) == "":
		return errors.New("acp: command is required")
	case strings.TrimSpace(o.Cwd) == "":
		return errors.New("acp: cwd is required")
	}

	mode := o.Permissions
	if mode == "" {
		mode = aghconfig.PermissionModeApproveReads
	}
	if err := mode.Validate("start.permissions"); err != nil {
		return err
	}

	for i, server := range o.MCPServers {
		if err := server.Validate(fmt.Sprintf("start.mcp_servers[%d]", i)); err != nil {
			return err
		}
	}

	return nil
}

// PromptRequest describes one prompt turn sent to an active ACP session.
type PromptRequest struct {
	TurnID  string
	Message string
}

// Validate ensures the prompt request can be sent to the agent.
func (r PromptRequest) Validate() error {
	switch {
	case strings.TrimSpace(r.TurnID) == "":
		return errors.New("acp: prompt turn id is required")
	case strings.TrimSpace(r.Message) == "":
		return errors.New("acp: prompt message is required")
	default:
		return nil
	}
}

// ACPCaps captures the usable capabilities exposed by an ACP agent.
type ACPCaps struct {
	SupportsLoadSession bool
	SupportedModes      []string
	SupportedModels     []string
}

// TokenUsage captures per-turn usage reported by the agent.
type TokenUsage struct {
	TurnID           string
	InputTokens      *int64
	OutputTokens     *int64
	TotalTokens      *int64
	ThoughtTokens    *int64
	CacheReadTokens  *int64
	CacheWriteTokens *int64
	ContextUsed      *int64
	ContextSize      *int64
	CostAmount       *float64
	CostCurrency     *string
	Timestamp        time.Time
}

// Merge overlays non-nil usage fields from other into the receiver.
func (u TokenUsage) Merge(other TokenUsage) TokenUsage {
	merged := u
	merged.TurnID = firstNonBlank(other.TurnID, merged.TurnID)
	merged.InputTokens = chooseInt64(other.InputTokens, merged.InputTokens)
	merged.OutputTokens = chooseInt64(other.OutputTokens, merged.OutputTokens)
	merged.TotalTokens = chooseInt64(other.TotalTokens, merged.TotalTokens)
	merged.ThoughtTokens = chooseInt64(other.ThoughtTokens, merged.ThoughtTokens)
	merged.CacheReadTokens = chooseInt64(other.CacheReadTokens, merged.CacheReadTokens)
	merged.CacheWriteTokens = chooseInt64(other.CacheWriteTokens, merged.CacheWriteTokens)
	merged.ContextUsed = chooseInt64(other.ContextUsed, merged.ContextUsed)
	merged.ContextSize = chooseInt64(other.ContextSize, merged.ContextSize)
	merged.CostAmount = chooseFloat64(other.CostAmount, merged.CostAmount)
	merged.CostCurrency = chooseString(other.CostCurrency, merged.CostCurrency)
	if !other.Timestamp.IsZero() {
		merged.Timestamp = other.Timestamp
	}
	return merged
}

// IsZero reports whether the usage payload carries any reported values.
func (u TokenUsage) IsZero() bool {
	return u.InputTokens == nil &&
		u.OutputTokens == nil &&
		u.TotalTokens == nil &&
		u.ThoughtTokens == nil &&
		u.CacheReadTokens == nil &&
		u.CacheWriteTokens == nil &&
		u.ContextUsed == nil &&
		u.ContextSize == nil &&
		u.CostAmount == nil &&
		u.CostCurrency == nil
}

// AgentEvent is the stream item exposed to session/.
type AgentEvent struct {
	Type       string
	SessionID  string
	TurnID     string
	RequestID  string
	Timestamp  time.Time
	Text       string
	Title      string
	ToolCallID string
	StopReason string
	Action     string
	Resource   string
	Decision   string
	Error      string
	Usage      *TokenUsage
	Raw        json.RawMessage
}

// AgentProcess represents one running ACP-backed agent subprocess.
type AgentProcess struct {
	PID       int
	AgentName string
	Command   string
	Args      []string
	Cwd       string
	SessionID string
	Caps      ACPCaps
	StartedAt time.Time

	cmd           *exec.Cmd
	conn          *acpsdk.Connection
	stderr        *lockedBuffer
	cancelProcess context.CancelFunc
	permissions   permissionPolicy
	terminals     *terminalManager

	waitMu        sync.RWMutex
	waitErr       error
	done          chan struct{}
	stopRequested bool
	stopMu        sync.RWMutex

	promptMu     sync.RWMutex
	activePrompt *activePromptState

	pendingPermissionMu  sync.Mutex
	pendingPermissions   map[string]*pendingPermission
	permissionRequestSeq uint64
	permissionTimeout    time.Duration
}

type activePromptState struct {
	turnID   string
	events   chan AgentEvent
	activity chan struct{}

	sendMu sync.Mutex
	closed bool

	usageMu sync.Mutex
	usage   TokenUsage
}

type pendingPermission struct {
	requestID string
	turnID    string
	response  chan permissionDecision
}

type lockedBuffer struct {
	mu sync.Mutex
	b  []byte
}

// Done returns a channel that closes when the subprocess exits.
func (p *AgentProcess) Done() <-chan struct{} {
	if p == nil || p.done == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return p.done
}

// Wait blocks until the subprocess exits and returns its final error state.
func (p *AgentProcess) Wait() error {
	if p == nil {
		return errors.New("acp: agent process is required")
	}
	<-p.Done()
	p.waitMu.RLock()
	defer p.waitMu.RUnlock()
	return p.waitErr
}

// Stderr returns the currently captured stderr output for the subprocess.
func (p *AgentProcess) Stderr() string {
	if p == nil || p.stderr == nil {
		return ""
	}
	return p.stderr.String()
}

func (p *AgentProcess) setWaitError(err error) {
	p.waitMu.Lock()
	defer p.waitMu.Unlock()
	p.waitErr = err
}

func (p *AgentProcess) stopWasRequested() bool {
	p.stopMu.RLock()
	defer p.stopMu.RUnlock()
	return p.stopRequested
}

func (p *AgentProcess) markStopRequested() {
	p.stopMu.Lock()
	defer p.stopMu.Unlock()
	p.stopRequested = true
}

func (p *AgentProcess) beginPrompt(turnID string, bufferSize int) (*activePromptState, error) {
	p.promptMu.Lock()
	defer p.promptMu.Unlock()
	if p.activePrompt != nil {
		return nil, errors.New("acp: prompt already in progress")
	}
	if bufferSize <= 0 {
		bufferSize = 1
	}
	active := &activePromptState{
		turnID:   turnID,
		events:   make(chan AgentEvent, bufferSize),
		activity: make(chan struct{}, 1),
	}
	p.activePrompt = active
	return active, nil
}

func (p *AgentProcess) endPrompt(active *activePromptState) {
	if active == nil {
		return
	}

	p.promptMu.Lock()
	if p.activePrompt == active {
		p.activePrompt = nil
	}
	p.promptMu.Unlock()

	active.sendMu.Lock()
	defer active.sendMu.Unlock()
	if !active.closed {
		active.closed = true
		close(active.events)
	}
}

func (p *AgentProcess) currentPrompt() *activePromptState {
	p.promptMu.RLock()
	defer p.promptMu.RUnlock()
	return p.activePrompt
}

func (p *AgentProcess) emitPromptEvent(event AgentEvent) {
	p.promptMu.RLock()
	active := p.activePrompt
	p.promptMu.RUnlock()
	if active == nil {
		return
	}

	active.sendMu.Lock()
	defer active.sendMu.Unlock()
	if active.closed {
		return
	}
	active.events <- event
	select {
	case active.activity <- struct{}{}:
	default:
	}
}

func (p *AgentProcess) mergePromptUsage(update TokenUsage) TokenUsage {
	active := p.currentPrompt()
	if active == nil {
		return update
	}
	active.usageMu.Lock()
	defer active.usageMu.Unlock()
	if active.usage.IsZero() {
		active.usage = update
		return active.usage
	}
	active.usage = active.usage.Merge(update)
	return active.usage
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.b = appendBounded(b.b, p, defaultTerminalOutputLimit)
	return len(p), nil
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.b)
}

func appendBounded(dst []byte, src []byte, limit int) []byte {
	if limit <= 0 {
		return nil
	}

	combined := append(dst, src...)
	if len(combined) <= limit {
		return combined
	}

	return append([]byte(nil), combined[len(combined)-limit:]...)
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func chooseInt64(primary, fallback *int64) *int64 {
	if primary != nil {
		return primary
	}
	return fallback
}

func chooseFloat64(primary, fallback *float64) *float64 {
	if primary != nil {
		return primary
	}
	return fallback
}

func chooseString(primary, fallback *string) *string {
	if primary != nil {
		return primary
	}
	return fallback
}
