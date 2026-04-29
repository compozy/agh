package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	core "github.com/pedronauck/agh/internal/api/core"
	aghconfig "github.com/pedronauck/agh/internal/config"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	mcppkg "github.com/pedronauck/agh/internal/mcp"
	mcpauth "github.com/pedronauck/agh/internal/mcp/auth"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/skills"
	taskpkg "github.com/pedronauck/agh/internal/task"
	toolspkg "github.com/pedronauck/agh/internal/tools"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type daemonNativeToolsDeps struct {
	Registry          func() toolspkg.Registry
	Skills            core.SkillsRegistry
	WorkspaceResolver workspacepkg.RuntimeResolver
	Network           core.NetworkService
	Tasks             taskpkg.Manager
}

type daemonNativeTools struct {
	deps daemonNativeToolsDeps
}

type nativeToolBinding struct {
	call         toolspkg.NativeToolFunc
	availability toolspkg.NativeAvailabilityFunc
}

func newDaemonNativeProvider(deps daemonNativeToolsDeps) (toolspkg.Provider, error) {
	adapter := &daemonNativeTools{deps: deps}
	bindings := adapter.bindings()
	descriptors := toolspkg.BuiltinNativeDescriptors()
	nativeTools := make([]toolspkg.NativeTool, 0, len(descriptors))
	for _, descriptor := range descriptors {
		binding, ok := bindings[descriptor.ID]
		if !ok {
			return nil, fmt.Errorf("daemon: missing native handler for %s", descriptor.ID)
		}
		nativeTools = append(nativeTools, toolspkg.NativeTool{
			Descriptor:   descriptor,
			Call:         binding.call,
			Availability: binding.availability,
		})
	}
	return toolspkg.NewNativeProvider(toolspkg.BuiltinSource(), nativeTools...)
}

func (d *Daemon) bootToolRegistry(_ context.Context, state *bootState) error {
	if state == nil {
		return errors.New("daemon: tool registry state is required")
	}
	if state.mcpServerCatalog == nil {
		state.mcpServerCatalog = newResourceCatalog(cloneDaemonMCPServer)
	}
	var registry *toolspkg.RuntimeRegistry
	provider, err := newDaemonNativeProvider(daemonNativeToolsDeps{
		Registry: func() toolspkg.Registry {
			return registry
		},
		Skills:            skillsRegistryAPI(state.skillsRegistry),
		WorkspaceResolver: state.workspaceResolver,
		Network:           state.deps.Network,
		Tasks:             state.deps.Tasks,
	})
	if err != nil {
		return fmt.Errorf("daemon: create native tool provider: %w", err)
	}
	policyInputs, err := nativeToolPolicyInputs(&state.cfg)
	if err != nil {
		return fmt.Errorf("daemon: build native tool policy inputs: %w", err)
	}
	var approvalBridge *toolApprovalBridge
	if _, ok := state.sessions.(sessionPermissionRequester); ok {
		approvalBridge = newToolApprovalBridge(
			func() sessionPermissionRequester {
				requester, ok := state.sessions.(sessionPermissionRequester)
				if !ok {
					return nil
				}
				return requester
			},
			state.cfg.Tools.Policy.ApprovalTimeout(),
		)
		policyInputs.ApprovalAvailable = true
	}
	toolsets, err := toolspkg.BuiltinToolsetCatalog()
	if err != nil {
		return fmt.Errorf("daemon: build native toolset catalog: %w", err)
	}
	providers := []toolspkg.Provider{provider}
	extensionProvider, err := newDaemonExtensionToolProvider(state)
	if err != nil {
		return fmt.Errorf("daemon: create extension tool provider: %w", err)
	}
	if extensionProvider != nil {
		providers = append(providers, extensionProvider)
	}
	mcpProvider, err := d.newDaemonMCPToolProvider(state)
	if err != nil {
		return fmt.Errorf("daemon: create mcp tool provider: %w", err)
	}
	if mcpProvider != nil {
		providers = append(providers, mcpProvider)
	}
	registry, err = toolspkg.NewRegistry(
		toolspkg.WithProviders(providers...),
		toolspkg.WithPolicyInputs(policyInputs, toolsets),
		toolspkg.WithApprovalBridge(approvalBridge),
		toolspkg.WithDefaultMaxResultBytes(state.cfg.Tools.DefaultMaxResultBytes),
	)
	if err != nil {
		return fmt.Errorf("daemon: create tool registry: %w", err)
	}
	state.toolRegistry = registry
	state.deps.ToolRegistry = registry
	return nil
}

