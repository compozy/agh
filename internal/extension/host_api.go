package extension

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/pedronauck/agh/internal/acp"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	"github.com/pedronauck/agh/internal/frontmatter"
	"github.com/pedronauck/agh/internal/memory"
	observepkg "github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/subprocess"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	// HostAPIRateLimitedCode is the protocol code for per-extension backpressure.
	HostAPIRateLimitedCode = -32002
	// HostAPIInvalidParamsCode is the JSON-RPC invalid params code used for bad request payloads.
	HostAPIInvalidParamsCode = -32602
	// HostAPIMethodNotFoundCode is the JSON-RPC method-not-found code for unknown Host API methods.
	HostAPIMethodNotFoundCode = -32601

	defaultHostAPIRateLimit    = 10
	defaultHostAPIBurst        = 20
	defaultHostAPIDefaultLimit = 100
	defaultHostAPIRecallLimit  = 10
	maxMemoryDescriptionLength = 160
	tagCommentPrefix           = "<!-- agh-tags:"
)

type hostAPIContextKey string

const hostAPIExtensionNameContextKey hostAPIContextKey = "extension.host_api.extension_name"

// HostAPIOption customizes a HostAPIHandler.
type HostAPIOption func(*HostAPIHandler)

// HostAPIHandler handles extension -> AGH Host API JSON-RPC requests.
type HostAPIHandler struct {
	sessions   hostAPISessionManager
	memory     *memory.Store
	observer   hostAPIObserver
	skills     hostAPISkillsRegistry
	workspaces workspacepkg.WorkspaceResolver
	capChecker *CapabilityChecker
	limiter    *hostAPIRateLimiter
	now        func() time.Time
	rateLimit  int
	rateBurst  int

	methods map[string]hostAPIMethodFunc
}

type hostAPIMethodFunc func(context.Context, json.RawMessage) (any, error)

type hostAPISessionManager interface {
	Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
	ListAll(ctx context.Context) ([]*session.SessionInfo, error)
	Status(ctx context.Context, id string) (*session.SessionInfo, error)
	Events(ctx context.Context, id string, query store.EventQuery) ([]store.SessionEvent, error)
	Stop(ctx context.Context, id string) error
	Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error)
}

type hostAPIObserver interface {
	Health(ctx context.Context) (observepkg.Health, error)
	QueryEvents(ctx context.Context, query store.EventSummaryQuery) ([]store.EventSummary, error)
}

type hostAPISkillsRegistry interface {
	List() []*skillspkg.Skill
	ForWorkspace(ctx context.Context, resolved workspacepkg.ResolvedWorkspace) ([]*skillspkg.Skill, error)
}

// WithHostAPICapabilityChecker injects the capability checker used for Host API authorization.
func WithHostAPICapabilityChecker(checker *CapabilityChecker) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.capChecker = checker
	}
}

// WithHostAPIWorkspaceResolver injects workspace resolution for workspace-scoped Host API methods.
func WithHostAPIWorkspaceResolver(resolver workspacepkg.WorkspaceResolver) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.workspaces = resolver
	}
}

// WithHostAPIRateLimit overrides the per-extension Host API token bucket settings.
func WithHostAPIRateLimit(limit int, burst int) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.rateLimit = limit
		handler.rateBurst = burst
	}
}

// WithHostAPINow overrides the handler clock, mainly for tests.
func WithHostAPINow(now func() time.Time) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.now = now
	}
}

