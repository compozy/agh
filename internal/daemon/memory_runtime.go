package daemon

import (
	"bufio"
	"bytes"
	"context"
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

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	"github.com/pedronauck/agh/internal/fileutil"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/memory"
	memcontract "github.com/pedronauck/agh/internal/memory/contract"
	extractorpkg "github.com/pedronauck/agh/internal/memory/extractor"
	"github.com/pedronauck/agh/internal/memory/prompts"
	localprovider "github.com/pedronauck/agh/internal/memory/provider/local"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	toolspkg "github.com/pedronauck/agh/internal/tools"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	memoryExtractorConsumeInterval = time.Second
	memoryExtractorStopTimeout     = 10 * time.Second
	memoryExtractorSyntheticTaskID = "memory-extractor"
)

type memoryExtractorSessionManager interface {
	Spawn(context.Context, session.SpawnOpts) (*session.Session, error)
	PromptSynthetic(context.Context, string, session.SyntheticPromptOpts) (<-chan acp.AgentEvent, error)
	StopWithCause(context.Context, string, session.StopCause, string) error
}

type daemonMemoryExtractor struct {
	runtime        *extractorpkg.Runtime
	consumer       *extractorpkg.InboxConsumer
	failuresDir    string
	proposalSink   extractorpkg.ProposalSink
	logger         *slog.Logger
	now            func() time.Time
	workspaceRoots *sync.Map

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func newDaemonMemoryExtractor(
	ctx context.Context,
	state *bootState,
	sessions SessionManager,
	now func() time.Time,
) (*daemonMemoryExtractor, error) {
	if state == nil || state.memoryStore == nil || !state.cfg.Memory.Enabled || !state.cfg.Memory.Extractor.Enabled {
		return nil, nil
	}
	forkSessions, ok := sessions.(memoryExtractorSessionManager)
	if !ok {
		return nil, errors.New("daemon: session manager does not implement memory extractor spawn surface")
	}
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	workspaceRoots := &sync.Map{}
	forked := &forkedMemoryExtractor{
		sessions:       forkSessions,
		defaultAgent:   firstNonEmptyString(state.cfg.Defaults.Agent, state.cfg.Memory.Dream.Agent),
		deadline:       state.cfg.Memory.Extractor.Deadline,
		logger:         state.logger,
		now:            now,
		workspaceRoots: workspaceRoots,
	}
	runtime, err := extractorpkg.NewRuntime(
		context.WithoutCancel(ctx),
		state.globalMemoryDir,
		forked,
		extractorpkg.WithEventSink(state.memoryStore),
		extractorpkg.WithLogger(state.logger),
		extractorpkg.WithClock(now),
		extractorpkg.WithCoalesceMax(state.cfg.Memory.Extractor.Queue.CoalesceMax),
		extractorpkg.WithInboxPath(state.cfg.Memory.Extractor.InboxPath),
	)
	if err != nil {
		return nil, fmt.Errorf("daemon: create memory extractor runtime: %w", err)
	}
	sink := &daemonMemoryProposalSink{
		base:              state.memoryStore,
		workspaceResolver: state.workspaceResolver,
	}
	consumer, err := extractorpkg.NewInboxConsumer(
		state.globalMemoryDir,
		sink,
		extractorpkg.WithConsumerEventSink(state.memoryStore),
		extractorpkg.WithConsumerLogger(state.logger),
		extractorpkg.WithConsumerClock(now),
		extractorpkg.WithConsumerInboxPath(state.cfg.Memory.Extractor.InboxPath),
		extractorpkg.WithConsumerFailurePath(state.cfg.Memory.Extractor.DLQPath),
	)
	if err != nil {
		return nil, fmt.Errorf("daemon: create memory extractor inbox consumer: %w", err)
	}
	return &daemonMemoryExtractor{
		runtime:        runtime,
		consumer:       consumer,
		failuresDir:    extractorFailureDir(state),
		proposalSink:   sink,
		logger:         state.logger,
		now:            now,
		workspaceRoots: workspaceRoots,
		done:           make(chan struct{}),
	}, nil
}

func extractorFailureDir(state *bootState) string {
	if state == nil {
		return ""
	}
	if path := strings.TrimSpace(state.cfg.Memory.Extractor.DLQPath); path != "" {
		return filepath.Clean(path)
	}
	return filepath.Join(state.globalMemoryDir, "_system", "extractor", "failures")
}

func (e *daemonMemoryExtractor) Start(ctx context.Context) error {
	if e == nil || e.consumer == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("daemon: memory extractor start context is required")
	}
	e.mu.Lock()
	if e.cancel != nil {
		e.mu.Unlock()
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	e.done = make(chan struct{})
	e.mu.Unlock()

	go func() {
		defer close(e.done)
		ticker := time.NewTicker(memoryExtractorConsumeInterval)
		defer ticker.Stop()
		e.consumeOnce(runCtx)
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				e.consumeOnce(runCtx)
			}
		}
	}()
	return nil
}

func (e *daemonMemoryExtractor) Close(ctx context.Context) error {
	if e == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("daemon: memory extractor close context is required")
	}
	e.mu.Lock()
	cancel := e.cancel
	done := e.done
	e.cancel = nil
	e.mu.Unlock()
	if cancel != nil {
		cancel()
		if done != nil {
			select {
			case <-done:
			case <-ctx.Done():
				return fmt.Errorf("daemon: wait memory extractor consumer: %w", ctx.Err())
			}
		}
	}
	if e.runtime != nil {
		if err := e.runtime.Close(ctx); err != nil {
			return err
		}
	}
	e.consumeOnce(ctx)
	return nil
}

