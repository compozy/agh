package task

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Normalize returns the normalized representation of the scope.
func (s Scope) Normalize() Scope {
	return Scope(strings.ToLower(strings.TrimSpace(string(s))))
}

// Validate reports whether the scope is one of the supported task scope values.
func (s Scope) Validate(path string) error {
	switch s.Normalize() {
	case ScopeGlobal, ScopeWorkspace:
		return nil
	case "":
		return fmt.Errorf("%w: %s is required", ErrValidation, path)
	default:
		return fmt.Errorf("%w: %s must be %q or %q: %q", ErrValidation, path, ScopeGlobal, ScopeWorkspace, s)
	}
}

// Normalize returns the normalized representation of the task status.
func (s TaskStatus) Normalize() TaskStatus {
	return TaskStatus(strings.ToLower(strings.TrimSpace(string(s))))
}

// Validate reports whether the task status is one of the supported lifecycle states.
func (s TaskStatus) Validate(path string) error {
	switch s.Normalize() {
	case TaskStatusPending,
		TaskStatusBlocked,
		TaskStatusReady,
		TaskStatusInProgress,
		TaskStatusCompleted,
		TaskStatusFailed,
		TaskStatusCancelled:
		return nil
	case "":
		return fmt.Errorf("%w: %s is required", ErrValidation, path)
	default:
		return fmt.Errorf(
			"%w: %s must be one of %q, %q, %q, %q, %q, %q, or %q: %q",
			ErrValidation,
			path,
			TaskStatusPending,
			TaskStatusBlocked,
			TaskStatusReady,
			TaskStatusInProgress,
			TaskStatusCompleted,
			TaskStatusFailed,
			TaskStatusCancelled,
			s,
		)
	}
}

// Normalize returns the normalized representation of the task-run status.
func (s TaskRunStatus) Normalize() TaskRunStatus {
	return TaskRunStatus(strings.ToLower(strings.TrimSpace(string(s))))
}

// Validate reports whether the task-run status is one of the supported lifecycle states.
func (s TaskRunStatus) Validate(path string) error {
	switch s.Normalize() {
	case TaskRunStatusQueued,
		TaskRunStatusClaimed,
		TaskRunStatusStarting,
		TaskRunStatusRunning,
		TaskRunStatusCompleted,
		TaskRunStatusFailed,
		TaskRunStatusCancelled:
		return nil
	case "":
		return fmt.Errorf("%w: %s is required", ErrValidation, path)
	default:
		return fmt.Errorf(
			"%w: %s must be one of %q, %q, %q, %q, %q, %q, or %q: %q",
			ErrValidation,
			path,
			TaskRunStatusQueued,
			TaskRunStatusClaimed,
			TaskRunStatusStarting,
			TaskRunStatusRunning,
			TaskRunStatusCompleted,
			TaskRunStatusFailed,
			TaskRunStatusCancelled,
			s,
		)
	}
}

// Normalize returns the normalized representation of the actor kind.
func (k ActorKind) Normalize() ActorKind {
	return ActorKind(strings.ToLower(strings.TrimSpace(string(k))))
}

// Validate reports whether the actor kind is supported.
func (k ActorKind) Validate(path string) error {
	switch k.Normalize() {
	case ActorKindHuman,
		ActorKindAgentSession,
		ActorKindAutomation,
		ActorKindExtension,
		ActorKindNetworkPeer,
		ActorKindDaemon:
		return nil
	case "":
		return fmt.Errorf("%w: %s is required", ErrValidation, path)
	default:
		return fmt.Errorf("%w: %s has unsupported value %q", ErrValidation, path, k)
	}
}

// Normalize returns the normalized representation of the owner kind.
func (k OwnerKind) Normalize() OwnerKind {
	return OwnerKind(strings.ToLower(strings.TrimSpace(string(k))))
}