// NewHostAPIHandler constructs a Host API handler with sensible defaults.
func NewHostAPIHandler(
	sessions hostAPISessionManager,
	memoryStore *memory.Store,
	observer hostAPIObserver,
	skillsRegistry hostAPISkillsRegistry,
	opts ...HostAPIOption,
) *HostAPIHandler {
	handler := &HostAPIHandler{
		sessions:   sessions,
		memory:     memoryStore,
		observer:   observer,
		skills:     skillsRegistry,
		capChecker: &CapabilityChecker{},
		rateLimit:  defaultHostAPIRateLimit,
		rateBurst:  defaultHostAPIBurst,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}

	for _, opt := range opts {
		if opt != nil {
			opt(handler)
		}
	}

	if handler.now == nil {
		handler.now = func() time.Time {
			return time.Now().UTC()
		}
	}
	if handler.capChecker == nil {
		handler.capChecker = &CapabilityChecker{}
	}
	handler.limiter = newHostAPIRateLimiter(handler.rateLimit, handler.rateBurst, handler.now)

	handler.methods = map[string]hostAPIMethodFunc{
		"memory/forget":   handler.handleMemoryForget,
		"memory/recall":   handler.handleMemoryRecall,
		"memory/store":    handler.handleMemoryStore,
		"observe/events":  handler.handleObserveEvents,
		"observe/health":  handler.handleObserveHealth,
		"sessions/create": handler.handleSessionsCreate,
		"sessions/events": handler.handleSessionsEvents,
		"sessions/list":   handler.handleSessionsList,
		"sessions/prompt": handler.handleSessionsPrompt,
		"sessions/status": handler.handleSessionsStatus,
		"sessions/stop":   handler.handleSessionsStop,
		"skills/list":     handler.handleSkillsList,
	}

	return handler
}

// Handle dispatches one Host API request for the named extension.
func (h *HostAPIHandler) Handle(ctx context.Context, extName string, method string, params json.RawMessage) (any, error) {
	if h == nil {
		return nil, errors.New("extension: host api handler is required")
	}
	if ctx == nil {
		return nil, errors.New("extension: host api context is required")
	}

	method = strings.TrimSpace(method)
	handler, ok := h.methods[method]
	if !ok {
		return nil, methodNotFoundRPCError(method)
	}

	if err := h.capChecker.CheckHostAPI(extName, method); err != nil {
		return nil, rpcCapabilityDenied(err)
	}
	if err := h.limiter.Allow(extName, method); err != nil {
		return nil, err
	}

	return handler(ctx, params)
}

// HandleMethod returns a subprocess-compatible handler for one Host API method.
func (h *HostAPIHandler) HandleMethod(method string) subprocess.HandlerFunc {
	method = strings.TrimSpace(method)
	return func(ctx context.Context, params json.RawMessage) (any, error) {
		return h.Handle(ctx, hostAPIExtensionNameFromContext(ctx), method, params)
	}
}

// MethodHandlers returns the subprocess-compatible handler set for every Host API method.
func (h *HostAPIHandler) MethodHandlers() map[string]subprocess.HandlerFunc {
	out := make(map[string]subprocess.HandlerFunc, len(h.methods))
	for method := range h.methods {
		out[method] = h.HandleMethod(method)
	}
	return out
}

type hostAPISessionsListParams = extensioncontract.SessionsListParams

type hostAPISessionCreateParams = extensioncontract.SessionsCreateParams

type hostAPISessionPromptParams = extensioncontract.SessionsPromptParams

type hostAPISessionTargetParams = extensioncontract.SessionTargetParams

type hostAPISessionEventsParams = extensioncontract.SessionEventsParams

type hostAPIMemoryStoreParams = extensioncontract.MemoryStoreParams

type hostAPIMemoryRecallParams = extensioncontract.MemoryRecallParams

type hostAPIMemoryForgetParams = extensioncontract.MemoryForgetParams

type hostAPIObserveEventsParams = extensioncontract.ObserveEventsParams

type hostAPISkillsListParams = extensioncontract.SkillsListParams

type hostAPISessionSummary = extensioncontract.SessionSummary

type hostAPISessionStatus = extensioncontract.SessionStatus

type hostAPISessionEvent = extensioncontract.SessionEvent

type hostAPISessionCreateResult = extensioncontract.SessionCreateResult

