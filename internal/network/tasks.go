package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	taskpkg "github.com/compozy/agh/internal/task"
)

const (
	tasksCapabilityDeniedKey = "capability_denied"
	tasksNetworkChannelKey   = "network_channel"
	tasksTaskIDKey           = "task_id"
)

const (
	networkTaskWriteCapability       = "task.write"
	taskIngressReasonChannelMismatch = "channel_mismatch"
	taskIngressReasonStaleChannel    = "stale_channel"

	networkTaskActionCreate  = "task.create"
	networkTaskActionUpdate  = "task.update"
	networkTaskActionCancel  = "task.cancel"
	networkTaskActionEnqueue = "task.run.enqueue"
)

var (
	// ErrTaskIngressUnavailable reports that the network runtime was not wired
	// with a task service.
	ErrTaskIngressUnavailable = errors.New("network: task ingress is not configured")
	// ErrTaskIngressPeerNotFound reports that the supplied peer is not currently
	// authenticated in the requested channel.
	ErrTaskIngressPeerNotFound = errors.New("network: task ingress peer not found")
	// ErrTaskIngressCapabilityDenied reports that the peer lacks the capability
	// needed for task ingress.
	ErrTaskIngressCapabilityDenied = errors.New("network: task ingress capability denied")
	// ErrTaskChannelMismatch reports a request whose bound or requested task
	// channel does not match the authenticated ingress channel.
	ErrTaskChannelMismatch = errors.New("network: task channel mismatch")
	// ErrTaskChannelStale reports a stored task binding that no longer validates
	// under the current channel grammar.
	ErrTaskChannelStale = errors.New("network: stale task channel")
)

// TaskService is the narrowed task-domain surface consumed by network ingress.
type TaskService interface {
	GetTask(ctx context.Context, id string, actor taskpkg.ActorContext) (*taskpkg.View, error)
	CreateTask(ctx context.Context, spec taskpkg.CreateTask, actor taskpkg.ActorContext) (*taskpkg.Task, error)
	UpdateTask(
		ctx context.Context,
		id string,
		patch taskpkg.Patch,
		actor taskpkg.ActorContext,
	) (*taskpkg.Task, error)
	CancelTask(
		ctx context.Context,
		id string,
		req taskpkg.CancelTask,
		actor taskpkg.ActorContext,
	) (*taskpkg.Task, error)
	EnqueueRun(ctx context.Context, spec taskpkg.EnqueueRun, actor taskpkg.ActorContext) (*taskpkg.Run, error)
}

// TaskIngressContext captures the trusted peer identity and delivery metadata
// that network ingress derives from the live runtime rather than the payload.
type TaskIngressContext struct {
	WorkspaceID string
	PeerID      string
	Channel     string
	RequestID   string
	Surface     Surface
	ThreadID    string
	DirectID    string
	WorkID      string
	ReplyTo     string
	TraceID     string
	CausationID string
}

// Validate reports whether the ingress context contains the mandatory peer and
// delivery identifiers.
func (c TaskIngressContext) Validate() error {
	if strings.TrimSpace(c.WorkspaceID) == "" {
		return fmt.Errorf("%w: workspace_id is required", ErrMissingField)
	}
	if err := ValidateWorkspaceID(strings.TrimSpace(c.WorkspaceID)); err != nil {
		return err
	}
	if strings.TrimSpace(c.PeerID) == "" {
		return fmt.Errorf("%w: peer id is required", ErrMissingField)
	}
	if err := ValidatePeerID(strings.TrimSpace(c.PeerID)); err != nil {
		return err
	}
	if strings.TrimSpace(c.Channel) == "" {
		return fmt.Errorf("%w: channel is required", ErrMissingField)
	}
	if err := ValidateChannel(strings.TrimSpace(c.Channel)); err != nil {
		return err
	}
	if strings.TrimSpace(c.RequestID) == "" {
		return fmt.Errorf("%w: request id is required", ErrMissingField)
	}
	if err := c.validateConversationMetadata(); err != nil {
		return err
	}
	return nil
}

