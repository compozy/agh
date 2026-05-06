package daemon

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	memcontract "github.com/pedronauck/agh/internal/memory/contract"

	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/diagnostics"
	mcpauth "github.com/pedronauck/agh/internal/mcp/auth"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/network"
	settingspkg "github.com/pedronauck/agh/internal/settings"
	aghupdate "github.com/pedronauck/agh/internal/update"
	"github.com/pedronauck/agh/internal/version"
)

type settingsRuntimeSurface struct {
	config         aghconfig.Config
	startedAt      time.Time
	sessions       SessionManager
	observer       Observer
	memoryStore    *memory.Store
	dreamTrigger   DreamTrigger
	automation     automationRuntime
	network        networkRuntime
	mcpAuthStore   mcpauth.TokenStore
	secretResolver mcpauth.SecretRefResolver
	extensions     interface {
		List(context.Context) ([]contract.ExtensionPayload, error)
	}
	now  func() time.Time
	pid  func() int
	info func() Info
}

var _ settingspkg.GeneralRuntimeProvider = (*settingsRuntimeSurface)(nil)
var _ settingspkg.MemoryRuntimeProvider = (*settingsRuntimeSurface)(nil)
var _ settingspkg.AutomationRuntimeProvider = (*settingsRuntimeSurface)(nil)
var _ settingspkg.NetworkRuntimeProvider = (*settingsRuntimeSurface)(nil)
var _ settingspkg.ObservabilityRuntimeProvider = (*settingsRuntimeSurface)(nil)
var _ settingspkg.ExtensionStatusProvider = (*settingsRuntimeSurface)(nil)
var _ settingspkg.TransportParityProvider = (*settingsRuntimeSurface)(nil)
var _ settingspkg.MCPAuthRuntimeProvider = (*settingsRuntimeSurface)(nil)

func newSettingsRuntimeSurface(d *Daemon, state *bootState) *settingsRuntimeSurface {
	if state == nil {
		return &settingsRuntimeSurface{}
	}

	now := time.Now
	pid := func() int { return 0 }
	info := func() Info { return Info{} }
	if d != nil {
		if d.now != nil {
			now = d.now
		}
		if d.pid != nil {
			pid = d.pid
		}
		info = d.settingsInfoSnapshot
	}

	var mcpAuthStore mcpauth.TokenStore
	if store, ok := state.registry.(mcpauth.TokenStore); ok {
		mcpAuthStore = store
	}
	var secretResolver mcpauth.SecretRefResolver
	if state.providerVault != nil {
		secretResolver = func(ctx context.Context, ref string) (string, error) {
			value, err := state.providerVault.ResolveRef(ctx, ref)
			if err != nil {
				return "", err
			}
			diagnostics.RegisterDynamicSecret(value)
			return value, nil
		}
	}

	return &settingsRuntimeSurface{
		config:         state.cfg,
		startedAt:      state.startedAt,
		sessions:       state.sessions,
		observer:       state.observer,
		memoryStore:    state.memoryStore,
		dreamTrigger:   dreamTriggerFromRuntime(state.dreamRuntime),
		automation:     state.automation,
		network:        state.network,
		mcpAuthStore:   mcpAuthStore,
		secretResolver: secretResolver,
		extensions:     state.deps.Extensions,
		now:            now,
		pid:            pid,
		info:           info,
	}
}

