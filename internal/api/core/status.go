package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/diagnostics"
	"github.com/pedronauck/agh/internal/doctor"
	observepkg "github.com/pedronauck/agh/internal/observe"
	authproviders "github.com/pedronauck/agh/internal/providers"
	"github.com/pedronauck/agh/internal/session"
	settingspkg "github.com/pedronauck/agh/internal/settings"
	skillspkg "github.com/pedronauck/agh/internal/skills"
)

const (
	statusApplyStateCurrent = "current"
	statusStateAvailable    = "available"
	statusStateConfigured   = "configured"
	statusStateOK           = "ok"
	statusStateRunning      = "running"
	statusStateWarn         = "warn"
	statusStateError        = "error"
)

// GetStatus returns the hard-cut runtime status payload shared by HTTP and UDS.
func (h *BaseHandlers) GetStatus(c *gin.Context) {
	payload, err := h.statusPayload(
		c.Request.Context(),
		firstNonEmptyString(c.Query("workspace_id"), c.Query("workspace")),
	)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, payload)
}

// GetDoctor returns the hard-cut diagnostic probe payload shared by HTTP and UDS.
func (h *BaseHandlers) GetDoctor(c *gin.Context) {
	opts := doctor.RunOptions{
		Only:    splitStatusFilter(c.QueryArray("only"), c.Query("only")),
		Exclude: splitStatusFilter(c.QueryArray("exclude"), c.Query("exclude")),
		Quiet:   strings.EqualFold(strings.TrimSpace(c.Query("quiet")), "true"),
		Env: doctor.ProbeEnv{
			Now: h.nowUTC,
		},
	}
	payload, err := h.doctorPayload(c.Request.Context(), opts)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, payload)
}

func (h *BaseHandlers) statusPayload(
	ctx context.Context,
	memoryWorkspace string,
) (contract.StatusPayload, error) {
	if ctx == nil {
		return contract.StatusPayload{}, errors.New("api: status context is required")
	}
	if h.Observer == nil {
		return contract.StatusPayload{}, errors.New("api: observer is required for status")
	}
	if h.Sessions == nil {
		return contract.StatusPayload{}, errors.New("api: session manager is required for status")
	}

	health, err := h.Observer.Health(ctx)
	if err != nil {
		return contract.StatusPayload{}, fmt.Errorf("api: collect observer health: %w", err)
	}
	sessionSummary, err := h.sessionAggregate(ctx)
	if err != nil {
		return contract.StatusPayload{}, err
	}
	memoryHealth, err := h.memoryHealthSnapshot(ctx, memoryWorkspace)
	if err != nil {
		return contract.StatusPayload{}, fmt.Errorf("api: collect memory health: %w", err)
	}
	automationHealth, err := h.automationHealth(ctx)
	if err != nil {
		return contract.StatusPayload{}, fmt.Errorf("api: collect automation health: %w", err)
	}
	networkStatus, err := h.runtimeNetworkStatusPayload(ctx)
	if err != nil {
		return contract.StatusPayload{}, fmt.Errorf("api: collect network status: %w", err)
	}
	providers, err := h.providerStatusPayloads(ctx)
	if err != nil {
		return contract.StatusPayload{}, fmt.Errorf("api: collect provider status: %w", err)
	}
	mcpServers, err := h.mcpServerStatusPayloads(ctx)
	if err != nil {
		return contract.StatusPayload{}, fmt.Errorf("api: collect MCP server status: %w", err)
	}

	return contract.StatusPayload{
		SchemaVersion: contract.StatusSchemaVersion,
		GeneratedAt:   h.nowUTC(),
		Daemon:        h.daemonStatusPayload(&health, sessionSummary.Total, networkStatus),
		Sessions:      sessionSummary,
		Health:        ObserveHealthPayloadFromHealth(&health),
		Memory:        memoryHealth,
		Automation:    automationHealth,
		Tasks:         TaskHealthPayloadFromObserve(health.Tasks),
		Bridges:       BridgeAggregateHealthPayloadFromObserve(health.Bridges),
		Providers:     providers,
		MCPServers:    mcpServers,
		Skills:        h.skillRuntimeStatusPayload(),
		Config:        h.configRuntimeStatusPayload(),
		LogTail:       h.logTailStatusPayload(ctx),
	}, nil
}