// Validate reports whether the owner kind is supported.
func (k OwnerKind) Validate(path string) error {
	switch k.Normalize() {
	case OwnerKindHuman,
		OwnerKindAgentSession,
		OwnerKindAutomation,
		OwnerKindExtension,
		OwnerKindNetworkPeer,
		OwnerKindPool:
		return nil
	case "":
		return fmt.Errorf("%w: %s is required", ErrValidation, path)
	default:
		return fmt.Errorf("%w: %s has unsupported value %q", ErrValidation, path, k)
	}
}

// Normalize returns the normalized representation of the origin kind.
func (k OriginKind) Normalize() OriginKind {
	return OriginKind(strings.ToLower(strings.TrimSpace(string(k))))
}

// Validate reports whether the origin kind is supported.
func (k OriginKind) Validate(path string) error {
	switch k.Normalize() {
	case OriginKindCLI,
		OriginKindWeb,
		OriginKindUDS,
		OriginKindHTTP,
		OriginKindAutomation,
		OriginKindExtension,
		OriginKindNetwork,
		OriginKindAgentSession,
		OriginKindDaemon:
		return nil
	case "":
		return fmt.Errorf("%w: %s is required", ErrValidation, path)
	default:
		return fmt.Errorf("%w: %s has unsupported value %q", ErrValidation, path, k)
	}
}

// Normalize returns the normalized representation of the dependency kind.
func (k DependencyKind) Normalize() DependencyKind {
	return DependencyKind(strings.ToLower(strings.TrimSpace(string(k))))
}

// Validate reports whether the dependency kind is supported.
func (k DependencyKind) Validate(path string) error {
	switch k.Normalize() {
	case DependencyKindBlocks:
		return nil
	case "":
		return fmt.Errorf("%w: %s is required", ErrValidation, path)
	default:
		return fmt.Errorf("%w: %s has unsupported value %q", ErrValidation, path, k)
	}
}

// Normalize returns the normalized representation of the session stop reason.
func (r StopReason) Normalize() StopReason {
	return StopReason(strings.ToLower(strings.TrimSpace(string(r))))
}

// Validate reports whether the stop reason is supported.
func (r StopReason) Validate(path string) error {
	switch r.Normalize() {
	case StopReasonCancellation, StopReasonShutdown, StopReasonOrphanedRun:
		return nil
	case "":
		return fmt.Errorf("%w: %s is required", ErrValidation, path)
	default:
		return fmt.Errorf("%w: %s has unsupported value %q", ErrValidation, path, r)
	}
}

// Validate reports whether the actor identity contains a supported kind and non-empty reference.
func (a ActorIdentity) Validate(path string) error {
	if err := a.Kind.Validate(nestedPath(path, "kind")); err != nil {
		return err
	}
	if strings.TrimSpace(a.Ref) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "ref"))
	}
	return nil
}

// Validate reports whether the ownership value contains a supported kind and non-empty reference.
func (o Ownership) Validate(path string) error {
	if err := o.Kind.Validate(nestedPath(path, "kind")); err != nil {
		return err
	}
	if strings.TrimSpace(o.Ref) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "ref"))
	}
	return nil
}

// IsZero reports whether the ownership value is empty.
func (o Ownership) IsZero() bool {
	return o.Kind.Normalize() == "" && strings.TrimSpace(o.Ref) == ""
}

// Validate reports whether the origin contains a supported kind and non-empty reference.
func (o Origin) Validate(path string) error {
	if err := o.Kind.Validate(nestedPath(path, "kind")); err != nil {
		return err
	}
	if strings.TrimSpace(o.Ref) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "ref"))
	}
	return nil
}

// Validate reports whether the authority flags are internally consistent.
func (a Authority) Validate(path string) error {
	if !a.Write && (a.CreateGlobal || a.CreateWorkspace) {
		return fmt.Errorf("%w: %s create permissions require write permission", ErrValidation, path)
	}
	return nil
}

// Validate reports whether the actor context contains a valid principal, origin, and authority envelope.
func (a ActorContext) Validate() error {
	if err := a.Actor.Validate("actor"); err != nil {
		return err
	}
	if err := a.Origin.Validate("origin"); err != nil {
		return err
	}
	if err := a.Authority.Validate("authority"); err != nil {
		return err
	}
	return nil
}