type hostAPISessionPromptResult = extensioncontract.SessionPromptResult

type hostAPIMemoryRecallEntry = extensioncontract.MemoryRecallEntry

type hostAPISkillSummary = extensioncontract.SkillSummary

func (h *HostAPIHandler) handleSessionsList(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.sessions == nil {
		return nil, errors.New("extension: session manager is not configured")
	}

	var params hostAPISessionsListParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	infos, err := h.sessions.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	filterWorkspaceID := ""
	filterWorkspaceRoot := ""
	if workspaceRef := strings.TrimSpace(params.Workspace); workspaceRef != "" {
		if h.workspaces != nil {
			resolved, resolveErr := h.workspaces.Resolve(ctx, workspaceRef)
			if resolveErr != nil {
				return nil, resolveErr
			}
			filterWorkspaceID = strings.TrimSpace(resolved.ID)
			filterWorkspaceRoot = strings.TrimSpace(resolved.RootDir)
		} else {
			filterWorkspaceID = workspaceRef
			filterWorkspaceRoot = workspaceRef
		}
	}

	result := make([]hostAPISessionSummary, 0, len(infos))
	for _, info := range infos {
		if info == nil {
			continue
		}
		if filterWorkspaceID != "" || filterWorkspaceRoot != "" {
			if info.WorkspaceID != filterWorkspaceID && info.Workspace != filterWorkspaceRoot {
				continue
			}
		}
		result = append(result, hostAPISessionSummary{
			ID:        info.ID,
			Name:      info.Name,
			Agent:     info.AgentName,
			Workspace: info.Workspace,
			State:     info.State,
			CreatedAt: info.CreatedAt,
		})
	}

	return result, nil
}

func (h *HostAPIHandler) handleSessionsCreate(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.sessions == nil {
		return nil, errors.New("extension: session manager is not configured")
	}

	var params hostAPISessionCreateParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.Agent) == "" {
		return nil, invalidParamsRPCError(errors.New("agent is required"))
	}

	sess, err := h.sessions.Create(ctx, session.CreateOpts{
		AgentName: strings.TrimSpace(params.Agent),
		Workspace: strings.TrimSpace(params.Workspace),
		Type:      session.SessionTypeSystem,
	})
	if err != nil {
		return nil, err
	}

	if prompt := strings.TrimSpace(params.Prompt); prompt != "" {
		if _, err := h.submitPrompt(ctx, sess.ID, prompt); err != nil {
			return nil, err
		}
	}

	return hostAPISessionCreateResult{SessionID: sess.ID}, nil
}

func (h *HostAPIHandler) handleSessionsPrompt(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPISessionPromptParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.SessionID) == "" {
		return nil, invalidParamsRPCError(errors.New("session_id is required"))
	}
	if strings.TrimSpace(params.Message) == "" {
		return nil, invalidParamsRPCError(errors.New("message is required"))
	}

	turnID, err := h.submitPrompt(ctx, params.SessionID, params.Message)
	if err != nil {
		return nil, err
	}

	return hostAPISessionPromptResult{TurnID: turnID}, nil
}

func (h *HostAPIHandler) handleSessionsStop(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPISessionTargetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if h.sessions == nil {
		return nil, errors.New("extension: session manager is not configured")
	}
	if strings.TrimSpace(params.SessionID) == "" {
		return nil, invalidParamsRPCError(errors.New("session_id is required"))
	}
	if err := h.sessions.Stop(ctx, params.SessionID); err != nil {
		return nil, err
	}
	return struct{}{}, nil
}

func (h *HostAPIHandler) handleSessionsStatus(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPISessionTargetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if h.sessions == nil {
		return nil, errors.New("extension: session manager is not configured")
	}
	if strings.TrimSpace(params.SessionID) == "" {
		return nil, invalidParamsRPCError(errors.New("session_id is required"))
	}

	info, err := h.sessions.Status(ctx, params.SessionID)
	if err != nil {
		return nil, err
	}
	return hostAPISessionStatusFromInfo(info), nil
}