func (h *BaseHandlers) doctorPayload(ctx context.Context, opts doctor.RunOptions) (contract.DoctorPayload, error) {
	start := h.nowUTC()
	registry := doctor.NewRegistry()
	if err := registry.Register(&doctor.ProbeFunc{
		ProbeID:       "runtime.status",
		ProbeCategory: contract.CategoryDaemon,
		RunFunc: func(ctx context.Context, _ *doctor.ProbeEnv) ([]contract.DiagnosticItem, error) {
			status, err := h.statusPayload(ctx, "")
			if err != nil {
				return nil, err
			}
			return diagnosticItemsFromStatus(&status, false), nil
		},
	}); err != nil {
		return contract.DoctorPayload{}, err
	}
	if err := registry.Register(&doctor.ProbeFunc{
		ProbeID:       "runtime.providers",
		ProbeCategory: contract.CategoryProvider,
		RunFunc: func(ctx context.Context, _ *doctor.ProbeEnv) ([]contract.DiagnosticItem, error) {
			providers, err := h.providerStatusPayloads(ctx)
			if err != nil {
				return nil, err
			}
			return providerDiagnosticItems(providers), nil
		},
	}); err != nil {
		return contract.DoctorPayload{}, err
	}
	runner, err := doctor.NewRunner(registry)
	if err != nil {
		return contract.DoctorPayload{}, err
	}
	items, err := runner.Run(ctx, opts)
	if err != nil {
		return contract.DoctorPayload{}, err
	}
	generatedAt := h.nowUTC()
	return contract.DoctorPayload{
		SchemaVersion: contract.StatusSchemaVersion,
		GeneratedAt:   generatedAt,
		DurationMS:    max(generatedAt.Sub(start).Milliseconds(), 0),
		Status:        diagnosticStatus(items),
		Summary:       doctorSummary(items),
		Items:         items,
	}, nil
}

func (h *BaseHandlers) daemonStatusPayload(
	health *observepkg.Health,
	totalSessions int,
	networkStatus *contract.NetworkStatusPayload,
) contract.DaemonStatusPayload {
	activeSessions := 0
	version := ""
	if health != nil {
		activeSessions = health.ActiveSessions
		version = health.Version
	}
	httpPort := h.HTTPPortValue()
	if httpPort <= 0 {
		httpPort = h.Config.HTTP.Port
	}
	return contract.DaemonStatusPayload{
		Status:         statusStateRunning,
		PID:            h.PID(),
		StartedAt:      h.StartedAt,
		Socket:         h.Config.Daemon.Socket,
		HTTPHost:       h.Config.HTTP.Host,
		HTTPPort:       httpPort,
		UserHomeDir:    h.daemonUserHomeDir(),
		ActiveSessions: activeSessions,
		TotalSessions:  totalSessions,
		Version:        version,
		Network:        networkStatus,
	}
}

func (h *BaseHandlers) runtimeNetworkStatusPayload(ctx context.Context) (*contract.NetworkStatusPayload, error) {
	if !h.Config.Network.Enabled {
		return h.networkStatusPayload(ctx)
	}
	if h.Network == nil {
		return &contract.NetworkStatusPayload{
			Enabled: true,
			Status:  memoryHealthStatusUnavailable,
		}, nil
	}
	payload, err := h.networkStatusPayload(ctx)
	if err != nil {
		return &contract.NetworkStatusPayload{
			Enabled: true,
			Status:  memoryHealthStatusUnavailable,
		}, nil
	}
	return payload, nil
}