func (d *Daemon) settingsInfoSnapshot() Info {
	if d == nil {
		return Info{}
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	return d.info
}

func (s *settingsRuntimeSurface) GeneralRuntimeStatus(
	ctx context.Context,
) (settingspkg.DaemonRuntimeStatus, error) {
	status := settingspkg.DaemonRuntimeStatus{
		Available: s.sessions != nil && s.observer != nil,
		Status:    "running",
		Socket:    strings.TrimSpace(s.config.Daemon.Socket),
		HTTPHost:  strings.TrimSpace(s.config.HTTP.Host),
		HTTPPort:  s.config.HTTP.Port,
	}

	info := s.currentInfo()
	if info.PID > 0 {
		status.PID = info.PID
	} else if s.pid != nil {
		status.PID = s.pid()
	}
	if !info.StartedAt.IsZero() {
		status.StartedAt = info.StartedAt
	} else {
		status.StartedAt = s.startedAt
	}
	if info.Port > 0 {
		status.HTTPPort = info.Port
	}

	if !status.Available {
		return status, nil
	}

	health, err := s.observer.Health(ctx)
	if err != nil {
		return settingspkg.DaemonRuntimeStatus{}, fmt.Errorf("daemon: settings general runtime health: %w", err)
	}
	sessions, err := s.sessions.ListAll(ctx)
	if err != nil {
		return settingspkg.DaemonRuntimeStatus{}, fmt.Errorf("daemon: settings general runtime sessions: %w", err)
	}

	status.UptimeSeconds = health.UptimeSeconds
	status.ActiveSessions = health.ActiveSessions
	status.ActiveAgents = health.ActiveAgents
	status.TotalSessions = len(sessions)
	status.Version = strings.TrimSpace(health.Version)
	return status, nil
}

func (s *settingsRuntimeSurface) MemoryHealthStatus(context.Context) (settingspkg.MemoryHealthStatus, error) {
	status := settingspkg.MemoryHealthStatus{
		Available:    s.memoryStore != nil,
		DreamEnabled: s.dreamTrigger != nil && s.dreamTrigger.Enabled(),
	}
	if s.memoryStore == nil {
		return status, nil
	}

	headers, err := s.memoryStore.Scan(memcontract.ScopeGlobal)
	if err != nil {
		return settingspkg.MemoryHealthStatus{}, fmt.Errorf("daemon: settings memory health scan: %w", err)
	}
	status.FileCount = len(headers)

	if s.dreamTrigger != nil {
		lastConsolidatedAt, err := s.dreamTrigger.LastConsolidatedAt()
		if err != nil {
			return settingspkg.MemoryHealthStatus{}, fmt.Errorf(
				"daemon: settings last consolidation timestamp: %w",
				err,
			)
		}
		if !lastConsolidatedAt.IsZero() {
			status.LastConsolidatedAt = &lastConsolidatedAt
		}
	}

	return status, nil
}

func (s *settingsRuntimeSurface) AutomationRuntimeStatus(
	ctx context.Context,
) (settingspkg.AutomationRuntimeStatus, error) {
	status := settingspkg.AutomationRuntimeStatus{Available: s.automation != nil}
	if s.automation == nil {
		return status, nil
	}

	managerStatus, err := s.automation.Status(ctx)
	if err != nil {
		return settingspkg.AutomationRuntimeStatus{}, fmt.Errorf("daemon: settings automation runtime: %w", err)
	}

	status.Running = managerStatus.Running
	status.SchedulerRunning = managerStatus.SchedulerRunning
	status.JobTotal = managerStatus.Jobs.Total
	status.JobEnabled = managerStatus.Jobs.Enabled
	status.TriggerTotal = managerStatus.Triggers.Total
	status.TriggerEnabled = managerStatus.Triggers.Enabled
	status.NextFire = managerStatus.NextFire
	if !managerStatus.LastSync.SyncedAt.IsZero() {
		status.LastSyncedAt = &managerStatus.LastSync.SyncedAt
	}
	return status, nil
}

func (s *settingsRuntimeSurface) NetworkRuntimeStatus(
	ctx context.Context,
) (settingspkg.NetworkRuntimeStatus, error) {
	if s.network == nil {
		if !s.config.Network.Enabled {
			return settingspkg.NetworkRuntimeStatus{
				Available: true,
				Enabled:   false,
				Status:    network.StatusDisabled,
			}, nil
		}
		return settingspkg.NetworkRuntimeStatus{}, errors.New("daemon: settings network runtime is unavailable")
	}

	runtimeStatus, err := s.network.Status(ctx)
	if err != nil {
		return settingspkg.NetworkRuntimeStatus{}, fmt.Errorf("daemon: settings network runtime: %w", err)
	}
	if runtimeStatus == nil {
		return settingspkg.NetworkRuntimeStatus{}, errors.New("daemon: settings network status is required")
	}

	return settingspkg.NetworkRuntimeStatus{
		Available:       true,
		Enabled:         runtimeStatus.Enabled,
		Status:          strings.TrimSpace(runtimeStatus.Status),
		ListenerHost:    strings.TrimSpace(runtimeStatus.ListenerHost),
		ListenerPort:    runtimeStatus.ListenerPort,
		LocalPeers:      runtimeStatus.LocalPeers,
		RemotePeers:     runtimeStatus.RemotePeers,
		Channels:        runtimeStatus.Channels,
		QueuedMessages:  runtimeStatus.QueuedMessages,
		QueuedSessions:  runtimeStatus.QueuedSessions,
		DeliveryWorkers: runtimeStatus.DeliveryWorkers,
	}, nil
}

func (s *settingsRuntimeSurface) ObservabilityRuntimeStatus(
	ctx context.Context,
) (settingspkg.ObservabilityRuntimeStatus, error) {
	status := settingspkg.ObservabilityRuntimeStatus{Available: s.observer != nil}
	if s.observer == nil {
		return status, nil
	}

	health, err := s.observer.Health(ctx)
	if err != nil {
		return settingspkg.ObservabilityRuntimeStatus{}, fmt.Errorf("daemon: settings observability runtime: %w", err)
	}

	status.Status = strings.TrimSpace(health.Status)
	status.GlobalDBSizeBytes = health.GlobalDBSizeBytes
	status.SessionDBSizeBytes = health.SessionDBSizeBytes
	status.ActiveSessions = health.ActiveSessions
	status.ActiveAgents = health.ActiveAgents
	status.UptimeSeconds = health.UptimeSeconds
	return status, nil
}

func (s *settingsRuntimeSurface) InstalledExtensions(
	ctx context.Context,
) ([]settingspkg.InstalledExtension, error) {
	if s.extensions == nil {
		return nil, nil
	}

	items, err := s.extensions.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("daemon: list installed extensions for settings: %w", err)
	}

	installed := make([]settingspkg.InstalledExtension, 0, len(items))
	for _, item := range items {
		installed = append(installed, settingspkg.InstalledExtension{
			Name:          strings.TrimSpace(item.Name),
			Version:       strings.TrimSpace(item.Version),
			Enabled:       item.Enabled,
			State:         strings.TrimSpace(item.State),
			Health:        strings.TrimSpace(item.Health),
			HealthMessage: strings.TrimSpace(item.HealthMessage),
			LastError:     strings.TrimSpace(item.LastError),
			RequiresEnv:   append([]string(nil), item.RequiresEnv...),
			MissingEnv:    append([]string(nil), item.MissingEnv...),
		})
	}
	return installed, nil
}