// Validate reports whether the task record contains the canonical persisted shape.
func (t Task) Validate() error {
	if strings.TrimSpace(t.ID) == "" {
		return fmt.Errorf("%w: task.id is required", ErrValidation)
	}
	if err := ValidateScopeBinding(t.Scope, t.WorkspaceID, "task", "workspace_id"); err != nil {
		return err
	}
	if strings.TrimSpace(t.ParentTaskID) != "" && strings.TrimSpace(t.ParentTaskID) == strings.TrimSpace(t.ID) {
		return fmt.Errorf("%w: task.parent_task_id cannot equal task.id", ErrValidation)
	}
	if strings.TrimSpace(t.Title) == "" {
		return fmt.Errorf("%w: task.title is required", ErrValidation)
	}
	if err := t.Status.Validate("task.status"); err != nil {
		return err
	}
	if err := t.CreatedBy.Validate("task.created_by"); err != nil {
		return err
	}
	if err := t.Origin.Validate("task.origin"); err != nil {
		return err
	}
	if t.Owner != nil {
		if err := t.Owner.Validate("task.owner"); err != nil {
			return err
		}
	}
	if err := ValidateMetadataSize(t.Metadata, "task.metadata"); err != nil {
		return err
	}
	return nil
}

// Validate reports whether the dependency edge contains the canonical persisted shape.
func (d TaskDependency) Validate() error {
	if strings.TrimSpace(d.TaskID) == "" {
		return fmt.Errorf("%w: task_dependency.task_id is required", ErrValidation)
	}
	if strings.TrimSpace(d.DependsOnTaskID) == "" {
		return fmt.Errorf("%w: task_dependency.depends_on_task_id is required", ErrValidation)
	}
	if strings.TrimSpace(d.TaskID) == strings.TrimSpace(d.DependsOnTaskID) {
		return fmt.Errorf("%w: task_dependency cannot depend on itself", ErrValidation)
	}
	if err := d.Kind.Validate("task_dependency.kind"); err != nil {
		return err
	}
	return nil
}

// Validate reports whether the task-run record contains the canonical persisted shape.
func (r TaskRun) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("%w: task_run.id is required", ErrValidation)
	}
	if strings.TrimSpace(r.TaskID) == "" {
		return fmt.Errorf("%w: task_run.task_id is required", ErrValidation)
	}
	if err := r.Status.Validate("task_run.status"); err != nil {
		return err
	}
	if r.Attempt <= 0 {
		return fmt.Errorf("%w: task_run.attempt must be positive: %d", ErrValidation, r.Attempt)
	}
	if r.ClaimedBy != nil {
		if err := r.ClaimedBy.Validate("task_run.claimed_by"); err != nil {
			return err
		}
	}
	if err := r.Origin.Validate("task_run.origin"); err != nil {
		return err
	}
	if r.Status.Normalize() == TaskRunStatusQueued && strings.TrimSpace(r.SessionID) != "" {
		return fmt.Errorf("%w: task_run.session_id must be empty while status is %q", ErrValidation, TaskRunStatusQueued)
	}
	if err := ValidateResultSize(r.Result, "task_run.result"); err != nil {
		return err
	}
	return nil
}

// Validate reports whether the audit event contains the canonical persisted shape.
func (e TaskEvent) Validate() error {
	if strings.TrimSpace(e.ID) == "" {
		return fmt.Errorf("%w: task_event.id is required", ErrValidation)
	}
	if strings.TrimSpace(e.TaskID) == "" {
		return fmt.Errorf("%w: task_event.task_id is required", ErrValidation)
	}
	if strings.TrimSpace(e.EventType) == "" {
		return fmt.Errorf("%w: task_event.event_type is required", ErrValidation)
	}
	if err := e.Actor.Validate("task_event.actor"); err != nil {
		return err
	}
	if err := e.Origin.Validate("task_event.origin"); err != nil {
		return err
	}
	if err := ValidatePayloadSize(e.Payload, "task_event.payload"); err != nil {
		return err
	}
	return nil
}