func (h *BaseHandlers) sessionAggregate(ctx context.Context) (contract.SessionAggregatePayload, error) {
	infos, err := h.Sessions.ListAll(ctx)
	if err != nil {
		return contract.SessionAggregatePayload{}, fmt.Errorf("api: list sessions for status: %w", err)
	}
	byStatus := make(map[string]int)
	byBadge := make(map[string]int)
	active := 0
	for _, info := range infos {
		if info == nil {
			continue
		}
		state := strings.TrimSpace(string(info.State))
		if state == "" {
			state = "unknown"
		}
		byStatus[state]++
		badge := string(session.BadgeForInfo(info))
		if badge == "" {
			badge = string(session.BadgeUnknown)
		}
		byBadge[badge]++
		if info.State == session.StateActive {
			active++
		}
	}
	return contract.SessionAggregatePayload{
		Active:   active,
		Total:    len(infos),
		ByStatus: byStatus,
		ByBadge:  byBadge,
	}, nil
}

func (h *BaseHandlers) providerStatusPayloads(ctx context.Context) ([]contract.ProviderStatusPayload, error) {
	response, err := h.providerListResponse(ctx)
	if err != nil {
		return nil, err
	}
	payloads := make([]contract.ProviderStatusPayload, 0, len(response.Providers))
	for _, provider := range response.Providers {
		status := provider.AuthStatus
		var lastProbeAt *time.Time
		if status.LastProbeAt != nil && !status.LastProbeAt.IsZero() {
			timestamp := status.LastProbeAt.UTC()
			lastProbeAt = &timestamp
		}
		payloads = append(payloads, contract.ProviderStatusPayload{
			Name:          strings.TrimSpace(provider.Name),
			DisplayName:   strings.TrimSpace(provider.DisplayName),
			Default:       provider.Default,
			Mode:          strings.TrimSpace(status.Mode),
			EnvPolicy:     strings.TrimSpace(status.EnvPolicy),
			HomePolicy:    strings.TrimSpace(status.HomePolicy),
			State:         strings.TrimSpace(status.State),
			Code:          strings.TrimSpace(status.Code),
			Message:       diagnostics.RedactAndBound(status.Message, maxDiagnosticPayloadBytes),
			StatusCommand: diagnostics.RedactAndBound(status.StatusCmd, maxDiagnosticPayloadBytes),
			LoginCommand:  diagnostics.RedactAndBound(status.LoginCmd, maxDiagnosticPayloadBytes),
			LastProbeAt:   lastProbeAt,
			SuggestedCommand: diagnostics.RedactAndBound(
				providerSuggestedCommand(provider.Name, status),
				maxDiagnosticPayloadBytes,
			),
		})
	}
	return payloads, nil
}

func (h *BaseHandlers) mcpServerStatusPayloads(ctx context.Context) ([]contract.MCPServerStatusPayload, error) {
	if h.Settings == nil {
		return nil, nil
	}
	envelope, err := h.Settings.ListCollection(ctx, settingspkg.CollectionRequest{
		Collection: settingspkg.CollectionMCPServers,
		Scope:      settingspkg.ScopeGlobal,
	})
	if err != nil {
		return nil, err
	}
	payloads := make([]contract.MCPServerStatusPayload, 0, len(envelope.MCPServers))
	for _, server := range envelope.MCPServers {
		payloads = append(payloads, mcpServerStatusPayload(server))
	}
	return payloads, nil
}

