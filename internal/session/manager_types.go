package session

import (
	"context"
	"log/slog"
	"sync"
	"time"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/sandbox"
	"github.com/compozy/agh/internal/session/inputqueue"
	"github.com/compozy/agh/internal/store"
	toolspkg "github.com/compozy/agh/internal/tools"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

// CreateOpts defines the inputs required to create a new session.
type CreateOpts struct {
	AgentName        string
	Provider         string
	Model            string
	ReasoningEffort  string
	SandboxRef       string
	DisableSandbox   bool
	Permissions      aghconfig.PermissionMode
	Name             string
	Workspace        string
	WorkspacePath    string
	Channel          string
	PromptOverlay    string
	Type             Type
	Lineage          *store.SessionLineage
	ParentSoulDigest string
	// AllowedToolsOverride is a concrete subset narrowing of the resolved agent tool policy.
	AllowedToolsOverride []string
}

// StoreOpener opens the per-session events store for a session directory.
type StoreOpener func(ctx context.Context, sessionID string, path string) (EventRecorder, error)

// IDGenerator returns unique identifiers for sessions and prompt turns.
type IDGenerator func() string

// HostedMCPLauncher mints and releases session-bound hosted MCP launch records.
type HostedMCPLauncher interface {
	Launch(ctx context.Context, req HostedMCPLaunchRequest) (aghconfig.MCPServer, error)
	CancelLaunch(sessionID string)
	ReleaseSession(sessionID string)
}

// HostedMCPLaunchRequest describes the session identity for a hosted MCP entry.
type HostedMCPLaunchRequest struct {
	SessionID   string
	WorkspaceID string
	AgentName   string
}

// ProviderSecretResolver resolves provider-bound secret refs at launch time.
type ProviderSecretResolver interface {
	ResolveRef(ctx context.Context, ref string) (string, error)
}

// Option customizes the session manager.
type Option func(*Manager)

// Manager owns active session lifecycle and runtime orchestration.
type Manager struct {
	mu           sync.RWMutex
	sessions     map[string]*Session
	pending      map[string]struct{}
	finalizing   map[string]chan struct{}
	promptDrains map[chan struct{}]struct{}
	spawnMu      sync.Mutex

	syntheticMu           sync.Mutex
	syntheticQueues       map[string][]queuedSyntheticPrompt
	syntheticDispatching  map[string]bool
	soulLocksMu           sync.Mutex
	soulLocks             map[string]chan struct{}
	sessionHealthHookMu   sync.Mutex
	sessionHealthHookLast map[string]time.Time
	transcriptCacheMu     sync.Mutex
	transcriptCache       map[string]*transcriptCacheEntry
	streamEventsMu        sync.Mutex
	streamEvents          *sessionEventBroadcaster

	logger                       *slog.Logger
	driver                       AgentDriver
	notifier                     Notifier
	networkPeers                 NetworkPeerLifecycle
	turnEndNotifier              TurnEndNotifier
	inputAugmenter               PromptInputAugmenter
	inputQueue                   *inputqueue.Service
	inputQueueStore              store.SessionInputQueueStore
	startupOverlay               StartupPromptOverlay
	hooks                        HookSet
	sandbox                      *sandbox.Registry
	agentResolver                AgentResolver
	providerSecrets              ProviderSecretResolver
	skillRegistry                SkillRegistry
	toolsetCatalog               toolspkg.ToolsetCatalog
	mcpResolver                  MCPResolver
	hostedMCP                    HostedMCPLauncher
	soulStore                    SoulSnapshotStore
	soulRunChecker               SoulRunActivityChecker
	sessionHealthStore           HealthStore
	sessionCatalog               store.SessionCatalog
	transcriptEpochStore         store.SessionTranscriptEpochStore
	ledgerMaterializer           LedgerMaterializer
	homePaths                    aghconfig.HomePaths
	workspace                    workspacepkg.RuntimeResolver
	openStore, openQueryStore    StoreOpener
	queryStoreExplicit           bool
	queryStoreRuntime            *queryStoreRuntime
	assembler                    PromptAssembler
	supervision                  aghconfig.SessionSupervisionConfig
	busyInput                    aghconfig.SessionBusyInputConfig
	sessionHealthStaleAfter      time.Duration
	lifecycleCtx                 context.Context
	now                          func() time.Time
	newSessionID                 IDGenerator
	newSandboxID                 IDGenerator
	newTurnID                    IDGenerator
	promptBufSize                int
	soulRefreshTimeout           time.Duration
	sessionHealthHookMinInterval time.Duration
}