// Validate reports whether the persisted idempotency record contains the canonical shape.
func (r TaskRunIdempotency) Validate() error {
	if strings.TrimSpace(r.IdempotencyKey) == "" {
		return fmt.Errorf("%w: task_run_idempotency.idempotency_key is required", ErrValidation)
	}
	if strings.TrimSpace(r.RunID) == "" {
		return fmt.Errorf("%w: task_run_idempotency.run_id is required", ErrValidation)
	}
	if err := r.Origin.Validate("task_run_idempotency.origin"); err != nil {
		return err
	}
	return nil
}

// Validate reports whether the create-task request is internally consistent.
func (r CreateTask) Validate(path string) error {
	if err := ValidateScopeBinding(r.Scope, r.WorkspaceID, path, "workspace_id"); err != nil {
		return err
	}
	if strings.TrimSpace(r.Title) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "title"))
	}
	if strings.TrimSpace(r.ParentTaskID) != "" && strings.TrimSpace(r.ID) != "" && strings.TrimSpace(r.ParentTaskID) == strings.TrimSpace(r.ID) {
		return fmt.Errorf("%w: %s cannot equal %s", ErrValidation, nestedPath(path, "parent_task_id"), nestedPath(path, "id"))
	}
	if r.Owner != nil {
		if err := r.Owner.Validate(nestedPath(path, "owner")); err != nil {
			return err
		}
	}
	if err := ValidateMetadataSize(r.Metadata, nestedPath(path, "metadata")); err != nil {
		return err
	}
	return nil
}

// Validate reports whether the task patch contains at least one mutable field and valid values.
func (p TaskPatch) Validate(path string) error {
	if p.Title == nil && p.Description == nil && p.Metadata == nil && p.NetworkChannel == nil && p.Owner == nil && !p.ClearOwner {
		return fmt.Errorf("%w: %s requires at least one mutable field", ErrValidation, path)
	}
	if p.Title != nil && strings.TrimSpace(*p.Title) == "" {
		return fmt.Errorf("%w: %s is required when provided", ErrValidation, nestedPath(path, "title"))
	}
	if p.Owner != nil && p.ClearOwner {
		return fmt.Errorf("%w: %s.owner and %s.clear_owner cannot both be set", ErrValidation, path, path)
	}
	if p.Owner != nil {
		if err := p.Owner.Validate(nestedPath(path, "owner")); err != nil {
			return err
		}
	}
	if p.Metadata != nil {
		if err := ValidateMetadataSize(*p.Metadata, nestedPath(path, "metadata")); err != nil {
			return err
		}
	}
	return nil
}

// Validate reports whether the task-cancellation request is internally consistent.
func (r CancelTask) Validate(path string) error {
	return ValidatePayloadSize(r.Metadata, nestedPath(path, "metadata"))
}

// Validate reports whether the dependency-create request is internally consistent.
func (r AddDependency) Validate(path string) error {
	dependency := TaskDependency{
		TaskID:          r.TaskID,
		DependsOnTaskID: r.DependsOnTaskID,
		Kind:            r.Kind,
	}
	if err := dependency.Validate(); err != nil {
		return fmt.Errorf("%w: %s", err, path)
	}
	return nil
}

// Validate reports whether the enqueue-run request is internally consistent.
func (r EnqueueRun) Validate(path string) error {
	if strings.TrimSpace(r.TaskID) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "task_id"))
	}
	return nil
}

// Validate reports whether the claim-run request is internally consistent.
func (r ClaimRun) Validate(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("%w: claim_run path is required", ErrValidation)
	}
	return nil
}

// Validate reports whether the start-run request is internally consistent.
func (r StartRun) Validate(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("%w: start_run path is required", ErrValidation)
	}
	return nil
}

// Validate reports whether the cancel-run request is internally consistent.
func (r CancelRun) Validate(path string) error {
	return ValidatePayloadSize(r.Metadata, nestedPath(path, "metadata"))
}

// Validate reports whether the run result respects the shared result-size guardrail.
func (r RunResult) Validate(path string) error {
	return ValidateResultSize(r.Value, nestedPath(path, "value"))
}