func (e *daemonMemoryExtractor) HandleSessionMessagePersisted(
	ctx context.Context,
	payload hookspkg.SessionMessagePersistedPayload,
) error {
	if e == nil || e.runtime == nil {
		return nil
	}
	if workspaceRoot := strings.TrimSpace(payload.Workspace); workspaceRoot != "" {
		sessionID := firstNonEmptyString(payload.SessionID, payload.RootSessionID)
		if sessionID != "" {
			e.workspaceRoots.Store(sessionID, workspaceRoot)
		}
	}
	return e.runtime.HandleSessionMessagePersisted(ctx, payload)
}

func (e *daemonMemoryExtractor) RecordToolWrite(sessionID string, turnSeq int64) {
	if e == nil || e.runtime == nil {
		return
	}
	e.runtime.RecordToolWrite(sessionID, turnSeq)
}

func (e *daemonMemoryExtractor) Status(context.Context) (contract.MemoryExtractorStatusPayload, error) {
	if e == nil || e.runtime == nil {
		return contract.MemoryExtractorStatusPayload{Status: contract.MemoryExtractorStateStopped}, nil
	}
	stats := e.runtime.Stats()
	status := contract.MemoryExtractorStateIdle
	if stats.Closed {
		status = contract.MemoryExtractorStateStopped
	} else if stats.InFlightSessions > 0 || stats.QueuedSessions > 0 {
		status = contract.MemoryExtractorStateRunning
	}
	failureCount, err := e.failureCount()
	if err != nil {
		return contract.MemoryExtractorStatusPayload{}, err
	}
	return contract.MemoryExtractorStatusPayload{
		Status:           status,
		QueuedSessions:   stats.QueuedSessions,
		InFlightSessions: stats.InFlightSessions,
		DroppedTurns:     intFromInt64(stats.DroppedTurns),
		CoalescedTurns:   intFromInt64(stats.CoalescedTurns),
		FailureCount:     failureCount,
	}, nil
}

func (e *daemonMemoryExtractor) ListFailures(context.Context) ([]contract.MemoryExtractorFailurePayload, error) {
	if e == nil {
		return []contract.MemoryExtractorFailurePayload{}, nil
	}
	failures, err := e.loadFailures()
	if err != nil {
		return nil, err
	}
	payloads := make([]contract.MemoryExtractorFailurePayload, 0, len(failures))
	for _, failure := range failures {
		payloads = append(payloads, failure.Payload)
	}
	return payloads, nil
}

func (e *daemonMemoryExtractor) Retry(
	ctx context.Context,
	req contract.MemoryExtractorRetryRequest,
) (contract.MemoryExtractorRetryResponse, error) {
	if e == nil || e.proposalSink == nil {
		return contract.MemoryExtractorRetryResponse{}, errors.New("daemon: memory extractor is not configured")
	}
	failures, err := e.loadFailures()
	if err != nil {
		return contract.MemoryExtractorRetryResponse{}, err
	}
	targetFailureID := strings.TrimSpace(req.FailureID)
	targetSessionID := strings.TrimSpace(req.SessionID)
	var response contract.MemoryExtractorRetryResponse
	for _, failure := range failures {
		if targetFailureID != "" && failure.Payload.ID != targetFailureID {
			continue
		}
		if targetSessionID != "" && failure.Payload.SessionID != targetSessionID {
			continue
		}
		if err := ctx.Err(); err != nil {
			return response, fmt.Errorf("daemon: retry memory extractor failures: %w", err)
		}
		candidates, decodeErr := failure.Candidates()
		if decodeErr != nil {
			response.Failed++
			continue
		}
		if len(candidates) == 0 {
			response.Failed++
			continue
		}
		var failed bool
		for _, candidate := range candidates {
			if _, proposeErr := e.proposalSink.ProposeCandidate(ctx, candidate); proposeErr != nil {
				failed = true
				break
			}
		}
		if failed {
			response.Failed++
			continue
		}
		if err := fileutil.AtomicRemoveFile(failure.Payload.Path); err != nil {
			return response, fmt.Errorf("daemon: remove retried extractor failure: %w", err)
		}
		response.Retried++
	}
	return response, nil
}

func (e *daemonMemoryExtractor) Drain(ctx context.Context) (contract.MemoryExtractorDrainResponse, error) {
	if e == nil || e.runtime == nil {
		return contract.MemoryExtractorDrainResponse{DrainedAt: e.nowUTC()}, nil
	}
	if err := e.runtime.Drain(ctx); err != nil {
		return contract.MemoryExtractorDrainResponse{}, err
	}
	e.consumeOnce(ctx)
	stats := e.runtime.Stats()
	return contract.MemoryExtractorDrainResponse{
		DrainedAt: e.nowUTC(),
		Remaining: stats.QueuedSessions + stats.InFlightSessions,
	}, nil
}

func (e *daemonMemoryExtractor) consumeOnce(ctx context.Context) {
	if e == nil || e.consumer == nil {
		return
	}
	if _, err := e.consumer.ConsumeOnce(ctx); err != nil && ctx.Err() == nil && e.logger != nil {
		e.logger.Warn("daemon: memory extractor inbox consume failed", "error", err)
	}
}