func (s *settingsRuntimeSurface) TransportParityStatus(
	context.Context,
) (settingspkg.TransportParityStatus, error) {
	httpMutationsAllowed := settingsHTTPMutationsAllowed(s.config.HTTP.Host)
	return settingspkg.TransportParityStatus{
		Known:          true,
		SettingsHTTP:   httpMutationsAllowed,
		SettingsUDS:    true,
		ExtensionsHTTP: httpMutationsAllowed,
		ExtensionsUDS:  true,
	}, nil
}

func (s *settingsRuntimeSurface) MCPAuthStatus(
	ctx context.Context,
	server aghconfig.MCPServer,
) (mcpauth.Status, error) {
	cfg, err := mcpauth.ServerConfigFromMCP(ctx, server, s.secretResolver)
	if err != nil {
		return mcpauth.Status{}, fmt.Errorf("daemon: resolve MCP auth config: %w", err)
	}
	if s.mcpAuthStore == nil {
		return mcpauth.Status{
			ServerName: strings.TrimSpace(cfg.ServerName),
			Status:     mcpauth.StatusNeedsLogin,
			RemoteURL:  strings.TrimSpace(cfg.RemoteURL),
			AuthType:   strings.TrimSpace(cfg.Type),
			ClientID:   strings.TrimSpace(cfg.ClientID),
			Scopes:     append([]string(nil), cfg.Scopes...),
			Diagnostic: "token store unavailable",
		}, nil
	}
	service, err := mcpauth.NewService(s.mcpAuthStore)
	if err != nil {
		return mcpauth.Status{}, err
	}
	status, err := service.Status(ctx, cfg)
	if err != nil {
		return mcpauth.Status{}, fmt.Errorf("daemon: load MCP auth status: %w", err)
	}
	return status, nil
}

