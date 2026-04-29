package acp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	aghconfig "github.com/pedronauck/agh/internal/config"
)

var (
	// ErrPermissionDenied reports that the configured static policy rejected an operation.
	ErrPermissionDenied = errors.New("acp: permission denied")
	// ErrPathOutsideWorkspace reports that a requested path escapes the session root.
	ErrPathOutsideWorkspace = errors.New("acp: path outside session workspace")
	// ErrToolBlockedForNetworkTurn reports that daemon-side network turn policy rejected a tool operation.
	ErrToolBlockedForNetworkTurn = errors.New("acp: tool blocked for network-originated turn")
	// ErrPendingPermissionNotFound reports that no waiting permission request matched the approval request.
	ErrPendingPermissionNotFound = errors.New("acp: pending permission not found")
	// ErrPendingPermissionConflict reports that a fallback lookup by turn ID matched multiple pending requests.
	ErrPendingPermissionConflict = errors.New("acp: pending permission lookup is ambiguous")
)

type permissionPolicy struct {
	mode aghconfig.PermissionMode
	root string
}

// ApproveRequest resolves one pending permission request.
type ApproveRequest struct {
	RequestID string `json:"request_id,omitempty"`
	TurnID    string `json:"turn_id,omitempty"`
	Decision  string `json:"decision"`
}

// RequestPermissionRequest asks the session permission bridge for a tool-call decision.
type RequestPermissionRequest = acpsdk.RequestPermissionRequest

// RequestPermissionResponse reports the selected or canceled permission outcome.
type RequestPermissionResponse = acpsdk.RequestPermissionResponse

// Validate ensures the approval request can be matched to a pending permission.
func (r ApproveRequest) Validate() error {
	if strings.TrimSpace(r.RequestID) == "" && strings.TrimSpace(r.TurnID) == "" {
		return errors.New("acp: request_id or turn_id is required")
	}
	if _, err := parsePermissionDecision(r.Decision); err != nil {
		return err
	}
	return nil
}

type permissionEventRaw struct {
	RequestID string                  `json:"request_id"`
	Decision  string                  `json:"decision,omitempty"`
	ToolInput json.RawMessage         `json:"tool_input,omitempty"`
	Options   []permissionEventOption `json:"options,omitempty"`
	ToolCall  permissionToolCallRaw   `json:"tool_call"`
}

type permissionEventOption struct {
	Decision string `json:"decision,omitempty"`
	OptionID string `json:"option_id"`
	Kind     string `json:"kind"`
	Label    string `json:"label,omitempty"`
}

type permissionToolCallRaw struct {
	ID        string                    `json:"id"`
	Kind      string                    `json:"kind,omitempty"`
	Title     string                    `json:"title,omitempty"`
	Status    string                    `json:"status,omitempty"`
	Locations []acpsdk.ToolCallLocation `json:"locations,omitempty"`
}

func newPermissionPolicy(mode aghconfig.PermissionMode, root string) (permissionPolicy, error) {
	effectiveMode := mode
	if effectiveMode == "" {
		effectiveMode = aghconfig.PermissionModeApproveReads
	}
	if err := effectiveMode.Validate("permissions.mode"); err != nil {
		return permissionPolicy{}, err
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return permissionPolicy{}, fmt.Errorf("acp: resolve permission root: %w", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return permissionPolicy{}, fmt.Errorf("acp: evaluate permission root %q: %w", absRoot, err)
	}

	return permissionPolicy{
		mode: effectiveMode,
		root: filepath.Clean(resolvedRoot),
	}, nil
}

func (p permissionPolicy) authorize(op permissionOperation) error {
	if p.isAllowed(op) {
		return nil
	}
	return fmt.Errorf("%w: %s blocked by %s", ErrPermissionDenied, op, p.mode)
}

func (p permissionPolicy) isAllowed(op permissionOperation) bool {
	switch p.mode {
	case aghconfig.PermissionModeApproveAll:
		return true
	case aghconfig.PermissionModeApproveReads:
		return op == permissionReadTextFile
	case aghconfig.PermissionModeDenyAll:
		return false
	default:
		return false
	}
}

func (p permissionPolicy) permissionDecision(request acpsdk.RequestPermissionRequest) (permissionDecision, bool) {
	if _, err := p.resolvePathList(request.ToolCall.Locations); err != nil {
		return decisionRejectOnce, false
	}

	switch p.mode {
	case aghconfig.PermissionModeApproveAll:
		return decisionAllowOnce, false
	case aghconfig.PermissionModeApproveReads:
		if request.ToolCall.Kind != nil && *request.ToolCall.Kind == acpsdk.ToolKindRead {
			return decisionAllowOnce, false
		}
		return decisionPending, true
	case aghconfig.PermissionModeDenyAll:
		return decisionPending, true
	default:
		return decisionRejectOnce, false
	}
}

func (p permissionPolicy) resolvePath(requestPath string) (string, error) {
	target := strings.TrimSpace(requestPath)
	if target == "" {
		return "", errors.New("acp: request path is required")
	}

	if !filepath.IsAbs(target) {
		target = filepath.Join(p.root, target)
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("acp: resolve request path %q: %w", requestPath, err)
	}

	resolvedTarget, err := resolveExistingAwarePath(absTarget)
	if err != nil {
		return "", err
	}
	if !isWithinRoot(p.root, resolvedTarget) {
		return "", fmt.Errorf("%w: %s", ErrPathOutsideWorkspace, requestPath)
	}

	return resolvedTarget, nil
}

func (p permissionPolicy) resolvePathList(locations []acpsdk.ToolCallLocation) ([]string, error) {
	if len(locations) == 0 {
		return nil, nil
	}

	resolved := make([]string, 0, len(locations))
	for _, location := range locations {
		path, err := p.resolvePath(location.Path)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, path)
	}
	return resolved, nil
}