func (c TaskIngressContext) validateConversationMetadata() error {
	surface := Surface(strings.TrimSpace(string(c.Surface)))
	if surface != "" {
		if err := surface.Validate(); err != nil {
			return err
		}
		ref := ConversationRef{
			WorkspaceID: strings.TrimSpace(c.WorkspaceID),
			Channel:     strings.TrimSpace(c.Channel),
			Surface:     surface,
			ThreadID:    strings.TrimSpace(c.ThreadID),
			DirectID:    strings.TrimSpace(c.DirectID),
		}
		if err := ValidateConversationRef(ref); err != nil {
			return err
		}
	} else if strings.TrimSpace(c.ThreadID) != "" || strings.TrimSpace(c.DirectID) != "" {
		return fmt.Errorf("%w: surface is required for conversation metadata", ErrMissingField)
	}
	if workID := strings.TrimSpace(c.WorkID); workID != "" {
		if err := ValidateWorkID(workID); err != nil {
			return err
		}
	}
	for field, value := range map[string]string{
		"reply_to":     c.ReplyTo,
		"trace_id":     c.TraceID,
		"causation_id": c.CausationID,
	} {
		if err := validateOptionalIdentifierField(taskIngressOptionalStringPtr(value), field); err != nil {
			return err
		}
	}
	return nil
}

func taskIngressOptionalStringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// WithManagerTaskService injects the daemon-owned task manager used for
// authenticated network task ingress.
func WithManagerTaskService(tasks TaskService) ManagerOption {
	return func(opts *managerOptions) {
		opts.tasks = tasks
	}
}

type resolvedTaskPeerContext struct {
	ingress TaskIngressContext
	peer    RemotePeerEntry
	actor   taskpkg.ActorContext
}

// CreateTaskFromPeer creates one task on behalf of an authenticated network
// peer after channel and capability validation succeed.
func (m *Manager) CreateTaskFromPeer(
	ctx context.Context,
	ingress TaskIngressContext,
	spec taskpkg.CreateTask,
) (*taskpkg.Task, error) {
	peerCtx, err := m.resolveTaskPeerContext(ctx, ingress, networkTaskActionCreate)
	if err != nil {
		return nil, err
	}
	if err := validateRequestedTaskChannel(peerCtx.ingress.Channel, spec.NetworkChannel); err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionCreate, err, map[string]any{
			tasksNetworkChannelKey: strings.TrimSpace(spec.NetworkChannel),
		})
	}

	record, err := m.tasks.CreateTask(ctx, spec, peerCtx.actor)
	if err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionCreate, err, map[string]any{
			tasksNetworkChannelKey: strings.TrimSpace(spec.NetworkChannel),
		})
	}
	m.recordTaskIngress(ctx, peerCtx.ingress, networkTaskActionCreate, AuditDirectionReceived, "", map[string]any{
		tasksTaskIDKey:         record.ID,
		tasksNetworkChannelKey: strings.TrimSpace(record.NetworkChannel),
	})
	return record, nil
}

// UpdateTaskFromPeer applies one mutable task patch through the task manager
// after enforcing channel-bound ingress rules.
func (m *Manager) UpdateTaskFromPeer(
	ctx context.Context,
	ingress TaskIngressContext,
	taskID string,
	patch taskpkg.Patch,
) (*taskpkg.Task, error) {
	peerCtx, err := m.resolveTaskPeerContext(ctx, ingress, networkTaskActionUpdate)
	if err != nil {
		return nil, err
	}
	view, err := m.tasks.GetTask(ctx, strings.TrimSpace(taskID), peerCtx.actor)
	if err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionUpdate, err, nil)
	}
	if err := enforceBoundTaskChannel(
		view.Task.ID,
		view.Task.NetworkChannel,
		peerCtx.ingress.Channel,
		&patch,
	); err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionUpdate, err, map[string]any{
			tasksTaskIDKey:         view.Task.ID,
			tasksNetworkChannelKey: strings.TrimSpace(view.Task.NetworkChannel),
		})
	}
	if patch.NetworkChannel != nil {
		if err := validateRequestedTaskChannel(peerCtx.ingress.Channel, *patch.NetworkChannel); err != nil {
			return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionUpdate, err, map[string]any{
				tasksTaskIDKey:         view.Task.ID,
				tasksNetworkChannelKey: strings.TrimSpace(*patch.NetworkChannel),
			})
		}
	}

	record, err := m.tasks.UpdateTask(ctx, strings.TrimSpace(taskID), patch, peerCtx.actor)
	if err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionUpdate, err, map[string]any{
			tasksTaskIDKey: view.Task.ID,
		})
	}
	m.recordTaskIngress(ctx, peerCtx.ingress, networkTaskActionUpdate, AuditDirectionReceived, "", map[string]any{
		tasksTaskIDKey:         record.ID,
		tasksNetworkChannelKey: strings.TrimSpace(record.NetworkChannel),
	})
	return record, nil
}

