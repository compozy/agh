package coordinator

import (
	"encoding/json"
	"slices"
	"strings"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	toolspkg "github.com/pedronauck/agh/internal/tools"
)

const (
	ReasonRunEnqueued        = "task_run_enqueued"
	ReasonRecovery           = "recovery"
	ReasonCoordinatorStopped = "coordinator_stopped"

	DecisionBootstrap        = "bootstrap"
	DecisionDisabled         = "disabled"
	DecisionGlobalScope      = "global_scope"
	DecisionMissingWorkspace = "missing_workspace"
	DecisionMissingChannel   = "missing_coordination_channel"
	DecisionNonExecutableRun = "non_executable_run"
	DecisionTaskRunMismatch  = "task_run_mismatch"
	DecisionExisting         = "existing_coordinator"
	DecisionDenied           = "denied"
	DecisionFailed           = "failed"
)

var (
	// operationalMessageKinds are the coordination-channel message kinds a
	// coordinator may use for worker conversation. Task ownership remains in
	// the task lease API.
	operationalMessageKinds = [...]string{
		"status",
		"request",
		"blocker",
		"handoff",
		"result",
		"review_request",
	}

	// toolAllowlist is the orchestration-safe surface granted to coordinator
	// sessions. Operator lifecycle verbs and coordinator-to-coordinator spawn
	// are intentionally absent. These must stay aligned with canonical builtin
	// ToolIDs because lineage permission policies validate concrete tool atoms.
	toolAllowlist = [...]string{
		toolspkg.ToolIDSessionDescribe.String(),
		toolspkg.ToolIDNetworkChannels.String(),
		toolspkg.ToolIDNetworkInbox.String(),
		toolspkg.ToolIDNetworkSend.String(),
		toolspkg.ToolIDTaskRunClaimNext.String(),
		toolspkg.ToolIDTaskRunHeartbeat.String(),
		toolspkg.ToolIDTaskRunComplete.String(),
		toolspkg.ToolIDTaskRunFail.String(),
		toolspkg.ToolIDTaskRunRelease.String(),
		toolspkg.ToolIDTaskCreate.String(),
	}
)

// OperationalMessageKinds returns the coordination-channel message kinds a
// coordinator may use for worker conversation.
func OperationalMessageKinds() []string {
	return slices.Clone(operationalMessageKinds[:])
}

// ToolAllowlist returns the orchestration-safe tool surface granted to
// coordinator sessions.
func ToolAllowlist() []string {
	return slices.Clone(toolAllowlist[:])
}

// Decision describes whether a task run is eligible to bootstrap a workspace
// coordinator.
type Decision struct {
	ShouldBootstrap       bool
	Reason                string
	WorkspaceID           string
	TaskID                string
	RunID                 string
	WorkflowID            string
	CoordinationChannelID string
}

// PromptInput captures the first-run situation given to a coordinator session.
type PromptInput struct {
	WorkspaceID           string
	TaskID                string
	RunID                 string
	WorkflowID            string
	CoordinationChannelID string
}

// DecideBootstrap evaluates the mechanical coordinator bootstrap rules. It
// does not check for already-running coordinator sessions; that singleton check
// belongs to the daemon runtime.
func DecideBootstrap(task taskpkg.Task, run taskpkg.Run, cfg aghconfig.CoordinatorConfig) Decision {
	decision := Decision{
		WorkspaceID:           strings.TrimSpace(task.WorkspaceID),
		TaskID:                strings.TrimSpace(task.ID),
		RunID:                 strings.TrimSpace(run.ID),
		WorkflowID:            workflowIDFromMetadata(run.Metadata),
		CoordinationChannelID: strings.TrimSpace(run.CoordinationChannelID),
	}
	if !cfg.Enabled {
		decision.Reason = DecisionDisabled
		return decision
	}
	if strings.TrimSpace(task.ID) == "" ||
		strings.TrimSpace(run.TaskID) == "" ||
		strings.TrimSpace(task.ID) != strings.TrimSpace(run.TaskID) {
		decision.Reason = DecisionTaskRunMismatch
		return decision
	}
	switch task.Scope.Normalize() {
	case taskpkg.ScopeGlobal:
		decision.Reason = DecisionGlobalScope
		return decision
	case taskpkg.ScopeWorkspace:
	default:
		decision.Reason = DecisionMissingWorkspace
		return decision
	}
	if decision.WorkspaceID == "" {
		decision.Reason = DecisionMissingWorkspace
		return decision
	}
	if !IsExecutableRunStatus(run.Status) {
		decision.Reason = DecisionNonExecutableRun
		return decision
	}
	if decision.CoordinationChannelID == "" {
		decision.Reason = DecisionMissingChannel
		return decision
	}
	decision.ShouldBootstrap = true
	decision.Reason = DecisionBootstrap
	return decision
}

// IsExecutableRunStatus reports whether a run still represents executable work
// that may need coordinator orchestration.
func IsExecutableRunStatus(status taskpkg.RunStatus) bool {
	switch status.Normalize() {
	case taskpkg.TaskRunStatusQueued,
		taskpkg.TaskRunStatusClaimed,
		taskpkg.TaskRunStatusStarting,
		taskpkg.TaskRunStatusRunning:
		return true
	default:
		return false
	}
}