func (e *daemonMemoryExtractor) failureCount() (int, error) {
	failures, err := e.loadFailures()
	if err != nil {
		return 0, err
	}
	return len(failures), nil
}

func (e *daemonMemoryExtractor) loadFailures() ([]extractorFailure, error) {
	dir := strings.TrimSpace(e.failuresDir)
	if dir == "" {
		return []extractorFailure{}, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []extractorFailure{}, nil
		}
		return nil, fmt.Errorf("daemon: read memory extractor failures: %w", err)
	}
	failures := make([]extractorFailure, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		failure, err := readExtractorFailure(path)
		if err != nil {
			return nil, err
		}
		failures = append(failures, failure)
	}
	slices.SortFunc(failures, func(a, b extractorFailure) int {
		if !a.Payload.CreatedAt.Equal(b.Payload.CreatedAt) {
			return a.Payload.CreatedAt.Compare(b.Payload.CreatedAt)
		}
		return strings.Compare(a.Payload.ID, b.Payload.ID)
	})
	return failures, nil
}

func (e *daemonMemoryExtractor) nowUTC() time.Time {
	if e != nil && e.now != nil {
		return e.now().UTC()
	}
	return time.Now().UTC()
}

type extractorFailure struct {
	Payload contract.MemoryExtractorFailurePayload
	Report  extractorFailureReport
}

type extractorFailureReport struct {
	Stage      string `json:"stage"`
	Source     string `json:"source"`
	Error      string `json:"error"`
	Content    string `json:"content"`
	RecordedAt string `json:"recorded_at"`
}

func readExtractorFailure(path string) (extractorFailure, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return extractorFailure{}, fmt.Errorf("daemon: read extractor failure %q: %w", path, err)
	}
	var report extractorFailureReport
	if err := json.Unmarshal(data, &report); err != nil {
		return extractorFailure{}, fmt.Errorf("daemon: decode extractor failure %q: %w", path, err)
	}
	createdAt := parseMemoryTime(report.RecordedAt)
	if createdAt.IsZero() {
		if info, statErr := os.Stat(path); statErr == nil {
			createdAt = info.ModTime().UTC()
		}
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	sessionID, workspaceID, agentName := failureCandidateMetadata(report.Content)
	return extractorFailure{
		Payload: contract.MemoryExtractorFailurePayload{
			ID:          strings.TrimSuffix(filepath.Base(path), ".json"),
			SessionID:   sessionID,
			WorkspaceID: workspaceID,
			AgentName:   agentName,
			Reason:      firstNonEmptyString(report.Error, report.Stage),
			Path:        path,
			CreatedAt:   createdAt,
		},
		Report: report,
	}, nil
}

func (f extractorFailure) Candidates() ([]memcontract.Candidate, error) {
	scanner := bufio.NewScanner(strings.NewReader(f.Report.Content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	candidates := make([]memcontract.Candidate, 0)
	for scanner.Scan() {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		var candidate memcontract.Candidate
		if err := json.Unmarshal([]byte(raw), &candidate); err != nil {
			return nil, fmt.Errorf("daemon: decode extractor failure candidate: %w", err)
		}
		candidates = append(candidates, candidate)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("daemon: scan extractor failure candidates: %w", err)
	}
	return candidates, nil
}

func failureCandidateMetadata(content string) (sessionID string, workspaceID string, agentName string) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		var candidate memcontract.Candidate
		if err := json.Unmarshal([]byte(raw), &candidate); err != nil {
			continue
		}
		if candidate.Metadata != nil {
			sessionID = firstNonEmptyString(sessionID, candidate.Metadata["session_id"])
		}
		workspaceID = firstNonEmptyString(workspaceID, candidate.WorkspaceID)
		agentName = firstNonEmptyString(agentName, candidate.AgentName)
		if sessionID != "" || workspaceID != "" || agentName != "" {
			return sessionID, workspaceID, agentName
		}
	}
	return "", "", ""
}

type daemonMemoryProposalSink struct {
	base              *memory.Store
	workspaceResolver workspacepkg.RuntimeResolver
}

func (s *daemonMemoryProposalSink) ProposeCandidate(
	ctx context.Context,
	candidate memcontract.Candidate,
) (memcontract.Decision, error) {
	if s == nil || s.base == nil {
		return memcontract.Decision{}, errors.New("daemon: memory store is not configured")
	}
	target, candidate, err := s.targetStore(ctx, candidate)
	if err != nil {
		return memcontract.Decision{}, err
	}
	return target.ProposeCandidate(ctx, candidate)
}

