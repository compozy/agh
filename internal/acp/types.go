// Package acp provides the AGH ACP client implementation.
package acp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/environment"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/subprocess"
)

const (
	// EventTypeUserMessage is emitted when the agent echoes a user message chunk.
	EventTypeUserMessage = "user_message"
	// EventTypeSyntheticReentry is emitted for daemon-originated synthetic prompt input.
	EventTypeSyntheticReentry = "synthetic_reentry"
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
	// EventTypeRuntimeProgress is emitted by AGH while a prompt remains active.
	EventTypeRuntimeProgress = "runtime_progress"
	// EventTypeRuntimeWarning is emitted by AGH when active prompt activity is stale.
	EventTypeRuntimeWarning = "runtime_warning"
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
	AdditionalDirs  []string
	Env             []string
	MCPServers      []aghconfig.MCPServer
	Permissions     aghconfig.PermissionMode
	SystemPrompt    string
	ResumeSessionID string
	Launcher        environment.Launcher
	ToolHost        environment.ToolHost
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
	for i, dir := range o.AdditionalDirs {
		trimmed := strings.TrimSpace(dir)
		if trimmed == "" {
			continue
		}
		if !filepath.IsAbs(trimmed) {
			return fmt.Errorf("acp: additional_dirs[%d] must be absolute", i)
		}
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
	TurnID                    string
	Message                   string
	Meta                      PromptMeta
	ActivityReporter          PromptActivityReporter
	ActivityHeartbeatInterval time.Duration
}

// Validate ensures the prompt request can be sent to the agent.
func (r PromptRequest) Validate() error {
	if strings.TrimSpace(r.TurnID) == "" {
		return errors.New("acp: prompt turn id is required")
	}
	if strings.TrimSpace(r.Message) == "" {
		return errors.New("acp: prompt message is required")
	}
	if r.ActivityHeartbeatInterval < 0 {
		return errors.New("acp: prompt activity heartbeat interval must be zero or positive")
	}
	if err := r.Meta.Validate(); err != nil {
		return err
	}
	return nil
}

// PromptActivityReporter receives runtime activity observed while a prompt is in flight.
type PromptActivityReporter func(PromptActivityReport)

// PromptActivityReport is a low-frequency runtime heartbeat emitted by the driver.
type PromptActivityReport struct {
	Timestamp time.Time
	Kind      string
	Detail    string
}

const (
	// PromptTurnSourceUser identifies a daemon prompt that originated from the
	// user-facing prompt surfaces.
	PromptTurnSourceUser = "user"
	// PromptTurnSourceNetwork identifies a daemon prompt that originated from an
	// AGH network envelope delivery.
	PromptTurnSourceNetwork = "network"
	// PromptTurnSourceSynthetic identifies a daemon-owned prompt turn injected by
	// internal runtime code.
	PromptTurnSourceSynthetic = "synthetic"
)

// PromptMeta carries structured, transport-stable metadata for one ACP prompt.
type PromptMeta struct {
	TurnSource string               `json:"turn_source,omitempty"`
	Network    *PromptNetworkMeta   `json:"network,omitempty"`
	Synthetic  *PromptSyntheticMeta `json:"synthetic,omitempty"`
}

// PromptNetworkMeta captures stable AGH network envelope correlation fields.
type PromptNetworkMeta struct {
	MessageID     string `json:"message_id,omitempty"`
	Kind          string `json:"kind,omitempty"`
	Channel       string `json:"channel,omitempty"`
	From          string `json:"from,omitempty"`
	To            string `json:"to,omitempty"`
	InteractionID string `json:"interaction_id,omitempty"`
	ReplyTo       string `json:"reply_to,omitempty"`
	TraceID       string `json:"trace_id,omitempty"`
	CausationID   string `json:"causation_id,omitempty"`
}

// PromptSyntheticMeta captures stable daemon-owned metadata for one synthetic prompt turn.
type PromptSyntheticMeta struct {
	TaskID    string `json:"task_id,omitempty"`
	TaskRunID string `json:"task_run_id,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Summary   string `json:"summary,omitempty"`
}

// Normalize returns a trimmed copy of the prompt metadata.
func (m PromptMeta) Normalize() PromptMeta {
	normalized := PromptMeta{
		TurnSource: strings.TrimSpace(m.TurnSource),
	}
	if m.Network != nil {
		network := m.Network.Normalize()
		if !network.IsZero() {
			normalized.Network = &network
		}
	}
	if m.Synthetic != nil {
		synthetic := m.Synthetic.Normalize()
		if !synthetic.IsZero() {
			normalized.Synthetic = &synthetic
		}
	}
	return normalized
}

// IsZero reports whether the prompt metadata carries any fields.
func (m PromptMeta) IsZero() bool {
	normalized := m.Normalize()
	return normalized.TurnSource == "" && normalized.Network == nil && normalized.Synthetic == nil
}

// Validate ensures the metadata shape is internally consistent.
func (m PromptMeta) Validate() error {
	normalized := m.Normalize()
	switch normalized.TurnSource {
	case "", PromptTurnSourceUser:
		if normalized.Network != nil || normalized.Synthetic != nil {
			return errors.New("acp: user prompt metadata cannot include network or synthetic fields")
		}
		return nil
	case PromptTurnSourceNetwork:
		if normalized.Synthetic != nil {
			return errors.New("acp: network prompt metadata cannot include synthetic fields")
		}
		return nil
	case PromptTurnSourceSynthetic:
		if normalized.Network != nil {
			return errors.New("acp: synthetic prompt metadata cannot include network fields")
		}
		if normalized.Synthetic == nil {
			return errors.New("acp: synthetic prompt metadata requires synthetic fields")
		}
		return normalized.Synthetic.Validate()
	default:
		return fmt.Errorf("acp: invalid prompt turn source %q", normalized.TurnSource)
	}
}

// Normalize returns a trimmed copy of the network metadata.
func (m PromptNetworkMeta) Normalize() PromptNetworkMeta {
	return PromptNetworkMeta{
		MessageID:     strings.TrimSpace(m.MessageID),
		Kind:          strings.TrimSpace(m.Kind),
		Channel:       strings.TrimSpace(m.Channel),
		From:          strings.TrimSpace(m.From),
		To:            strings.TrimSpace(m.To),
		InteractionID: strings.TrimSpace(m.InteractionID),
		ReplyTo:       strings.TrimSpace(m.ReplyTo),
		TraceID:       strings.TrimSpace(m.TraceID),
		CausationID:   strings.TrimSpace(m.CausationID),
	}
}

// IsZero reports whether the network metadata carries any fields.
func (m PromptNetworkMeta) IsZero() bool {
	normalized := m.Normalize()
	return normalized == (PromptNetworkMeta{})
}

// Normalize returns a trimmed copy of the synthetic metadata.
func (m PromptSyntheticMeta) Normalize() PromptSyntheticMeta {
	return PromptSyntheticMeta{
		TaskID:    strings.TrimSpace(m.TaskID),
		TaskRunID: strings.TrimSpace(m.TaskRunID),
		Reason:    strings.TrimSpace(m.Reason),
		Summary:   strings.TrimSpace(m.Summary),
	}
}

// IsZero reports whether the synthetic metadata carries any fields.
func (m PromptSyntheticMeta) IsZero() bool {
	normalized := m.Normalize()
	return normalized == (PromptSyntheticMeta{})
}

// Validate ensures the synthetic metadata carries the minimum wake-up identity.
func (m PromptSyntheticMeta) Validate() error {
	normalized := m.Normalize()
	if normalized.Reason == "" {
		return errors.New("acp: synthetic prompt metadata requires a reason")
	}
	return nil
}

// Caps captures the usable capabilities exposed by an ACP agent.
type Caps struct {
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

// RuntimeActivity is the structured activity payload attached to AGH runtime
// progress and warning events.
type RuntimeActivity struct {
	TurnID             string     `json:"turn_id,omitempty"`
	TurnSource         string     `json:"turn_source,omitempty"`
	TurnStartedAt      *time.Time `json:"turn_started_at,omitempty"`
	LastActivityAt     *time.Time `json:"last_activity_at,omitempty"`
	LastActivityKind   string     `json:"last_activity_kind,omitempty"`
	LastActivityDetail string     `json:"last_activity_detail,omitempty"`
	CurrentTool        string     `json:"current_tool,omitempty"`
	ToolCallID         string     `json:"tool_call_id,omitempty"`
	LastProgressAt     *time.Time `json:"last_progress_at,omitempty"`
	IterationCurrent   int        `json:"iteration_current,omitempty"`
	IterationMax       int        `json:"iteration_max,omitempty"`
	IdleSeconds        int64      `json:"idle_seconds,omitempty"`
	ElapsedSeconds     int64      `json:"elapsed_seconds,omitempty"`
}

// Merge overlays non-nil usage fields from other into the receiver.
func (u TokenUsage) Merge(other TokenUsage) TokenUsage {
	merged := u
	if turnID := strings.TrimSpace(other.TurnID); turnID != "" {
		merged.TurnID = turnID
	}
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
	Failure    *store.SessionFailure
	Synthetic  *PromptSyntheticMeta
	Usage      *TokenUsage
	Runtime    *RuntimeActivity
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
	Caps      Caps
	StartedAt time.Time

	managed       *subprocess.Process
	handle        environment.Handle
	toolHostMu    sync.Mutex
	toolHost      environment.ToolHost
	cmd           *exec.Cmd
	conn          *acpsdk.Connection
	stderr        *lockedBuffer
	processCtx    context.Context
	cancelProcess context.CancelFunc
	permissions   permissionPolicy
	terminals     *terminalManager

	terminalOwnershipMu sync.RWMutex
	terminalOwnership   map[string]terminalOwnership

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

	systemPromptMu   sync.Mutex
	systemPrompt     string
	systemPromptSent bool

	turnSourceProviderMu sync.RWMutex
	turnSourceProvider   func() string
}

type activePromptState struct {
	turnID   string
	events   chan AgentEvent
	activity chan struct{}

	sendMu sync.Mutex
	closed bool

	seenToolCalls        map[string]struct{}
	pendingToolResults   []AgentEvent
	pendingToolResultIDs map[string]struct{}

	usageMu sync.Mutex
	usage   TokenUsage
}

const maxPendingToolResults = 128

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
	return p.done
}

// Wait blocks until the subprocess exits and returns its final error state.
func (p *AgentProcess) Wait() error {
	<-p.Done()
	p.waitMu.RLock()
	defer p.waitMu.RUnlock()
	return p.waitErr
}

// Stderr returns the currently captured stderr output for the subprocess.
func (p *AgentProcess) Stderr() string {
	if p.handle != nil {
		return p.handle.Stderr()
	}
	if p.managed != nil {
		return p.managed.Stderr()
	}
	if p.stderr == nil {
		return ""
	}
	return p.stderr.String()
}

// ToolHost returns the environment-owned tool host used by this process.
func (p *AgentProcess) ToolHost() environment.ToolHost {
	if p == nil {
		return nil
	}
	p.toolHostMu.Lock()
	defer p.toolHostMu.Unlock()
	return p.toolHost
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
		turnID:               turnID,
		events:               make(chan AgentEvent, bufferSize),
		activity:             make(chan struct{}, 1),
		seenToolCalls:        make(map[string]struct{}),
		pendingToolResultIDs: make(map[string]struct{}),
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

// SetTurnSourceProvider configures a daemon-local callback that reports the current turn provenance.
func (p *AgentProcess) SetTurnSourceProvider(provider func() string) {
	if p == nil {
		return
	}

	p.turnSourceProviderMu.Lock()
	defer p.turnSourceProviderMu.Unlock()
	p.turnSourceProvider = provider
}

func (p *AgentProcess) currentTurnSource() string {
	if p == nil {
		return ""
	}

	p.turnSourceProviderMu.RLock()
	provider := p.turnSourceProvider
	p.turnSourceProviderMu.RUnlock()
	if provider == nil {
		return ""
	}
	return strings.TrimSpace(provider())
}

func (p *AgentProcess) isNetworkTurn() bool {
	return p.currentTurnSource() == networkCommandName
}

func (p *AgentProcess) nextPromptText(message string) string {
	userMessage := strings.TrimSpace(message)

	p.systemPromptMu.Lock()
	defer p.systemPromptMu.Unlock()

	systemPrompt := strings.TrimSpace(p.systemPrompt)
	if p.systemPromptSent || systemPrompt == "" {
		return userMessage
	}

	p.systemPromptSent = true
	return strings.TrimSpace(
		"Session instructions (treat as system guidance for this conversation):\n\n" +
			systemPrompt +
			"\n\nUser request:\n\n" +
			userMessage,
	)
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
	if active.deferToolResultLocked(event) {
		return
	}
	if event.Type == EventTypeDone {
		active.flushDeferredToolResultsLocked()
	}
	active.sendEventLocked(event)
	if event.Type == EventTypeToolCall {
		active.markToolCallSeenLocked(event.ToolCallID)
		active.flushDeferredToolResultsForToolLocked(event.ToolCallID)
	}
}

func (a *activePromptState) deferToolResultLocked(event AgentEvent) bool {
	if a == nil || event.Type != EventTypeToolResult {
		return false
	}
	toolCallID := strings.TrimSpace(event.ToolCallID)
	if toolCallID == "" {
		return false
	}
	if _, ok := a.seenToolCalls[toolCallID]; ok {
		return false
	}
	if _, ok := a.pendingToolResultIDs[toolCallID]; ok {
		return true
	}
	if len(a.pendingToolResults) >= maxPendingToolResults {
		a.dropOldestPendingToolResultLocked()
	}
	a.pendingToolResults = append(a.pendingToolResults, event)
	a.pendingToolResultIDs[toolCallID] = struct{}{}
	return true
}

func (a *activePromptState) markToolCallSeenLocked(toolCallID string) {
	if a == nil {
		return
	}
	trimmed := strings.TrimSpace(toolCallID)
	if trimmed == "" {
		return
	}
	a.seenToolCalls[trimmed] = struct{}{}
}

func (a *activePromptState) flushDeferredToolResultsForToolLocked(toolCallID string) {
	if a == nil {
		return
	}
	trimmed := strings.TrimSpace(toolCallID)
	if trimmed == "" || len(a.pendingToolResults) == 0 {
		return
	}

	remaining := a.pendingToolResults[:0]
	remainingIDs := make(map[string]struct{}, len(a.pendingToolResultIDs))
	for _, event := range a.pendingToolResults {
		eventToolCallID := strings.TrimSpace(event.ToolCallID)
		if eventToolCallID == trimmed {
			a.sendEventLocked(event)
			continue
		}
		remaining = append(remaining, event)
		if eventToolCallID != "" {
			remainingIDs[eventToolCallID] = struct{}{}
		}
	}
	a.pendingToolResults = remaining
	a.pendingToolResultIDs = remainingIDs
}

func (a *activePromptState) flushDeferredToolResultsLocked() {
	if a == nil || len(a.pendingToolResults) == 0 {
		return
	}
	pending := a.pendingToolResults
	a.pendingToolResults = nil
	a.pendingToolResultIDs = make(map[string]struct{})
	for _, event := range pending {
		a.sendEventLocked(event)
	}
}

func (a *activePromptState) dropOldestPendingToolResultLocked() {
	if a == nil || len(a.pendingToolResults) == 0 {
		return
	}
	oldest := a.pendingToolResults[0]
	delete(a.pendingToolResultIDs, strings.TrimSpace(oldest.ToolCallID))
	a.pendingToolResults = append(a.pendingToolResults[:0], a.pendingToolResults[1:]...)
}

func (a *activePromptState) sendEventLocked(event AgentEvent) {
	if a == nil {
		return
	}
	a.events <- event
	select {
	case a.activity <- struct{}{}:
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

	dst = append(dst[:len(dst):len(dst)], src...)
	combined := dst
	if len(combined) <= limit {
		return combined
	}

	return append([]byte(nil), combined[len(combined)-limit:]...)
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