// CancelTaskFromPeer requests manager-owned task cancellation after validating
// the authenticated peer context and task channel binding.
func (m *Manager) CancelTaskFromPeer(
	ctx context.Context,
	ingress TaskIngressContext,
	taskID string,
	req taskpkg.CancelTask,
) (*taskpkg.Task, error) {
	peerCtx, err := m.resolveTaskPeerContext(ctx, ingress, networkTaskActionCancel)
	if err != nil {
		return nil, err
	}
	view, err := m.tasks.GetTask(ctx, strings.TrimSpace(taskID), peerCtx.actor)
	if err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionCancel, err, nil)
	}
	if err := enforceBoundTaskChannel(
		view.Task.ID,
		view.Task.NetworkChannel,
		peerCtx.ingress.Channel,
		nil,
	); err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionCancel, err, map[string]any{
			tasksTaskIDKey:         view.Task.ID,
			tasksNetworkChannelKey: strings.TrimSpace(view.Task.NetworkChannel),
		})
	}

	record, err := m.tasks.CancelTask(ctx, strings.TrimSpace(taskID), req, peerCtx.actor)
	if err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionCancel, err, map[string]any{
			tasksTaskIDKey: view.Task.ID,
		})
	}
	m.recordTaskIngress(ctx, peerCtx.ingress, networkTaskActionCancel, AuditDirectionReceived, "", map[string]any{
		tasksTaskIDKey: record.ID,
	})
	return record, nil
}

// EnqueueRunFromPeer enqueues one task run from an authenticated network peer
// while preserving origin-scoped idempotency inside the task manager.
func (m *Manager) EnqueueRunFromPeer(
	ctx context.Context,
	ingress TaskIngressContext,
	spec taskpkg.EnqueueRun,
) (*taskpkg.Run, error) {
	peerCtx, err := m.resolveTaskPeerContext(ctx, ingress, networkTaskActionEnqueue)
	if err != nil {
		return nil, err
	}
	view, err := m.tasks.GetTask(ctx, strings.TrimSpace(spec.TaskID), peerCtx.actor)
	if err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionEnqueue, err, nil)
	}
	if err := enforceBoundTaskChannel(
		view.Task.ID,
		view.Task.NetworkChannel,
		peerCtx.ingress.Channel,
		nil,
	); err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionEnqueue, err, map[string]any{
			tasksTaskIDKey:         view.Task.ID,
			tasksNetworkChannelKey: strings.TrimSpace(view.Task.NetworkChannel),
		})
	}
	if err := validateRequestedTaskChannel(peerCtx.ingress.Channel, spec.NetworkChannel); err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionEnqueue, err, map[string]any{
			tasksTaskIDKey:         view.Task.ID,
			tasksNetworkChannelKey: strings.TrimSpace(spec.NetworkChannel),
		})
	}
	spec, err = withNetworkRunMetadata(spec, peerCtx.ingress)
	if err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionEnqueue, err, map[string]any{
			tasksTaskIDKey: view.Task.ID,
		})
	}

	run, err := m.tasks.EnqueueRun(ctx, spec, peerCtx.actor)
	if err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionEnqueue, err, map[string]any{
			tasksTaskIDKey:    view.Task.ID,
			"idempotency_key": strings.TrimSpace(spec.IdempotencyKey),
		})
	}
	m.recordTaskIngress(ctx, peerCtx.ingress, networkTaskActionEnqueue, AuditDirectionReceived, "", map[string]any{
		tasksTaskIDKey:         run.TaskID,
		"run_id":               run.ID,
		"idempotency_key":      strings.TrimSpace(run.IdempotencyKey),
		tasksNetworkChannelKey: strings.TrimSpace(run.NetworkChannel),
	})
	return run, nil
}