func resolveExistingAwarePath(target string) (string, error) {
	cleanTarget := filepath.Clean(target)
	if _, err := os.Stat(cleanTarget); err == nil {
		resolved, resolveErr := filepath.EvalSymlinks(cleanTarget)
		if resolveErr != nil {
			return "", fmt.Errorf("acp: evaluate request path %q: %w", cleanTarget, resolveErr)
		}
		return filepath.Clean(resolved), nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("acp: stat request path %q: %w", cleanTarget, err)
	}

	parent := filepath.Dir(cleanTarget)
	existingParent, err := firstExistingAncestor(parent)
	if err != nil {
		return "", err
	}
	resolvedAncestor, err := filepath.EvalSymlinks(existingParent)
	if err != nil {
		return "", fmt.Errorf("acp: evaluate ancestor %q: %w", existingParent, err)
	}
	relativeParent, err := filepath.Rel(existingParent, parent)
	if err != nil {
		return "", fmt.Errorf("acp: resolve ancestor relationship for %q: %w", cleanTarget, err)
	}
	resolvedParent := filepath.Join(resolvedAncestor, relativeParent)
	return filepath.Join(resolvedParent, filepath.Base(cleanTarget)), nil
}

func firstExistingAncestor(path string) (string, error) {
	current := filepath.Clean(path)
	for {
		if _, err := os.Stat(current); err == nil {
			return current, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("acp: stat ancestor %q: %w", current, err)
		}

		next := filepath.Dir(current)
		if next == current {
			return "", fmt.Errorf("acp: no existing ancestor for %q", path)
		}
		current = next
	}
}

func isWithinRoot(root, target string) bool {
	cleanRoot := filepath.Clean(root)
	cleanTarget := filepath.Clean(target)
	if cleanRoot == cleanTarget {
		return true
	}
	return strings.HasPrefix(cleanTarget, cleanRoot+string(os.PathSeparator))
}

func parsePermissionDecision(raw string) (permissionDecision, error) {
	switch permissionDecision(strings.TrimSpace(raw)) {
	case decisionAllowOnce, decisionAllowAlways, decisionRejectOnce, decisionRejectAlways:
		return permissionDecision(strings.TrimSpace(raw)), nil
	default:
		return "", fmt.Errorf("acp: invalid decision %q", raw)
	}
}

func selectPermissionOutcome(
	options []acpsdk.PermissionOption,
	decision permissionDecision,
) (acpsdk.RequestPermissionOutcome, permissionDecision) {
	var preferred []acpsdk.PermissionOptionKind
	switch decision {
	case decisionAllowAlways:
		preferred = []acpsdk.PermissionOptionKind{
			acpsdk.PermissionOptionKindAllowAlways,
			acpsdk.PermissionOptionKindAllowOnce,
		}
	case decisionAllowOnce:
		preferred = []acpsdk.PermissionOptionKind{
			acpsdk.PermissionOptionKindAllowOnce,
			acpsdk.PermissionOptionKindAllowAlways,
		}
	case decisionRejectAlways:
		preferred = []acpsdk.PermissionOptionKind{
			acpsdk.PermissionOptionKindRejectAlways,
			acpsdk.PermissionOptionKindRejectOnce,
		}
	case decisionRejectOnce:
		preferred = []acpsdk.PermissionOptionKind{
			acpsdk.PermissionOptionKindRejectOnce,
			acpsdk.PermissionOptionKindRejectAlways,
		}
	default:
		return acpsdk.NewRequestPermissionOutcomeCancelled(), ""
	}

	for _, kind := range preferred {
		for _, option := range options {
			if option.Kind == kind {
				return acpsdk.NewRequestPermissionOutcomeSelected(option.OptionId), permissionDecisionFromKind(kind)
			}
		}
	}

	return acpsdk.NewRequestPermissionOutcomeCancelled(), ""
}