func (h *HostAPIHandler) handleSessionsEvents(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPISessionEventsParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if h.sessions == nil {
		return nil, errors.New("extension: session manager is not configured")
	}
	if strings.TrimSpace(params.SessionID) == "" {
		return nil, invalidParamsRPCError(errors.New("session_id is required"))
	}

	events, err := h.sessions.Events(ctx, params.SessionID, store.EventQuery{
		Type:          strings.TrimSpace(params.Type),
		AgentName:     strings.TrimSpace(params.AgentName),
		TurnID:        strings.TrimSpace(params.TurnID),
		Since:         params.Since,
		Limit:         params.Limit,
		AfterSequence: params.Offset,
	})
	if err != nil {
		return nil, err
	}

	result := make([]hostAPISessionEvent, 0, len(events))
	for _, event := range events {
		result = append(result, hostAPISessionEvent{
			Type:      event.Type,
			Timestamp: event.Timestamp,
			Data:      decodeJSONValue(event.Content),
		})
	}
	return result, nil
}

func (h *HostAPIHandler) handleMemoryStore(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPIMemoryStoreParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.Key) == "" {
		return nil, invalidParamsRPCError(errors.New("key is required"))
	}
	if strings.TrimSpace(params.Content) == "" {
		return nil, invalidParamsRPCError(errors.New("content is required"))
	}

	storeHandle, scope, err := h.memoryStoreFor(ctx, string(params.Scope), params.Workspace)
	if err != nil {
		return nil, err
	}

	filename := normalizeMemoryFilename(params.Key)
	doc, err := renderMemoryDocument(hostAPIMemoryDocument{
		Key:       filename,
		Scope:     scope,
		Content:   params.Content,
		Tags:      params.Tags,
		AgentName: hostAPIExtensionNameFromContext(ctx),
	})
	if err != nil {
		return nil, err
	}
	if err := storeHandle.Write(scope, filename, []byte(doc)); err != nil {
		return nil, err
	}
	return struct{}{}, nil
}

func (h *HostAPIHandler) handleMemoryRecall(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPIMemoryRecallParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	query := strings.TrimSpace(params.Query)
	if query == "" {
		return nil, invalidParamsRPCError(errors.New("query is required"))
	}

	sources, err := h.memorySourcesForRecall(ctx, string(params.Scope), params.Workspace)
	if err != nil {
		return nil, err
	}

	results := make([]hostAPIMemoryRecallEntry, 0)
	for _, source := range sources {
		headers, scanErr := source.store.Scan(source.scope)
		if scanErr != nil {
			return nil, scanErr
		}
		for _, header := range headers {
			content, readErr := source.store.Read(source.scope, header.Filename)
			if readErr != nil {
				return nil, readErr
			}
			body, tags := extractMemoryBodyAndTags(content)
			score := scoreMemoryRecall(query, header, body, tags)
			if score <= 0 {
				continue
			}
			results = append(results, hostAPIMemoryRecallEntry{
				Key:     header.Filename,
				Content: body,
				Score:   score,
			})
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Key < results[j].Key
		}
		return results[i].Score > results[j].Score
	})

	limit := params.Limit
	if limit <= 0 {
		limit = defaultHostAPIRecallLimit
	}
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func (h *HostAPIHandler) handleMemoryForget(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPIMemoryForgetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.Key) == "" {
		return nil, invalidParamsRPCError(errors.New("key is required"))
	}

	storeHandle, scope, err := h.memoryStoreFor(ctx, string(params.Scope), params.Workspace)
	if err != nil {
		return nil, err
	}
	if err := storeHandle.Delete(scope, normalizeMemoryFilename(params.Key)); err != nil {
		return nil, err
	}
	return struct{}{}, nil
}

