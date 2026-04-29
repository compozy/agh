package mcp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/tools"
)

const (
	// HostedServerName is the single stdio MCP server name injected into ACP sessions.
	HostedServerName = "agh-hosted-tools"
	hostedNonceBytes = 32
	hostedBindBytes  = 24
)

var (
	ErrHostedDisabled          = errors.New("mcp: hosted MCP is disabled")
	ErrHostedSessionRequired   = errors.New("mcp: hosted MCP session id is required")
	ErrHostedNonceRequired     = errors.New("mcp: hosted MCP bind nonce is required")
	ErrHostedNonceInvalid      = errors.New("mcp: hosted MCP bind nonce is invalid")
	ErrHostedNonceExpired      = errors.New("mcp: hosted MCP bind nonce expired")
	ErrHostedBindRequired      = errors.New("mcp: hosted MCP bind id is required")
	ErrHostedBindNotFound      = errors.New("mcp: hosted MCP bind not found")
	ErrHostedPeerInvalid       = errors.New("mcp: hosted MCP peer validation failed")
	ErrHostedBinaryInvalid     = errors.New("mcp: hosted MCP binary validation failed")
	ErrHostedRegistryRequired  = errors.New("mcp: hosted MCP registry is required")
	ErrHostedProjectionChanged = errors.New("mcp: hosted MCP projection changed")
)

// HostedRegistry resolves the daemon-owned tool registry at call time.
type HostedRegistry func() tools.Registry

// HostedConfig configures hosted MCP launch and bind validation.
type HostedConfig struct {
	Enabled        bool
	BindNonceTTL   time.Duration
	ExpectedBinary string
	HomePaths      aghconfig.HomePaths
	Registry       HostedRegistry
	Logger         *slog.Logger
	Now            func() time.Time
	NonceReader    func([]byte) error
}

// HostedService owns session-scoped hosted MCP launch and bind lifecycle.
type HostedService struct {
	mu sync.Mutex

	enabled        bool
	bindNonceTTL   time.Duration
	expectedBinary string
	homePaths      aghconfig.HomePaths
	registry       HostedRegistry
	logger         *slog.Logger
	now            func() time.Time
	nonceReader    func([]byte) error

	launches map[string]*hostedLaunchRecord
	binds    map[string]*hostedBindRecord
}

// HostedLaunchRequest describes one session-bound MCP launch.
type HostedLaunchRequest struct {
	SessionID   string
	WorkspaceID string
	AgentName   string
}

// HostedBindRequest is sent by the stdio proxy immediately after ACP starts it.
type HostedBindRequest struct {
	SessionID string `json:"session_id"`
	Nonce     string `json:"bind_nonce"`
}

// HostedBindResponse proves a hosted proxy has bound to one launch record.
type HostedBindResponse struct {
	BindID string           `json:"bind_id"`
	Scope  tools.Scope      `json:"scope"`
	Tools  []tools.ToolView `json:"tools"`
	Digest string           `json:"digest"`
}

// HostedProjectionResponse is a full session-callable projection snapshot.
type HostedProjectionResponse struct {
	Tools  []tools.ToolView `json:"tools"`
	Digest string           `json:"digest"`
}