func (m *Manager) resolveTaskPeerContext(
	ctx context.Context,
	ingress TaskIngressContext,
	action string,
) (resolvedTaskPeerContext, error) {
	if ctx == nil {
		return resolvedTaskPeerContext{}, errors.New("network: task ingress context is required")
	}
	if m == nil {
		return resolvedTaskPeerContext{}, errors.New("network: manager is required")
	}
	if err := ingress.Validate(); err != nil {
		return resolvedTaskPeerContext{}, err
	}
	if m.tasks == nil {
		return resolvedTaskPeerContext{}, m.rejectTaskIngress(ctx, ingress, action, ErrTaskIngressUnavailable, nil)
	}
	if m.peers == nil {
		return resolvedTaskPeerContext{}, m.rejectTaskIngress(ctx, ingress, action, ErrTaskIngressPeerNotFound, nil)
	}
	ingress.WorkspaceID = strings.TrimSpace(ingress.WorkspaceID)
	ingress.Channel = strings.TrimSpace(ingress.Channel)
	ingress.PeerID = strings.TrimSpace(ingress.PeerID)

	peer, ok := m.peers.RemoteByPeer(ingress.WorkspaceID, ingress.Channel, ingress.PeerID, m.now().UTC())
	if !ok {
		return resolvedTaskPeerContext{}, m.rejectTaskIngress(ctx, ingress, action, ErrTaskIngressPeerNotFound, nil)
	}
	if !containsString(peer.PeerCard.Capabilities, networkTaskWriteCapability) {
		return resolvedTaskPeerContext{}, m.rejectTaskIngress(ctx, ingress, action, ErrTaskIngressCapabilityDenied, nil)
	}

	actor, err := taskpkg.DeriveNetworkPeerActorContext(
		strings.TrimSpace(ingress.PeerID),
		networkTaskOriginRef(ingress),
	)
	if err != nil {
		return resolvedTaskPeerContext{}, m.rejectTaskIngress(ctx, ingress, action, err, nil)
	}
	return resolvedTaskPeerContext{
		ingress: ingress,
		peer:    peer,
		actor:   actor,
	}, nil
}

func (m *Manager) rejectTaskIngress(
	ctx context.Context,
	ingress TaskIngressContext,
	action string,
	err error,
	payload any,
) error {
	if err == nil {
		return nil
	}
	m.recordTaskIngress(ctx, ingress, action, AuditDirectionRejected, taskIngressReason(err), payload)
	return err
}

func (m *Manager) recordTaskIngress(
	ctx context.Context,
	ingress TaskIngressContext,
	action string,
	direction string,
	reason string,
	payload any,
) {
	if m == nil || m.auditor == nil {
		return
	}
	writer, ok := m.auditor.(TaskIngressAuditWriter)
	if !ok {
		return
	}
	if err := writer.RecordTaskIngress(ctx, TaskIngressAudit{
		WorkspaceID: strings.TrimSpace(ingress.WorkspaceID),
		Action:      strings.TrimSpace(action),
		Direction:   strings.TrimSpace(direction),
		PeerID:      strings.TrimSpace(ingress.PeerID),
		Channel:     strings.TrimSpace(ingress.Channel),
		RequestID:   strings.TrimSpace(ingress.RequestID),
		Reason:      strings.TrimSpace(reason),
		Payload:     payload,
	}); err != nil {
		m.logger.Warn(
			"network.audit.record_task_ingress_failed",
			"action",
			action,
			"peer_id",
			ingress.PeerID,
			"request_id",
			ingress.RequestID,
			"error",
			err,
		)
	}
}

func validateRequestedTaskChannel(ingressChannel string, requestedChannel string) error {
	trimmed := strings.TrimSpace(requestedChannel)
	if trimmed == "" {
		return nil
	}
	if err := ValidateChannel(trimmed); err != nil {
		return err
	}
	if trimmed != strings.TrimSpace(ingressChannel) {
		return fmt.Errorf(
			"%w: requested channel %q does not match ingress channel %q",
			ErrTaskChannelMismatch,
			trimmed,
			strings.TrimSpace(ingressChannel),
		)
	}
	return nil
}

func enforceBoundTaskChannel(
	taskID string,
	boundChannel string,
	ingressChannel string,
	patch *taskpkg.Patch,
) error {
	trimmedBound := strings.TrimSpace(boundChannel)
	if trimmedBound == "" {
		return nil
	}
	if err := ValidateChannel(trimmedBound); err != nil {
		if patch != nil && patchAllowsStaleChannelRepair(strings.TrimSpace(ingressChannel), *patch) {
			return nil
		}
		return fmt.Errorf(
			"%w: task %q channel %q no longer validates",
			ErrTaskChannelStale,
			strings.TrimSpace(taskID),
			trimmedBound,
		)
	}
	if trimmedBound != strings.TrimSpace(ingressChannel) {
		return fmt.Errorf(
			"%w: task %q channel %q does not match ingress channel %q",
			ErrTaskChannelMismatch,
			strings.TrimSpace(taskID),
			trimmedBound,
			strings.TrimSpace(ingressChannel),
		)
	}
	return nil
}