func (s *daemonMemoryProposalSink) targetStore(
	ctx context.Context,
	candidate memcontract.Candidate,
) (*memory.Store, memcontract.Candidate, error) {
	scope := candidate.Scope.Normalize()
	if scope == "" {
		scope = candidate.Frontmatter.Scope.Normalize()
	}
	switch scope {
	case "", memcontract.ScopeGlobal:
		candidate.Scope = memcontract.ScopeGlobal
		candidate.Frontmatter.Scope = memcontract.ScopeGlobal
		return s.base, candidate, nil
	case memcontract.ScopeWorkspace, memcontract.ScopeAgent:
		workspaceRoot, workspaceID, err := s.resolveWorkspace(ctx, candidate)
		if err != nil {
			return nil, candidate, err
		}
		candidate.WorkspaceID = workspaceID
		candidate.Scope = scope
		candidate.Frontmatter.Scope = scope
		store := s.base.ForWorkspace(workspaceRoot)
		if scope == memcontract.ScopeAgent {
			tier := candidate.AgentTier.Normalize()
			if tier == "" {
				tier = memcontract.AgentTierWorkspace
			}
			candidate.AgentTier = tier
			candidate.Frontmatter.AgentTier = tier
			store = store.ForAgent(workspaceID, candidate.AgentName, tier)
		}
		return store, candidate, nil
	default:
		return nil, candidate, fmt.Errorf("daemon: unsupported memory scope %q", candidate.Scope)
	}
}

func (s *daemonMemoryProposalSink) resolveWorkspace(
	ctx context.Context,
	candidate memcontract.Candidate,
) (root string, workspaceID string, err error) {
	if candidate.Metadata != nil {
		root = strings.TrimSpace(candidate.Metadata["workspace_root"])
	}
	workspaceID = strings.TrimSpace(candidate.WorkspaceID)
	if root != "" {
		identity, identityErr := workspacepkg.EnsureIdentity(ctx, root)
		if identityErr != nil {
			return "", "", fmt.Errorf("daemon: resolve memory candidate workspace identity: %w", identityErr)
		}
		return root, identity.WorkspaceID, nil
	}
	if workspaceID == "" {
		return "", "", errors.New("daemon: workspace memory candidate requires workspace id")
	}
	if s.workspaceResolver == nil {
		return "", "", errors.New("daemon: workspace resolver is not configured")
	}
	resolved, err := s.workspaceResolver.Resolve(ctx, workspaceID)
	if err != nil {
		return "", "", fmt.Errorf("daemon: resolve memory candidate workspace %q: %w", workspaceID, err)
	}
	return resolved.RootDir, firstNonEmptyString(resolved.WorkspaceID, resolved.ID, workspaceID), nil
}

type forkedMemoryExtractor struct {
	sessions       memoryExtractorSessionManager
	defaultAgent   string
	deadline       time.Duration
	logger         *slog.Logger
	now            func() time.Time
	workspaceRoots *sync.Map
}

func (e *forkedMemoryExtractor) Extract(
	ctx context.Context,
	turn memcontract.TurnRecord,
) ([]memcontract.Candidate, error) {
	if e == nil || e.sessions == nil {
		return nil, errors.New("daemon: memory extractor sessions are not configured")
	}
	runCtx := context.WithoutCancel(ctx)
	if e.deadline > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(runCtx, e.deadline)
		defer cancel()
	}
	prompt, err := renderMemoryExtractorPrompt(turn)
	if err != nil {
		return nil, err
	}
	child, err := e.sessions.Spawn(runCtx, session.SpawnOpts{
		ParentSessionID:    turn.SessionID,
		AgentName:          firstNonEmptyString(turn.AgentID, e.defaultAgent),
		Name:               "Memory extractor",
		PromptOverlay:      memoryExtractorOverlay(),
		SpawnRole:          session.SpawnRoleMemoryExtractor,
		TTL:                e.extractorTTL(),
		AllowStoppedParent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("daemon: spawn memory extractor session: %w", err)
	}
	defer e.stopChild(ctx, child.ID)

	events, err := e.sessions.PromptSynthetic(runCtx, child.ID, session.SyntheticPromptOpts{
		Message: prompt,
		Metadata: acp.PromptSyntheticMeta{
			TaskID:  memoryExtractorSyntheticTaskID,
			Reason:  "memory_extractor",
			Summary: "extract durable Memory v2 candidates",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("daemon: prompt memory extractor session: %w", err)
	}
	output, err := collectMemoryExtractorOutput(runCtx, events)
	if err != nil {
		return nil, err
	}
	candidates, err := parseMemoryExtractorCandidates(output, turn, e.workspaceRoot(turn.SessionID), e.nowUTC())
	if err != nil {
		return nil, err
	}
	return candidates, nil
}

func (e *forkedMemoryExtractor) Drain(context.Context) error {
	return nil
}

func (e *forkedMemoryExtractor) extractorTTL() time.Duration {
	if e.deadline > 0 {
		return e.deadline + memoryExtractorStopTimeout
	}
	return 2 * time.Minute
}

func (e *forkedMemoryExtractor) stopChild(parentCtx context.Context, id string) {
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(parentCtx), memoryExtractorStopTimeout)
	defer cancel()
	if err := e.sessions.StopWithCause(stopCtx, id, session.CauseCompleted, "memory extractor completed"); err != nil &&
		e.logger != nil {
		e.logger.Warn("daemon: stop memory extractor child failed", "session_id", id, "error", err)
	}
}

func (e *forkedMemoryExtractor) workspaceRoot(sessionID string) string {
	if e == nil || e.workspaceRoots == nil {
		return ""
	}
	defer e.workspaceRoots.Delete(sessionID)
	value, ok := e.workspaceRoots.Load(sessionID)
	if !ok {
		return ""
	}
	root, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(root)
}

func (e *forkedMemoryExtractor) nowUTC() time.Time {
	if e != nil && e.now != nil {
		return e.now().UTC()
	}
	return time.Now().UTC()
}

func renderMemoryExtractorPrompt(turn memcontract.TurnRecord) (string, error) {
	tmpl, err := prompts.ParseTemplate(prompts.NameExtract, prompts.VersionV1)
	if err != nil {
		return "", err
	}
	policy, err := prompts.Load(prompts.NameWhatNotToSave, prompts.VersionV1)
	if err != nil {
		return "", err
	}
	var rendered bytes.Buffer
	data := map[string]any{
		"WhatNotToSave": policy.Content,
		"Turn":          turn,
		"Transcript":    renderMemoryTranscript(turn.Snapshot),
	}
	if err := tmpl.Execute(&rendered, data); err != nil {
		return "", fmt.Errorf("daemon: render memory extractor prompt: %w", err)
	}
	return rendered.String(), nil
}

func renderMemoryTranscript(snapshot memcontract.TranscriptSnapshot) string {
	var buf strings.Builder
	for _, message := range snapshot.Messages {
		role := strings.TrimSpace(message.Role)
		if role == "" {
			role = "unknown"
		}
		if _, err := fmt.Fprintf(
			&buf,
			"- sequence=%d role=%s at=%s\n%s\n",
			message.Sequence,
			role,
			message.At.UTC().Format(time.RFC3339Nano),
			strings.TrimSpace(message.Content),
		); err != nil {
			return buf.String()
		}
	}
	return buf.String()
}

func memoryExtractorOverlay() string {
	return strings.TrimSpace(`
You are an AGH internal Memory v2 extractor child session.
Return only JSONL candidates that match the requested schema.
Do not modify files, run commands, or include commentary outside JSONL.
`)
}

func collectMemoryExtractorOutput(ctx context.Context, events <-chan acp.AgentEvent) (string, error) {
	var output strings.Builder
	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("daemon: collect memory extractor output: %w", ctx.Err())
		case event, ok := <-events:
			if !ok {
				return output.String(), nil
			}
			switch event.Type {
			case acp.EventTypeAgentMessage:
				output.WriteString(event.Text)
				if !strings.HasSuffix(event.Text, "\n") {
					output.WriteByte('\n')
				}
			case acp.EventTypeError:
				return "", fmt.Errorf("daemon: memory extractor agent error: %s", strings.TrimSpace(event.Error))
			}
		}
	}
}