func permissionDecisionFromKind(kind acpsdk.PermissionOptionKind) permissionDecision {
	switch kind {
	case acpsdk.PermissionOptionKindAllowOnce:
		return decisionAllowOnce
	case acpsdk.PermissionOptionKindAllowAlways:
		return decisionAllowAlways
	case acpsdk.PermissionOptionKindRejectOnce:
		return decisionRejectOnce
	case acpsdk.PermissionOptionKindRejectAlways:
		return decisionRejectAlways
	default:
		return ""
	}
}

func buildPermissionEventRaw(
	requestID string,
	decision permissionDecision,
	request acpsdk.RequestPermissionRequest,
) json.RawMessage {
	options := make([]permissionEventOption, 0, len(request.Options))
	for _, option := range request.Options {
		options = append(options, permissionEventOption{
			Decision: string(permissionDecisionFromKind(option.Kind)),
			OptionID: string(option.OptionId),
			Kind:     string(option.Kind),
			Label:    option.Name,
		})
	}

	var toolInput json.RawMessage
	if request.ToolCall.RawInput != nil {
		if marshaled, err := json.Marshal(request.ToolCall.RawInput); err == nil {
			toolInput = marshaled
		}
	}

	toolCall := permissionToolCallRaw{
		ID:        strings.TrimSpace(string(request.ToolCall.ToolCallId)),
		Locations: append([]acpsdk.ToolCallLocation(nil), request.ToolCall.Locations...),
	}
	if request.ToolCall.Kind != nil {
		toolCall.Kind = string(*request.ToolCall.Kind)
	}
	if request.ToolCall.Title != nil {
		toolCall.Title = strings.TrimSpace(*request.ToolCall.Title)
	}
	if request.ToolCall.Status != nil {
		toolCall.Status = string(*request.ToolCall.Status)
	}

	payload := permissionEventRaw{
		RequestID: requestID,
		Options:   options,
		ToolInput: toolInput,
		ToolCall:  toolCall,
	}
	if decision != "" && decision != decisionPending {
		payload.Decision = string(decision)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fallbackPermissionEventRaw(requestID, decision)
	}
	return data
}

func (p *AgentProcess) registerPendingPermission(
	turnID string,
	request acpsdk.RequestPermissionRequest,
) (string, *pendingPermission) {
	p.pendingPermissionMu.Lock()
	defer p.pendingPermissionMu.Unlock()

	if p.pendingPermissions == nil {
		p.pendingPermissions = make(map[string]*pendingPermission)
	}
	requestID := p.allocatePermissionRequestIDLocked(turnID, request)
	pending := &pendingPermission{
		requestID: requestID,
		turnID:    strings.TrimSpace(turnID),
		response:  make(chan permissionDecision, 1),
	}
	p.pendingPermissions[requestID] = pending
	return requestID, pending
}

func (p *AgentProcess) nextPermissionRequestID(turnID string, request acpsdk.RequestPermissionRequest) string {
	p.pendingPermissionMu.Lock()
	defer p.pendingPermissionMu.Unlock()

	if p.pendingPermissions == nil {
		p.pendingPermissions = make(map[string]*pendingPermission)
	}
	return p.allocatePermissionRequestIDLocked(turnID, request)
}

func (p *AgentProcess) allocatePermissionRequestIDLocked(
	turnID string,
	request acpsdk.RequestPermissionRequest,
) string {
	base := strings.TrimSpace(permissionRequestIDFromMeta(request.Meta))
	if base == "" {
		toolCallID := strings.TrimSpace(string(request.ToolCall.ToolCallId))
		if toolCallID != "" {
			base = toolCallID
			if strings.TrimSpace(turnID) != "" {
				base = strings.TrimSpace(turnID) + ":" + toolCallID
			}
		}
	}
	if base == "" {
		name := strings.TrimSpace(permissionRequestName(turnID, request))
		if name != "" {
			base = name
		}
	}
	if base == "" {
		base = EventTypePermission
	}

	requestID := base
	for {
		if _, exists := p.pendingPermissions[requestID]; !exists {
			return requestID
		}
		p.permissionRequestSeq++
		requestID = fmt.Sprintf("%s:%d", base, p.permissionRequestSeq)
	}
}

