package local

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"sync"
	"time"

	memcontract "github.com/pedronauck/agh/internal/memory/contract"
)

// Name is the bundled local MemoryProvider registration name.
const Name = "local"

// Backend is the contract-typed substrate the local provider needs from AGH's
// memory store without depending on controller or recall internals directly.
type Backend interface {
	EnsureDirs() error
	LoadPromptIndex(scope memcontract.Scope) (content string, truncated bool, err error)
	List(scope memcontract.Scope) ([]memcontract.Header, error)
	Recall(
		ctx context.Context,
		query memcontract.Query,
		opts memcontract.RecallOptions,
	) (memcontract.Packaged, error)
	ApplyDecision(ctx context.Context, decision memcontract.Decision) error
	ForWorkspace(workspaceRoot string) Backend
	ForAgent(workspaceID string, agentName string, tier memcontract.AgentTier) Backend
}

// Provider implements the bundled local MemoryProvider over Store seams.
type Provider struct {
	backend Backend
	now     func() time.Time

	mu          sync.RWMutex
	workspaceID string
	config      map[string]any
	logger      *slog.Logger
	initialized bool
	shutdown    bool
}

var _ memcontract.MemoryProvider = (*Provider)(nil)

// Option customizes the bundled local Provider.
type Option func(*Provider)

// WithClock injects a deterministic clock.
func WithClock(now func() time.Time) Option {
	return func(provider *Provider) {
		if now != nil {
			provider.now = now
		}
	}
}

// WithLogger injects a provider logger used until Initialize overrides it.
func WithLogger(logger *slog.Logger) Option {
	return func(provider *Provider) {
		if logger != nil {
			provider.logger = logger
		}
	}
}

// New constructs a bundled local provider over the supplied memory store.
func New(backend Backend, opts ...Option) *Provider {
	provider := &Provider{
		backend: backend,
		logger:  slog.Default(),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(provider)
		}
	}
	return provider
}