func patchAllowsStaleChannelRepair(ingressChannel string, patch taskpkg.Patch) bool {
	if patch.NetworkChannel == nil {
		return false
	}
	trimmed := strings.TrimSpace(*patch.NetworkChannel)
	if trimmed == "" {
		return true
	}
	return trimmed == strings.TrimSpace(ingressChannel)
}

func networkTaskOriginRef(ingress TaskIngressContext) string {
	return fmt.Sprintf(
		"workspace:%s/channel:%s/peer:%s",
		strings.TrimSpace(ingress.WorkspaceID),
		strings.TrimSpace(ingress.Channel),
		strings.TrimSpace(ingress.PeerID),
	)
}

func withNetworkRunMetadata(spec taskpkg.EnqueueRun, ingress TaskIngressContext) (taskpkg.EnqueueRun, error) {
	values, err := networkRunMetadataValues(ingress)
	if err != nil {
		return taskpkg.EnqueueRun{}, err
	}
	metadata, err := mergeTrustedNetworkMetadata(spec.Metadata, values)
	if err != nil {
		return taskpkg.EnqueueRun{}, err
	}
	spec.Metadata = metadata
	return spec, nil
}

func networkRunMetadataValues(ingress TaskIngressContext) (map[string]string, error) {
	workID := strings.TrimSpace(ingress.WorkID)
	if workID == "" {
		return nil, fmt.Errorf("%w: network work_id is required", ErrMissingField)
	}
	values := map[string]string{
		"network_workspace_id": strings.TrimSpace(ingress.WorkspaceID),
		"network_work_id":      workID,
		"network_message_id":   strings.TrimSpace(ingress.RequestID),
		tasksNetworkChannelKey: strings.TrimSpace(ingress.Channel),
		"network_surface":      strings.TrimSpace(string(ingress.Surface)),
		"network_reply_to":     strings.TrimSpace(ingress.ReplyTo),
		"network_trace_id":     strings.TrimSpace(ingress.TraceID),
		"network_causation_id": strings.TrimSpace(ingress.CausationID),
	}
	switch ingress.Surface {
	case SurfaceThread:
		values["network_thread_id"] = strings.TrimSpace(ingress.ThreadID)
	case SurfaceDirect:
		values["network_direct_id"] = strings.TrimSpace(ingress.DirectID)
	}
	return values, nil
}

func mergeTrustedNetworkMetadata(raw json.RawMessage, values map[string]string) (json.RawMessage, error) {
	metadata := make(map[string]any)
	if len(raw) > 0 && strings.TrimSpace(string(raw)) != "" {
		if err := json.Unmarshal(raw, &metadata); err != nil {
			return nil, fmt.Errorf("%w: task run metadata must be a JSON object: %w", ErrInvalidField, err)
		}
		if metadata == nil {
			metadata = make(map[string]any)
		}
	}
	for key, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if existing, ok := metadata[key]; ok {
			existingValue, ok := existing.(string)
			if !ok || strings.TrimSpace(existingValue) != value {
				return nil, fmt.Errorf("%w: %s is server-derived network metadata", ErrInvalidField, key)
			}
		}
		metadata[key] = value
	}
	if len(metadata) == 0 {
		return nil, nil
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("network: marshal task run metadata: %w", err)
	}
	return payload, nil
}

func taskIngressReason(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrTaskChannelMismatch):
		return taskIngressReasonChannelMismatch
	case errors.Is(err, ErrTaskChannelStale):
		return taskIngressReasonStaleChannel
	case errors.Is(err, ErrTaskIngressCapabilityDenied):
		return tasksCapabilityDeniedKey
	case errors.Is(err, ErrTaskIngressPeerNotFound):
		return "peer_not_found"
	case errors.Is(err, ErrTaskIngressUnavailable):
		return "task_ingress_unavailable"
	case errors.Is(err, taskpkg.ErrTaskNotFound):
		return "task_not_found"
	case errors.Is(err, taskpkg.ErrValidation):
		return "validation_failed"
	case errors.Is(err, taskpkg.ErrPermissionDenied):
		return "permission_denied"
	case errors.Is(err, taskpkg.ErrStaleNetworkChannel):
		return taskIngressReasonStaleChannel
	case errors.Is(err, ErrMissingField), errors.Is(err, ErrInvalidField):
		return "invalid_request"
	default:
		return "task_ingress_failed"
	}
}