func (p *AgentProcess) clearPendingPermission(requestID string) {
	p.pendingPermissionMu.Lock()
	defer p.pendingPermissionMu.Unlock()

	if p.pendingPermissions == nil {
		return
	}
	delete(p.pendingPermissions, strings.TrimSpace(requestID))
}

// ResolvePermission delivers a user decision to a waiting permission request.
func (p *AgentProcess) ResolvePermission(req ApproveRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}

	decision, err := parsePermissionDecision(req.Decision)
	if err != nil {
		return err
	}

	p.pendingPermissionMu.Lock()
	requestID, pending, err := p.lookupPendingPermissionLocked(req)
	if err != nil {
		p.pendingPermissionMu.Unlock()
		return err
	}
	delete(p.pendingPermissions, requestID)
	p.pendingPermissionMu.Unlock()

	pending.response <- decision
	return nil
}

// RequestPermission reuses the ACP client-side permission path for daemon-originated tool approvals.
func (p *AgentProcess) RequestPermission(
	ctx context.Context,
	req RequestPermissionRequest,
) (RequestPermissionResponse, error) {
	if p == nil {
		return RequestPermissionResponse{}, errors.New("acp: agent process is required")
	}
	if ctx == nil {
		return RequestPermissionResponse{}, errors.New("acp: permission context is required")
	}
	select {
	case <-p.Done():
		return RequestPermissionResponse{}, errors.New("acp: agent process is stopped")
	default:
	}
	return p.handleRequestPermission(ctx, req)
}

func (p *AgentProcess) permissionTimeoutOrDefault() time.Duration {
	if p.permissionTimeout <= 0 {
		return 5 * time.Minute
	}
	return p.permissionTimeout
}

func (p *AgentProcess) lookupPendingPermissionLocked(req ApproveRequest) (string, *pendingPermission, error) {
	if p.pendingPermissions == nil {
		return "", nil, ErrPendingPermissionNotFound
	}

	if requestID := strings.TrimSpace(req.RequestID); requestID != "" {
		pending, ok := p.pendingPermissions[requestID]
		if !ok {
			return "", nil, fmt.Errorf("%w: %s", ErrPendingPermissionNotFound, requestID)
		}
		return requestID, pending, nil
	}

	turnID := strings.TrimSpace(req.TurnID)
	if turnID == "" {
		return "", nil, ErrPendingPermissionNotFound
	}

	var (
		matchedID string
		matched   *pendingPermission
	)
	for requestID, pending := range p.pendingPermissions {
		if pending == nil || pending.turnID != turnID {
			continue
		}
		if matched != nil {
			return "", nil, fmt.Errorf("%w: %s", ErrPendingPermissionConflict, turnID)
		}
		matchedID = requestID
		matched = pending
	}
	if matched == nil {
		return "", nil, fmt.Errorf("%w: %s", ErrPendingPermissionNotFound, turnID)
	}
	return matchedID, matched, nil
}

func permissionRequestIDFromMeta(meta any) string {
	record, ok := meta.(map[string]any)
	if !ok {
		return ""
	}

	for _, key := range []string{"request_id", "requestId", "id"} {
		if value, ok := record[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func permissionRequestName(turnID string, request acpsdk.RequestPermissionRequest) string {
	parts := make([]string, 0, 2)
	if trimmed := strings.TrimSpace(turnID); trimmed != "" {
		parts = append(parts, trimmed)
	}
	if title := toolCallTitle(request.ToolCall); title != "" {
		parts = append(parts, title)
	} else if kind := toolCallKind(request.ToolCall); kind != "" {
		parts = append(parts, kind)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ":")
}

func toolCallTitle(toolCall acpsdk.RequestPermissionToolCall) string {
	if toolCall.Title == nil {
		return ""
	}
	return strings.TrimSpace(*toolCall.Title)
}

func toolCallKind(toolCall acpsdk.RequestPermissionToolCall) string {
	if toolCall.Kind == nil {
		return ""
	}
	return strings.TrimSpace(string(*toolCall.Kind))
}