func mcpServerStatusPayload(server settingspkg.MCPServerItem) contract.MCPServerStatusPayload {
	payload := contract.MCPServerStatusPayload{
		Name:          strings.TrimSpace(server.Name),
		Scope:         strings.TrimSpace(string(server.Scope)),
		WorkspaceID:   strings.TrimSpace(server.WorkspaceID),
		Transport:     strings.TrimSpace(string(server.Transport)),
		RuntimeStatus: statusStateConfigured,
	}
	if server.AuthStatus != nil {
		payload.AuthStatus = strings.TrimSpace(string(server.AuthStatus.Status))
	}
	if server.RuntimeStatus == nil {
		return payload
	}
	runtimeStatus := *server.RuntimeStatus
	payload.Configured = runtimeStatus.Configured
	payload.Initialized = runtimeStatus.Initialized
	payload.State = strings.TrimSpace(string(runtimeStatus.State))
	payload.Probe = strings.TrimSpace(string(runtimeStatus.Probe))
	payload.ToolCount = runtimeStatus.ToolCount
	payload.Reason = diagnostics.RedactAndBound(runtimeStatus.Reason, maxDiagnosticPayloadBytes)
	payload.Diagnostic = diagnostics.RedactAndBound(runtimeStatus.Diagnostic, maxDiagnosticPayloadBytes)
	payload.RuntimeStatus = mcpRuntimeStatus(runtimeStatus.State)
	return payload
}

func mcpRuntimeStatus(state settingspkg.MCPServerRuntimeState) string {
	switch state {
	case settingspkg.MCPServerRuntimeStateReady:
		return statusStateRunning
	case settingspkg.MCPServerRuntimeStateAuthRequired,
		settingspkg.MCPServerRuntimeStateAuthExpired,
		settingspkg.MCPServerRuntimeStateAuthInvalid,
		settingspkg.MCPServerRuntimeStateAuthRefreshFailed:
		return "auth_required"
	case settingspkg.MCPServerRuntimeStateConfigError,
		settingspkg.MCPServerRuntimeStatePermissionDenied,
		settingspkg.MCPServerRuntimeStateRuntimeUnavailable:
		return memoryHealthStatusUnavailable
	default:
		return statusStateConfigured
	}
}

func (h *BaseHandlers) skillRuntimeStatusPayload() contract.SkillRuntimeStatusPayload {
	if h.SkillsRegistry == nil {
		return contract.SkillRuntimeStatusPayload{RuntimeAvailable: false}
	}
	skills := h.SkillsRegistry.List()
	payload := contract.SkillRuntimeStatusPayload{
		RuntimeAvailable: true,
		DiscoveredCount:  len(skills),
	}
	for _, skill := range skills {
		if skill == nil {
			continue
		}
		if !skill.Enabled {
			payload.DisabledCount++
		}
		payload.Diagnostics = append(
			payload.Diagnostics,
			SkillDiagnosticPayloadsFromDiagnostics(skillspkg.DiagnosticsForSkill(skill))...,
		)
	}
	return payload
}

func (h *BaseHandlers) configRuntimeStatusPayload() contract.ConfigRuntimeStatusPayload {
	cfg := h.Config
	payload := contract.ConfigRuntimeStatusPayload{
		Status:          statusStateOK,
		Validated:       true,
		HomeDir:         strings.TrimSpace(h.HomePaths.HomeDir),
		ConfigFile:      strings.TrimSpace(h.HomePaths.ConfigFile),
		RestartRequired: false,
		ApplyState:      statusApplyStateCurrent,
	}
	if err := cfg.Validate(); err != nil {
		payload.Status = statusStateError
		payload.Validated = false
		payload.ValidationError = diagnostics.RedactAndBound(err.Error(), maxDiagnosticPayloadBytes)
	}
	return payload
}

func (h *BaseHandlers) logTailStatusPayload(ctx context.Context) contract.LogTailStatusPayload {
	if h.Settings == nil {
		return contract.LogTailStatusPayload{
			Available: false,
			Status:    memoryHealthStatusUnavailable,
		}
	}
	envelope, err := h.Settings.GetSection(ctx, settingspkg.SectionRequest{
		Section: settingspkg.SectionObservability,
		Scope:   settingspkg.ScopeGlobal,
	})
	if err != nil || envelope.Observability == nil {
		return contract.LogTailStatusPayload{
			Available: false,
			Status:    memoryHealthStatusUnavailable,
		}
	}
	if envelope.Observability.LogTailSupport.Available {
		return contract.LogTailStatusPayload{
			Available: true,
			Status:    statusStateAvailable,
		}
	}
	return contract.LogTailStatusPayload{
		Available: false,
		Status:    memoryHealthStatusDisabled,
	}
}

