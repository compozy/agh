package settings

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

// WorkspaceResolver resolves and lists registered workspaces for settings flows.
type WorkspaceResolver interface {
	Resolve(ctx context.Context, idOrNameOrPath string) (workspacepkg.ResolvedWorkspace, error)
	List(ctx context.Context) ([]workspacepkg.Workspace, error)
}

// GeneralRuntimeProvider returns general daemon runtime metadata.
type GeneralRuntimeProvider interface {
	GeneralRuntimeStatus(ctx context.Context) (DaemonRuntimeStatus, error)
}

// MemoryRuntimeProvider returns memory runtime metadata.
type MemoryRuntimeProvider interface {
	MemoryHealthStatus(ctx context.Context) (MemoryHealthStatus, error)
}

// SkillsRuntime exposes the global skills registry state used by settings.
type SkillsRuntime interface {
	List() []*skillspkg.Skill
	SetEnabled(name string, resolved *workspacepkg.ResolvedWorkspace, enabled bool) error
}

// AutomationRuntimeProvider returns automation runtime metadata.
type AutomationRuntimeProvider interface {
	AutomationRuntimeStatus(ctx context.Context) (AutomationRuntimeStatus, error)
}

// NetworkRuntimeProvider returns network runtime metadata.
type NetworkRuntimeProvider interface {
	NetworkRuntimeStatus(ctx context.Context) (NetworkRuntimeStatus, error)
}

// ObservabilityRuntimeProvider returns observability runtime metadata.
type ObservabilityRuntimeProvider interface {
	ObservabilityRuntimeStatus(ctx context.Context) (ObservabilityRuntimeStatus, error)
}

// ExtensionStatusProvider returns installed extension summaries.
type ExtensionStatusProvider interface {
	InstalledExtensions(ctx context.Context) ([]InstalledExtension, error)
}

// TransportParityProvider returns settings transport parity metadata.
type TransportParityProvider interface {
	TransportParityStatus(ctx context.Context) (TransportParityStatus, error)
}

// Dependencies captures the runtime dependencies required by the settings service.
type Dependencies struct {
	WorkspaceResolver          WorkspaceResolver
	GeneralRuntime             GeneralRuntimeProvider
	MemoryRuntime              MemoryRuntimeProvider
	SkillsRuntime              SkillsRuntime
	AutomationRuntime          AutomationRuntimeProvider
	NetworkRuntime             NetworkRuntimeProvider
	ObservabilityRuntime       ObservabilityRuntimeProvider
	Extensions                 ExtensionStatusProvider
	TransportParity            TransportParityProvider
	RestartActionAvailable     bool
	ConsolidateActionAvailable bool
	LogTailAvailable           bool
	CommandLookPath            func(string) (string, error)
	LookupEnv                  func(string) (string, bool)
}

type service struct {
	homePaths                  aghconfig.HomePaths
	workspaceResolver          WorkspaceResolver
	generalRuntime             GeneralRuntimeProvider
	memoryRuntime              MemoryRuntimeProvider
	skillsRuntime              SkillsRuntime
	automationRuntime          AutomationRuntimeProvider
	networkRuntime             NetworkRuntimeProvider
	observabilityRuntime       ObservabilityRuntimeProvider
	extensions                 ExtensionStatusProvider
	transportParity            TransportParityProvider
	restartActionAvailable     bool
	consolidateActionAvailable bool
	logTailAvailable           bool
	commandLookPath            func(string) (string, error)
	lookupEnv                  func(string) (string, bool)
}

var _ Service = (*service)(nil)

// NewService constructs the daemon-facing settings orchestration service.
func NewService(homePaths aghconfig.HomePaths, deps Dependencies) (Service, error) {
	if strings.TrimSpace(homePaths.HomeDir) == "" {
		return nil, errors.New("settings: home paths are required")
	}

	commandLookPath := deps.CommandLookPath
	if commandLookPath == nil {
		commandLookPath = exec.LookPath
	}
	lookupEnv := deps.LookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	return &service{
		homePaths:                  homePaths,
		workspaceResolver:          deps.WorkspaceResolver,
		generalRuntime:             deps.GeneralRuntime,
		memoryRuntime:              deps.MemoryRuntime,
		skillsRuntime:              deps.SkillsRuntime,
		automationRuntime:          deps.AutomationRuntime,
		networkRuntime:             deps.NetworkRuntime,
		observabilityRuntime:       deps.ObservabilityRuntime,
		extensions:                 deps.Extensions,
		transportParity:            deps.TransportParity,
		restartActionAvailable:     deps.RestartActionAvailable,
		consolidateActionAvailable: deps.ConsolidateActionAvailable,
		logTailAvailable:           deps.LogTailAvailable,
		commandLookPath:            commandLookPath,
		lookupEnv:                  lookupEnv,
	}, nil
}

func (s *service) normalizeReadScope(scope ScopeKind, workspaceID string) (ScopeKind, string, error) {
	normalized := scope
	if normalized == "" {
		normalized = ScopeGlobal
	}
	if err := normalized.Validate(); err != nil {
		return "", "", validationError(err)
	}

	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	if normalized == ScopeGlobal && trimmedWorkspaceID != "" {
		return "", "", conflictError(errors.New("settings: workspace_id requires workspace scope"))
	}
	return normalized, trimmedWorkspaceID, nil
}

func (s *service) resolveWorkspace(
	ctx context.Context,
	scope ScopeKind,
	workspaceID string,
) (*workspacepkg.ResolvedWorkspace, error) {
	if scope != ScopeWorkspace {
		return nil, nil
	}
	if strings.TrimSpace(workspaceID) == "" {
		return nil, conflictError(errors.New("settings: workspace scope requires a workspace_id"))
	}
	if s.workspaceResolver == nil {
		return nil, errors.New("settings: workspace resolver is required for workspace scope")
	}

	resolved, err := s.workspaceResolver.Resolve(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return &resolved, nil
}

func (s *service) loadConfig(
	ctx context.Context,
	scope ScopeKind,
	workspaceID string,
) (aghconfig.Config, *workspacepkg.ResolvedWorkspace, error) {
	normalizedScope, normalizedWorkspaceID, err := s.normalizeReadScope(scope, workspaceID)
	if err != nil {
		return aghconfig.Config{}, nil, err
	}

	resolved, err := s.resolveWorkspace(ctx, normalizedScope, normalizedWorkspaceID)
	if err != nil {
		return aghconfig.Config{}, nil, err
	}

	if resolved != nil {
		cfg, loadErr := aghconfig.LoadForHome(s.homePaths, aghconfig.WithWorkspaceRoot(resolved.RootDir))
		return cfg, resolved, loadErr
	}

	cfg, loadErr := aghconfig.LoadForHome(s.homePaths)
	return cfg, nil, loadErr
}

func workspaceConfigPath(root string) string {
	return filepath.Join(strings.TrimSpace(root), aghconfig.DirName, aghconfig.ConfigName)
}