// Initialize prepares local memory directories and records workspace metadata.
func (p *Provider) Initialize(ctx context.Context, init memcontract.ProviderInit) error {
	if err := p.checkContext(ctx); err != nil {
		return err
	}
	if p.backend == nil {
		return errors.New("memory provider local: backend is required")
	}
	if err := p.backend.EnsureDirs(); err != nil {
		return fmt.Errorf("memory provider local: initialize backend: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.workspaceID = strings.TrimSpace(init.WorkspaceID)
	p.config = maps.Clone(init.Config)
	if init.Logger != nil {
		p.logger = init.Logger
	}
	p.initialized = true
	p.shutdown = false
	return nil
}

// SystemPromptBlock returns the prompt-safe local MEMORY.md block for one scope.
func (p *Provider) SystemPromptBlock(
	ctx context.Context,
	req memcontract.SnapshotRequest,
) (memcontract.SnapshotResult, error) {
	if err := p.checkReady(ctx); err != nil {
		return memcontract.SnapshotResult{}, err
	}
	backend, scope, err := p.backendForSnapshot(req)
	if err != nil {
		return memcontract.SnapshotResult{}, err
	}
	markdown, _, err := backend.LoadPromptIndex(scope)
	if err != nil {
		return memcontract.SnapshotResult{}, err
	}
	ageMs, err := p.scopeAgeMs(backend, scope)
	if err != nil {
		return memcontract.SnapshotResult{}, err
	}
	return memcontract.SnapshotResult{Markdown: markdown, AgeMs: ageMs}, nil
}

// Recall delegates deterministic read-path packaging to the local store.
func (p *Provider) Recall(
	ctx context.Context,
	req memcontract.RecallRequest,
) (memcontract.RecallResult, error) {
	if err := p.checkReady(ctx); err != nil {
		return memcontract.RecallResult{}, err
	}
	query := req.Query
	if strings.TrimSpace(query.WorkspaceID) == "" {
		query.WorkspaceID = p.currentWorkspaceID()
	}
	packaged, err := p.backend.Recall(ctx, query, req.Options)
	if err != nil {
		return memcontract.RecallResult{}, err
	}
	return memcontract.RecallResult{Packaged: packaged}, nil
}

// Prefetch is a no-op for the bundled local provider.
func (p *Provider) Prefetch(ctx context.Context, _ memcontract.PrefetchRequest) error {
	return p.checkReady(ctx)
}

// SyncTurn is a no-op for the bundled local provider.
func (p *Provider) SyncTurn(ctx context.Context, _ memcontract.TurnRecord) error {
	return p.checkReady(ctx)
}

// OnSessionEnd is a no-op for the bundled local provider.
func (p *Provider) OnSessionEnd(ctx context.Context, _ memcontract.SessionEndRecord) error {
	return p.checkReady(ctx)
}

// OnSessionSwitch is a no-op for the bundled local provider.
func (p *Provider) OnSessionSwitch(ctx context.Context, _ memcontract.SessionSwitchRecord) error {
	return p.checkReady(ctx)
}

// OnPreCompress returns no local pre-compression hint.
func (p *Provider) OnPreCompress(
	ctx context.Context,
	_ memcontract.PreCompressRequest,
) (memcontract.PreCompressHint, error) {
	if err := p.checkReady(ctx); err != nil {
		return memcontract.PreCompressHint{}, err
	}
	return memcontract.PreCompressHint{}, nil
}

// OnMemoryWrite applies a controller decision through the local store.
//
//nolint:gocritic // MemoryProvider requires a value WriteRecord for interface compatibility.
func (p *Provider) OnMemoryWrite(ctx context.Context, rec memcontract.WriteRecord) error {
	if err := p.checkReady(ctx); err != nil {
		return err
	}
	target := p.backendForWriteRecord(&rec)
	if err := target.ApplyDecision(ctx, rec.Decision); err != nil {
		return fmt.Errorf("memory provider local: apply write decision: %w", err)
	}
	return nil
}

// Shutdown marks the local provider unavailable for future lifecycle calls.
func (p *Provider) Shutdown(ctx context.Context) error {
	if err := p.checkContext(ctx); err != nil {
		return err
	}
	if p == nil {
		return errors.New("memory provider local: provider is required")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.shutdown = true
	return nil
}

func (p *Provider) backendForSnapshot(
	req memcontract.SnapshotRequest,
) (Backend, memcontract.Scope, error) {
	scope := req.Scope.Normalize()
	if scope == "" {
		scope = memcontract.ScopeGlobal
	}
	if err := scope.Validate(); err != nil {
		return nil, "", fmt.Errorf("memory provider local: snapshot scope: %w", err)
	}
	backend := p.backendForWorkspace(req.WorkspaceRoot)
	if scope != memcontract.ScopeAgent {
		if scope == memcontract.ScopeWorkspace {
			return backend, scope, nil
		}
		return p.backend, scope, nil
	}
	agentName := strings.TrimSpace(req.AgentName)
	if agentName == "" {
		return nil, "", errors.New("memory provider local: snapshot agent name is required")
	}
	tier := req.AgentTier.Normalize()
	if tier == "" {
		tier = memcontract.AgentTierWorkspace
	}
	if err := tier.Validate(); err != nil {
		return nil, "", fmt.Errorf("memory provider local: snapshot agent tier: %w", err)
	}
	workspaceID := strings.TrimSpace(req.WorkspaceID)
	if workspaceID == "" {
		workspaceID = p.currentWorkspaceID()
	}
	return backend.ForAgent(workspaceID, agentName, tier), scope, nil
}

func (p *Provider) backendForWriteRecord(rec *memcontract.WriteRecord) Backend {
	if rec == nil {
		return p.backend
	}
	frontmatter := rec.Decision.Frontmatter
	if frontmatter.Scope.Normalize() != memcontract.ScopeAgent {
		return p.backend
	}
	agentName := strings.TrimSpace(frontmatter.AgentName)
	if agentName == "" {
		agentName = strings.TrimSpace(rec.Candidate.AgentName)
	}
	tier := frontmatter.AgentTier.Normalize()
	if tier == "" {
		tier = rec.Candidate.AgentTier.Normalize()
	}
	workspaceID := strings.TrimSpace(rec.Candidate.WorkspaceID)
	if workspaceID == "" {
		workspaceID = p.currentWorkspaceID()
	}
	return p.backend.ForAgent(workspaceID, agentName, tier)
}

func (p *Provider) backendForWorkspace(workspaceRoot string) Backend {
	if p == nil || p.backend == nil {
		return nil
	}
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		return p.backend
	}
	return p.backend.ForWorkspace(root)
}

func (p *Provider) scopeAgeMs(backend Backend, scope memcontract.Scope) (int64, error) {
	headers, err := backend.List(scope)
	if err != nil {
		return 0, err
	}
	var newest time.Time
	for _, header := range headers {
		if header.ModTime.After(newest) {
			newest = header.ModTime
		}
	}
	if newest.IsZero() {
		return 0, nil
	}
	age := p.now().Sub(newest)
	if age < 0 {
		return 0, nil
	}
	return age.Milliseconds(), nil
}

func (p *Provider) currentWorkspaceID() string {
	if p == nil {
		return ""
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.workspaceID
}

func (p *Provider) checkReady(ctx context.Context) error {
	if err := p.checkContext(ctx); err != nil {
		return err
	}
	p.mu.RLock()
	initialized := p.initialized
	shutdown := p.shutdown
	p.mu.RUnlock()
	if !initialized {
		return errors.New("memory provider local: provider is not initialized")
	}
	if shutdown {
		return errors.New("memory provider local: provider is shut down")
	}
	return nil
}

func (p *Provider) checkContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("memory provider local: context is required")
	}
	if p == nil {
		return errors.New("memory provider local: provider is required")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("memory provider local: context error: %w", err)
	}
	return nil
}