type extractedMemoryLine struct {
	Type      string `json:"type"`
	Scope     string `json:"scope"`
	AgentTier string `json:"agent_tier"`
	Content   string `json:"content"`
	Evidence  string `json:"evidence"`
	Entity    string `json:"entity"`
	Attribute string `json:"attribute"`
}

func parseMemoryExtractorCandidates(
	output string,
	turn memcontract.TurnRecord,
	workspaceRoot string,
	submittedAt time.Time,
) ([]memcontract.Candidate, error) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	candidates := make([]memcontract.Candidate, 0)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		raw := normalizeExtractorJSONLLine(scanner.Text())
		if raw == "" {
			continue
		}
		var line extractedMemoryLine
		if err := json.Unmarshal([]byte(raw), &line); err != nil {
			return nil, fmt.Errorf("daemon: decode memory extractor line %d: %w", lineNumber, err)
		}
		candidate, err := candidateFromExtractedLine(line, turn, workspaceRoot, submittedAt)
		if err != nil {
			return nil, fmt.Errorf("daemon: normalize memory extractor line %d: %w", lineNumber, err)
		}
		candidates = append(candidates, candidate)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("daemon: scan memory extractor output: %w", err)
	}
	return candidates, nil
}

func normalizeExtractorJSONLLine(raw string) string {
	line := strings.TrimSpace(raw)
	switch line {
	case "", "```", "```json", "```jsonl":
		return ""
	default:
		return line
	}
}

func candidateFromExtractedLine(
	line extractedMemoryLine,
	turn memcontract.TurnRecord,
	workspaceRoot string,
	submittedAt time.Time,
) (memcontract.Candidate, error) {
	memoryType := memcontract.Type(strings.TrimSpace(line.Type)).Normalize()
	if err := memoryType.Validate(); err != nil {
		return memcontract.Candidate{}, err
	}
	scope := memcontract.Scope(strings.TrimSpace(line.Scope)).Normalize()
	if scope == "" {
		defaultScope, err := memcontract.DefaultScopeForType(memoryType)
		if err != nil {
			return memcontract.Candidate{}, err
		}
		scope = defaultScope
	}
	if err := scope.Validate(); err != nil {
		return memcontract.Candidate{}, err
	}
	content := strings.TrimSpace(line.Content)
	if content == "" {
		return memcontract.Candidate{}, errors.New("content is required")
	}
	agentTier := memcontract.AgentTier(strings.TrimSpace(line.AgentTier)).Normalize()
	if scope == memcontract.ScopeAgent && agentTier == "" {
		agentTier = memcontract.AgentTierWorkspace
	}
	agentName := ""
	if scope == memcontract.ScopeAgent {
		agentName = strings.TrimSpace(turn.AgentID)
	}
	metadata := map[string]string{
		"evidence": line.Evidence,
	}
	if workspaceRoot != "" {
		metadata["workspace_root"] = workspaceRoot
	}
	return memcontract.Candidate{
		WorkspaceID: strings.TrimSpace(turn.WorkspaceID),
		Scope:       scope,
		AgentName:   agentName,
		AgentTier:   agentTier,
		Origin:      memcontract.OriginExtractor,
		Content:     content,
		Frontmatter: memcontract.Header{
			Name:        memoryCandidateName(line, content),
			Description: strings.TrimSpace(line.Evidence),
			Type:        memoryType,
			Scope:       scope,
			AgentName:   agentName,
			AgentTier:   agentTier,
			Provenance: &memcontract.Provenance{
				SourceSessionIDs: []string{turn.SessionID},
				SourceActor:      memcontract.OriginExtractor,
				Confidence:       "candidate",
				CreatedAt:        submittedAt.UTC(),
				UpdatedAt:        submittedAt.UTC(),
			},
		},
		Entity:      strings.TrimSpace(line.Entity),
		Attribute:   strings.TrimSpace(line.Attribute),
		Metadata:    metadata,
		SubmittedAt: submittedAt.UTC(),
	}, nil
}