func (d *Daemon) newDaemonMCPToolProvider(state *bootState) (toolspkg.Provider, error) {
	if state == nil {
		return nil, nil
	}
	resolver := mcppkg.ServerResolverFunc(func(context.Context) ([]aghconfig.MCPServer, error) {
		return daemonMCPServerConfigs(state), nil
	})
	options := []mcppkg.CallExecutorOption{}
	if d != nil && d.getenv != nil {
		options = append(options, mcppkg.WithSecretLookup(d.getenv))
	}
	if store, ok := state.registry.(mcpauth.TokenStore); ok {
		options = append(options, mcppkg.WithTokenStore(store))
	}
	executor, err := mcppkg.NewMCPCallExecutor(resolver, options...)
	if err != nil {
		return nil, err
	}
	return toolspkg.NewMCPProvider(
		toolspkg.MCPSourceListerFunc(func(context.Context) ([]toolspkg.SourceRef, error) {
			return daemonMCPSources(state), nil
		}),
		executor,
		executor,
	)
}

func newDaemonExtensionToolProvider(state *bootState) (toolspkg.Provider, error) {
	if state == nil || state.registry == nil {
		return nil, nil
	}
	dbSource, ok := state.registry.(extensionDBSource)
	if !ok || dbSource.DB() == nil {
		return nil, nil
	}
	return extensionpkg.NewExtensionToolProvider(
		extensionpkg.NewRegistry(dbSource.DB()),
		func() extensionpkg.ExtensionToolRuntime {
			runtime := state.currentExtensionRuntime()
			if runtime == nil {
				return nil
			}
			toolRuntime, ok := runtime.(extensionpkg.ExtensionToolRuntime)
			if !ok {
				return nil
			}
			return toolRuntime
		},
	)
}

