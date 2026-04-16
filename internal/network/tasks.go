package network

import (
	"context"
	"errors"
	"fmt"
	"strings"

	taskpkg "github.com/pedronauck/agh/internal/task"
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
	PeerID    string
	Channel   string
	RequestID string
}

// Validate reports whether the ingress context contains the mandatory peer and
// delivery identifiers.
func (c TaskIngressContext) Validate() error {
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
	return nil
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
			"network_channel": strings.TrimSpace(spec.NetworkChannel),
		})
	}

	record, err := m.tasks.CreateTask(ctx, spec, peerCtx.actor)
	if err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionCreate, err, map[string]any{
			"network_channel": strings.TrimSpace(spec.NetworkChannel),
		})
	}
	m.recordTaskIngress(ctx, peerCtx.ingress, networkTaskActionCreate, AuditDirectionReceived, "", map[string]any{
		"task_id":         record.ID,
		"network_channel": strings.TrimSpace(record.NetworkChannel),
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
			"task_id":         view.Task.ID,
			"network_channel": strings.TrimSpace(view.Task.NetworkChannel),
		})
	}
	if patch.NetworkChannel != nil {
		if err := validateRequestedTaskChannel(peerCtx.ingress.Channel, *patch.NetworkChannel); err != nil {
			return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionUpdate, err, map[string]any{
				"task_id":         view.Task.ID,
				"network_channel": strings.TrimSpace(*patch.NetworkChannel),
			})
		}
	}

	record, err := m.tasks.UpdateTask(ctx, strings.TrimSpace(taskID), patch, peerCtx.actor)
	if err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionUpdate, err, map[string]any{
			"task_id": view.Task.ID,
		})
	}
	m.recordTaskIngress(ctx, peerCtx.ingress, networkTaskActionUpdate, AuditDirectionReceived, "", map[string]any{
		"task_id":         record.ID,
		"network_channel": strings.TrimSpace(record.NetworkChannel),
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
			"task_id":         view.Task.ID,
			"network_channel": strings.TrimSpace(view.Task.NetworkChannel),
		})
	}

	record, err := m.tasks.CancelTask(ctx, strings.TrimSpace(taskID), req, peerCtx.actor)
	if err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionCancel, err, map[string]any{
			"task_id": view.Task.ID,
		})
	}
	m.recordTaskIngress(ctx, peerCtx.ingress, networkTaskActionCancel, AuditDirectionReceived, "", map[string]any{
		"task_id": record.ID,
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
			"task_id":         view.Task.ID,
			"network_channel": strings.TrimSpace(view.Task.NetworkChannel),
		})
	}
	if err := validateRequestedTaskChannel(peerCtx.ingress.Channel, spec.NetworkChannel); err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionEnqueue, err, map[string]any{
			"task_id":         view.Task.ID,
			"network_channel": strings.TrimSpace(spec.NetworkChannel),
		})
	}

	run, err := m.tasks.EnqueueRun(ctx, spec, peerCtx.actor)
	if err != nil {
		return nil, m.rejectTaskIngress(ctx, peerCtx.ingress, networkTaskActionEnqueue, err, map[string]any{
			"task_id":         view.Task.ID,
			"idempotency_key": strings.TrimSpace(spec.IdempotencyKey),
		})
	}
	m.recordTaskIngress(ctx, peerCtx.ingress, networkTaskActionEnqueue, AuditDirectionReceived, "", map[string]any{
		"task_id":         run.TaskID,
		"run_id":          run.ID,
		"idempotency_key": strings.TrimSpace(run.IdempotencyKey),
		"network_channel": strings.TrimSpace(run.NetworkChannel),
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

	peer, ok := m.peers.RemoteByPeer(ingress.Channel, ingress.PeerID, m.now().UTC())
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
		Action:    strings.TrimSpace(action),
		Direction: strings.TrimSpace(direction),
		PeerID:    strings.TrimSpace(ingress.PeerID),
		Channel:   strings.TrimSpace(ingress.Channel),
		RequestID: strings.TrimSpace(ingress.RequestID),
		Reason:    strings.TrimSpace(reason),
		Payload:   payload,
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
	return fmt.Sprintf("peer:%s/channel:%s", strings.TrimSpace(ingress.PeerID), strings.TrimSpace(ingress.Channel))
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
		return "capability_denied"
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