func (h *HostAPIHandler) handleObserveHealth(ctx context.Context, _ json.RawMessage) (any, error) {
	if h.observer == nil {
		return nil, errors.New("extension: observer is not configured")
	}
	return h.observer.Health(ctx)
}

func (h *HostAPIHandler) handleObserveEvents(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.observer == nil {
		return nil, errors.New("extension: observer is not configured")
	}

	var params hostAPIObserveEventsParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	events, err := h.observer.QueryEvents(ctx, store.EventSummaryQuery{
		SessionID: strings.TrimSpace(params.SessionID),
		AgentName: strings.TrimSpace(params.AgentName),
		Type:      strings.TrimSpace(params.Type),
		Since:     params.Since,
		Limit:     params.Limit,
	})
	if err != nil {
		return nil, err
	}

	result := make([]hostAPISessionEvent, 0, len(events))
	for _, event := range events {
		result = append(result, hostAPISessionEvent{
			Type:      event.Type,
			Timestamp: event.Timestamp,
			Data: map[string]any{
				"session_id": event.SessionID,
				"agent_name": event.AgentName,
				"summary":    event.Summary,
			},
		})
	}

	return result, nil
}

func (h *HostAPIHandler) handleSkillsList(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.skills == nil {
		return nil, errors.New("extension: skills registry is not configured")
	}

	var params hostAPISkillsListParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	var (
		skills []*skillspkg.Skill
		err    error
	)
	if workspaceRef := strings.TrimSpace(params.Workspace); workspaceRef != "" {
		if h.workspaces == nil {
			return nil, errors.New("extension: workspace resolver is not configured")
		}
		resolved, resolveErr := h.workspaces.Resolve(ctx, workspaceRef)
		if resolveErr != nil {
			return nil, resolveErr
		}
		skills, err = h.skills.ForWorkspace(ctx, resolved)
	} else {
		skills = h.skills.List()
	}
	if err != nil {
		return nil, err
	}

	result := make([]hostAPISkillSummary, 0, len(skills))
	for _, skill := range skills {
		if skill == nil {
			continue
		}
		result = append(result, hostAPISkillSummary{
			Name:        skill.Meta.Name,
			Description: skill.Meta.Description,
			Source:      skillspkg.SkillSourceName(skill.Source),
		})
	}
	return result, nil
}

func (h *HostAPIHandler) submitPrompt(ctx context.Context, sessionID string, message string) (string, error) {
	if h.sessions == nil {
		return "", errors.New("extension: session manager is not configured")
	}

	lastSequence, err := h.latestSessionSequence(ctx, sessionID)
	if err != nil {
		return "", err
	}

	promptCtx := context.WithoutCancel(ctx)
	eventsCh, err := h.sessions.Prompt(promptCtx, sessionID, message)
	if err != nil {
		return "", err
	}
	go drainAgentEvents(eventsCh)

	events, err := h.sessions.Events(ctx, sessionID, store.EventQuery{
		Type:          acp.EventTypeUserMessage,
		Limit:         1,
		AfterSequence: lastSequence,
	})
	if err != nil {
		return "", err
	}
	if len(events) == 0 || strings.TrimSpace(events[0].TurnID) == "" {
		return "", errors.New("extension: prompt turn id not found after prompt submission")
	}
	return strings.TrimSpace(events[0].TurnID), nil
}

func (h *HostAPIHandler) latestSessionSequence(ctx context.Context, sessionID string) (int64, error) {
	events, err := h.sessions.Events(ctx, sessionID, store.EventQuery{Limit: 1})
	if err != nil {
		return 0, err
	}
	if len(events) == 0 {
		return 0, nil
	}
	return events[len(events)-1].Sequence, nil
}

type hostAPIMemorySource struct {
	store *memory.Store
	scope memory.Scope
}