func splitStatusFilter(values []string, raw string) []string {
	values = append(values, raw)
	out := make([]string, 0, len(values))
	for _, value := range values {
		for part := range strings.SplitSeq(value, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	return out
}

func diagnosticItemsFromStatus(status *contract.StatusPayload, includeProviders bool) []contract.DiagnosticItem {
	if status == nil {
		return nil
	}
	items := []contract.DiagnosticItem{
		daemonDiagnosticItem(status),
		configDiagnosticItem(status.Config),
		automationDiagnosticItem(status.Automation),
		bridgeDiagnosticItem(status.Bridges),
		networkDiagnosticItem(status.Daemon.Network),
		skillDiagnosticItem(status.Skills),
		logTailDiagnosticItem(status.LogTail),
		taskDiagnosticItem(status.Tasks),
	}
	if includeProviders {
		items = append(items, providerDiagnosticItems(status.Providers)...)
	}
	for _, server := range status.MCPServers {
		items = append(items, mcpServerDiagnosticItem(server))
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items
}

func providerDiagnosticItems(providers []contract.ProviderStatusPayload) []contract.DiagnosticItem {
	items := make([]contract.DiagnosticItem, 0, len(providers))
	for _, provider := range providers {
		items = append(items, providerDiagnosticItem(provider))
	}
	return items
}

func providerSuggestedCommand(providerName string, status contract.ProviderAuthStatusPayload) string {
	classification := authproviders.Classification{
		State:   authproviders.ProviderAuthState(strings.TrimSpace(status.State)),
		Code:    strings.TrimSpace(status.Code),
		Message: strings.TrimSpace(status.Message),
	}
	return authproviders.SuggestedCommand(providerName, classification)
}

func daemonDiagnosticItem(status *contract.StatusPayload) contract.DiagnosticItem {
	if strings.TrimSpace(status.Daemon.Status) == statusStateRunning {
		return diagnostics.NewItem(
			"doctor.daemon.status",
			contract.CodeDaemonStatusOK,
			contract.CategoryDaemon,
			"Daemon is running",
			"AGH daemon process and status transport are responding.",
			contract.SeverityOK,
			contract.FreshnessLive,
			diagnostics.WithEvidence(map[string]any{
				"pid":             status.Daemon.PID,
				"active_sessions": status.Daemon.ActiveSessions,
				"total_sessions":  status.Daemon.TotalSessions,
			}),
		)
	}
	return diagnostics.NewItem(
		"doctor.daemon.status",
		contract.CodeDaemonStateSuspect,
		contract.CategoryDaemon,
		"Daemon state is suspect",
		"AGH daemon returned a non-running status.",
		contract.SeverityWarn,
		contract.FreshnessLive,
		diagnostics.WithEvidence(map[string]any{modelCatalogStatusSegment: status.Daemon.Status}),
	)
}

func configDiagnosticItem(status contract.ConfigRuntimeStatusPayload) contract.DiagnosticItem {
	if status.Validated {
		return diagnostics.NewItem(
			"doctor.config.validate",
			contract.CodeConfigValidated,
			contract.CategoryConfig,
			"Config validates",
			"Runtime config validates against the current schema.",
			contract.SeverityOK,
			contract.FreshnessLive,
			diagnostics.WithEvidence(map[string]any{"apply_state": status.ApplyState}),
		)
	}
	message := strings.TrimSpace(status.ValidationError)
	if message == "" {
		message = "Runtime config failed validation."
	}
	return diagnostics.NewItem(
		"doctor.config.validate",
		contract.CodeConfigValidateFailed,
		contract.CategoryConfig,
		"Config validation failed",
		message,
		contract.SeverityCritical,
		contract.FreshnessLive,
	)
}

func automationDiagnosticItem(status contract.AutomationHealthPayload) contract.DiagnosticItem {
	if !status.Enabled {
		return diagnostics.NewItem(
			"doctor.scheduler.status",
			contract.CodeSchedulerPaused,
			contract.CategoryTask,
			"Automation scheduler is disabled",
			"Automation is disabled in runtime config.",
			contract.SeverityInfo,
			contract.FreshnessLive,
		)
	}
	if status.SchedulerRunning {
		return diagnostics.NewItem(
			"doctor.scheduler.status",
			contract.CodeSchedulerReady,
			contract.CategoryTask,
			"Automation scheduler is running",
			"Scheduled automation is available.",
			contract.SeverityOK,
			contract.FreshnessLive,
			diagnostics.WithEvidence(map[string]any{
				"jobs":     status.Jobs.Total,
				"triggers": status.Triggers.Total,
			}),
		)
	}
	return diagnostics.NewItem(
		"doctor.scheduler.status",
		contract.CodeSchedulerPaused,
		contract.CategoryTask,
		"Automation scheduler is not running",
		"Automation is enabled but the scheduler is not reporting a running state.",
		contract.SeverityWarn,
		contract.FreshnessLive,
	)
}

func bridgeDiagnosticItem(status contract.BridgeAggregateHealthPayload) contract.DiagnosticItem {
	if status.StatusCounts.Error > 0 || status.AuthFailuresTotal > 0 {
		return diagnostics.NewItem(
			"doctor.bridge.status",
			contract.CodeBridgeHealthUnavailable,
			contract.CategoryBridge,
			"Bridge health has failures",
			"One or more bridge instances report errors or authentication failures.",
			contract.SeverityError,
			contract.FreshnessLive,
			diagnostics.WithEvidence(bridgeDiagnosticEvidence(status)),
		)
	}
	if status.StatusCounts.Degraded > 0 || status.DeliveryBacklog > 0 || status.DeliveryFailuresTotal > 0 {
		return diagnostics.NewItem(
			"doctor.bridge.status",
			contract.CodeBridgeHealthUnavailable,
			contract.CategoryBridge,
			"Bridge health is degraded",
			"Bridge delivery is not fully healthy.",
			contract.SeverityWarn,
			contract.FreshnessLive,
			diagnostics.WithEvidence(bridgeDiagnosticEvidence(status)),
		)
	}
	return diagnostics.NewItem(
		"doctor.bridge.status",
		contract.CodeBridgeReady,
		contract.CategoryBridge,
		"Bridge health is ready",
		"Bridge registry health has no reported failures.",
		contract.SeverityOK,
		contract.FreshnessLive,
		diagnostics.WithEvidence(bridgeDiagnosticEvidence(status)),
	)
}

func bridgeDiagnosticEvidence(status contract.BridgeAggregateHealthPayload) map[string]any {
	return map[string]any{
		"instances":       status.TotalInstances,
		"routes":          status.RouteCount,
		"backlog":         status.DeliveryBacklog,
		"delivery_errors": status.DeliveryFailuresTotal,
		"auth_errors":     status.AuthFailuresTotal,
	}
}

func networkDiagnosticItem(status *contract.NetworkStatusPayload) contract.DiagnosticItem {
	if status == nil || !status.Enabled {
		return diagnostics.NewItem(
			"doctor.network.status",
			contract.CodeNetworkDisabled,
			contract.CategoryNetwork,
			"Network is disabled",
			"AGH Network is not enabled in runtime config.",
			contract.SeverityInfo,
			contract.FreshnessLive,
		)
	}
	return diagnostics.NewItem(
		"doctor.network.status",
		contract.CodeNetworkReady,
		contract.CategoryNetwork,
		"Network status is available",
		"AGH Network status is available from the daemon.",
		contract.SeverityOK,
		contract.FreshnessLive,
		diagnostics.WithEvidence(map[string]any{
			"status":   status.Status,
			"channels": status.Channels,
			"peers":    status.LocalPeers + status.RemotePeers,
		}),
	)
}

func skillDiagnosticItem(status contract.SkillRuntimeStatusPayload) contract.DiagnosticItem {
	if status.RuntimeAvailable {
		return diagnostics.NewItem(
			"doctor.skills.status",
			contract.CodeSkillRegistryReady,
			contract.CategoryExtension,
			"Skill registry is available",
			"Skill registry is loaded and can be queried.",
			contract.SeverityOK,
			contract.FreshnessLive,
			diagnostics.WithEvidence(map[string]any{
				"discovered": status.DiscoveredCount,
				"disabled":   status.DisabledCount,
			}),
		)
	}
	return diagnostics.NewItem(
		"doctor.skills.status",
		contract.CodeSkillNotFound,
		contract.CategoryExtension,
		"Skill registry is unavailable",
		"Skill registry was not configured for this daemon.",
		contract.SeverityWarn,
		contract.FreshnessLive,
	)
}

func logTailDiagnosticItem(status contract.LogTailStatusPayload) contract.DiagnosticItem {
	if status.Available {
		return diagnostics.NewItem(
			"doctor.logs.tail",
			contract.CodeDaemonStatusOK,
			contract.CategoryDaemon,
			"Log tail is available",
			"Runtime log-tail support is available.",
			contract.SeverityOK,
			contract.FreshnessLive,
		)
	}
	return diagnostics.NewItem(
		"doctor.logs.tail",
		contract.CodeDaemonStateSuspect,
		contract.CategoryDaemon,
		"Log tail is unavailable",
		"Runtime log-tail support is not currently available.",
		contract.SeverityInfo,
		contract.FreshnessLive,
		diagnostics.WithEvidence(map[string]any{"status": status.Status}),
	)
}

func taskDiagnosticItem(status contract.TaskHealthPayload) contract.DiagnosticItem {
	if len(status.StuckRuns) > 0 {
		return diagnostics.NewItem(
			"doctor.tasks.health",
			contract.CodeTaskRunStuck,
			contract.CategoryTask,
			"Task runs are stuck",
			"One or more task runs exceeded the configured health threshold.",
			contract.SeverityWarn,
			contract.FreshnessLive,
			diagnostics.WithEvidence(map[string]any{"stuck_runs": len(status.StuckRuns)}),
		)
	}
	if status.ActiveOrphanRuns > 0 {
		return diagnostics.NewItem(
			"doctor.tasks.health",
			contract.CodeTaskRunOrphan,
			contract.CategoryTask,
			"Task runs are orphaned",
			"One or more active task runs no longer have a valid owner.",
			contract.SeverityWarn,
			contract.FreshnessLive,
			diagnostics.WithEvidence(map[string]any{"active_orphan_runs": status.ActiveOrphanRuns}),
		)
	}
	return diagnostics.NewItem(
		"doctor.tasks.health",
		contract.CodeSchedulerReady,
		contract.CategoryTask,
		"Task health is ready",
		"Task queue health has no stuck or orphaned active runs.",
		contract.SeverityOK,
		contract.FreshnessLive,
		diagnostics.WithEvidence(map[string]any{"queue_depth": status.QueueDepthTotal}),
	)
}

func providerDiagnosticItem(status contract.ProviderStatusPayload) contract.DiagnosticItem {
	severity, code := providerDiagnosticSeverityAndCode(status.State)
	title := "Provider auth is ready"
	if severity != contract.SeverityOK {
		title = "Provider auth needs attention"
	}
	message := status.Message
	if strings.TrimSpace(message) == "" {
		message = fmt.Sprintf("Provider %q auth state is %q.", status.Name, status.State)
	}
	return diagnostics.NewItem(
		"doctor.provider."+status.Name,
		code,
		contract.CategoryProvider,
		title,
		message,
		severity,
		contract.FreshnessLive,
		diagnostics.WithSuggestedCommand(status.SuggestedCommand),
		diagnostics.WithEvidence(map[string]any{
			"provider": status.Name,
			"state":    status.State,
			"mode":     status.Mode,
		}),
	)
}

func providerDiagnosticSeverityAndCode(state string) (string, string) {
	switch strings.TrimSpace(state) {
	case contract.ProviderAuthStateAuthenticated, contract.ProviderAuthStateNone:
		return contract.SeverityOK, contract.CodeProviderAuthenticated
	case contract.ProviderAuthStateNeedsLogin:
		return contract.SeverityWarn, contract.CodeProviderNotAuthenticated
	case contract.ProviderAuthStateMissingCLI:
		return contract.SeverityError, contract.CodeProviderCLIMissing
	case contract.ProviderAuthStateMissingCredential:
		return contract.SeverityError, contract.CodeProviderCredentialUnresolved
	case contract.ProviderAuthStatePermissionDenied:
		return contract.SeverityError, contract.CodeProviderPermissionDenied
	case contract.ProviderAuthStateRateLimited:
		return contract.SeverityWarn, contract.CodeProviderRateLimited
	case contract.ProviderAuthStateTransient:
		return contract.SeverityWarn, contract.CodeProviderTransientFailure
	default:
		return contract.SeverityWarn, contract.CodeProviderClassificationUnknown
	}
}

func mcpServerDiagnosticItem(status contract.MCPServerStatusPayload) contract.DiagnosticItem {
	severity, code := mcpServerDiagnosticSeverityAndCode(status)
	title := "MCP server is ready"
	if severity != contract.SeverityOK {
		title = "MCP server needs attention"
	}
	message := fmt.Sprintf("MCP server %q runtime status is %q.", status.Name, status.RuntimeStatus)
	if strings.TrimSpace(status.Diagnostic) != "" {
		message = status.Diagnostic
	}
	return diagnostics.NewItem(
		"doctor.mcp."+status.Name,
		code,
		contract.CategoryMCP,
		title,
		message,
		severity,
		contract.FreshnessLive,
		diagnostics.WithEvidence(map[string]any{
			"server": status.Name,
			"state":  status.State,
			"probe":  status.Probe,
		}),
	)
}

func mcpServerDiagnosticSeverityAndCode(status contract.MCPServerStatusPayload) (string, string) {
	switch strings.TrimSpace(status.RuntimeStatus) {
	case "running":
		return contract.SeverityOK, contract.CodeMCPServerReady
	case "auth_required":
		return contract.SeverityWarn, contract.CodeMCPAuthRequired
	case "unavailable":
		return contract.SeverityError, contract.CodeMCPServerUnavailable
	default:
		return contract.SeverityInfo, contract.CodeMCPServerUnavailable
	}
}

func diagnosticStatus(items []contract.DiagnosticItem) string {
	result := statusStateOK
	for _, item := range items {
		switch item.Severity {
		case contract.SeverityCritical, contract.SeverityError:
			return statusStateError
		case contract.SeverityWarn:
			result = statusStateWarn
		case contract.SeverityInfo:
			if result == statusStateOK {
				result = contract.SeverityInfo
			}
		}
	}
	return result
}

func doctorSummary(items []contract.DiagnosticItem) contract.DoctorSummaryPayload {
	counts := make(map[string]int)
	for _, item := range items {
		counts[item.Severity]++
	}
	return contract.DoctorSummaryPayload{
		Total:            len(items),
		CountsBySeverity: counts,
	}
}
