package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	"github.com/pedronauck/agh/internal/heartbeat"
	mcppkg "github.com/pedronauck/agh/internal/mcp"
	mcpauth "github.com/pedronauck/agh/internal/mcp/auth"
	memorypkg "github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	toolspkg "github.com/pedronauck/agh/internal/tools"
	builtintools "github.com/pedronauck/agh/internal/tools/builtin"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type daemonNativeToolsDeps struct {
	Registry          func() toolspkg.Registry
	Skills            core.SkillsRegistry
	Sessions          core.SessionManager
	Workspaces        core.WorkspaceService
	WorkspaceResolver workspacepkg.RuntimeResolver
	Network           core.NetworkService
	NetworkStore      core.NetworkStore
	Tasks             taskpkg.Manager
	MemoryStore       *memorypkg.Store
	Bridges           core.BridgeService
	HomePaths         aghconfig.HomePaths
	Observer          core.Observer
	HookBindings      hookBindingPublisher
	AgentCatalog      core.AgentCatalog
	HeartbeatStatus   core.HeartbeatStatusService
	HeartbeatWake     core.HeartbeatWakeService
	SessionHealth     core.SessionHealthReader
	WakeEvents        core.HeartbeatWakeEventReader
	Automation        core.AutomationManager
	AutomationRuntime func() core.AutomationManager
	ExtensionRegistry *extensionpkg.Registry
	ExtensionRuntime  func() extensionRuntime
	ExtensionMarket   aghconfig.ExtensionsMarketplaceConfig
	ExtensionSources  extensionMarketplaceSourceLoader
	AgentSkills       agentSkillPublisher
	ToolMCP           toolMCPPublisher
	MCPAuth           func() toolspkg.MCPAuthStatusProvider
	Bundles           bundleResourcePublisher
}

type daemonNativeTools struct {
	deps *daemonNativeToolsDeps
}

type nativeToolBinding struct {
	call         toolspkg.NativeToolFunc
	availability toolspkg.NativeAvailabilityFunc
}

const defaultNativeWakeEventLimit = 10

func newDaemonNativeProvider(deps *daemonNativeToolsDeps) (toolspkg.Provider, error) {
	if deps == nil {
		return nil, errors.New("daemon: native tool dependencies are required")
	}
	adapter := &daemonNativeTools{deps: deps}
	bindings := adapter.bindings()
	descriptors := builtintools.NativeDescriptors()
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
	return toolspkg.NewNativeProvider(builtintools.Source(), nativeTools...)
}

func (d *Daemon) bootToolRegistry(_ context.Context, state *bootState) error {
	if state == nil {
		return errors.New("daemon: tool registry state is required")
	}
	if state.mcpServerCatalog == nil {
		state.mcpServerCatalog = newResourceCatalog(cloneDaemonMCPServer)
	}
	var registry *toolspkg.RuntimeRegistry
	var mcpAuth toolspkg.MCPAuthStatusProvider
	deps := d.nativeToolsDeps(state, func() toolspkg.Registry {
		return registry
	})
	deps.MCPAuth = func() toolspkg.MCPAuthStatusProvider {
		return mcpAuth
	}
	provider, err := newDaemonNativeProvider(&deps)
	if err != nil {
		return fmt.Errorf("daemon: create native tool provider: %w", err)
	}
	approvalTokens := toolspkg.NewApprovalTokenStore(state.cfg.Tools.Policy.ApprovalTimeout())
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
			approvalTokens,
		)
	} else {
		approvalBridge = newToolApprovalBridge(nil, state.cfg.Tools.Policy.ApprovalTimeout(), approvalTokens)
	}
	toolsets, err := builtintools.ToolsetCatalog()
	if err != nil {
		return fmt.Errorf("daemon: build native toolset catalog: %w", err)
	}
	policyResolver, err := newNativeToolPolicyResolverForBoot(state)
	if err != nil {
		return fmt.Errorf("daemon: build native tool policy resolver: %w", err)
	}
	providers := []toolspkg.Provider{provider}
	extensionProvider, err := newDaemonExtensionToolProvider(state)
	if err != nil {
		return fmt.Errorf("daemon: create extension tool provider: %w", err)
	}
	if extensionProvider != nil {
		providers = append(providers, extensionProvider)
	}
	mcpProvider, mcpAuthProvider, err := d.newDaemonMCPToolProvider(state)
	if err != nil {
		return fmt.Errorf("daemon: create mcp tool provider: %w", err)
	}
	mcpAuth = mcpAuthProvider
	if mcpProvider != nil {
		providers = append(providers, mcpProvider)
	}
	registry, err = toolspkg.NewRegistry(
		toolspkg.WithProviders(providers...),
		toolspkg.WithPolicyInputResolver(policyResolver, toolsets),
		toolspkg.WithApprovalBridge(approvalBridge),
		toolspkg.WithDefaultMaxResultBytes(state.cfg.Tools.DefaultMaxResultBytes),
	)
	if err != nil {
		return fmt.Errorf("daemon: create tool registry: %w", err)
	}
	state.toolRegistry = registry
	state.toolsets = registry
	state.toolApprovals = approvalTokens
	state.deps.ToolRegistry = registry
	state.deps.Toolsets = registry
	state.deps.ToolApprovals = approvalTokens
	return nil
}

func (d *Daemon) nativeToolsDeps(
	state *bootState,
	registryRef func() toolspkg.Registry,
) daemonNativeToolsDeps {
	return daemonNativeToolsDeps{
		Registry:          registryRef,
		Skills:            skillsRegistryAPI(state.skillsRegistry),
		Sessions:          state.sessions,
		Workspaces:        state.workspaceResolver,
		WorkspaceResolver: state.workspaceResolver,
		Network:           state.deps.Network,
		NetworkStore:      state.registry,
		Tasks:             state.deps.Tasks,
		MemoryStore:       state.memoryStore,
		Bridges:           state.deps.Bridges,
		HomePaths:         d.homePaths,
		Observer:          state.observer,
		HookBindings:      state.hookBindings,
		AgentCatalog: agentCatalogDependency(state.agentCatalog, agentSidecarCatalogs{
			soul:      state.soulCatalog,
			heartbeat: state.heartbeatCatalog,
		}),
		HeartbeatStatus: state.deps.HeartbeatStatus,
		HeartbeatWake:   state.deps.HeartbeatWake,
		SessionHealth:   state.deps.SessionHealth,
		WakeEvents:      state.deps.WakeEvents,
		Automation:      state.deps.Automation,
		AutomationRuntime: func() core.AutomationManager {
			return state.deps.Automation
		},
		ExtensionRegistry: extensionRegistryDependency(state.registry),
		ExtensionRuntime:  state.currentExtensionRuntime,
		ExtensionMarket:   state.cfg.Extensions.Marketplace,
		AgentSkills:       state.agentSkillResources,
		ToolMCP:           state.toolMCPResources,
		Bundles:           state.bundleResources,
	}
}