// Validate reports whether the run failure contains a message and bounded metadata.
func (r RunFailure) Validate(path string) error {
	if strings.TrimSpace(r.Error) == "" {
		return fmt.Errorf("%w: %s is required", ErrValidation, nestedPath(path, "error"))
	}
	return ValidatePayloadSize(r.Metadata, nestedPath(path, "metadata"))
}

// Validate reports whether the task-query filters are internally consistent.
func (q TaskQuery) Validate(path string) error {
	if q.Scope.Normalize() != "" {
		if err := ValidateScopeBinding(q.Scope, q.WorkspaceID, path, "workspace_id"); err != nil {
			return err
		}
	}
	if q.Status.Normalize() != "" {
		if err := q.Status.Validate(nestedPath(path, "status")); err != nil {
			return err
		}
	}
	if q.OwnerKind.Normalize() != "" {
		if err := q.OwnerKind.Validate(nestedPath(path, "owner_kind")); err != nil {
			return err
		}
	}
	if q.Limit < 0 {
		return fmt.Errorf("%w: %s must be zero or positive: %d", ErrValidation, nestedPath(path, "limit"), q.Limit)
	}
	return nil
}

// Validate reports whether the task-run query filters are internally consistent.
func (q TaskRunQuery) Validate(path string) error {
	if q.Status.Normalize() != "" {
		if err := q.Status.Validate(nestedPath(path, "status")); err != nil {
			return err
		}
	}
	if q.Limit < 0 {
		return fmt.Errorf("%w: %s must be zero or positive: %d", ErrValidation, nestedPath(path, "limit"), q.Limit)
	}
	return nil
}

// Validate reports whether the task-event query filters are internally consistent.
func (q TaskEventQuery) Validate(path string) error {
	if q.Limit < 0 {
		return fmt.Errorf("%w: %s must be zero or positive: %d", ErrValidation, nestedPath(path, "limit"), q.Limit)
	}
	return nil
}

// Validate reports whether the session-start request contains the task and run context required by the bridge.
func (r StartTaskSession) Validate() error {
	if err := r.Task.Validate(); err != nil {
		return err
	}
	if err := r.Run.Validate(); err != nil {
		return err
	}
	if err := r.Actor.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.Run.TaskID) != strings.TrimSpace(r.Task.ID) {
		return fmt.Errorf("%w: start_task_session.run.task_id must match start_task_session.task.id", ErrValidation)
	}
	return nil
}

// Validate reports whether the session reference returned by the bridge is usable.
func (r SessionRef) Validate() error {
	if strings.TrimSpace(r.SessionID) == "" {
		return fmt.Errorf("%w: session_ref.session_id is required", ErrValidation)
	}
	return nil
}

// ValidateScopeBinding enforces the canonical scope/workspace invariant shared by task-domain records.
func ValidateScopeBinding(scope Scope, workspaceBinding string, path string, workspaceField string) error {
	scopePath := nestedPath(path, "scope")
	if err := scope.Validate(scopePath); err != nil {
		return err
	}

	workspacePath := nestedPath(path, workspaceField)
	switch scope.Normalize() {
	case ScopeGlobal:
		if strings.TrimSpace(workspaceBinding) != "" {
			return fmt.Errorf(
				"%w: %s must be empty when %s is %q",
				ErrInvalidScopeBinding,
				workspacePath,
				scopePath,
				ScopeGlobal,
			)
		}
	case ScopeWorkspace:
		if strings.TrimSpace(workspaceBinding) == "" {
			return fmt.Errorf(
				"%w: %s is required when %s is %q",
				ErrInvalidScopeBinding,
				workspacePath,
				scopePath,
				ScopeWorkspace,
			)
		}
	}

	return nil
}