func memoryCandidateName(line extractedMemoryLine, content string) string {
	entity := strings.TrimSpace(line.Entity)
	attribute := strings.TrimSpace(line.Attribute)
	switch {
	case entity != "" && attribute != "":
		return entity + " " + attribute
	case entity != "":
		return entity
	default:
		words := strings.Fields(content)
		if len(words) > 8 {
			words = words[:8]
		}
		return strings.Join(words, " ")
	}
}

type daemonMemoryProviderService struct {
	registry *extensionpkg.MemoryProviderRegistry
}

func (s daemonMemoryProviderService) List(
	ctx context.Context,
	workspaceID string,
) ([]contract.MemoryProviderPayload, error) {
	if s.registry == nil {
		return nil, errors.New("daemon: memory provider registry is not configured")
	}
	active, activeErr := s.registry.Select(ctx, workspaceID, "")
	if activeErr != nil && !isMemoryProviderNotFound(activeErr) {
		return nil, activeErr
	}
	registrations := s.registry.List()
	payloads := make([]contract.MemoryProviderPayload, 0, len(registrations))
	for _, registration := range registrations {
		payloads = append(payloads, memoryProviderPayload(registration, registration.Name == active.Name))
	}
	return payloads, nil
}

func (s daemonMemoryProviderService) Get(
	ctx context.Context,
	workspaceID string,
	name string,
) (contract.MemoryProviderPayload, error) {
	if s.registry == nil {
		return contract.MemoryProviderPayload{}, errors.New("daemon: memory provider registry is not configured")
	}
	active, activeErr := s.registry.Select(ctx, workspaceID, "")
	if activeErr != nil && !isMemoryProviderNotFound(activeErr) {
		return contract.MemoryProviderPayload{}, activeErr
	}
	registration, err := s.registry.Select(ctx, workspaceID, name)
	if err != nil {
		if isMemoryProviderNotFound(err) {
			return contract.MemoryProviderPayload{}, fmt.Errorf("%w: %s", os.ErrNotExist, err.Error())
		}
		return contract.MemoryProviderPayload{}, err
	}
	return memoryProviderPayload(registration, registration.Name == active.Name), nil
}

func (s daemonMemoryProviderService) Select(
	ctx context.Context,
	workspaceID string,
	name string,
) (contract.MemoryProviderPayload, error) {
	if s.registry == nil {
		return contract.MemoryProviderPayload{}, errors.New("daemon: memory provider registry is not configured")
	}
	if err := s.registry.SetActive(ctx, workspaceID, name); err != nil {
		if isMemoryProviderNotFound(err) {
			return contract.MemoryProviderPayload{}, fmt.Errorf("%w: %s", os.ErrNotExist, err.Error())
		}
		return contract.MemoryProviderPayload{}, err
	}
	return s.Get(ctx, workspaceID, name)
}

func (s daemonMemoryProviderService) Enable(
	ctx context.Context,
	workspaceID string,
	name string,
	_ string,
) (contract.MemoryProviderLifecycleResponse, error) {
	provider, err := s.Get(ctx, workspaceID, name)
	return contract.MemoryProviderLifecycleResponse{Provider: provider, Changed: false}, err
}

func (s daemonMemoryProviderService) Disable(
	ctx context.Context,
	workspaceID string,
	name string,
	_ string,
) (contract.MemoryProviderLifecycleResponse, error) {
	provider, err := s.Get(ctx, workspaceID, name)
	return contract.MemoryProviderLifecycleResponse{Provider: provider, Changed: false}, err
}

func memoryProviderPayload(
	registration extensionpkg.MemoryProviderRegistration,
	active bool,
) contract.MemoryProviderPayload {
	status := contract.MemoryProviderStateStandby
	if active {
		status = contract.MemoryProviderStateActive
	}
	return contract.MemoryProviderPayload{
		Name:    strings.TrimSpace(registration.Name),
		Status:  status,
		Active:  active,
		Builtin: registration.Bundled,
		Tools:   append([]string(nil), registration.ToolNames...),
	}
}

func isMemoryProviderNotFound(err error) bool {
	var typed *extensionpkg.MemoryProviderNotFoundError
	return errors.As(err, &typed)
}