func (d *Daemon) newDaemonMCPToolProvider(
	state *bootState,
) (toolspkg.Provider, toolspkg.MCPAuthStatusProvider, error) {
	if state == nil {
		return nil, nil, nil
	}
	resolver := mcppkg.ServerResolverFunc(func(context.Context) ([]aghconfig.MCPServer, error) {
		return daemonMCPServerConfigs(state), nil
	})
	options := []mcppkg.CallExecutorOption{}
	if d != nil && d.getenv != nil {
		options = append(options, mcppkg.WithSecretLookup(d.getenv))
	}
	if state.providerVault != nil {
		options = append(options, mcppkg.WithSecretResolver(state.providerVault))
	}
	if store, ok := state.registry.(mcpauth.TokenStore); ok {
		options = append(options, mcppkg.WithTokenStore(store))
	}
	executor, err := mcppkg.NewMCPCallExecutor(resolver, options...)
	if err != nil {
		return nil, nil, err
	}
	provider, err := toolspkg.NewMCPProvider(
		toolspkg.MCPSourceListerFunc(func(context.Context) ([]toolspkg.SourceRef, error) {
			return daemonMCPSources(state), nil
		}),
		executor,
		executor,
	)
	if err != nil {
		return nil, nil, err
	}
	return provider, executor, nil
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

type nativeToolAvailabilitySet struct {
	registry         toolspkg.NativeAvailabilityFunc
	skills           toolspkg.NativeAvailabilityFunc
	network          toolspkg.NativeAvailabilityFunc
	sessions         toolspkg.NativeAvailabilityFunc
	sessionHealth    toolspkg.NativeAvailabilityFunc
	heartbeatStatus  toolspkg.NativeAvailabilityFunc
	heartbeatWake    toolspkg.NativeAvailabilityFunc
	workspaces       toolspkg.NativeAvailabilityFunc
	workspaceDetails toolspkg.NativeAvailabilityFunc
	tasks            toolspkg.NativeAvailabilityFunc
	memory           toolspkg.NativeAvailabilityFunc
	observe          toolspkg.NativeAvailabilityFunc
	bridges          toolspkg.NativeAvailabilityFunc
	config           toolspkg.NativeAvailabilityFunc
	hookRead         toolspkg.NativeAvailabilityFunc
	hookMutation     toolspkg.NativeAvailabilityFunc
	automation       toolspkg.NativeAvailabilityFunc
	extensions       toolspkg.NativeAvailabilityFunc
	mcpAuth          toolspkg.NativeAvailabilityFunc
}

func (n *daemonNativeTools) bindings() map[toolspkg.ToolID]nativeToolBinding {
	availability := n.nativeToolAvailability()
	bindings := make(map[toolspkg.ToolID]nativeToolBinding, 32)
	addNativeToolBindings(bindings, n.registryToolBindings(availability.registry))
	addNativeToolBindings(bindings, n.skillToolBindings(availability.skills))
	addNativeToolBindings(bindings, n.networkToolBindings(availability.network))
	addNativeToolBindings(bindings, n.sessionToolBindings(availability.sessions))
	addNativeToolBindings(
		bindings,
		n.authoredContextToolBindings(
			availability.sessionHealth,
			availability.heartbeatStatus,
			availability.heartbeatWake,
		),
	)
	addNativeToolBindings(bindings, n.workspaceToolBindings(availability.workspaces, availability.workspaceDetails))
	addNativeToolBindings(bindings, n.memoryToolBindings(availability.memory))
	addNativeToolBindings(bindings, n.observeToolBindings(availability.observe))
	addNativeToolBindings(bindings, n.bridgeToolBindings(availability.bridges))
	addNativeToolBindings(bindings, n.taskToolBindings(availability.tasks))
	addNativeToolBindings(bindings, n.autonomyToolBindings(availability.tasks))
	addNativeToolBindings(bindings, n.configToolBindings(availability.config))
	addNativeToolBindings(bindings, n.hookToolBindings(availability.hookRead, availability.hookMutation))
	addNativeToolBindings(bindings, n.automationToolBindings(availability.automation))
	addNativeToolBindings(bindings, n.extensionToolBindings(availability.extensions))
	addNativeToolBindings(bindings, n.mcpAuthToolBindings(availability.mcpAuth))
	return bindings
}

func (n *daemonNativeTools) nativeToolAvailability() nativeToolAvailabilitySet {
	configReady := func() bool {
		return strings.TrimSpace(n.deps.HomePaths.ConfigFile) != ""
	}
	return nativeToolAvailabilitySet{
		registry: n.registryAvailability(),
		skills:   n.dependencyAvailability(func() bool { return n.deps.Skills != nil }),
		network:  n.dependencyAvailability(func() bool { return n.deps.Network != nil }),
		sessions: n.dependencyAvailability(func() bool { return n.deps.Sessions != nil }),
		sessionHealth: n.dependencyAvailability(func() bool {
			return n.deps.SessionHealth != nil
		}),
		heartbeatStatus: n.dependencyAvailability(func() bool {
			return n.deps.HeartbeatStatus != nil && n.deps.WorkspaceResolver != nil
		}),
		heartbeatWake: n.dependencyAvailability(func() bool {
			return n.deps.HeartbeatWake != nil && n.deps.WorkspaceResolver != nil
		}),
		workspaces: n.dependencyAvailability(func() bool {
			return n.deps.Workspaces != nil
		}),
		workspaceDetails: n.dependencyAvailability(func() bool {
			return n.deps.Workspaces != nil && n.deps.Sessions != nil
		}),
		memory: n.dependencyAvailability(func() bool { return n.deps.MemoryStore != nil }),
		observe: n.dependencyAvailability(func() bool {
			return n.deps.Observer != nil
		}),
		bridges:  n.dependencyAvailability(func() bool { return n.deps.Bridges != nil }),
		tasks:    n.dependencyAvailability(func() bool { return n.deps.Tasks != nil }),
		config:   n.dependencyAvailability(configReady),
		hookRead: n.dependencyAvailability(func() bool { return n.deps.Observer != nil }),
		hookMutation: n.dependencyAvailability(func() bool {
			return configReady() && n.deps.Observer != nil && n.deps.HookBindings != nil
		}),
		automation: n.dependencyAvailability(func() bool { return n.automationManager() != nil }),
		extensions: n.dependencyAvailability(func() bool {
			return n.deps.ExtensionRegistry != nil && strings.TrimSpace(n.deps.HomePaths.HomeDir) != ""
		}),
		mcpAuth: n.dependencyAvailability(func() bool { return n.mcpAuthProvider() != nil }),
	}
}

func extensionRegistryDependency(registry Registry) *extensionpkg.Registry {
	if registry == nil {
		return nil
	}
	dbSource, ok := registry.(extensionDBSource)
	if !ok || dbSource.DB() == nil {
		return nil
	}
	return extensionpkg.NewRegistry(dbSource.DB())
}

func addNativeToolBindings(
	dst map[toolspkg.ToolID]nativeToolBinding,
	src map[toolspkg.ToolID]nativeToolBinding,
) {
	maps.Copy(dst, src)
}

func (n *daemonNativeTools) registryToolBindings(
	availability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDToolList: {
			call:         n.toolList,
			availability: availability,
		},
		toolspkg.ToolIDToolSearch: {
			call:         n.toolSearch,
			availability: availability,
		},
		toolspkg.ToolIDToolInfo: {
			call:         n.toolInfo,
			availability: availability,
		},
	}
}

func (n *daemonNativeTools) skillToolBindings(
	availability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDSkillList: {
			call:         n.skillList,
			availability: availability,
		},
		toolspkg.ToolIDSkillSearch: {
			call:         n.skillSearch,
			availability: availability,
		},
		toolspkg.ToolIDSkillView: {
			call:         n.skillView,
			availability: availability,
		},
	}
}

func (n *daemonNativeTools) networkToolBindings(
	availability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDNetworkStatus: {
			call:         n.networkStatus,
			availability: availability,
		},
		toolspkg.ToolIDNetworkChannels: {
			call:         n.networkChannels,
			availability: availability,
		},
		toolspkg.ToolIDNetworkInbox: {
			call:         n.networkInbox,
			availability: availability,
		},
		toolspkg.ToolIDNetworkPeers: {
			call:         n.networkPeers,
			availability: availability,
		},
		toolspkg.ToolIDNetworkSend: {
			call:         n.networkSend,
			availability: availability,
		},
	}
}

func (n *daemonNativeTools) sessionToolBindings(
	availability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDSessionList: {
			call:         n.sessionList,
			availability: availability,
		},
		toolspkg.ToolIDSessionStatus: {
			call:         n.sessionStatus,
			availability: availability,
		},
		toolspkg.ToolIDSessionHistory: {
			call:         n.sessionHistory,
			availability: availability,
		},
		toolspkg.ToolIDSessionEvents: {
			call:         n.sessionEvents,
			availability: availability,
		},
		toolspkg.ToolIDSessionDescribe: {
			call:         n.sessionDescribe,
			availability: availability,
		},
	}
}

func (n *daemonNativeTools) authoredContextToolBindings(
	healthAvailability toolspkg.NativeAvailabilityFunc,
	statusAvailability toolspkg.NativeAvailabilityFunc,
	wakeAvailability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDSessionHealth: {
			call:         n.sessionHealth,
			availability: healthAvailability,
		},
		toolspkg.ToolIDAgentHeartbeatStatus: {
			call:         n.agentHeartbeatStatus,
			availability: statusAvailability,
		},
		toolspkg.ToolIDAgentHeartbeatWake: {
			call:         n.agentHeartbeatWake,
			availability: wakeAvailability,
		},
	}
}

func (n *daemonNativeTools) workspaceToolBindings(
	availability toolspkg.NativeAvailabilityFunc,
	describeAvailability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDWorkspaceList: {
			call:         n.workspaceList,
			availability: availability,
		},
		toolspkg.ToolIDWorkspaceInfo: {
			call:         n.workspaceInfo,
			availability: availability,
		},
		toolspkg.ToolIDWorkspaceDescribe: {
			call:         n.workspaceDescribe,
			availability: describeAvailability,
		},
	}
}

func (n *daemonNativeTools) memoryToolBindings(
	availability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDMemoryList: {
			call:         n.memoryList,
			availability: availability,
		},
		toolspkg.ToolIDMemoryRead: {
			call:         n.memoryRead,
			availability: availability,
		},
		toolspkg.ToolIDMemorySearch: {
			call:         n.memorySearch,
			availability: availability,
		},
		toolspkg.ToolIDMemoryHistory: {
			call:         n.memoryHistory,
			availability: availability,
		},
	}
}

func (n *daemonNativeTools) observeToolBindings(
	availability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDObserveEvents: {
			call:         n.observeEvents,
			availability: availability,
		},
		toolspkg.ToolIDObserveMetrics: {
			call:         n.observeMetrics,
			availability: availability,
		},
		toolspkg.ToolIDObserveSearch: {
			call:         n.observeSearch,
			availability: availability,
		},
	}
}

func (n *daemonNativeTools) bridgeToolBindings(
	availability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDBridgesList: {
			call:         n.bridgesList,
			availability: availability,
		},
		toolspkg.ToolIDBridgesStatus: {
			call:         n.bridgesStatus,
			availability: availability,
		},
	}
}

func (n *daemonNativeTools) taskToolBindings(
	availability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDTaskList: {
			call:         n.taskList,
			availability: availability,
		},
		toolspkg.ToolIDTaskRead: {
			call:         n.taskRead,
			availability: availability,
		},
		toolspkg.ToolIDTaskCreate: {
			call:         n.taskCreate,
			availability: availability,
		},
		toolspkg.ToolIDTaskChildCreate: {
			call:         n.taskChildCreate,
			availability: availability,
		},
		toolspkg.ToolIDTaskUpdate: {
			call:         n.taskUpdate,
			availability: availability,
		},
		toolspkg.ToolIDTaskCancel: {
			call:         n.taskCancel,
			availability: availability,
		},
		toolspkg.ToolIDTaskRunList: {
			call:         n.taskRunList,
			availability: availability,
		},
	}
}

func (n *daemonNativeTools) autonomyToolBindings(
	availability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDTaskRunClaimNext: {
			call:         n.autonomyClaimNext,
			availability: availability,
		},
		toolspkg.ToolIDTaskRunHeartbeat: {
			call:         n.autonomyHeartbeat,
			availability: availability,
		},
		toolspkg.ToolIDTaskRunComplete: {
			call:         n.autonomyComplete,
			availability: availability,
		},
		toolspkg.ToolIDTaskRunFail: {
			call:         n.autonomyFail,
			availability: availability,
		},
		toolspkg.ToolIDTaskRunRelease: {
			call:         n.autonomyRelease,
			availability: availability,
		},
	}
}