func settingsHTTPMutationsAllowed(host string) bool {
	normalized := strings.Trim(strings.TrimSpace(host), "[]")
	if strings.EqualFold(normalized, "localhost") {
		return true
	}
	ip := net.ParseIP(normalized)
	return ip != nil && ip.IsLoopback()
}

func (s *settingsRuntimeSurface) currentInfo() Info {
	if s == nil || s.info == nil {
		return Info{}
	}
	return s.info()
}

type settingsRestartController struct {
	daemon *Daemon
}

var _ core.SettingsRestartController = settingsRestartController{}

func (c settingsRestartController) RequestRestart(ctx context.Context) (core.SettingsRestartOperation, error) {
	if c.daemon == nil {
		return core.SettingsRestartOperation{}, errors.New("daemon: settings restart controller is required")
	}

	operation, err := c.daemon.RequestRestart(ctx)
	if err != nil {
		return core.SettingsRestartOperation{}, err
	}
	return settingsRestartOperationFromDaemon(operation), nil
}

func (c settingsRestartController) GetRestartOperation(
	ctx context.Context,
	operationID string,
) (core.SettingsRestartOperation, error) {
	if c.daemon == nil {
		return core.SettingsRestartOperation{}, errors.New("daemon: settings restart controller is required")
	}

	operation, err := c.daemon.GetRestartOperation(ctx, operationID)
	if err != nil {
		if errors.Is(err, ErrRestartOperationNotFound) {
			return core.SettingsRestartOperation{}, core.NewSettingsNotFoundError(err)
		}
		return core.SettingsRestartOperation{}, err
	}
	return settingsRestartOperationFromDaemon(operation), nil
}

func settingsRestartOperationFromDaemon(operation RestartOperation) core.SettingsRestartOperation {
	return core.SettingsRestartOperation{
		OperationID:        strings.TrimSpace(operation.OperationID),
		Status:             strings.TrimSpace(string(operation.Status)),
		OldPID:             operation.OldPID,
		OldStartedAt:       operation.OldStartedAt,
		OldSocketPath:      strings.TrimSpace(operation.OldSocketPath),
		NewPID:             operation.NewPID,
		ActiveSessionCount: operation.ActiveSessionCount,
		FailureReason:      strings.TrimSpace(operation.FailureReason),
		StartedAt:          operation.StartedAt,
		UpdatedAt:          operation.UpdatedAt,
		CompletedAt:        operation.CompletedAt,
	}
}

type settingsUpdateController struct {
	manager settingsUpdateManager
}

var _ core.SettingsUpdateController = settingsUpdateController{}

type settingsUpdateManager interface {
	Check(context.Context, aghupdate.CheckOptions) (aghupdate.State, *aghupdate.Release, error)
}

func (c settingsUpdateController) GetUpdate(ctx context.Context) (core.SettingsUpdateStatus, error) {
	if c.manager == nil {
		return core.SettingsUpdateStatus{}, errors.New("daemon: settings update manager is required")
	}

	state, _, err := c.manager.Check(ctx, aghupdate.CheckOptions{AllowCachedOnFailure: true})
	if err != nil && strings.TrimSpace(state.Message) == "" && strings.TrimSpace(state.LastError) == "" {
		return core.SettingsUpdateStatus{}, err
	}

	return core.SettingsUpdateStatus{
		Supported:      state.Supported,
		Managed:        state.Managed,
		InstallMethod:  strings.TrimSpace(state.InstallMethod),
		CurrentVersion: strings.TrimSpace(state.CurrentVersion),
		LatestVersion:  strings.TrimSpace(state.LatestVersion),
		Available:      state.Available,
		Status:         strings.TrimSpace(string(state.Status)),
		Recommendation: strings.TrimSpace(state.Recommendation),
		ReleaseURL:     strings.TrimSpace(state.ReleaseURL),
		CheckedAt:      state.CheckedAt,
		LastError:      strings.TrimSpace(state.LastError),
	}, nil
}

func newSettingsUpdateManager(d *Daemon) (*aghupdate.Manager, error) {
	if d == nil {
		return nil, errors.New("daemon: settings update daemon is required")
	}

	return aghupdate.NewManager(aghupdate.Config{
		HomePaths:      d.homePaths,
		CurrentVersion: version.Current().Version,
		ExecutablePath: d.executable,
		Getenv:         os.Getenv,
	})
}