func newDaemonMemoryProviderRegistry(
	ctx context.Context,
	state *bootState,
) (*extensionpkg.MemoryProviderRegistry, error) {
	if state == nil || state.memoryStore == nil || !state.cfg.Memory.Enabled {
		return nil, nil
	}
	opts := []extensionpkg.MemoryProviderRegistryOption{
		extensionpkg.WithMemoryProviderReservedTools(
			toolspkg.ToolIDMemoryList.String(),
			toolspkg.ToolIDMemoryShow.String(),
			toolspkg.ToolIDMemorySearch.String(),
			toolspkg.ToolIDMemoryPropose.String(),
			toolspkg.ToolIDMemoryNote.String(),
		),
		extensionpkg.WithMemoryProviderRegistryClock(func() time.Time {
			return time.Now().UTC()
		}),
	}
	if eventWriter, ok := state.registry.(store.EventSummaryStore); ok {
		opts = append(opts, extensionpkg.WithMemoryProviderEventSummaryStore(eventWriter))
	}
	registry := extensionpkg.NewMemoryProviderRegistry(opts...)
	if state.localMemoryProvider != nil {
		if err := registry.Register(ctx, extensionpkg.MemoryProviderRegistration{
			Name:     localprovider.Name,
			Version:  "builtin",
			Provider: state.localMemoryProvider,
			Bundled:  true,
		}); err != nil {
			return nil, err
		}
	}
	activeProvider := firstNonEmptyString(state.cfg.Memory.Provider.Name, localprovider.Name)
	if err := registry.SetActive(ctx, "", activeProvider); err != nil {
		return nil, err
	}
	return registry, nil
}

type daemonMemorySessionLedgerService struct {
	rootDir          string
	unboundPartition string
	now              func() time.Time
}

func newDaemonMemorySessionLedgerService(state *bootState, now func() time.Time) *daemonMemorySessionLedgerService {
	if state == nil || !state.cfg.Memory.Enabled {
		return nil
	}
	root := strings.TrimSpace(state.cfg.Memory.Session.LedgerRoot)
	if root == "" {
		return nil
	}
	unbound := strings.TrimSpace(state.cfg.Memory.Session.UnboundPartition)
	if unbound == "" {
		unbound = "_unbound"
	}
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	return &daemonMemorySessionLedgerService{rootDir: root, unboundPartition: unbound, now: now}
}

func (s *daemonMemorySessionLedgerService) Get(
	ctx context.Context,
	sessionID string,
) (contract.MemorySessionLedgerResponse, error) {
	path, err := s.locate(ctx, sessionID)
	if err != nil {
		return contract.MemorySessionLedgerResponse{}, err
	}
	return readSessionLedger(path)
}

func (s *daemonMemorySessionLedgerService) Replay(
	ctx context.Context,
	sessionID string,
	req contract.MemorySessionReplayRequest,
) (contract.MemorySessionReplayResponse, error) {
	ledger, err := s.Get(ctx, sessionID)
	if err != nil {
		return contract.MemorySessionReplayResponse{}, err
	}
	events := make([]contract.MemorySessionLedgerEntryPayload, 0, len(ledger.Events))
	for _, event := range ledger.Events {
		if !req.IncludeToolEvents && isToolLedgerEvent(event.EventType) {
			continue
		}
		if !req.IncludeMemory && strings.Contains(strings.ToLower(event.EventType), "memory") {
			continue
		}
		events = append(events, event)
	}
	return contract.MemorySessionReplayResponse{SessionID: ledger.Meta.SessionID, Events: events}, nil
}

func (s *daemonMemorySessionLedgerService) Prune(
	ctx context.Context,
	req contract.MemorySessionsPruneRequest,
) (contract.MemorySessionsPruneResponse, error) {
	if req.OlderThanHours <= 0 {
		return contract.MemorySessionsPruneResponse{}, errors.New("older_than_hours must be positive")
	}
	paths, err := s.listPaths(ctx)
	if err != nil {
		return contract.MemorySessionsPruneResponse{}, err
	}
	cutoff := s.now().UTC().Add(-time.Duration(req.OlderThanHours) * time.Hour)
	response := contract.MemorySessionsPruneResponse{DryRun: req.DryRun}
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return response, fmt.Errorf("daemon: prune memory session ledgers: %w", err)
		}
		ledger, err := readSessionLedger(path)
		if err != nil {
			return response, err
		}
		stopTime := ledger.Meta.CreatedAt
		if ledger.Meta.StoppedAt != nil {
			stopTime = *ledger.Meta.StoppedAt
		}
		if !stopTime.Before(cutoff) {
			continue
		}
		response.PrunedSessions++
		response.PrunedEvents += len(ledger.Events)
		if req.DryRun {
			continue
		}
		if err := os.RemoveAll(filepath.Dir(path)); err != nil {
			return response, fmt.Errorf("daemon: prune ledger %q: %w", path, err)
		}
	}
	return response, nil
}

func (s *daemonMemorySessionLedgerService) Repair(
	context.Context,
) (contract.MemorySessionsRepairResponse, error) {
	return contract.MemorySessionsRepairResponse{CompletedAt: s.now().UTC()}, nil
}

func (s *daemonMemorySessionLedgerService) locate(ctx context.Context, sessionID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", errors.New("session_id is required")
	}
	paths, err := s.listPaths(ctx)
	if err != nil {
		return "", err
	}
	for _, path := range paths {
		if filepath.Base(filepath.Dir(path)) == sessionID {
			return path, nil
		}
	}
	return "", fmt.Errorf("%w: memory session ledger %q", os.ErrNotExist, sessionID)
}

