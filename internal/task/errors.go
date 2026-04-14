package task

import "errors"

var (
	// ErrTaskNotFound reports that no persisted task matched the lookup.
	ErrTaskNotFound = errors.New("task: task not found")
	// ErrTaskRunNotFound reports that no persisted task run matched the lookup.
	ErrTaskRunNotFound = errors.New("task: task run not found")
	// ErrTaskRunIdempotencyNotFound reports that no persisted task-run idempotency record matched the lookup.
	ErrTaskRunIdempotencyNotFound = errors.New("task: task run idempotency not found")
	// ErrTaskDependencyNotFound reports that no persisted dependency edge matched the lookup.
	ErrTaskDependencyNotFound = errors.New("task: task dependency not found")
	// ErrTaskEventNotFound reports that no persisted task event matched the lookup.
	ErrTaskEventNotFound = errors.New("task: task event not found")
	// ErrValidation reports that a task-domain payload or state failed validation.
	ErrValidation = errors.New("task: validation failed")
	// ErrImmutableField reports that a caller attempted to change an immutable task field.
	ErrImmutableField = errors.New("task: immutable field")
	// ErrInvalidScopeBinding reports that a scope and workspace binding combination is invalid.
	ErrInvalidScopeBinding = errors.New("task: invalid scope binding")
	// ErrPayloadTooLarge reports that a JSON payload exceeded the task-domain size guardrails.
	ErrPayloadTooLarge = errors.New("task: payload too large")
	// ErrGraphLimitExceeded reports that a task hierarchy or dependency operation exceeded a bounded limit.
	ErrGraphLimitExceeded = errors.New("task: graph limit exceeded")
	// ErrCycleDetected reports that a dependency insert would introduce a cycle.
	ErrCycleDetected = errors.New("task: dependency cycle detected")
	// ErrInvalidStatusTransition reports that a task or run lifecycle transition is not allowed.
	ErrInvalidStatusTransition = errors.New("task: invalid status transition")
	// ErrSessionAlreadyBound reports that a run already owns a session binding.
	ErrSessionAlreadyBound = errors.New("task: session already bound")
	// ErrSessionAttachNotAllowed reports that a run cannot attach an existing session in its current state.
	ErrSessionAttachNotAllowed = errors.New("task: session attach not allowed")
	// ErrStaleNetworkChannel reports that a stored task or run channel no longer passes the active validator.
	ErrStaleNetworkChannel = errors.New("task: stale network channel")
	// ErrPermissionDenied reports that the resolved actor context lacks authority for the requested task action.
	ErrPermissionDenied = errors.New("task: permission denied")
)