func daemonMCPServerConfigs(state *bootState) []aghconfig.MCPServer {
	if state == nil {
		return nil
	}
	servers := make([]aghconfig.MCPServer, 0, len(state.cfg.MCPServers))
	seen := map[string]struct{}{}
	add := func(server aghconfig.MCPServer) {
		name := strings.TrimSpace(server.Name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		servers = append(servers, cloneDaemonMCPServer(server))
	}
	for _, server := range state.cfg.MCPServers {
		add(server)
	}
	providerNames := make([]string, 0, len(state.cfg.Providers))
	for name := range state.cfg.Providers {
		providerNames = append(providerNames, name)
	}
	slices.Sort(providerNames)
	for _, name := range providerNames {
		for _, server := range state.cfg.Providers[name].MCPServers {
			add(server)
		}
	}
	if state.mcpServerCatalog != nil {
		for _, record := range state.mcpServerCatalog.Snapshot() {
			add(record.Spec)
		}
	}
	return servers
}

func daemonMCPSources(state *bootState) []toolspkg.SourceRef {
	if state == nil {
		return nil
	}
	sources := make([]toolspkg.SourceRef, 0, len(state.cfg.MCPServers))
	seen := map[string]struct{}{}
	add := func(server aghconfig.MCPServer, source toolspkg.SourceRef) {
		name := strings.TrimSpace(server.Name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		source.Kind = toolspkg.SourceMCP
		source.Owner = name
		source.RawServerName = name
		sources = append(sources, source)
	}
	for _, server := range state.cfg.MCPServers {
		add(server, toolspkg.SourceRef{})
	}
	providerNames := make([]string, 0, len(state.cfg.Providers))
	for name := range state.cfg.Providers {
		providerNames = append(providerNames, name)
	}
	slices.Sort(providerNames)
	for _, name := range providerNames {
		for _, server := range state.cfg.Providers[name].MCPServers {
			add(server, toolspkg.SourceRef{})
		}
	}
	if state.mcpServerCatalog != nil {
		for _, record := range state.mcpServerCatalog.Snapshot() {
			add(record.Spec, toolspkg.SourceRef{
				ResourceID:      record.ID,
				ResourceVersion: fmt.Sprint(record.Version),
				WorkspaceID:     record.Scope.ID,
				Scope:           string(record.Scope.Kind),
			})
		}
	}
	return sources
}

func (n *daemonNativeTools) bindings() map[toolspkg.ToolID]nativeToolBinding {
	registryAvailability := n.registryAvailability()
	skillAvailability := n.dependencyAvailability(func() bool { return n.deps.Skills != nil })
	networkAvailability := n.dependencyAvailability(func() bool { return n.deps.Network != nil })
	taskAvailability := n.dependencyAvailability(func() bool { return n.deps.Tasks != nil })
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDToolList: {
			call:         n.toolList,
			availability: registryAvailability,
		},
		toolspkg.ToolIDToolSearch: {
			call:         n.toolSearch,
			availability: registryAvailability,
		},
		toolspkg.ToolIDToolInfo: {
			call:         n.toolInfo,
			availability: registryAvailability,
		},
		toolspkg.ToolIDSkillList: {
			call:         n.skillList,
			availability: skillAvailability,
		},
		toolspkg.ToolIDSkillSearch: {
			call:         n.skillSearch,
			availability: skillAvailability,
		},
		toolspkg.ToolIDSkillView: {
			call:         n.skillView,
			availability: skillAvailability,
		},
		toolspkg.ToolIDNetworkPeers: {
			call:         n.networkPeers,
			availability: networkAvailability,
		},
		toolspkg.ToolIDNetworkSend: {
			call:         n.networkSend,
			availability: networkAvailability,
		},
		toolspkg.ToolIDTaskList: {
			call:         n.taskList,
			availability: taskAvailability,
		},
		toolspkg.ToolIDTaskRead: {
			call:         n.taskRead,
			availability: taskAvailability,
		},
		toolspkg.ToolIDTaskCreate: {
			call:         n.taskCreate,
			availability: taskAvailability,
		},
		toolspkg.ToolIDTaskChildCreate: {
			call:         n.taskChildCreate,
			availability: taskAvailability,
		},
		toolspkg.ToolIDTaskUpdate: {
			call:         n.taskUpdate,
			availability: taskAvailability,
		},
		toolspkg.ToolIDTaskCancel: {
			call:         n.taskCancel,
			availability: taskAvailability,
		},
		toolspkg.ToolIDTaskRunList: {
			call:         n.taskRunList,
			availability: taskAvailability,
		},
	}
}

func (n *daemonNativeTools) registryAvailability() toolspkg.NativeAvailabilityFunc {
	return func(context.Context, toolspkg.Scope) toolspkg.Availability {
		if n.registry() == nil {
			return toolspkg.Unavailable(toolspkg.ReasonDependencyMissing)
		}
		return toolspkg.Available()
	}
}

func (n *daemonNativeTools) dependencyAvailability(ready func() bool) toolspkg.NativeAvailabilityFunc {
	return func(context.Context, toolspkg.Scope) toolspkg.Availability {
		if ready == nil || !ready() {
			return toolspkg.Unavailable(toolspkg.ReasonDependencyMissing)
		}
		return toolspkg.Available()
	}
}

func (n *daemonNativeTools) registry() toolspkg.Registry {
	if n == nil || n.deps.Registry == nil {
		return nil
	}
	return n.deps.Registry()
}

func (n *daemonNativeTools) toolList(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input toolListInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	views, err := n.registry().List(ctx, scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	views = limitToolViews(views, input.Limit)
	return structuredResult(map[string]any{"tools": views}, fmt.Sprintf("%d tools", len(views)))
}

func (n *daemonNativeTools) toolSearch(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input toolSearchInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	views, err := n.registry().Search(ctx, scope, toolspkg.SearchQuery{
		Query: input.Query,
		Limit: input.Limit,
	})
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	return structuredResult(map[string]any{"tools": views}, fmt.Sprintf("%d tools", len(views)))
}

func (n *daemonNativeTools) toolInfo(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input toolInfoInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	id := toolspkg.ToolID(strings.TrimSpace(input.ToolID))
	view, err := n.registry().Get(ctx, scope, id)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	return structuredResult(map[string]any{"tool": view}, view.Descriptor.ID.String())
}

func (n *daemonNativeTools) skillList(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input skillListInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	skillList, err := n.skillsFor(ctx, scope, input.WorkspaceID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := core.SkillPayloadsFromSkills(limitSkills(skillList, input.Limit))
	return structuredResult(map[string]any{"skills": payload}, fmt.Sprintf("%d skills", len(payload)))
}

func (n *daemonNativeTools) skillSearch(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input skillSearchInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	skillList, err := n.skillsFor(ctx, scope, input.WorkspaceID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	filtered := searchSkills(skillList, input.Query)
	payload := core.SkillPayloadsFromSkills(limitSkills(filtered, input.Limit))
	return structuredResult(map[string]any{"skills": payload}, fmt.Sprintf("%d skills", len(payload)))
}

func (n *daemonNativeTools) skillView(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input skillViewInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	skill, err := n.resolveSkill(ctx, scope, input.WorkspaceID, input.Name)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	content, err := n.deps.Skills.LoadContent(ctx, skill)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := map[string]any{
		"skill":   core.SkillPayloadFromSkill(skill),
		"content": content,
	}
	result, err := structuredResult(payload, content)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	result.Content = []toolspkg.ToolContent{{Type: "text", Text: content}}
	return result, nil
}

func (n *daemonNativeTools) networkPeers(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input networkPeersInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	peers, err := n.deps.Network.ListPeers(ctx, input.Channel)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	return structuredResult(map[string]any{"peers": peers}, fmt.Sprintf("%d peers", len(peers)))
}

func (n *daemonNativeTools) networkSend(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input networkSendInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	sessionID := firstNonEmpty(input.SessionID, req.SessionID, scope.SessionID)
	messageID, err := n.deps.Network.Send(ctx, network.SendRequest{
		SessionID:     sessionID,
		Channel:       strings.TrimSpace(input.Channel),
		Kind:          network.Kind(strings.TrimSpace(input.Kind)),
		To:            stringPtr(input.To),
		Body:          cloneJSON(input.Body),
		InteractionID: stringPtr(input.InteractionID),
		ReplyTo:       stringPtr(input.ReplyTo),
		TraceID:       stringPtr(input.TraceID),
		CausationID:   stringPtr(input.CausationID),
		ExpiresAt:     input.ExpiresAt,
		ID:            stringPtr(input.ID),
		Ext:           cloneExtensionMap(input.Ext),
	})
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	return structuredResult(map[string]any{"message_id": messageID}, messageID)
}

func (n *daemonNativeTools) taskList(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input taskListInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	actor, err := actorContextFromScope(scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	query := input.query(scope)
	summaries, err := n.deps.Tasks.ListTasks(ctx, query, actor)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	return structuredResult(map[string]any{"tasks": summaries}, fmt.Sprintf("%d tasks", len(summaries)))
}

func (n *daemonNativeTools) taskRead(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input taskReadInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	actor, err := actorContextFromScope(scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	view, err := n.deps.Tasks.GetTask(ctx, input.TaskID, actor)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	return structuredResult(map[string]any{"task": view}, view.Summary.Title)
}

func (n *daemonNativeTools) taskCreate(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input taskCreateInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	actor, err := actorContextFromScope(scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	created, err := n.deps.Tasks.CreateTask(ctx, input.spec(scope), actor)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	return structuredResult(map[string]any{"task": created}, created.Title)
}

func (n *daemonNativeTools) taskChildCreate(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input taskChildCreateInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	actor, err := actorContextFromScope(scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	created, err := n.deps.Tasks.CreateChildTask(ctx, input.ParentTaskID, input.spec(scope), actor)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	return structuredResult(map[string]any{"task": created}, created.Title)
}

func (n *daemonNativeTools) taskUpdate(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input taskUpdateInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	actor, err := actorContextFromScope(scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	updated, err := n.deps.Tasks.UpdateTask(ctx, input.TaskID, input.patch(), actor)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	return structuredResult(map[string]any{"task": updated}, updated.Title)
}

func (n *daemonNativeTools) taskCancel(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input taskCancelInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	actor, err := actorContextFromScope(scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	canceled, err := n.deps.Tasks.CancelTask(ctx, input.TaskID, input.cancel(), actor)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	return structuredResult(map[string]any{"task": canceled}, canceled.Title)
}

func (n *daemonNativeTools) taskRunList(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input taskRunListInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	actor, err := actorContextFromScope(scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	runs, err := n.deps.Tasks.ListTaskRuns(ctx, input.TaskID, input.query(), actor)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	return structuredResult(map[string]any{"runs": runs}, fmt.Sprintf("%d runs", len(runs)))
}

func (n *daemonNativeTools) skillsFor(
	ctx context.Context,
	scope toolspkg.Scope,
	workspaceID string,
) ([]*skills.Skill, error) {
	if n.deps.Skills == nil {
		return nil, errors.New("daemon: skills registry is required")
	}
	workspaceID = firstNonEmpty(workspaceID, scope.WorkspaceID)
	if workspaceID == "" {
		return n.deps.Skills.List(), nil
	}
	if n.deps.WorkspaceResolver == nil {
		return nil, errors.New("daemon: workspace resolver is required for workspace skills")
	}
	resolved, err := n.deps.WorkspaceResolver.Resolve(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return n.deps.Skills.ForWorkspace(ctx, &resolved)
}

func (n *daemonNativeTools) resolveSkill(
	ctx context.Context,
	scope toolspkg.Scope,
	workspaceID string,
	name string,
) (*skills.Skill, error) {
	trimmedName := strings.TrimSpace(name)
	workspaceID = firstNonEmpty(workspaceID, scope.WorkspaceID)
	if workspaceID == "" {
		skill, ok := n.deps.Skills.Get(trimmedName)
		if !ok {
			return nil, fmt.Errorf("daemon: skill %q not found", trimmedName)
		}
		return skill, nil
	}
	skillList, err := n.skillsFor(ctx, scope, workspaceID)
	if err != nil {
		return nil, err
	}
	for _, skill := range skillList {
		if skill != nil && skill.Meta.Name == trimmedName {
			return skill, nil
		}
	}
	return nil, fmt.Errorf("daemon: skill %q not found", trimmedName)
}

type toolListInput struct {
	Limit int `json:"limit,omitempty"`
}

type toolSearchInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

type toolInfoInput struct {
	ToolID string `json:"tool_id"`
}

type skillListInput struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type skillSearchInput struct {
	Query       string `json:"query"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type skillViewInput struct {
	Name        string `json:"name"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

type networkPeersInput struct {
	Channel string `json:"channel,omitempty"`
}

type networkSendInput struct {
	SessionID     string               `json:"session_id,omitempty"`
	Channel       string               `json:"channel"`
	Kind          string               `json:"kind"`
	To            string               `json:"to,omitempty"`
	Body          json.RawMessage      `json:"body"`
	InteractionID string               `json:"interaction_id,omitempty"`
	ReplyTo       string               `json:"reply_to,omitempty"`
	TraceID       string               `json:"trace_id,omitempty"`
	CausationID   string               `json:"causation_id,omitempty"`
	ExpiresAt     *int64               `json:"expires_at,omitempty"`
	ID            string               `json:"id,omitempty"`
	Ext           network.ExtensionMap `json:"ext,omitempty"`
}

type taskListInput struct {
	Scope          string `json:"scope,omitempty"`
	WorkspaceID    string `json:"workspace_id,omitempty"`
	Status         string `json:"status,omitempty"`
	Priority       string `json:"priority,omitempty"`
	ApprovalState  string `json:"approval_state,omitempty"`
	OwnerKind      string `json:"owner_kind,omitempty"`
	OwnerRef       string `json:"owner_ref,omitempty"`
	ParentTaskID   string `json:"parent_task_id,omitempty"`
	NetworkChannel string `json:"network_channel,omitempty"`
	Search         string `json:"search,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

func (i taskListInput) query(scope toolspkg.Scope) taskpkg.Query {
	query := taskpkg.Query{
		Scope:          taskpkg.Scope(strings.TrimSpace(i.Scope)),
		WorkspaceID:    strings.TrimSpace(i.WorkspaceID),
		Status:         taskpkg.Status(strings.TrimSpace(i.Status)),
		Priority:       taskpkg.Priority(strings.TrimSpace(i.Priority)),
		ApprovalState:  taskpkg.ApprovalState(strings.TrimSpace(i.ApprovalState)),
		OwnerKind:      taskpkg.OwnerKind(strings.TrimSpace(i.OwnerKind)),
		OwnerRef:       strings.TrimSpace(i.OwnerRef),
		ParentTaskID:   strings.TrimSpace(i.ParentTaskID),
		NetworkChannel: strings.TrimSpace(i.NetworkChannel),
		Search:         strings.TrimSpace(i.Search),
		Limit:          i.Limit,
	}
	if query.WorkspaceID == "" && scope.WorkspaceID != "" {
		switch query.Scope.Normalize() {
		case "", taskpkg.ScopeWorkspace:
			query.Scope = taskpkg.ScopeWorkspace
			query.WorkspaceID = strings.TrimSpace(scope.WorkspaceID)
		}
	}
	return query
}

type taskReadInput struct {
	TaskID string `json:"task_id"`
}

type taskCreateInput struct {
	ID             string             `json:"id,omitempty"`
	Identifier     string             `json:"identifier,omitempty"`
	Scope          string             `json:"scope"`
	WorkspaceID    string             `json:"workspace_id,omitempty"`
	NetworkChannel string             `json:"network_channel,omitempty"`
	Title          string             `json:"title"`
	Description    string             `json:"description,omitempty"`
	Priority       string             `json:"priority,omitempty"`
	MaxAttempts    *int               `json:"max_attempts,omitempty"`
	Draft          bool               `json:"draft,omitempty"`
	ApprovalPolicy string             `json:"approval_policy,omitempty"`
	Owner          *taskpkg.Ownership `json:"owner,omitempty"`
	Metadata       json.RawMessage    `json:"metadata,omitempty"`
}

func (i taskCreateInput) spec(scope toolspkg.Scope) taskpkg.CreateTask {
	taskScope := taskpkg.Scope(strings.TrimSpace(i.Scope))
	workspaceID := strings.TrimSpace(i.WorkspaceID)
	if workspaceID == "" && taskScope.Normalize() == taskpkg.ScopeWorkspace {
		workspaceID = strings.TrimSpace(scope.WorkspaceID)
	}
	return taskpkg.CreateTask{
		ID:             strings.TrimSpace(i.ID),
		Identifier:     strings.TrimSpace(i.Identifier),
		Scope:          taskScope,
		WorkspaceID:    workspaceID,
		NetworkChannel: strings.TrimSpace(i.NetworkChannel),
		Title:          strings.TrimSpace(i.Title),
		Description:    strings.TrimSpace(i.Description),
		Priority:       taskpkg.Priority(strings.TrimSpace(i.Priority)),
		MaxAttempts:    cloneIntPtr(i.MaxAttempts),
		Draft:          i.Draft,
		ApprovalPolicy: taskpkg.ApprovalPolicy(strings.TrimSpace(i.ApprovalPolicy)),
		Owner:          cloneTaskOwner(i.Owner),
		Metadata:       cloneJSON(i.Metadata),
	}
}

type taskChildCreateInput struct {
	ParentTaskID string `json:"parent_task_id"`
	taskCreateInput
}

func (i taskChildCreateInput) spec(scope toolspkg.Scope) taskpkg.CreateTask {
	spec := i.taskCreateInput.spec(scope)
	spec.ParentTaskID = strings.TrimSpace(i.ParentTaskID)
	return spec
}

type taskUpdateInput struct {
	TaskID         string             `json:"task_id"`
	Title          *string            `json:"title,omitempty"`
	Description    *string            `json:"description,omitempty"`
	Priority       *string            `json:"priority,omitempty"`
	MaxAttempts    *int               `json:"max_attempts,omitempty"`
	ApprovalPolicy *string            `json:"approval_policy,omitempty"`
	Metadata       *json.RawMessage   `json:"metadata,omitempty"`
	NetworkChannel *string            `json:"network_channel,omitempty"`
	Owner          *taskpkg.Ownership `json:"owner,omitempty"`
	ClearOwner     bool               `json:"clear_owner,omitempty"`
}

func (i taskUpdateInput) patch() taskpkg.Patch {
	return taskpkg.Patch{
		Title:          cloneStringPtr(i.Title),
		Description:    cloneStringPtr(i.Description),
		Priority:       taskPriorityPtr(i.Priority),
		MaxAttempts:    cloneIntPtr(i.MaxAttempts),
		ApprovalPolicy: taskApprovalPolicyPtr(i.ApprovalPolicy),
		Metadata:       cloneRawMessagePtr(i.Metadata),
		NetworkChannel: cloneStringPtr(i.NetworkChannel),
		Owner:          cloneTaskOwner(i.Owner),
		ClearOwner:     i.ClearOwner,
	}
}

type taskCancelInput struct {
	TaskID   string          `json:"task_id"`
	Reason   string          `json:"reason,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

func (i taskCancelInput) cancel() taskpkg.CancelTask {
	return taskpkg.CancelTask{
		Reason:   strings.TrimSpace(i.Reason),
		Metadata: cloneJSON(i.Metadata),
	}
}

type taskRunListInput struct {
	TaskID                string `json:"task_id"`
	Status                string `json:"status,omitempty"`
	SessionID             string `json:"session_id,omitempty"`
	CoordinationChannelID string `json:"coordination_channel_id,omitempty"`
	Limit                 int    `json:"limit,omitempty"`
}

func (i taskRunListInput) query() taskpkg.RunQuery {
	return taskpkg.RunQuery{
		TaskID:                strings.TrimSpace(i.TaskID),
		Status:                taskpkg.RunStatus(strings.TrimSpace(i.Status)),
		SessionID:             strings.TrimSpace(i.SessionID),
		CoordinationChannelID: strings.TrimSpace(i.CoordinationChannelID),
		Limit:                 i.Limit,
	}
}

func decodeNativeInput(req toolspkg.CallRequest, dst any) error {
	raw := req.Input
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = json.RawMessage(`{}`)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			req.ToolID,
			fmt.Sprintf("tool %q input is invalid", req.ToolID),
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonSchemaInvalid,
		)
	}
	return nil
}

func structuredResult(value any, preview string) (toolspkg.ToolResult, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return toolspkg.ToolResult{}, fmt.Errorf("daemon: marshal native tool result: %w", err)
	}
	result := toolspkg.ToolResult{
		Structured: data,
		Preview:    strings.TrimSpace(preview),
	}
	if result.Preview != "" {
		result.Content = []toolspkg.ToolContent{{Type: "text", Text: result.Preview}}
	}
	return result, nil
}

func actorContextFromScope(scope toolspkg.Scope) (taskpkg.ActorContext, error) {
	if sessionID := strings.TrimSpace(scope.SessionID); sessionID != "" {
		return taskpkg.DeriveAgentSessionActorContext(sessionID)
	}
	return taskpkg.DeriveDaemonActorContext("native-tools", "tool.registry")
}

func searchSkills(skillList []*skills.Skill, query string) []*skills.Skill {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return skillList
	}
	filtered := make([]*skills.Skill, 0, len(skillList))
	for _, skill := range skillList {
		if skill == nil {
			continue
		}
		values := []string{
			skill.Meta.Name,
			skill.Meta.Description,
			skill.Meta.Version,
			skills.SkillSourceName(skill.Source),
			skill.InstalledFrom,
		}
		if slices.ContainsFunc(values, func(value string) bool {
			return strings.Contains(strings.ToLower(value), needle)
		}) {
			filtered = append(filtered, skill)
		}
	}
	return filtered
}

func limitSkills(skillList []*skills.Skill, limit int) []*skills.Skill {
	if limit <= 0 || limit >= len(skillList) {
		return skillList
	}
	return skillList[:limit]
}

func limitToolViews(views []toolspkg.ToolView, limit int) []toolspkg.ToolView {
	if limit <= 0 || limit >= len(views) {
		return views
	}
	return views[:limit]
}

func stringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := strings.TrimSpace(*value)
	return &cloned
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneRawMessagePtr(value *json.RawMessage) *json.RawMessage {
	if value == nil {
		return nil
	}
	cloned := cloneJSON(*value)
	return &cloned
}

func taskPriorityPtr(value *string) *taskpkg.Priority {
	if value == nil {
		return nil
	}
	priority := taskpkg.Priority(strings.TrimSpace(*value))
	return &priority
}

func taskApprovalPolicyPtr(value *string) *taskpkg.ApprovalPolicy {
	if value == nil {
		return nil
	}
	policy := taskpkg.ApprovalPolicy(strings.TrimSpace(*value))
	return &policy
}

func cloneTaskOwner(owner *taskpkg.Ownership) *taskpkg.Ownership {
	if owner == nil {
		return nil
	}
	cloned := *owner
	cloned.Kind = taskpkg.OwnerKind(strings.TrimSpace(string(cloned.Kind)))
	cloned.Ref = strings.TrimSpace(cloned.Ref)
	return &cloned
}

func cloneJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

func cloneExtensionMap(src network.ExtensionMap) network.ExtensionMap {
	if len(src) == 0 {
		return nil
	}
	dst := make(network.ExtensionMap, len(src))
	for key, value := range src {
		dst[key] = cloneJSON(value)
	}
	return dst
}

func nativeToolPolicyInputs(cfg *aghconfig.Config) (toolspkg.PolicyInputs, error) {
	if cfg == nil {
		return toolspkg.PolicyInputs{}, errors.New("daemon: native tool config is required")
	}
	trustedSources := make([]toolspkg.SourceGrant, 0, len(cfg.Tools.Policy.TrustedSources))
	for _, raw := range cfg.Tools.Policy.TrustedSources {
		grant, err := toolspkg.ParseSourceGrant(raw)
		if err != nil {
			return toolspkg.PolicyInputs{}, err
		}
		trustedSources = append(trustedSources, grant)
	}
	return toolspkg.PolicyInputs{
		ToolsDisabled:        !cfg.Tools.Enabled,
		SystemPermissionMode: nativeToolPermissionMode(cfg.Permissions.Mode),
		ExternalDefault:      nativeToolExternalDefault(cfg.Tools.Policy.ExternalDefault),
		TrustedSources:       trustedSources,
	}, nil
}

func nativeToolPermissionMode(mode aghconfig.PermissionMode) toolspkg.PermissionMode {
	switch mode {
	case aghconfig.PermissionModeDenyAll:
		return toolspkg.PermissionModeDenyAll
	case aghconfig.PermissionModeApproveReads:
		return toolspkg.PermissionModeApproveReads
	case aghconfig.PermissionModeApproveAll:
		return toolspkg.PermissionModeApproveAll
	default:
		return ""
	}
}

func nativeToolExternalDefault(value aghconfig.ToolsExternalDefault) toolspkg.ExternalDefault {
	switch value {
	case aghconfig.ToolsExternalDefaultAsk:
		return toolspkg.ExternalDefaultAsk
	case aghconfig.ToolsExternalDefaultEnabled:
		return toolspkg.ExternalDefaultEnabled
	default:
		return toolspkg.ExternalDefaultDisabled
	}
}