// HostedCallRequest invokes one hosted MCP tool through the registry.
type HostedCallRequest struct {
	BindID        string          `json:"bind_id"`
	ToolName      string          `json:"tool_name"`
	ToolCallID    string          `json:"tool_call_id,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	Input         json.RawMessage `json:"input,omitempty"`
}

// HostedCallResponse carries the canonical registry result envelope.
type HostedCallResponse struct {
	Result tools.ToolResult `json:"result"`
}

// HostedReleaseRequest releases one active bind record.
type HostedReleaseRequest struct {
	BindID string `json:"bind_id"`
}

type hostedLaunchRecord struct {
	sessionID     string
	workspaceID   string
	agentName     string
	nonceHash     string
	expiresAt     time.Time
	expectedBin   string
	createdAt     time.Time
	consumed      bool
	correlationID string
}

type hostedBindRecord struct {
	bindID        string
	sessionID     string
	workspaceID   string
	agentName     string
	expectedBin   string
	peer          PeerInfo
	createdAt     time.Time
	correlationID string
}

// NewHostedService builds a hosted MCP lifecycle service.
func NewHostedService(cfg HostedConfig) (*HostedService, error) {
	now := cfg.Now
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	nonceReader := cfg.NonceReader
	if nonceReader == nil {
		nonceReader = func(dst []byte) error {
			_, err := rand.Read(dst)
			return err
		}
	}
	ttl := cfg.BindNonceTTL
	if ttl <= 0 {
		ttl = aghconfig.DefaultHostedMCPBindNonceTTLSeconds * time.Second
	}
	expected, err := normalizeExecutablePath(cfg.ExpectedBinary)
	if err != nil {
		return nil, fmt.Errorf("mcp: normalize hosted MCP binary: %w", err)
	}
	return &HostedService{
		enabled:        cfg.Enabled,
		bindNonceTTL:   ttl,
		expectedBinary: expected,
		homePaths:      cfg.HomePaths,
		registry:       cfg.Registry,
		logger:         cfg.Logger,
		now:            now,
		nonceReader:    nonceReader,
		launches:       make(map[string]*hostedLaunchRecord),
		binds:          make(map[string]*hostedBindRecord),
	}, nil
}

// Launch mints a session-bound, single-use hosted MCP launch record.
func (s *HostedService) Launch(ctx context.Context, req HostedLaunchRequest) (aghconfig.MCPServer, error) {
	if err := ctxErr(ctx); err != nil {
		return aghconfig.MCPServer{}, err
	}
	if s == nil || !s.enabled {
		return aghconfig.MCPServer{}, ErrHostedDisabled
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return aghconfig.MCPServer{}, ErrHostedSessionRequired
	}
	nonce, err := s.randomToken(hostedNonceBytes)
	if err != nil {
		return aghconfig.MCPServer{}, fmt.Errorf("mcp: mint hosted MCP nonce: %w", err)
	}
	correlation, err := s.randomToken(hostedBindBytes)
	if err != nil {
		return aghconfig.MCPServer{}, fmt.Errorf("mcp: mint hosted MCP correlation id: %w", err)
	}
	now := s.now().UTC()
	record := &hostedLaunchRecord{
		sessionID:     sessionID,
		workspaceID:   strings.TrimSpace(req.WorkspaceID),
		agentName:     strings.TrimSpace(req.AgentName),
		nonceHash:     tokenHash(nonce),
		expiresAt:     now.Add(s.bindNonceTTL),
		expectedBin:   s.expectedBinary,
		createdAt:     now,
		correlationID: correlation,
	}
	s.mu.Lock()
	s.launches[sessionID] = record
	s.mu.Unlock()

	env := map[string]string{}
	if home := strings.TrimSpace(s.homePaths.HomeDir); home != "" {
		env["AGH_HOME"] = home
	}
	return aghconfig.MCPServer{
		Name:      HostedServerName,
		Transport: aghconfig.MCPServerTransportStdio,
		Command:   s.expectedBinary,
		Args:      []string{"tool", "mcp", "--session", sessionID, "--bind-nonce", nonce},
		Env:       env,
	}, nil
}

// CancelLaunch removes an unbound launch record for a failed session start.
func (s *HostedService) CancelLaunch(sessionID string) {
	if s == nil {
		return
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	s.mu.Lock()
	if record, ok := s.launches[sessionID]; ok && !record.consumed {
		delete(s.launches, sessionID)
	}
	s.mu.Unlock()
}

// ReleaseSession removes all hosted state for a stopped session.
func (s *HostedService) ReleaseSession(sessionID string) {
	if s == nil {
		return
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	s.mu.Lock()
	delete(s.launches, sessionID)
	for bindID, record := range s.binds {
		if record.sessionID == sessionID {
			delete(s.binds, bindID)
		}
	}
	s.mu.Unlock()
}

// Bind consumes a launch nonce after validating the Unix peer and expected binary.
func (s *HostedService) Bind(ctx context.Context, req HostedBindRequest, peer PeerInfo) (HostedBindResponse, error) {
	if err := ctxErr(ctx); err != nil {
		return HostedBindResponse{}, err
	}
	if s == nil || !s.enabled {
		return HostedBindResponse{}, ErrHostedDisabled
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return HostedBindResponse{}, ErrHostedSessionRequired
	}
	nonce := strings.TrimSpace(req.Nonce)
	if nonce == "" {
		return HostedBindResponse{}, ErrHostedNonceRequired
	}
	if err := s.validatePeer(peer); err != nil {
		return HostedBindResponse{}, err
	}
	bindID, err := s.randomToken(hostedBindBytes)
	if err != nil {
		return HostedBindResponse{}, fmt.Errorf("mcp: mint hosted MCP bind id: %w", err)
	}
	record, err := s.consumeLaunch(sessionID, nonce, bindID, peer)
	if err != nil {
		return HostedBindResponse{}, err
	}
	projection, err := s.projection(ctx, record)
	if err != nil {
		s.ReleaseBind(bindID)
		return HostedBindResponse{}, err
	}
	return HostedBindResponse{
		BindID: bindID,
		Scope:  record.scope(),
		Tools:  projection.Tools,
		Digest: projection.Digest,
	}, nil
}

// Projection returns the current session-callable tool projection for one bind.
func (s *HostedService) Projection(
	ctx context.Context,
	bindID string,
	peer PeerInfo,
) (HostedProjectionResponse, error) {
	record, err := s.recordForBind(ctx, bindID, peer)
	if err != nil {
		return HostedProjectionResponse{}, err
	}
	return s.projection(ctx, record)
}

// Call routes a hosted MCP tool call through the registry dispatch pipeline.
func (s *HostedService) Call(ctx context.Context, req HostedCallRequest, peer PeerInfo) (HostedCallResponse, error) {
	record, err := s.recordForBind(ctx, req.BindID, peer)
	if err != nil {
		return HostedCallResponse{}, err
	}
	toolID := tools.ToolID(strings.TrimSpace(req.ToolName))
	if err := toolID.Validate(); err != nil {
		return HostedCallResponse{}, err
	}
	input := cloneRaw(req.Input)
	if len(input) == 0 {
		input = json.RawMessage(`{}`)
	}
	registry := s.currentRegistry()
	if registry == nil {
		return HostedCallResponse{}, ErrHostedRegistryRequired
	}
	result, err := registry.Call(ctx, record.scope(), tools.CallRequest{
		ToolID:        toolID,
		ToolCallID:    strings.TrimSpace(req.ToolCallID),
		SessionID:     record.sessionID,
		WorkspaceID:   record.workspaceID,
		AgentName:     record.agentName,
		CorrelationID: strings.TrimSpace(req.CorrelationID),
		Input:         input,
	})
	if err != nil {
		return HostedCallResponse{}, err
	}
	return HostedCallResponse{Result: result}, nil
}

// ReleaseBind removes one active bind record.
func (s *HostedService) ReleaseBind(bindID string) {
	if s == nil {
		return
	}
	bindID = strings.TrimSpace(bindID)
	if bindID == "" {
		return
	}
	s.mu.Lock()
	delete(s.binds, bindID)
	s.mu.Unlock()
}

func (s *HostedService) consumeLaunch(
	sessionID string,
	nonce string,
	bindID string,
	peer PeerInfo,
) (*hostedBindRecord, error) {
	now := s.now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	launch, ok := s.launches[sessionID]
	if !ok || launch == nil || launch.consumed || !constantHashEqual(launch.nonceHash, tokenHash(nonce)) {
		return nil, ErrHostedNonceInvalid
	}
	if !launch.expiresAt.After(now) {
		delete(s.launches, sessionID)
		return nil, ErrHostedNonceExpired
	}
	launch.consumed = true
	delete(s.launches, sessionID)
	record := &hostedBindRecord{
		bindID:        bindID,
		sessionID:     launch.sessionID,
		workspaceID:   launch.workspaceID,
		agentName:     launch.agentName,
		expectedBin:   launch.expectedBin,
		peer:          peer,
		createdAt:     now,
		correlationID: launch.correlationID,
	}
	s.binds[bindID] = record
	return record.clone(), nil
}

func (s *HostedService) recordForBind(
	ctx context.Context,
	bindID string,
	peer PeerInfo,
) (*hostedBindRecord, error) {
	if err := ctxErr(ctx); err != nil {
		return nil, err
	}
	if s == nil || !s.enabled {
		return nil, ErrHostedDisabled
	}
	if strings.TrimSpace(bindID) == "" {
		return nil, ErrHostedBindRequired
	}
	if err := s.validatePeer(peer); err != nil {
		return nil, err
	}
	s.mu.Lock()
	record := s.binds[strings.TrimSpace(bindID)]
	s.mu.Unlock()
	if record == nil {
		return nil, ErrHostedBindNotFound
	}
	if err := record.validatePeer(peer); err != nil {
		return nil, err
	}
	return record.clone(), nil
}

func (s *HostedService) projection(ctx context.Context, record *hostedBindRecord) (HostedProjectionResponse, error) {
	if record == nil {
		return HostedProjectionResponse{}, ErrHostedBindNotFound
	}
	registry := s.currentRegistry()
	if registry == nil {
		return HostedProjectionResponse{}, ErrHostedRegistryRequired
	}
	views, err := registry.List(ctx, record.scope())
	if err != nil {
		return HostedProjectionResponse{}, err
	}
	slices.SortFunc(views, func(left, right tools.ToolView) int {
		return strings.Compare(left.Descriptor.ID.String(), right.Descriptor.ID.String())
	})
	return HostedProjectionResponse{
		Tools:  cloneToolViews(views),
		Digest: hostedProjectionDigest(views),
	}, nil
}

func (s *HostedService) validatePeer(peer PeerInfo) error {
	if !peer.Supported {
		return fmt.Errorf("%w: unsupported peer credential inspection", ErrHostedPeerInvalid)
	}
	if peer.PID <= 0 || peer.UID < 0 {
		return fmt.Errorf("%w: missing peer credentials", ErrHostedPeerInvalid)
	}
	if currentUID := os.Getuid(); currentUID >= 0 && peer.UID != currentUID {
		return fmt.Errorf("%w: uid mismatch", ErrHostedPeerInvalid)
	}
	peerBinary, err := normalizeExecutablePath(peer.ExecutablePath)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrHostedBinaryInvalid, err)
	}
	if peerBinary != s.expectedBinary {
		return fmt.Errorf("%w: peer executable mismatch", ErrHostedBinaryInvalid)
	}
	return nil
}

func (s *HostedService) currentRegistry() tools.Registry {
	if s == nil || s.registry == nil {
		return nil
	}
	return s.registry()
}

func (s *HostedService) randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if err := s.nonceReader(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (r *hostedBindRecord) scope() tools.Scope {
	if r == nil {
		return tools.Scope{}
	}
	return tools.Scope{
		SessionID:   r.sessionID,
		WorkspaceID: r.workspaceID,
		AgentName:   r.agentName,
	}
}

func (r *hostedBindRecord) clone() *hostedBindRecord {
	if r == nil {
		return nil
	}
	cloned := *r
	return &cloned
}

func (r *hostedBindRecord) validatePeer(peer PeerInfo) error {
	if r == nil {
		return ErrHostedBindNotFound
	}
	if peer.PID != r.peer.PID || peer.UID != r.peer.UID {
		return fmt.Errorf("%w: peer credential changed", ErrHostedPeerInvalid)
	}
	peerBinary, err := normalizeExecutablePath(peer.ExecutablePath)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrHostedBinaryInvalid, err)
	}
	if peerBinary != r.expectedBin {
		return fmt.Errorf("%w: peer executable mismatch", ErrHostedBinaryInvalid)
	}
	return nil
}

func hostedProjectionDigest(views []tools.ToolView) string {
	payload, err := json.Marshal(views)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func constantHashEqual(left string, right string) bool {
	if len(left) != len(right) {
		return false
	}
	var diff byte
	for i := range left {
		diff |= left[i] ^ right[i]
	}
	return diff == 0
}

func normalizeExecutablePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("executable path is required")
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}
	evaluated, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	return filepath.Clean(evaluated), nil
}

func cloneToolViews(src []tools.ToolView) []tools.ToolView {
	if src == nil {
		return nil
	}
	out := make([]tools.ToolView, len(src))
	copy(out, src)
	for i := range out {
		out[i].Descriptor.InputSchema = cloneRaw(out[i].Descriptor.InputSchema)
		out[i].Descriptor.OutputSchema = cloneRaw(out[i].Descriptor.OutputSchema)
		out[i].Descriptor.Toolsets = append([]tools.ToolsetID(nil), out[i].Descriptor.Toolsets...)
		out[i].Descriptor.Tags = append([]string(nil), out[i].Descriptor.Tags...)
		out[i].Descriptor.SearchHints = append([]string(nil), out[i].Descriptor.SearchHints...)
		out[i].Availability.ReasonCodes = append([]tools.ReasonCode(nil), out[i].Availability.ReasonCodes...)
		out[i].Decision.ReasonCodes = append([]tools.ReasonCode(nil), out[i].Decision.ReasonCodes...)
	}
	return out
}

func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("mcp: context is required")
	}
	return ctx.Err()
}