// ExecutableRunStatuses returns every open run state considered by recovery.
func ExecutableRunStatuses() []taskpkg.RunStatus {
	return []taskpkg.RunStatus{
		taskpkg.TaskRunStatusQueued,
		taskpkg.TaskRunStatusClaimed,
		taskpkg.TaskRunStatusStarting,
		taskpkg.TaskRunStatusRunning,
	}
}

// PermissionPolicy returns the restricted coordinator root permission policy.
func PermissionPolicy(channelIDs ...string) store.SessionPermissionPolicy {
	policy := store.SessionPermissionPolicy{
		Tools:           ToolAllowlist(),
		NetworkChannels: nonEmptyAtoms(channelIDs...),
	}
	return store.NormalizeSessionPermissionPolicy(policy)
}

// ToolAllowed reports whether a concrete tool/action is coordinator-safe.
func ToolAllowed(tool string) bool {
	return slices.Contains(toolAllowlist[:], strings.TrimSpace(tool))
}

// SpawnRoleAllowed reports whether a coordinator may request the given child
// role through the public safe-spawn API.
func SpawnRoleAllowed(role string) bool {
	return strings.TrimSpace(strings.ToLower(role)) != string(session.SessionTypeCoordinator)
}

// Lineage builds root lineage metadata for a managed coordinator session.
func Lineage(
	now time.Time,
	cfg aghconfig.CoordinatorConfig,
	policy store.SessionPermissionPolicy,
) *store.SessionLineage {
	ttl := now.UTC().Add(cfg.DefaultTTL)
	return &store.SessionLineage{
		SpawnRole:    string(session.SessionTypeCoordinator),
		TTLExpiresAt: &ttl,
		SpawnBudget: store.SessionSpawnBudget{
			MaxChildren:           cfg.MaxChildren,
			MaxDepth:              session.DefaultSpawnMaxDepth,
			TTLSeconds:            int64(cfg.DefaultTTL.Seconds()),
			MaxActivePerWorkspace: cfg.MaxActivePerWorkspace,
		},
		PermissionPolicy: store.NormalizeSessionPermissionPolicy(policy),
	}
}

// HealthySession reports whether a session snapshot is an active coordinator
// for the workspace.
func HealthySession(info *session.Info, workspaceID string, now time.Time) bool {
	if info == nil {
		return false
	}
	if info.Type != session.SessionTypeCoordinator {
		return false
	}
	if strings.TrimSpace(info.WorkspaceID) != strings.TrimSpace(workspaceID) {
		return false
	}
	switch info.State {
	case session.StateStarting, session.StateActive:
	default:
		return false
	}
	lineage := store.NormalizeSessionLineage(info.ID, info.Lineage)
	if lineage.TTLExpiresAt != nil && !lineage.TTLExpiresAt.After(now.UTC()) {
		return false
	}
	return true
}

// PromptOverlay assembles the coordinator's first-run situation and available
// public API surface.
func PromptOverlay(input PromptInput) string {
	var b strings.Builder
	b.WriteString("You are the AGH workspace coordinator for executable task runs.\n\n")
	b.WriteString("Current run context:\n")
	writePromptLine(&b, "workspace_id", input.WorkspaceID)
	writePromptLine(&b, "task_id", input.TaskID)
	writePromptLine(&b, "run_id", input.RunID)
	writePromptLine(&b, "workflow_id", input.WorkflowID)
	writePromptLine(&b, "coordination_channel_id", input.CoordinationChannelID)
	b.WriteString("\nUse public AGH agent APIs only:\n")
	b.WriteString("- `agh me context` for the Situation Surface.\n")
	b.WriteString("- `agh task create|start` to persist task intent and enqueue active work.\n")
	b.WriteString("- `agh task next|heartbeat|complete|fail|release` for task ownership and terminal status.\n")
	b.WriteString("- `agh ch list|recv|send|reply` for operational worker communication.\n")
	b.WriteString("- `agh spawn` for bounded worker delegation.\n")
	b.WriteString("\nCreating a task only records intent. When the objective asks for active execution, ")
	b.WriteString("start the executable task so AGH enqueues a run and can route worker agents.\n")
	b.WriteString("\nChannel communication is operational only. Use the run coordination channel for ")
	b.WriteString(strings.Join(operationalMessageKinds[:], ", "))
	b.WriteString(" messages when conversation is useful. Do not use channel messages as task ownership state.\n")
	b.WriteString("Never spawn another coordinator. ")
	b.WriteString("Worker delegation must stay inside safe-spawn permissions and task approvals.\n")
	return strings.TrimSpace(b.String())
}

func writePromptLine(b *strings.Builder, key string, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	b.WriteString("- ")
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(trimmed)
	b.WriteByte('\n')
}

func nonEmptyAtoms(values ...string) []string {
	atoms := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			atoms = append(atoms, trimmed)
		}
	}
	return atoms
}

func workflowIDFromMetadata(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return ""
	}
	value, ok := metadata["workflow_id"].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