func (h *HostAPIHandler) memorySourcesForRecall(
	ctx context.Context,
	rawScope string,
	rawWorkspace string,
) ([]hostAPIMemorySource, error) {
	if h.memory == nil {
		return nil, errors.New("extension: memory store is not configured")
	}

	scope := memory.Scope(strings.TrimSpace(rawScope)).Normalize()
	switch scope {
	case "":
		sources := []hostAPIMemorySource{{store: h.memory, scope: memory.ScopeGlobal}}
		workspaceRoot, err := h.resolveWorkspaceRoot(ctx, rawWorkspace)
		if err != nil {
			return nil, err
		}
		if workspaceRoot != "" {
			sources = append(sources, hostAPIMemorySource{
				store: h.memory.ForWorkspace(workspaceRoot),
				scope: memory.ScopeWorkspace,
			})
		}
		return sources, nil
	case memory.ScopeGlobal:
		return []hostAPIMemorySource{{store: h.memory, scope: memory.ScopeGlobal}}, nil
	case memory.ScopeWorkspace:
		storeHandle, _, err := h.memoryStoreFor(ctx, rawScope, rawWorkspace)
		if err != nil {
			return nil, err
		}
		return []hostAPIMemorySource{{store: storeHandle, scope: memory.ScopeWorkspace}}, nil
	default:
		return nil, invalidParamsRPCError(fmt.Errorf("memory scope must be one of global or workspace"))
	}
}

func (h *HostAPIHandler) memoryStoreFor(ctx context.Context, rawScope string, rawWorkspace string) (*memory.Store, memory.Scope, error) {
	if h.memory == nil {
		return nil, "", errors.New("extension: memory store is not configured")
	}

	scope := memory.Scope(strings.TrimSpace(rawScope)).Normalize()
	workspaceRoot, err := h.resolveWorkspaceRoot(ctx, rawWorkspace)
	if err != nil {
		return nil, "", err
	}
	if scope == "" {
		if workspaceRoot != "" {
			scope = memory.ScopeWorkspace
		} else {
			scope = memory.ScopeGlobal
		}
	}

	switch scope {
	case memory.ScopeGlobal:
		return h.memory, memory.ScopeGlobal, nil
	case memory.ScopeWorkspace:
		if workspaceRoot == "" {
			return nil, "", invalidParamsRPCError(errors.New("workspace is required for workspace memory scope"))
		}
		return h.memory.ForWorkspace(workspaceRoot), memory.ScopeWorkspace, nil
	default:
		return nil, "", invalidParamsRPCError(fmt.Errorf("memory scope must be one of global or workspace"))
	}
}

func (h *HostAPIHandler) resolveWorkspaceRoot(ctx context.Context, rawWorkspace string) (string, error) {
	if strings.TrimSpace(rawWorkspace) == "" {
		return "", nil
	}
	if h.workspaces == nil {
		return "", invalidParamsRPCError(errors.New("workspace resolver is not configured"))
	}
	resolved, err := h.workspaces.Resolve(ctx, rawWorkspace)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resolved.RootDir), nil
}

type hostAPIMemoryDocument struct {
	Key       string
	Scope     memory.Scope
	Content   string
	Tags      []string
	AgentName string
}

func renderMemoryDocument(doc hostAPIMemoryDocument) (string, error) {
	header := memory.MemoryHeader{
		Name:        memoryNameFromFilename(doc.Key),
		Description: memoryDescriptionFromContent(doc.Content),
		Type:        memoryTypeForScope(doc.Scope, doc.Tags),
		AgentName:   strings.TrimSpace(doc.AgentName),
	}
	if err := header.Validate(); err != nil {
		return "", invalidParamsRPCError(err)
	}

	metadata, err := yaml.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("extension: marshal memory frontmatter: %w", err)
	}

	var builder strings.Builder
	builder.WriteString("---\n")
	builder.Write(metadata)
	builder.WriteString("---\n\n")
	body := strings.TrimSpace(doc.Content)
	tags := normalizeUniqueStrings(doc.Tags)
	if len(tags) > 0 {
		builder.WriteString(tagCommentPrefix)
		builder.WriteByte(' ')
		builder.WriteString(strings.Join(tags, ", "))
		builder.WriteString(" -->\n\n")
	}
	builder.WriteString(body)
	return builder.String(), nil
}