func (s *daemonMemorySessionLedgerService) listPaths(ctx context.Context) ([]string, error) {
	if s == nil || strings.TrimSpace(s.rootDir) == "" {
		return nil, errors.New("daemon: memory session ledger root is not configured")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("daemon: list memory session ledgers: %w", err)
	}
	pattern := filepath.Join(s.rootDir, "*", "*", "ledger.jsonl")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("daemon: glob memory session ledgers: %w", err)
	}
	slices.Sort(paths)
	return paths, nil
}

type sessionLedgerMetaLine struct {
	Type          string `json:"type"`
	Version       int    `json:"version"`
	SessionID     string `json:"session_id"`
	WorkspaceID   string `json:"workspace_id"`
	SpawnParentID string `json:"spawn_parent_id,omitempty"`
	RootSessionID string `json:"root_session_id,omitempty"`
	SpawnDepth    int    `json:"spawn_depth,omitempty"`
	StartedAt     string `json:"started_at,omitempty"`
	EndedAt       string `json:"ended_at,omitempty"`
}

type sessionLedgerEventLine struct {
	Type      string          `json:"type"`
	Sequence  int64           `json:"sequence"`
	EventType string          `json:"event_type"`
	Content   json.RawMessage `json:"content,omitempty"`
	Timestamp string          `json:"timestamp,omitempty"`
}

type sessionLedgerLineType struct {
	Type string `json:"type"`
}

func readSessionLedger(path string) (contract.MemorySessionLedgerResponse, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return contract.MemorySessionLedgerResponse{}, fmt.Errorf(
			"daemon: read memory session ledger %q: %w",
			path,
			err,
		)
	}
	checksum := sha256.Sum256(data)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	response := contract.MemorySessionLedgerResponse{
		Events: make([]contract.MemorySessionLedgerEntryPayload, 0),
	}
	for scanner.Scan() {
		raw := bytes.TrimSpace(scanner.Bytes())
		if len(raw) == 0 {
			continue
		}
		var lineType sessionLedgerLineType
		if err := json.Unmarshal(raw, &lineType); err != nil {
			return contract.MemorySessionLedgerResponse{}, fmt.Errorf("daemon: decode ledger line type: %w", err)
		}
		switch lineType.Type {
		case "ledger_meta":
			var meta sessionLedgerMetaLine
			if err := json.Unmarshal(raw, &meta); err != nil {
				return contract.MemorySessionLedgerResponse{}, fmt.Errorf("daemon: decode ledger meta: %w", err)
			}
			response.Meta = contract.MemorySessionLedgerMetaPayload{
				Version:         meta.Version,
				SessionID:       strings.TrimSpace(meta.SessionID),
				WorkspaceID:     strings.TrimSpace(meta.WorkspaceID),
				RootSessionID:   strings.TrimSpace(meta.RootSessionID),
				ParentSessionID: strings.TrimSpace(meta.SpawnParentID),
				SpawnDepth:      meta.SpawnDepth,
				Path:            path,
				Checksum:        hex.EncodeToString(checksum[:]),
				CreatedAt:       parseMemoryTime(meta.StartedAt),
			}
			if stoppedAt := parseMemoryTime(meta.EndedAt); !stoppedAt.IsZero() {
				response.Meta.StoppedAt = &stoppedAt
			}
		case "session_event":
			var event sessionLedgerEventLine
			if err := json.Unmarshal(raw, &event); err != nil {
				return contract.MemorySessionLedgerResponse{}, fmt.Errorf("daemon: decode ledger event: %w", err)
			}
			response.Events = append(response.Events, contract.MemorySessionLedgerEntryPayload{
				Sequence:  event.Sequence,
				EventType: strings.TrimSpace(event.EventType),
				EmittedAt: parseMemoryTime(event.Timestamp),
				Payload:   ledgerPayload(event.Content),
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return contract.MemorySessionLedgerResponse{}, fmt.Errorf("daemon: scan memory session ledger: %w", err)
	}
	if response.Meta.SessionID == "" {
		return contract.MemorySessionLedgerResponse{}, fmt.Errorf("%w: memory session ledger %q", os.ErrInvalid, path)
	}
	if response.Meta.CreatedAt.IsZero() {
		response.Meta.CreatedAt = time.Now().UTC()
	}
	return response, nil
}

func ledgerPayload(raw json.RawMessage) map[string]any {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err == nil {
		return payload
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return map[string]any{"raw": string(raw)}
	}
	return map[string]any{"value": value}
}

func isToolLedgerEvent(eventType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(eventType))
	return normalized == acp.EventTypeToolCall ||
		normalized == acp.EventTypeToolResult ||
		strings.Contains(normalized, "tool")
}

func parseMemoryTime(raw string) time.Time {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err == nil {
		return parsed.UTC()
	}
	parsed, err = time.Parse(time.RFC3339, value)
	if err == nil {
		return parsed.UTC()
	}
	return time.Time{}
}

func intFromInt64(value int64) int {
	if value > int64(^uint(0)>>1) {
		return int(^uint(0) >> 1)
	}
	return int(value)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

var _ memcontract.MemoryProvider = (*localprovider.Provider)(nil)
var _ extractorpkg.ProposalSink = (*daemonMemoryProposalSink)(nil)