// ValidateImmutableTaskFields reports whether an update attempted to change immutable task fields.
func ValidateImmutableTaskFields(current Task, next Task) error {
	if !sameActorIdentity(current.CreatedBy, next.CreatedBy) {
		return fmt.Errorf("%w: %s", ErrImmutableField, TaskFieldCreatedBy)
	}
	if !sameOrigin(current.Origin, next.Origin) {
		return fmt.Errorf("%w: %s", ErrImmutableField, TaskFieldOrigin)
	}
	if current.Scope.Normalize() != next.Scope.Normalize() {
		return fmt.Errorf("%w: %s", ErrImmutableField, TaskFieldScope)
	}
	if strings.TrimSpace(current.WorkspaceID) != strings.TrimSpace(next.WorkspaceID) {
		return fmt.Errorf("%w: %s", ErrImmutableField, TaskFieldWorkspaceID)
	}
	if strings.TrimSpace(current.ParentTaskID) != strings.TrimSpace(next.ParentTaskID) {
		return fmt.Errorf("%w: %s", ErrImmutableField, TaskFieldParentTaskID)
	}
	return nil
}

// ValidateMetadataSize reports whether metadata JSON respects the shared 16 KiB guardrail.
func ValidateMetadataSize(payload json.RawMessage, path string) error {
	return validateJSONSize(payload, MaxMetadataBytes, path)
}

// ValidatePayloadSize reports whether a persisted JSON payload respects the shared 64 KiB guardrail.
func ValidatePayloadSize(payload json.RawMessage, path string) error {
	return validateJSONSize(payload, MaxPayloadBytes, path)
}

// ValidateResultSize reports whether a persisted run result respects the shared 64 KiB guardrail.
func ValidateResultSize(payload json.RawMessage, path string) error {
	return validateJSONSize(payload, MaxResultBytes, path)
}

// ValidateHierarchyDepth reports whether the supplied task depth stays within the bounded hierarchy limit.
func ValidateHierarchyDepth(depth int) error {
	return validateBoundedCount(depth, MaxHierarchyDepth, "hierarchy depth")
}

// ValidateDependencyCount reports whether the supplied dependency count stays within the bounded edge limit.
func ValidateDependencyCount(count int) error {
	return validateBoundedCount(count, MaxDependencyCount, "dependency count")
}

// ValidateDirectChildCount reports whether the supplied direct-child count stays within the bounded fan-out limit.
func ValidateDirectChildCount(count int) error {
	return validateBoundedCount(count, MaxDirectChildren, "direct child count")
}

func validateJSONSize(payload json.RawMessage, maxBytes int, path string) error {
	if len(payload) == 0 {
		return nil
	}

	trimmed := bytesTrimSpace(payload)
	if !json.Valid(trimmed) {
		return fmt.Errorf("%w: %s must contain valid JSON", ErrValidation, path)
	}
	if len(trimmed) > maxBytes {
		return fmt.Errorf("%w: %s exceeds %d bytes", ErrPayloadTooLarge, path, maxBytes)
	}
	return nil
}

func validateBoundedCount(count int, maxCount int, label string) error {
	if count < 0 {
		return fmt.Errorf("%w: %s cannot be negative: %d", ErrValidation, label, count)
	}
	if count > maxCount {
		return fmt.Errorf("%w: %s exceeds %d: %d", ErrGraphLimitExceeded, label, maxCount, count)
	}
	return nil
}

func nestedPath(path string, field string) string {
	trimmedPath := strings.TrimSpace(path)
	trimmedField := strings.TrimSpace(field)
	if trimmedPath == "" {
		return trimmedField
	}
	if trimmedField == "" {
		return trimmedPath
	}
	return trimmedPath + "." + trimmedField
}

func sameActorIdentity(left ActorIdentity, right ActorIdentity) bool {
	return left.Kind.Normalize() == right.Kind.Normalize() && strings.TrimSpace(left.Ref) == strings.TrimSpace(right.Ref)
}

func sameOrigin(left Origin, right Origin) bool {
	return left.Kind.Normalize() == right.Kind.Normalize() && strings.TrimSpace(left.Ref) == strings.TrimSpace(right.Ref)
}

func bytesTrimSpace(payload []byte) []byte {
	return []byte(strings.TrimSpace(string(payload)))
}