func memoryTypeForScope(scope memory.Scope, tags []string) memory.MemoryType {
	for _, tag := range normalizeUniqueStrings(tags) {
		switch memory.MemoryType(tag).Normalize() {
		case memory.MemoryTypeUser, memory.MemoryTypeFeedback, memory.MemoryTypeProject, memory.MemoryTypeReference:
			return memory.MemoryType(tag).Normalize()
		}
	}
	if scope == memory.ScopeWorkspace {
		return memory.MemoryTypeProject
	}
	return memory.MemoryTypeUser
}

func memoryNameFromFilename(filename string) string {
	base := strings.TrimSuffix(filepath.Base(strings.TrimSpace(filename)), filepath.Ext(strings.TrimSpace(filename)))
	if base == "" {
		return ""
	}

	normalized := strings.NewReplacer("-", " ", "_", " ", ".", " ").Replace(base)
	parts := strings.Fields(normalized)
	for i, part := range parts {
		parts[i] = titleCaseWord(part)
	}
	return strings.Join(parts, " ")
}

func titleCaseWord(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) == 1 {
		return strings.ToUpper(trimmed)
	}
	return strings.ToUpper(trimmed[:1]) + strings.ToLower(trimmed[1:])
}

func memoryDescriptionFromContent(content string) string {
	firstLine := strings.TrimSpace(strings.Split(strings.TrimSpace(content), "\n")[0])
	if len(firstLine) <= maxMemoryDescriptionLength {
		return firstLine
	}
	return strings.TrimSpace(firstLine[:maxMemoryDescriptionLength]) + "..."
}

func normalizeMemoryFilename(key string) string {
	filename := strings.TrimSpace(key)
	if filepath.Ext(filename) == "" {
		filename += ".md"
	}
	return filename
}

func extractMemoryBodyAndTags(content []byte) (string, []string) {
	body := strings.TrimSpace(string(content))
	parts, err := frontmatter.Split(content)
	if err == nil {
		body = strings.TrimSpace(parts.Body)
	}
	if !strings.HasPrefix(body, tagCommentPrefix) {
		return body, nil
	}

	lineEnd := strings.IndexByte(body, '\n')
	if lineEnd < 0 {
		lineEnd = len(body)
	}
	comment := strings.TrimSpace(body[:lineEnd])
	body = strings.TrimSpace(strings.TrimPrefix(body[lineEnd:], "\n"))

	comment = strings.TrimPrefix(comment, tagCommentPrefix)
	comment = strings.TrimSuffix(comment, "-->")
	comment = strings.TrimSpace(comment)
	if comment == "" {
		return body, nil
	}
	return body, normalizeUniqueStrings(strings.Split(comment, ","))
}

func scoreMemoryRecall(query string, header memory.MemoryHeader, body string, tags []string) float64 {
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	if normalizedQuery == "" {
		return 0
	}

	haystack := strings.ToLower(strings.Join([]string{
		header.Filename,
		header.Name,
		header.Description,
		header.AgentName,
		strings.Join(tags, " "),
		body,
	}, " "))

	score := 0.0
	if strings.Contains(haystack, normalizedQuery) {
		score += 4
	}

	for _, token := range strings.Fields(normalizedQuery) {
		if strings.Contains(haystack, token) {
			score++
		}
	}

	return score
}