func (n *daemonNativeTools) configToolBindings(
	availability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDConfigShow: {
			call:         n.configShow,
			availability: availability,
		},
		toolspkg.ToolIDConfigList: {
			call:         n.configList,
			availability: availability,
		},
		toolspkg.ToolIDConfigGet: {
			call:         n.configGet,
			availability: availability,
		},
		toolspkg.ToolIDConfigSet: {
			call:         n.configSet,
			availability: availability,
		},
		toolspkg.ToolIDConfigUnset: {
			call:         n.configUnset,
			availability: availability,
		},
		toolspkg.ToolIDConfigDiff: {
			call:         n.configDiff,
			availability: availability,
		},
		toolspkg.ToolIDConfigPath: {
			call:         n.configPath,
			availability: availability,
		},
	}
}

func (n *daemonNativeTools) hookToolBindings(
	readAvailability toolspkg.NativeAvailabilityFunc,
	mutationAvailability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDHooksList: {
			call:         n.hooksList,
			availability: readAvailability,
		},
		toolspkg.ToolIDHooksInfo: {
			call:         n.hooksInfo,
			availability: readAvailability,
		},
		toolspkg.ToolIDHooksEvents: {
			call:         n.hooksEvents,
			availability: readAvailability,
		},
		toolspkg.ToolIDHooksRuns: {
			call:         n.hooksRuns,
			availability: readAvailability,
		},
		toolspkg.ToolIDHooksCreate: {
			call:         n.hooksCreate,
			availability: mutationAvailability,
		},
		toolspkg.ToolIDHooksUpdate: {
			call:         n.hooksUpdate,
			availability: mutationAvailability,
		},
		toolspkg.ToolIDHooksDelete: {
			call:         n.hooksDelete,
			availability: mutationAvailability,
		},
		toolspkg.ToolIDHooksEnable: {
			call:         n.hooksEnable,
			availability: mutationAvailability,
		},
		toolspkg.ToolIDHooksDisable: {
			call:         n.hooksDisable,
			availability: mutationAvailability,
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

func (n *daemonNativeTools) mcpAuthProvider() toolspkg.MCPAuthStatusProvider {
	if n == nil || n.deps.MCPAuth == nil {
		return nil
	}
	return n.deps.MCPAuth()
}

func (n *daemonNativeTools) automationManager() core.AutomationManager {
	if n == nil || n.deps == nil {
		return nil
	}
	if n.deps.Automation != nil {
		return n.deps.Automation
	}
	if n.deps.AutomationRuntime == nil {
		return nil
	}
	return n.deps.AutomationRuntime()
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

func (n *daemonNativeTools) networkStatus(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input struct{}
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	status, err := n.deps.Network.Status(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := core.NetworkStatusPayloadFromStatus(status)
	if payload == nil {
		return toolspkg.ToolResult{}, errors.New("daemon: network status is required")
	}
	return structuredResult(map[string]any{"network": payload}, payload.Status)
}

func (n *daemonNativeTools) networkChannels(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input struct{}
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	var channels any
	var count int
	if n.deps.Sessions != nil && n.deps.NetworkStore != nil {
		payload, err := core.NetworkChannelPayloads(ctx, n.deps.Network, n.deps.Sessions, n.deps.NetworkStore)
		if err != nil {
			return toolspkg.ToolResult{}, err
		}
		channels = payload
		count = len(payload)
	} else {
		infos, err := n.deps.Network.ListChannels(ctx)
		if err != nil {
			return toolspkg.ToolResult{}, err
		}
		payload := core.NetworkChannelPayloadsFromInfos(infos)
		channels = payload
		count = len(payload)
	}
	return structuredResult(map[string]any{"channels": channels}, fmt.Sprintf("%d channels", count))
}

func (n *daemonNativeTools) networkInbox(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input networkInboxInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	sessionID := firstNonEmpty(input.SessionID, req.SessionID, scope.SessionID)
	if sessionID == "" {
		return toolspkg.ToolResult{}, nativeRequiredInputError(req.ToolID, "session_id")
	}
	messages, err := n.deps.Network.Inbox(ctx, sessionID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := core.NetworkEnvelopePayloadsFromEnvelopes(messages)
	return structuredResult(map[string]any{"messages": payload}, fmt.Sprintf("%d messages", len(payload)))
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
	sendReq, err := core.NetworkSendRequestFromPayload(contract.NetworkSendRequest{
		SessionID:   sessionID,
		Channel:     strings.TrimSpace(input.Channel),
		Surface:     strings.TrimSpace(input.Surface),
		ThreadID:    strings.TrimSpace(input.ThreadID),
		DirectID:    strings.TrimSpace(input.DirectID),
		Kind:        strings.TrimSpace(input.Kind),
		To:          strings.TrimSpace(input.To),
		Body:        cloneJSON(input.Body),
		WorkID:      strings.TrimSpace(input.WorkID),
		ReplyTo:     strings.TrimSpace(input.ReplyTo),
		TraceID:     strings.TrimSpace(input.TraceID),
		CausationID: strings.TrimSpace(input.CausationID),
		ExpiresAt:   input.ExpiresAt,
		ID:          strings.TrimSpace(input.ID),
		Ext:         map[string]json.RawMessage(cloneExtensionMap(input.Ext)),
	})
	if err != nil {
		return toolspkg.ToolResult{}, nativeNetworkSendToolError(req.ToolID, err)
	}
	messageID, err := n.deps.Network.Send(ctx, sendReq)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	return structuredResult(map[string]any{"message_id": messageID}, messageID)
}

func (n *daemonNativeTools) sessionList(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input sessionListInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	infos, err := n.deps.Sessions.ListAll(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	workspaceRef := firstNonEmpty(input.Workspace, scope.WorkspaceID)
	if workspaceRef != "" {
		workspaceID, err := n.workspaceID(ctx, workspaceRef)
		if err != nil {
			return toolspkg.ToolResult{}, err
		}
		payload := core.SessionPayloadsForWorkspace(infos, workspaceID)
		return structuredResult(
			map[string]any{"sessions": limitSessionPayloads(payload, input.Limit)},
			fmt.Sprintf("%d sessions", len(payload)),
		)
	}
	payload := core.SessionPayloadsFromInfos(infos)
	return structuredResult(
		map[string]any{"sessions": limitSessionPayloads(payload, input.Limit)},
		fmt.Sprintf("%d sessions", len(payload)),
	)
}

func (n *daemonNativeTools) sessionStatus(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input sessionIDInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	sessionID, err := requiredNativeString(req.ToolID, "session_id", input.SessionID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	info, err := n.deps.Sessions.Status(ctx, sessionID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := core.SessionPayloadFromInfo(info)
	return structuredResult(map[string]any{"session": payload}, payload.ID)
}

func (n *daemonNativeTools) sessionHealth(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input sessionIDInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	sessionID, err := requiredNativeString(req.ToolID, "session_id", input.SessionID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	health, err := n.deps.SessionHealth.GetSessionHealth(ctx, sessionID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload, err := contract.SessionHealthPayloadFromDomain(health)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	if err := contract.ValidateAuthoredContextRedacted(payload); err != nil {
		return toolspkg.ToolResult{}, err
	}
	return structuredResult(map[string]any{"health": payload}, string(payload.Health))
}

func (n *daemonNativeTools) agentHeartbeatStatus(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input agentHeartbeatStatusInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	target, err := n.authoredAgentTarget(ctx, req.ToolID, input.WorkspaceID, input.AgentName)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	status, err := n.deps.HeartbeatStatus.Status(ctx, heartbeat.StatusRequest{
		Target:               target.heartbeatAuthoringTarget(),
		SessionID:            strings.TrimSpace(input.SessionID),
		IncludeSessionHealth: input.IncludeSessionHealth,
	})
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload, err := contract.HeartbeatStatusResponseFromResult(&status)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	if input.IncludeRecentWakeEvents && n.deps.WakeEvents != nil {
		events, err := n.deps.WakeEvents.ListHeartbeatWakeEvents(ctx, heartbeat.WakeEventListQuery{
			WorkspaceID: target.workspaceID,
			AgentName:   target.agentName,
			SessionID:   strings.TrimSpace(input.SessionID),
			Limit:       defaultNativeWakeEventLimit,
		})
		if err != nil {
			return toolspkg.ToolResult{}, err
		}
		payload.WakeEvents = make([]contract.HeartbeatWakeEventPayload, 0, len(events))
		for _, event := range events {
			converted, convertErr := contract.HeartbeatWakeEventPayloadFromDomain(event)
			if convertErr != nil {
				return toolspkg.ToolResult{}, convertErr
			}
			payload.WakeEvents = append(payload.WakeEvents, converted)
		}
	}
	if err := contract.ValidateAuthoredContextRedacted(payload); err != nil {
		return toolspkg.ToolResult{}, err
	}
	return structuredResult(map[string]any{"heartbeat": payload}, string(payload.ValidationStatus))
}

func (n *daemonNativeTools) agentHeartbeatWake(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input agentHeartbeatWakeInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	target, err := n.authoredAgentTarget(ctx, req.ToolID, input.WorkspaceID, input.AgentName)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	sessionID, err := requiredNativeString(req.ToolID, "session_id", input.SessionID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	source := heartbeat.WakeSource(strings.TrimSpace(input.Source))
	if source == "" {
		source = heartbeat.WakeSourceManual
	}
	decision, err := n.deps.HeartbeatWake.Wake(ctx, heartbeat.WakeRequest{
		WorkspaceID: target.workspaceID,
		AgentName:   target.agentName,
		SessionID:   sessionID,
		Source:      source,
		DryRun:      input.DryRun,
	})
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload, err := contract.HeartbeatWakeDecisionPayloadFromDomain(decision)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	response := contract.HeartbeatWakeResponse{Decision: payload}
	if err := contract.ValidateAuthoredContextRedacted(response); err != nil {
		return toolspkg.ToolResult{}, err
	}
	return structuredResult(map[string]any{"wake": response}, string(payload.Result))
}

func (n *daemonNativeTools) sessionEvents(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	input, query, err := decodeSessionEventQueryInput(req)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	info, err := n.deps.Sessions.Status(ctx, input.SessionID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	events, err := n.deps.Sessions.Events(ctx, input.SessionID, query)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := make([]any, 0, len(events))
	for _, event := range events {
		payload = append(payload, core.SessionEventPayloadFromEvent(event, info))
	}
	return structuredResult(map[string]any{"events": payload}, fmt.Sprintf("%d events", len(payload)))
}

func (n *daemonNativeTools) sessionHistory(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	input, query, err := decodeSessionEventQueryInput(req)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	info, err := n.deps.Sessions.Status(ctx, input.SessionID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	history, err := n.deps.Sessions.History(ctx, input.SessionID, query)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := sessionHistoryPayload(history, info)
	return structuredResult(map[string]any{"history": payload}, fmt.Sprintf("%d turns", len(payload)))
}

func (n *daemonNativeTools) sessionDescribe(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	input, query, err := decodeSessionEventQueryInput(req)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	info, err := n.deps.Sessions.Status(ctx, input.SessionID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	events, err := n.deps.Sessions.Events(ctx, input.SessionID, query)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	history, err := n.deps.Sessions.History(ctx, input.SessionID, query)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	eventPayload := make([]any, 0, len(events))
	for _, event := range events {
		eventPayload = append(eventPayload, core.SessionEventPayloadFromEvent(event, info))
	}
	return structuredResult(map[string]any{
		"session": core.SessionPayloadFromInfo(info),
		"events":  eventPayload,
		"history": sessionHistoryPayload(history, info),
	}, info.ID)
}

func (n *daemonNativeTools) workspaceList(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input struct{}
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	workspaces, err := n.deps.Workspaces.List(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := make([]any, 0, len(workspaces))
	for _, workspace := range workspaces {
		payload = append(payload, core.WorkspacePayloadFromWorkspace(workspace))
	}
	return structuredResult(map[string]any{"workspaces": payload}, fmt.Sprintf("%d workspaces", len(payload)))
}

func (n *daemonNativeTools) workspaceInfo(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input workspaceRefInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	ref, err := requiredNativeString(req.ToolID, "workspace", input.Workspace)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	workspace, err := n.deps.Workspaces.Get(ctx, ref)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := core.WorkspacePayloadFromWorkspace(workspace)
	return structuredResult(map[string]any{"workspace": payload}, payload.ID)
}

func (n *daemonNativeTools) workspaceDescribe(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input workspaceRefInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	ref, err := requiredNativeString(req.ToolID, "workspace", input.Workspace)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	resolved, err := n.deps.Workspaces.Resolve(ctx, ref)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	sessions, err := n.deps.Sessions.ListAll(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	agents, err := n.workspaceAgents(ctx, &resolved)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	return structuredResult(map[string]any{
		"workspace": core.WorkspacePayloadFromWorkspace(resolved.Workspace),
		"sessions":  core.SessionPayloadsForWorkspace(sessions, resolved.ID),
		"agents":    core.AgentPayloadsFromDefs(agents),
		"skills":    core.WorkspaceSkillPayloads(resolved.Skills),
		"providers": core.SessionProviderOptionPayloadsFromConfig(&resolved.Config),
	}, resolved.ID)
}

func (n *daemonNativeTools) memoryList(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryListInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload, err := n.memoryHeaderPayloads(ctx, scope, input.Scope, input.Workspace)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryToolError(req.ToolID, err)
	}
	payload = limitMemoryPayloads(payload, input.Limit)
	return structuredResult(map[string]any{"memories": payload}, fmt.Sprintf("%d memories", len(payload)))
}

func (n *daemonNativeTools) memoryRead(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryReadInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	location, err := n.resolveMemoryLocation(ctx, scope, req.ToolID, input.Filename, input.Scope, input.Workspace)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryToolError(req.ToolID, err)
	}
	content, err := location.Store.Read(location.Scope, location.Filename)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryToolError(req.ToolID, err)
	}
	redactedContent := taskpkg.RedactClaimTokens(string(content))
	return structuredResult(map[string]any{
		"filename":  location.Filename,
		"scope":     location.Scope,
		"workspace": location.Workspace,
		"content":   redactedContent,
		"redacted":  redactedContent != string(content),
	}, location.Filename)
}

func (n *daemonNativeTools) memorySearch(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memorySearchInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	query, err := requiredNativeString(req.ToolID, "query", firstNonEmpty(input.Query, input.Q))
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	memoryScope, workspace, err := n.memoryScopeAndWorkspace(ctx, scope, input.Scope, input.Workspace)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryToolError(req.ToolID, err)
	}
	results, err := n.deps.MemoryStore.Search(ctx, query, memorypkg.SearchOptions{
		Scope:     memoryScope,
		Workspace: workspace,
		Limit:     input.Limit,
	})
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryToolError(req.ToolID, err)
	}
	payload := redactMemorySearchResults(results)
	return structuredResult(map[string]any{"results": payload}, fmt.Sprintf("%d memory results", len(payload)))
}

func (n *daemonNativeTools) memoryHistory(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input memoryHistoryInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	memoryScope, workspace, err := n.memoryScopeAndWorkspace(ctx, scope, input.Scope, input.Workspace)
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryToolError(req.ToolID, err)
	}
	since, err := parseNativeOptionalRFC3339(req.ToolID, "since", input.Since)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	records, err := n.deps.MemoryStore.History(ctx, memorypkg.OperationHistoryQuery{
		Scope:     memoryScope,
		Workspace: workspace,
		Operation: memorypkg.Operation(strings.TrimSpace(input.Operation)),
		Since:     since,
		Limit:     input.Limit,
	})
	if err != nil {
		return toolspkg.ToolResult{}, nativeMemoryToolError(req.ToolID, err)
	}
	payload := core.MemoryOperationPayloads(records)
	for i := range payload {
		payload[i].Summary = taskpkg.RedactClaimTokens(payload[i].Summary)
	}
	return structuredResult(map[string]any{"operations": payload}, fmt.Sprintf("%d memory operations", len(payload)))
}

func (n *daemonNativeTools) observeEvents(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	input, query, err := decodeObserveEventQueryInput(req)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	events, err := n.deps.Observer.QueryEvents(ctx, query)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := observeEventPayloads(events)
	payload = limitObservePayloads(payload, input.Limit)
	return structuredResult(map[string]any{"events": payload}, fmt.Sprintf("%d events", len(payload)))
}

func (n *daemonNativeTools) observeMetrics(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input struct{}
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	health, err := n.deps.Observer.Health(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := redactObserveHealthPayload(core.ObserveHealthPayloadFromHealth(&health))
	return structuredResult(map[string]any{"health": payload}, payload.Status)
}

func (n *daemonNativeTools) observeSearch(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	input, query, err := decodeObserveSearchInput(req)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	query.Limit = 0
	events, err := n.deps.Observer.QueryEvents(ctx, query)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := filterObserveEvents(observeEventPayloads(events), input.Query)
	payload = limitObservePayloads(payload, input.Limit)
	return structuredResult(map[string]any{"events": payload}, fmt.Sprintf("%d events", len(payload)))
}

func (n *daemonNativeTools) bridgesList(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input struct{}
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	instances, err := n.deps.Bridges.ListInstances(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	health, err := n.bridgeHealthMap(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := make([]contract.BridgePayload, 0, len(instances))
	for _, instance := range instances {
		payload = append(payload, redactedBridgePayload(instance))
		mergeBridgeDegradation(health, instance)
	}
	return structuredResult(map[string]any{
		"bridges":       payload,
		"bridge_health": health,
		"redacted":      true,
	}, fmt.Sprintf("%d bridges", len(payload)))
}

func (n *daemonNativeTools) bridgesStatus(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input bridgeStatusInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	health, err := n.bridgeHealthMap(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	if bridgeID := strings.TrimSpace(input.BridgeID); bridgeID != "" {
		instance, err := n.deps.Bridges.GetInstance(ctx, bridgeID)
		if err != nil {
			return toolspkg.ToolResult{}, err
		}
		mergeBridgeDegradation(health, *instance)
		return structuredResult(map[string]any{
			"bridge":   redactedBridgePayload(*instance),
			"health":   health[strings.TrimSpace(instance.ID)],
			"redacted": true,
		}, string(instance.Status))
	}
	instances, err := n.deps.Bridges.ListInstances(ctx)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	payload := make([]contract.BridgePayload, 0, len(instances))
	statusCounts := make(map[string]int)
	for _, instance := range instances {
		payload = append(payload, redactedBridgePayload(instance))
		statusCounts[string(instance.Status)]++
		mergeBridgeDegradation(health, instance)
	}
	return structuredResult(map[string]any{
		"bridges":       payload,
		"bridge_health": health,
		"status_counts": statusCounts,
		"redacted":      true,
	}, fmt.Sprintf("%d bridges", len(payload)))
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

func (n *daemonNativeTools) autonomyClaimNext(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input autonomyClaimNextInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	actor, sessionID, err := autonomyActorContext(req.ToolID, scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	criteria, err := input.criteria(scope, sessionID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutonomyToolError(req.ToolID, err)
	}
	result, err := n.deps.Tasks.ClaimNextRun(ctx, criteria, actor)
	if err != nil {
		if errors.Is(err, taskpkg.ErrNoClaimableRun) {
			return structuredResult(map[string]any{"claimed": false}, "no claimable task runs")
		}
		return toolspkg.ToolResult{}, nativeAutonomyToolError(req.ToolID, err)
	}
	if result == nil {
		return toolspkg.ToolResult{}, errors.New("daemon: task-run claim returned an empty result")
	}
	payload := core.AgentTaskClaimPayloadFromResult(result)
	return structuredResult(
		map[string]any{"claimed": true, "claim": payload},
		fmt.Sprintf("claimed %s", payload.Lease.RunID),
	)
}

func (n *daemonNativeTools) autonomyHeartbeat(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input autonomyHeartbeatInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	actor, sessionID, err := autonomyActorContext(req.ToolID, scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	runID, err := requiredNativeString(req.ToolID, "run_id", input.RunID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	leaseDuration, err := autonomyLeaseDuration(input.LeaseSeconds)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutonomyToolError(req.ToolID, err)
	}
	handle, err := n.lookupAutonomyLease(ctx, req.ToolID, sessionID, runID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	run, err := n.deps.Tasks.HeartbeatRunLease(ctx, taskpkg.LeaseHeartbeat{
		RunID:         runID,
		ClaimToken:    handle.ClaimToken,
		LeaseDuration: leaseDuration,
	}, actor)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutonomyToolError(req.ToolID, err)
	}
	lease := core.AgentTaskLeasePayloadFromRun(run, nil)
	return structuredResult(map[string]any{"lease": lease}, fmt.Sprintf("heartbeat %s", lease.RunID))
}

func (n *daemonNativeTools) autonomyComplete(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input autonomyCompleteInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	actor, sessionID, err := autonomyActorContext(req.ToolID, scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	runID, err := requiredNativeString(req.ToolID, "run_id", input.RunID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	result := taskpkg.RunResult{Value: cloneJSON(input.Result)}
	if err := result.Validate("run_result"); err != nil {
		return toolspkg.ToolResult{}, nativeAutonomyToolError(req.ToolID, err)
	}
	handle, err := n.lookupAutonomyLease(ctx, req.ToolID, sessionID, runID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	run, err := n.deps.Tasks.CompleteRunLease(ctx, taskpkg.LeaseCompletion{
		RunID:      runID,
		ClaimToken: handle.ClaimToken,
		Result:     result,
	}, actor)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutonomyToolError(req.ToolID, err)
	}
	lease := core.AgentTaskLeasePayloadFromRun(run, nil)
	return structuredResult(map[string]any{"lease": lease}, fmt.Sprintf("completed %s", lease.RunID))
}

func (n *daemonNativeTools) autonomyFail(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input autonomyFailInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	actor, sessionID, err := autonomyActorContext(req.ToolID, scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	runID, err := requiredNativeString(req.ToolID, "run_id", input.RunID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	failure := taskpkg.RunFailure{
		Error:    strings.TrimSpace(input.Error),
		Metadata: cloneJSON(input.Metadata),
	}
	if err := failure.Validate("run_failure"); err != nil {
		return toolspkg.ToolResult{}, nativeAutonomyToolError(req.ToolID, err)
	}
	handle, err := n.lookupAutonomyLease(ctx, req.ToolID, sessionID, runID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	run, err := n.deps.Tasks.FailRunLease(ctx, taskpkg.LeaseFailure{
		RunID:      runID,
		ClaimToken: handle.ClaimToken,
		Failure:    failure,
	}, actor)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutonomyToolError(req.ToolID, err)
	}
	lease := core.AgentTaskLeasePayloadFromRun(run, nil)
	return structuredResult(map[string]any{"lease": lease}, fmt.Sprintf("failed %s", lease.RunID))
}

func (n *daemonNativeTools) autonomyRelease(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input autonomyReleaseInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	actor, sessionID, err := autonomyActorContext(req.ToolID, scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	runID, err := requiredNativeString(req.ToolID, "run_id", input.RunID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	handle, err := n.lookupAutonomyLease(ctx, req.ToolID, sessionID, runID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	run, err := n.deps.Tasks.ReleaseRunLease(ctx, taskpkg.LeaseRelease{
		RunID:      runID,
		ClaimToken: handle.ClaimToken,
		Reason:     strings.TrimSpace(input.Reason),
	}, actor)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutonomyToolError(req.ToolID, err)
	}
	lease := core.AgentTaskLeasePayloadFromRun(run, nil)
	return structuredResult(map[string]any{"lease": lease}, fmt.Sprintf("released %s", lease.RunID))
}

func (n *daemonNativeTools) skillsFor(
	ctx context.Context,
	scope toolspkg.Scope,
	workspaceID string,
) ([]*skills.Skill, error) {
	if n.deps.Skills == nil {
		return nil, errors.New("daemon: skills registry is required")
	}
	agentName := strings.TrimSpace(scope.AgentName)
	workspaceID = firstNonEmpty(workspaceID, scope.WorkspaceID)
	if workspaceID == "" {
		if agentName != "" {
			return n.deps.Skills.ForAgent(ctx, nil, agentName)
		}
		return n.deps.Skills.List(), nil
	}
	if n.deps.WorkspaceResolver == nil {
		return nil, errors.New("daemon: workspace resolver is required for workspace skills")
	}
	resolved, err := n.deps.WorkspaceResolver.Resolve(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if agentName != "" {
		return n.deps.Skills.ForAgent(ctx, &resolved, agentName)
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

func (n *daemonNativeTools) workspaceID(ctx context.Context, ref string) (string, error) {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return "", nil
	}
	if n.deps.Workspaces == nil {
		return trimmed, nil
	}
	workspace, err := n.deps.Workspaces.Get(ctx, trimmed)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(workspace.ID), nil
}

type nativeAuthoredAgentTarget struct {
	workspaceID     string
	workspaceRoot   string
	agentName       string
	agentPath       string
	heartbeatConfig aghconfig.HeartbeatConfig
}

func (n *daemonNativeTools) authoredAgentTarget(
	ctx context.Context,
	toolID toolspkg.ToolID,
	workspaceRef string,
	agentName string,
) (nativeAuthoredAgentTarget, error) {
	workspaceID, err := requiredNativeString(toolID, "workspace_id", workspaceRef)
	if err != nil {
		return nativeAuthoredAgentTarget{}, err
	}
	name, err := requiredNativeString(toolID, "agent_name", agentName)
	if err != nil {
		return nativeAuthoredAgentTarget{}, err
	}
	if n.deps.WorkspaceResolver == nil {
		return nativeAuthoredAgentTarget{}, errors.New("daemon: workspace resolver is required")
	}
	resolved, err := n.deps.WorkspaceResolver.Resolve(ctx, workspaceID)
	if err != nil {
		return nativeAuthoredAgentTarget{}, err
	}
	root := strings.TrimSpace(resolved.RootDir)
	if root == "" {
		return nativeAuthoredAgentTarget{}, workspacepkg.ErrWorkspaceRootMissing
	}
	return nativeAuthoredAgentTarget{
		workspaceID:     strings.TrimSpace(resolved.ID),
		workspaceRoot:   root,
		agentName:       name,
		agentPath:       nativeAuthoredAgentPath(&resolved, name),
		heartbeatConfig: resolved.Config.Agents.Heartbeat,
	}, nil
}

func (t nativeAuthoredAgentTarget) heartbeatAuthoringTarget() heartbeat.AuthoringTarget {
	return heartbeat.AuthoringTarget{
		WorkspaceID:   t.workspaceID,
		WorkspaceRoot: nativeAuthoredSourceRoot(t.workspaceRoot, t.agentPath),
		AgentName:     t.agentName,
		AgentPath:     t.agentPath,
		Config:        t.heartbeatConfig,
	}
}

func nativeAuthoredSourceRoot(workspaceRoot string, agentPath string) string {
	root := strings.TrimSpace(workspaceRoot)
	source := strings.TrimSpace(agentPath)
	if source == "" || !filepath.IsAbs(source) || nativePathWithinRoot(root, source) {
		return root
	}
	if derived := nativeTrustedRootFromAgentSourcePath(source); derived != "" {
		return derived
	}
	return root
}

func nativeTrustedRootFromAgentSourcePath(agentPath string) string {
	cleaned := filepath.Clean(strings.TrimSpace(agentPath))
	if !strings.EqualFold(filepath.Base(cleaned), "AGENT.md") {
		return ""
	}
	agentDir := filepath.Dir(cleaned)
	agentsDir := filepath.Dir(agentDir)
	if filepath.Base(agentsDir) != aghconfig.AgentsDirName {
		return ""
	}
	root := filepath.Dir(agentsDir)
	if filepath.Base(root) == aghconfig.DirName {
		return filepath.Dir(root)
	}
	return root
}

func nativePathWithinRoot(root string, sourcePath string) bool {
	trimmedRoot := strings.TrimSpace(root)
	trimmedSource := strings.TrimSpace(sourcePath)
	if trimmedRoot == "" || trimmedSource == "" {
		return false
	}
	absRoot, err := filepath.Abs(filepath.Clean(trimmedRoot))
	if err != nil {
		return false
	}
	sourceForRoot := filepath.Clean(trimmedSource)
	if !filepath.IsAbs(sourceForRoot) {
		sourceForRoot = filepath.Join(absRoot, sourceForRoot)
	}
	absSource, err := filepath.Abs(sourceForRoot)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absSource)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func nativeAuthoredAgentPath(workspace *workspacepkg.ResolvedWorkspace, agentName string) string {
	name := strings.TrimSpace(agentName)
	if workspace == nil {
		return ""
	}
	for _, agent := range workspace.Agents {
		if strings.TrimSpace(agent.Name) == name && strings.TrimSpace(agent.SourcePath) != "" {
			return strings.TrimSpace(agent.SourcePath)
		}
	}
	if root := strings.TrimSpace(workspace.RootDir); root != "" && name != "" {
		return filepath.Join(root, aghconfig.DirName, aghconfig.AgentsDirName, name, "AGENT.md")
	}
	return ""
}

func (n *daemonNativeTools) workspaceAgents(
	ctx context.Context,
	resolved *workspacepkg.ResolvedWorkspace,
) ([]aghconfig.AgentDef, error) {
	if resolved == nil {
		return nil, errors.New("daemon: resolved workspace is required")
	}
	merged := make(map[string]aghconfig.AgentDef, len(resolved.Agents))
	for _, agent := range resolved.Agents {
		name := strings.TrimSpace(agent.Name)
		if name == "" {
			continue
		}
		merged[name] = agent
	}
	if n.deps.AgentCatalog != nil {
		catalogAgents, err := n.deps.AgentCatalog.ListAgents(ctx)
		if err != nil {
			return nil, err
		}
		for _, agent := range catalogAgents {
			name := strings.TrimSpace(agent.Name)
			if name == "" {
				continue
			}
			if _, exists := merged[name]; exists {
				continue
			}
			merged[name] = agent
		}
	}
	names := make([]string, 0, len(merged))
	for name := range merged {
		names = append(names, name)
	}
	slices.Sort(names)
	agents := make([]aghconfig.AgentDef, 0, len(names))
	for _, name := range names {
		agents = append(agents, merged[name])
	}
	return agents, nil
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

type networkInboxInput struct {
	SessionID string `json:"session_id,omitempty"`
}

type networkSendInput struct {
	SessionID   string               `json:"session_id,omitempty"`
	Channel     string               `json:"channel"`
	Surface     string               `json:"surface,omitempty"`
	ThreadID    string               `json:"thread_id,omitempty"`
	DirectID    string               `json:"direct_id,omitempty"`
	Kind        string               `json:"kind"`
	To          string               `json:"to,omitempty"`
	Body        json.RawMessage      `json:"body"`
	WorkID      string               `json:"work_id,omitempty"`
	ReplyTo     string               `json:"reply_to,omitempty"`
	TraceID     string               `json:"trace_id,omitempty"`
	CausationID string               `json:"causation_id,omitempty"`
	ExpiresAt   *int64               `json:"expires_at,omitempty"`
	ID          string               `json:"id,omitempty"`
	Ext         network.ExtensionMap `json:"ext,omitempty"`
}

type sessionListInput struct {
	Workspace string `json:"workspace,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type sessionIDInput struct {
	SessionID string `json:"session_id"`
}

type sessionEventQueryInput struct {
	SessionID     string `json:"session_id"`
	Type          string `json:"type,omitempty"`
	AgentName     string `json:"agent_name,omitempty"`
	TurnID        string `json:"turn_id,omitempty"`
	AfterSequence int64  `json:"after_sequence,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Since         string `json:"since,omitempty"`
}

type agentHeartbeatStatusInput struct {
	WorkspaceID             string `json:"workspace_id"`
	AgentName               string `json:"agent_name"`
	SessionID               string `json:"session_id,omitempty"`
	IncludeSessionHealth    bool   `json:"include_session_health,omitempty"`
	IncludeRecentWakeEvents bool   `json:"include_recent_wake_events,omitempty"`
}

type agentHeartbeatWakeInput struct {
	WorkspaceID string `json:"workspace_id"`
	AgentName   string `json:"agent_name"`
	SessionID   string `json:"session_id"`
	Source      string `json:"source,omitempty"`
	DryRun      bool   `json:"dry_run,omitempty"`
}

func (i sessionEventQueryInput) eventQuery(id toolspkg.ToolID) (store.EventQuery, error) {
	query := store.EventQuery{
		Type:          strings.TrimSpace(i.Type),
		AgentName:     strings.TrimSpace(i.AgentName),
		TurnID:        strings.TrimSpace(i.TurnID),
		AfterSequence: i.AfterSequence,
		Limit:         i.Limit,
	}
	if rawSince := strings.TrimSpace(i.Since); rawSince != "" {
		since, err := time.Parse(time.RFC3339, rawSince)
		if err != nil {
			return store.EventQuery{}, toolspkg.NewToolError(
				toolspkg.ErrorCodeInvalidInput,
				id,
				"session event since must be an RFC3339 timestamp",
				fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
				toolspkg.ReasonSchemaInvalid,
			)
		}
		query.Since = since
	}
	if err := query.Validate(); err != nil {
		return store.EventQuery{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			id,
			"session event query is invalid",
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonSchemaInvalid,
		)
	}
	return query, nil
}

type workspaceRefInput struct {
	Workspace string `json:"workspace"`
}

type memoryListInput struct {
	Scope     string `json:"scope,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type memoryReadInput struct {
	Filename  string `json:"filename"`
	Scope     string `json:"scope,omitempty"`
	Workspace string `json:"workspace,omitempty"`
}

type memorySearchInput struct {
	Query     string `json:"query,omitempty"`
	Q         string `json:"q,omitempty"`
	Scope     string `json:"scope,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type memoryHistoryInput struct {
	Scope     string `json:"scope,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	Operation string `json:"operation,omitempty"`
	Since     string `json:"since,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type memoryToolLocation struct {
	Store     *memorypkg.Store
	Scope     memorypkg.Scope
	Workspace string
	Filename  string
}

type memoryHeaderPayload struct {
	Filename    string          `json:"filename"`
	Name        string          `json:"name"`
	Type        memorypkg.Type  `json:"type"`
	Scope       memorypkg.Scope `json:"scope"`
	Workspace   string          `json:"workspace,omitempty"`
	AgentName   string          `json:"agent_name,omitempty"`
	Description string          `json:"description,omitempty"`
	ModTime     time.Time       `json:"mod_time"`
}

type observeEventQueryInput struct {
	SessionID string `json:"session_id,omitempty"`
	AgentName string `json:"agent_name,omitempty"`
	Type      string `json:"type,omitempty"`
	Since     string `json:"since,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

func (i observeEventQueryInput) eventSummaryQuery(id toolspkg.ToolID) (store.EventSummaryQuery, error) {
	since, err := parseNativeOptionalRFC3339(id, "since", i.Since)
	if err != nil {
		return store.EventSummaryQuery{}, err
	}
	query := store.EventSummaryQuery{
		SessionID: strings.TrimSpace(i.SessionID),
		AgentName: strings.TrimSpace(i.AgentName),
		Type:      strings.TrimSpace(i.Type),
		Since:     since,
		Limit:     i.Limit,
	}
	if err := query.Validate(); err != nil {
		return store.EventSummaryQuery{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			id,
			"observe event query is invalid",
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonSchemaInvalid,
		)
	}
	return query, nil
}

type observeSearchInput struct {
	Query string `json:"query"`
	observeEventQueryInput
}

type bridgeStatusInput struct {
	BridgeID string `json:"bridge_id,omitempty"`
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

type autonomyClaimNextInput struct {
	WorkspaceID          string   `json:"workspace_id,omitempty"`
	RequiredCapabilities []string `json:"required_capabilities,omitempty"`
	PriorityMin          int      `json:"priority_min,omitempty"`
	LeaseSeconds         int64    `json:"lease_seconds,omitempty"`
}

func (i autonomyClaimNextInput) criteria(scope toolspkg.Scope, sessionID string) (taskpkg.ClaimCriteria, error) {
	leaseDuration, err := autonomyLeaseDuration(i.LeaseSeconds)
	if err != nil {
		return taskpkg.ClaimCriteria{}, err
	}
	workspaceID := strings.TrimSpace(i.WorkspaceID)
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(scope.WorkspaceID)
	}
	return taskpkg.ClaimCriteria{
		WorkspaceID:      workspaceID,
		ClaimerSessionID: sessionID,
		ClaimedBy: &taskpkg.ActorIdentity{
			Kind: taskpkg.ActorKindAgentSession,
			Ref:  sessionID,
		},
		AgentName:            strings.TrimSpace(scope.AgentName),
		RequiredCapabilities: trimNativeStrings(i.RequiredCapabilities),
		PriorityMin:          i.PriorityMin,
		LeaseDuration:        leaseDuration,
	}, nil
}

type autonomyHeartbeatInput struct {
	RunID        string `json:"run_id"`
	LeaseSeconds int64  `json:"lease_seconds,omitempty"`
}

type autonomyCompleteInput struct {
	RunID  string          `json:"run_id"`
	Result json.RawMessage `json:"result,omitempty"`
}

type autonomyFailInput struct {
	RunID    string          `json:"run_id"`
	Error    string          `json:"error"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type autonomyReleaseInput struct {
	RunID  string `json:"run_id"`
	Reason string `json:"reason,omitempty"`
}

func decodeNativeInput(req toolspkg.CallRequest, dst any) error {
	raw := req.Input
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = json.RawMessage(`{}`)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
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

func requiredNativeString(id toolspkg.ToolID, field string, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nativeRequiredInputError(id, field)
	}
	return trimmed, nil
}

func nativeRequiredInputError(id toolspkg.ToolID, field string) error {
	return toolspkg.NewToolError(
		toolspkg.ErrorCodeInvalidInput,
		id,
		fmt.Sprintf("%s is required", field),
		toolspkg.ErrToolInvalidInput,
		toolspkg.ReasonSchemaInvalid,
	)
}

func autonomyActorContext(id toolspkg.ToolID, scope toolspkg.Scope) (taskpkg.ActorContext, string, error) {
	sessionID := strings.TrimSpace(scope.SessionID)
	if sessionID == "" {
		return taskpkg.ActorContext{}, "", toolspkg.NewToolError(
			toolspkg.ErrorCodeDenied,
			id,
			"autonomy tool requires a caller session",
			fmt.Errorf("%w: session_id is required", toolspkg.ErrToolDenied),
			toolspkg.ReasonAutonomySessionRequired,
		)
	}
	actor, err := taskpkg.DeriveAgentSessionActorContext(sessionID)
	if err != nil {
		return taskpkg.ActorContext{}, "", nativeAutonomyToolError(id, err)
	}
	return actor, sessionID, nil
}

func (n *daemonNativeTools) lookupAutonomyLease(
	ctx context.Context,
	id toolspkg.ToolID,
	sessionID string,
	runID string,
) (taskpkg.AutonomyLeaseHandle, error) {
	authority, ok := n.deps.Tasks.(taskpkg.AutonomyLeaseAuthority)
	if !ok {
		return taskpkg.AutonomyLeaseHandle{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeUnavailable,
			id,
			"autonomy lease authority is unavailable",
			fmt.Errorf("%w: task autonomy lease authority is unavailable", toolspkg.ErrToolUnavailable),
			toolspkg.ReasonBackendUnhealthy,
		)
	}
	handle, err := authority.LookupActiveRunForSession(ctx, sessionID, runID)
	if err != nil {
		return taskpkg.AutonomyLeaseHandle{}, nativeAutonomyToolError(id, err)
	}
	return handle, nil
}

func autonomyLeaseDuration(seconds int64) (time.Duration, error) {
	switch {
	case seconds < 0:
		return 0, fmt.Errorf("%w: lease_seconds must be zero or positive: %d", taskpkg.ErrValidation, seconds)
	case seconds == 0:
		return 0, nil
	case seconds > int64(taskpkg.MaxRunLeaseDuration/time.Second):
		return 0, fmt.Errorf(
			"%w: lease_seconds exceeds %d",
			taskpkg.ErrValidation,
			int64(taskpkg.MaxRunLeaseDuration/time.Second),
		)
	default:
		return time.Duration(seconds) * time.Second, nil
	}
}

func nativeAutonomyToolError(id toolspkg.ToolID, err error) error {
	if err == nil {
		return nil
	}
	if reason, ok := taskpkg.AutonomyReasonOf(err); ok {
		code, toolReason, cause := autonomyToolErrorCodeAndReason(reason)
		return toolspkg.NewToolError(
			code,
			id,
			taskpkg.RedactClaimTokens(err.Error()),
			fmt.Errorf("%w: %w", cause, err),
			toolReason,
		)
	}
	switch {
	case errors.Is(err, taskpkg.ErrValidation),
		errors.Is(err, taskpkg.ErrInvalidScopeBinding),
		errors.Is(err, taskpkg.ErrImmutableField):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			id,
			taskpkg.RedactClaimTokens(err.Error()),
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonSchemaInvalid,
		)
	case errors.Is(err, taskpkg.ErrActiveRunLease):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeConflict,
			id,
			taskpkg.RedactClaimTokens(err.Error()),
			fmt.Errorf("%w: %w", toolspkg.ErrToolConflict, err),
			toolspkg.ReasonAutonomyLeaseAlreadyHeld,
		)
	case errors.Is(err, taskpkg.ErrPermissionDenied):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeDenied,
			id,
			taskpkg.RedactClaimTokens(err.Error()),
			fmt.Errorf("%w: %w", toolspkg.ErrToolDenied, err),
			toolspkg.ReasonSessionDenied,
		)
	case errors.Is(err, taskpkg.ErrInvalidClaimToken),
		errors.Is(err, taskpkg.ErrLeaseExpired),
		errors.Is(err, taskpkg.ErrInvalidStatusTransition):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeConflict,
			id,
			taskpkg.RedactClaimTokens(err.Error()),
			fmt.Errorf("%w: %w", toolspkg.ErrToolConflict, err),
			toolspkg.ReasonAutonomyLeaseExpired,
		)
	default:
		return err
	}
}

func nativeNetworkSendToolError(id toolspkg.ToolID, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, contract.ErrRawClaimTokenMetadata) {
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			id,
			"network send payload must not contain raw claim_token fields",
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonNetworkRawTokenRejected,
		)
	}
	if errors.Is(err, core.ErrNetworkValidation) {
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			id,
			err.Error(),
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonSchemaInvalid,
		)
	}
	return err
}

func autonomyToolErrorCodeAndReason(reason taskpkg.AutonomyReasonCode) (
	toolspkg.ErrorCode,
	toolspkg.ReasonCode,
	error,
) {
	switch reason {
	case taskpkg.AutonomySessionRequired:
		return toolspkg.ErrorCodeDenied, toolspkg.ReasonAutonomySessionRequired, toolspkg.ErrToolDenied
	case taskpkg.AutonomyForeignRun:
		return toolspkg.ErrorCodeDenied, toolspkg.ReasonAutonomyForeignRun, toolspkg.ErrToolDenied
	case taskpkg.AutonomyNoActiveLease:
		return toolspkg.ErrorCodeConflict, toolspkg.ReasonAutonomyNoActiveLease, toolspkg.ErrToolConflict
	case taskpkg.AutonomyLeaseExpired:
		return toolspkg.ErrorCodeConflict, toolspkg.ReasonAutonomyLeaseExpired, toolspkg.ErrToolConflict
	case taskpkg.AutonomyLeaseAlreadyHeld:
		return toolspkg.ErrorCodeConflict, toolspkg.ReasonAutonomyLeaseAlreadyHeld, toolspkg.ErrToolConflict
	default:
		return toolspkg.ErrorCodeConflict, toolspkg.ReasonAutonomyLeaseExpired, toolspkg.ErrToolConflict
	}
}

func trimNativeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		if next := strings.TrimSpace(value); next != "" {
			trimmed = append(trimmed, next)
		}
	}
	return trimmed
}

func decodeSessionEventQueryInput(req toolspkg.CallRequest) (sessionEventQueryInput, store.EventQuery, error) {
	var input sessionEventQueryInput
	if err := decodeNativeInput(req, &input); err != nil {
		return sessionEventQueryInput{}, store.EventQuery{}, err
	}
	sessionID, err := requiredNativeString(req.ToolID, "session_id", input.SessionID)
	if err != nil {
		return sessionEventQueryInput{}, store.EventQuery{}, err
	}
	input.SessionID = sessionID
	query, err := input.eventQuery(req.ToolID)
	if err != nil {
		return sessionEventQueryInput{}, store.EventQuery{}, err
	}
	return input, query, nil
}

func (n *daemonNativeTools) memoryHeaderPayloads(
	ctx context.Context,
	callerScope toolspkg.Scope,
	rawScope string,
	rawWorkspace string,
) ([]memoryHeaderPayload, error) {
	scope, err := core.ParseOptionalMemoryScope(rawScope)
	if err != nil {
		return nil, err
	}
	workspaceRef := firstNonEmpty(rawWorkspace, callerScope.WorkspaceID)
	locations := []memoryToolLocation{{Store: n.deps.MemoryStore, Scope: memorypkg.ScopeGlobal}}
	switch scope {
	case memorypkg.ScopeGlobal:
		locations = locations[:1]
	case memorypkg.ScopeWorkspace:
		workspace, err := n.memoryWorkspaceRoot(ctx, workspaceRef)
		if err != nil {
			return nil, err
		}
		locations = []memoryToolLocation{
			{Store: n.deps.MemoryStore.ForWorkspace(workspace), Scope: memorypkg.ScopeWorkspace, Workspace: workspace},
		}
	default:
		if strings.TrimSpace(workspaceRef) != "" {
			workspace, err := n.memoryWorkspaceRoot(ctx, workspaceRef)
			if err != nil {
				return nil, err
			}
			locations = append(locations, memoryToolLocation{
				Store:     n.deps.MemoryStore.ForWorkspace(workspace),
				Scope:     memorypkg.ScopeWorkspace,
				Workspace: workspace,
			})
		}
	}
	payload := make([]memoryHeaderPayload, 0)
	for _, location := range locations {
		headers, err := location.Store.Scan(location.Scope)
		if err != nil {
			return nil, err
		}
		for _, header := range headers {
			payload = append(payload, memoryHeaderPayloadFromHeader(header, location.Scope, location.Workspace))
		}
	}
	sort.SliceStable(payload, func(i, j int) bool {
		if payload[i].ModTime.Equal(payload[j].ModTime) {
			return payload[i].Filename < payload[j].Filename
		}
		return payload[i].ModTime.After(payload[j].ModTime)
	})
	return payload, nil
}

func (n *daemonNativeTools) resolveMemoryLocation(
	ctx context.Context,
	callerScope toolspkg.Scope,
	id toolspkg.ToolID,
	filename string,
	rawScope string,
	rawWorkspace string,
) (memoryToolLocation, error) {
	trimmedFilename, err := requiredNativeString(id, "filename", filename)
	if err != nil {
		return memoryToolLocation{}, err
	}
	scope, err := core.ParseOptionalMemoryScope(rawScope)
	if err != nil {
		return memoryToolLocation{}, err
	}
	workspaceRef := firstNonEmpty(rawWorkspace, callerScope.WorkspaceID)
	if scope != "" {
		location, err := n.memoryStoreFor(ctx, scope, workspaceRef)
		if err != nil {
			return memoryToolLocation{}, err
		}
		exists, err := location.Store.Exists(location.Scope, trimmedFilename)
		if err != nil {
			return memoryToolLocation{}, err
		}
		if !exists {
			return memoryToolLocation{}, fmt.Errorf("%w: memory %q not found", os.ErrNotExist, trimmedFilename)
		}
		location.Filename = trimmedFilename
		return location, nil
	}
	candidates := []memoryToolLocation{
		{Store: n.deps.MemoryStore, Scope: memorypkg.ScopeGlobal, Filename: trimmedFilename},
	}
	if strings.TrimSpace(workspaceRef) != "" {
		workspace, err := n.memoryWorkspaceRoot(ctx, workspaceRef)
		if err != nil {
			return memoryToolLocation{}, err
		}
		candidates = append(candidates, memoryToolLocation{
			Store:     n.deps.MemoryStore.ForWorkspace(workspace),
			Scope:     memorypkg.ScopeWorkspace,
			Workspace: workspace,
			Filename:  trimmedFilename,
		})
	}
	matches := make([]memoryToolLocation, 0, len(candidates))
	for _, candidate := range candidates {
		exists, err := candidate.Store.Exists(candidate.Scope, trimmedFilename)
		if err != nil {
			return memoryToolLocation{}, err
		}
		if exists {
			matches = append(matches, candidate)
		}
	}
	switch len(matches) {
	case 0:
		return memoryToolLocation{}, fmt.Errorf("%w: memory %q not found", os.ErrNotExist, trimmedFilename)
	case 1:
		return matches[0], nil
	default:
		return memoryToolLocation{}, core.NewMemoryValidationError(
			fmt.Errorf("memory %q exists in multiple scopes; set scope explicitly", trimmedFilename),
		)
	}
}

func (n *daemonNativeTools) memoryStoreFor(
	ctx context.Context,
	scope memorypkg.Scope,
	workspaceRef string,
) (memoryToolLocation, error) {
	switch scope.Normalize() {
	case memorypkg.ScopeGlobal:
		return memoryToolLocation{Store: n.deps.MemoryStore, Scope: memorypkg.ScopeGlobal}, nil
	case memorypkg.ScopeWorkspace:
		workspace, err := n.memoryWorkspaceRoot(ctx, workspaceRef)
		if err != nil {
			return memoryToolLocation{}, err
		}
		return memoryToolLocation{
			Store:     n.deps.MemoryStore.ForWorkspace(workspace),
			Scope:     memorypkg.ScopeWorkspace,
			Workspace: workspace,
		}, nil
	default:
		return memoryToolLocation{}, core.NewMemoryValidationError(fmt.Errorf("unsupported scope %q", scope))
	}
}

func (n *daemonNativeTools) memoryScopeAndWorkspace(
	ctx context.Context,
	callerScope toolspkg.Scope,
	rawScope string,
	rawWorkspace string,
) (memorypkg.Scope, string, error) {
	scope, err := core.ParseOptionalMemoryScope(rawScope)
	if err != nil {
		return "", "", err
	}
	workspaceRef := firstNonEmpty(rawWorkspace, callerScope.WorkspaceID)
	if scope == memorypkg.ScopeWorkspace || strings.TrimSpace(workspaceRef) != "" {
		workspace, err := n.memoryWorkspaceRoot(ctx, workspaceRef)
		if err != nil {
			return "", "", err
		}
		return scope, workspace, nil
	}
	return scope, "", nil
}

func (n *daemonNativeTools) memoryWorkspaceRoot(ctx context.Context, ref string) (string, error) {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return core.ResolveMemoryWorkspace(trimmed)
	}
	if n.deps.Workspaces != nil {
		workspace, err := n.deps.Workspaces.Get(ctx, trimmed)
		switch {
		case err == nil && strings.TrimSpace(workspace.RootDir) != "":
			return core.ResolveMemoryWorkspace(workspace.RootDir)
		case err == nil:
			return core.ResolveMemoryWorkspace(trimmed)
		case !errors.Is(err, workspacepkg.ErrWorkspaceNotFound):
			return "", err
		}
	}
	return core.ResolveMemoryWorkspace(trimmed)
}

func memoryHeaderPayloadFromHeader(
	header memorypkg.Header,
	scope memorypkg.Scope,
	workspace string,
) memoryHeaderPayload {
	return memoryHeaderPayload{
		Filename:    strings.TrimSpace(header.Filename),
		Name:        taskpkg.RedactClaimTokens(strings.TrimSpace(header.Name)),
		Type:        header.Type.Normalize(),
		Scope:       scope.Normalize(),
		Workspace:   strings.TrimSpace(workspace),
		AgentName:   strings.TrimSpace(header.AgentName),
		Description: taskpkg.RedactClaimTokens(strings.TrimSpace(header.Description)),
		ModTime:     header.ModTime.UTC(),
	}
}

func limitMemoryPayloads(items []memoryHeaderPayload, limit int) []memoryHeaderPayload {
	if limit <= 0 || limit >= len(items) {
		return items
	}
	return items[:limit]
}

func redactMemorySearchResults(results []memorypkg.SearchResult) []memorypkg.SearchResult {
	payload := make([]memorypkg.SearchResult, 0, len(results))
	for _, result := range results {
		next := result
		next.Name = taskpkg.RedactClaimTokens(strings.TrimSpace(next.Name))
		next.Description = taskpkg.RedactClaimTokens(strings.TrimSpace(next.Description))
		next.Snippet = taskpkg.RedactClaimTokens(strings.TrimSpace(next.Snippet))
		next.Workspace = strings.TrimSpace(next.Workspace)
		next.ModTime = next.ModTime.UTC()
		payload = append(payload, next)
	}
	return payload
}

func nativeMemoryToolError(id toolspkg.ToolID, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, memorypkg.ErrValidation):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			id,
			err.Error(),
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonSchemaInvalid,
		)
	case errors.Is(err, os.ErrNotExist):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeNotFound,
			id,
			err.Error(),
			fmt.Errorf("%w: %w", toolspkg.ErrToolNotFound, err),
			toolspkg.ReasonToolUnknown,
		)
	default:
		return err
	}
}

func decodeObserveEventQueryInput(req toolspkg.CallRequest) (observeEventQueryInput, store.EventSummaryQuery, error) {
	var input observeEventQueryInput
	if err := decodeNativeInput(req, &input); err != nil {
		return observeEventQueryInput{}, store.EventSummaryQuery{}, err
	}
	query, err := input.eventSummaryQuery(req.ToolID)
	if err != nil {
		return observeEventQueryInput{}, store.EventSummaryQuery{}, err
	}
	return input, query, nil
}

func decodeObserveSearchInput(req toolspkg.CallRequest) (observeSearchInput, store.EventSummaryQuery, error) {
	var input observeSearchInput
	if err := decodeNativeInput(req, &input); err != nil {
		return observeSearchInput{}, store.EventSummaryQuery{}, err
	}
	if _, err := requiredNativeString(req.ToolID, "query", input.Query); err != nil {
		return observeSearchInput{}, store.EventSummaryQuery{}, err
	}
	query, err := input.eventSummaryQuery(req.ToolID)
	if err != nil {
		return observeSearchInput{}, store.EventSummaryQuery{}, err
	}
	return input, query, nil
}

func parseNativeOptionalRFC3339(id toolspkg.ToolID, field string, raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, nil
	}
	timestamp, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			id,
			fmt.Sprintf("%s must be an RFC3339 timestamp", field),
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonSchemaInvalid,
		)
	}
	return timestamp, nil
}

func observeEventPayloads(events []store.EventSummary) []contract.ObserveEventPayload {
	payload := make([]contract.ObserveEventPayload, 0, len(events))
	for _, event := range events {
		item := core.ObserveEventPayloadFromEvent(event)
		item.Summary = taskpkg.RedactClaimTokens(strings.TrimSpace(item.Summary))
		payload = append(payload, item)
	}
	return payload
}

func redactObserveHealthPayload(payload contract.ObserveHealthPayload) contract.ObserveHealthPayload {
	payload.Retention.LastSweepError = taskpkg.RedactClaimTokens(strings.TrimSpace(payload.Retention.LastSweepError))
	for i := range payload.Failures.Recent {
		payload.Failures.Recent[i].Summary = taskpkg.RedactClaimTokens(
			strings.TrimSpace(payload.Failures.Recent[i].Summary),
		)
		payload.Failures.Recent[i].CrashBundlePath = taskpkg.RedactClaimTokens(
			strings.TrimSpace(payload.Failures.Recent[i].CrashBundlePath),
		)
	}
	for i := range payload.AgentProbes {
		payload.AgentProbes[i].Command = taskpkg.RedactClaimTokens(strings.TrimSpace(payload.AgentProbes[i].Command))
		payload.AgentProbes[i].Executable = taskpkg.RedactClaimTokens(
			strings.TrimSpace(payload.AgentProbes[i].Executable),
		)
		payload.AgentProbes[i].Error = taskpkg.RedactClaimTokens(strings.TrimSpace(payload.AgentProbes[i].Error))
	}
	for i := range payload.Activities {
		payload.Activities[i].LastActivityDetail = taskpkg.RedactClaimTokens(
			strings.TrimSpace(payload.Activities[i].LastActivityDetail),
		)
		payload.Activities[i].CurrentTool = taskpkg.RedactClaimTokens(
			strings.TrimSpace(payload.Activities[i].CurrentTool),
		)
		payload.Activities[i].ToolCallID = taskpkg.RedactClaimTokens(
			strings.TrimSpace(payload.Activities[i].ToolCallID),
		)
		payload.Activities[i].StallReason = taskpkg.RedactClaimTokens(
			strings.TrimSpace(payload.Activities[i].StallReason),
		)
	}
	return payload
}

func filterObserveEvents(
	events []contract.ObserveEventPayload,
	query string,
) []contract.ObserveEventPayload {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return events
	}
	filtered := make([]contract.ObserveEventPayload, 0, len(events))
	for _, event := range events {
		values := []string{event.ID, event.SessionID, event.Type, event.AgentName, event.Summary}
		if slices.ContainsFunc(values, func(value string) bool {
			return strings.Contains(strings.ToLower(value), needle)
		}) {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func limitObservePayloads(
	events []contract.ObserveEventPayload,
	limit int,
) []contract.ObserveEventPayload {
	if limit <= 0 || limit >= len(events) {
		return events
	}
	return events[:limit]
}

func (n *daemonNativeTools) bridgeHealthMap(ctx context.Context) (map[string]contract.BridgeHealthPayload, error) {
	health := make(map[string]contract.BridgeHealthPayload)
	if n.deps.Observer == nil {
		return health, nil
	}
	observed, err := n.deps.Observer.QueryBridgeHealth(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range observed {
		payload := core.BridgeHealthPayloadFromObserve(item)
		payload.LastError = taskpkg.RedactClaimTokens(strings.TrimSpace(payload.LastError))
		health[strings.TrimSpace(item.BridgeInstanceID)] = payload
	}
	return health, nil
}

func redactedBridgePayload(instance bridgepkg.BridgeInstance) contract.BridgePayload {
	payload := core.BridgePayloadFromBridgeInstance(instance)
	payload.ProviderConfig = nil
	if payload.Degradation != nil {
		payload.Degradation.Message = taskpkg.RedactClaimTokens(strings.TrimSpace(payload.Degradation.Message))
	}
	return payload
}

func mergeBridgeDegradation(
	health map[string]contract.BridgeHealthPayload,
	instance bridgepkg.BridgeInstance,
) {
	key := strings.TrimSpace(instance.ID)
	item := health[key]
	if instance.Degradation != nil {
		degradation := *instance.Degradation
		degradation.Message = taskpkg.RedactClaimTokens(strings.TrimSpace(degradation.Message))
		item.Degradation = &degradation
	} else {
		item.Degradation = nil
	}
	health[key] = item
}

func sessionHistoryPayload(history []store.TurnHistory, info *session.Info) []any {
	payload := make([]any, 0, len(history))
	for _, turn := range history {
		events := make([]any, 0, len(turn.Events))
		for _, event := range turn.Events {
			events = append(events, core.SessionEventPayloadFromEvent(event, info))
		}
		payload = append(payload, map[string]any{
			"turn_id": turn.TurnID,
			"events":  events,
		})
	}
	return payload
}

func limitSessionPayloads[T any](items []T, limit int) []T {
	if limit <= 0 || limit >= len(items) {
		return items
	}
	return items[:limit]
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