func hostAPISessionStatusFromInfo(info *session.SessionInfo) hostAPISessionStatus {
	if info == nil {
		return hostAPISessionStatus{}
	}
	return hostAPISessionStatus{
		SessionID:    info.ID,
		Name:         info.Name,
		Agent:        info.AgentName,
		WorkspaceID:  info.WorkspaceID,
		Workspace:    info.Workspace,
		State:        info.State,
		StopReason:   info.StopReason,
		StopDetail:   info.StopDetail,
		ACPSessionID: info.ACPSessionID,
		CreatedAt:    info.CreatedAt,
		UpdatedAt:    info.UpdatedAt,
	}
}

func decodeHostAPIParams(raw json.RawMessage, target any) error {
	if target == nil {
		return errors.New("extension: host api params target is required")
	}
	payload := raw
	if len(payload) == 0 || strings.TrimSpace(string(payload)) == "" || string(payload) == "null" {
		payload = json.RawMessage(`{}`)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return invalidParamsRPCError(fmt.Errorf("decode params: %w", err))
	}
	return nil
}

func decodeJSONValue(raw string) any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
		return decoded
	}
	return trimmed
}

func invalidParamsRPCError(err error) error {
	if err == nil {
		return subprocess.NewRPCError(HostAPIInvalidParamsCode, "Invalid params", nil)
	}
	return subprocess.NewRPCError(HostAPIInvalidParamsCode, "Invalid params", map[string]string{"error": err.Error()})
}

func methodNotFoundRPCError(method string) error {
	return subprocess.NewRPCError(HostAPIMethodNotFoundCode, "Method not found", map[string]string{"method": strings.TrimSpace(method)})
}

func rpcCapabilityDenied(err error) error {
	var denied *ErrCapabilityDenied
	if !errors.As(err, &denied) {
		return err
	}
	return subprocess.NewRPCError(denied.Code(), "Capability denied", denied.Data)
}

func withHostAPIExtensionName(ctx context.Context, extName string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, hostAPIExtensionNameContextKey, strings.TrimSpace(extName))
}

func hostAPIExtensionNameFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(hostAPIExtensionNameContextKey).(string)
	return strings.TrimSpace(value)
}

func drainAgentEvents(events <-chan acp.AgentEvent) {
	for range events {
	}
}

type hostAPIRateLimiter struct {
	mu      sync.Mutex
	now     func() time.Time
	limit   int
	burst   int
	entries map[string]hostAPIRateState
}

type hostAPIRateState struct {
	tokens    float64
	updatedAt time.Time
}

func newHostAPIRateLimiter(limit int, burst int, now func() time.Time) *hostAPIRateLimiter {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &hostAPIRateLimiter{
		now:     now,
		limit:   limit,
		burst:   burst,
		entries: make(map[string]hostAPIRateState),
	}
}

func (l *hostAPIRateLimiter) Allow(extName string, method string) error {
	if l == nil || l.limit <= 0 || l.burst <= 0 {
		return nil
	}

	key := strings.TrimSpace(extName)
	if key == "" {
		key = "unknown"
	}
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	state := l.entries[key]
	if state.updatedAt.IsZero() {
		state.tokens = float64(l.burst)
		state.updatedAt = now
	}

	elapsed := now.Sub(state.updatedAt).Seconds()
	if elapsed > 0 {
		state.tokens = minFloat(float64(l.burst), state.tokens+(elapsed*float64(l.limit)))
		state.updatedAt = now
	}

	if state.tokens >= 1 {
		state.tokens--
		l.entries[key] = state
		return nil
	}

	needed := 1 - state.tokens
	retryAfter := time.Duration((needed / float64(l.limit)) * float64(time.Second))
	if retryAfter < time.Millisecond {
		retryAfter = time.Millisecond
	}
	l.entries[key] = state

	return subprocess.NewRPCError(HostAPIRateLimitedCode, "Rate limited", map[string]any{
		"scope":          "host_api." + strings.TrimSpace(method),
		"retry_after_ms": retryAfter.Milliseconds(),
		"limit":          l.limit,
		"burst":          l.burst,
	})
}

func minFloat(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}
