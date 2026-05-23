package presets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	bridgepkg "github.com/compozy/agh/internal/bridges"
	eventspkg "github.com/compozy/agh/internal/events"
	"github.com/compozy/agh/internal/notifications"
	"github.com/compozy/agh/internal/store"
)

const defaultDispatchTimeout = 10 * time.Second

// Store persists notification presets.
type Store interface {
	ListPresets(ctx context.Context, query Query) ([]Preset, error)
	GetPreset(ctx context.Context, name string) (Preset, error)
	CreatePreset(ctx context.Context, preset Preset) (Preset, error)
	UpdatePreset(ctx context.Context, name string, req UpdateRequest) (Preset, error)
	DeletePreset(ctx context.Context, name string) error
	EnsureBuiltInPresets(ctx context.Context, defaults []Preset) error
}

// BridgeRuntime is the bridge surface needed for preset fanout.
type BridgeRuntime interface {
	GetBridgeInstance(ctx context.Context, id string) (bridgepkg.BridgeInstance, error)
	ResolveBridgeTarget(
		ctx context.Context,
		bridgeID string,
		query string,
	) (bridgepkg.ResolveBridgeTargetResult, error)
	DeliverBridge(
		ctx context.Context,
		extensionName string,
		req bridgepkg.DeliveryRequest,
	) (bridgepkg.DeliveryAck, error)
}

// Service owns preset CRUD and cursor-backed fanout.
type Service struct {
	store   Store
	cursors *notifications.Service
	bridges BridgeRuntime
	events  store.EventSummaryStore
	logger  *slog.Logger
	now     func() time.Time
	timeout time.Duration
}

// Config wires a preset service.
type Config struct {
	Store   Store
	Cursors notifications.CursorStore
	Bridges BridgeRuntime
	Events  store.EventSummaryStore
	Logger  *slog.Logger
	Now     func() time.Time
	Timeout time.Duration
}

func NewService(cfg Config) *Service {
	now := cfg.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultDispatchTimeout
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		store:   cfg.Store,
		cursors: notifications.NewService(cfg.Cursors),
		bridges: cfg.Bridges,
		events:  cfg.Events,
		logger:  logger,
		now:     now,
		timeout: timeout,
	}
}

func (s *Service) List(ctx context.Context, query Query) ([]Preset, error) {
	if err := s.checkStore(); err != nil {
		return nil, err
	}
	items, err := s.store.ListPresets(ctx, query.Normalize())
	if err != nil {
		return nil, fmt.Errorf("notifications: list presets: %w", err)
	}
	return items, nil
}

func (s *Service) Get(ctx context.Context, name string) (Preset, error) {
	if err := s.checkStore(); err != nil {
		return Preset{}, err
	}
	preset, err := s.store.GetPreset(ctx, normalizePresetName(name))
	if err != nil {
		return Preset{}, fmt.Errorf(
			"notifications: get preset %q: %w",
			normalizePresetName(name),
			err,
		)
	}
	return preset, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Preset, error) {
	if err := s.checkStore(); err != nil {
		return Preset{}, err
	}
	preset, err := req.Normalize(s.now())
	if err != nil {
		return Preset{}, err
	}
	created, err := s.store.CreatePreset(ctx, preset)
	if err != nil {
		return Preset{}, fmt.Errorf("notifications: create preset %q: %w", preset.Name, err)
	}
	if err := s.recordPresetLifecycleEvent(ctx, eventspkg.NotificationPresetCreated, created); err != nil {
		return Preset{}, err
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, name string, req UpdateRequest) (Preset, error) {
	if err := s.checkStore(); err != nil {
		return Preset{}, err
	}
	if !req.HasMutableField() {
		return Preset{}, fmt.Errorf(
			"%w: update requires at least one mutable field",
			ErrInvalidPreset,
		)
	}
	req = normalizeUpdateRequest(req, s.now())
	updated, err := s.store.UpdatePreset(ctx, normalizePresetName(name), req)
	if err != nil {
		return Preset{}, fmt.Errorf(
			"notifications: update preset %q: %w",
			normalizePresetName(name),
			err,
		)
	}
	if err := s.recordPresetLifecycleEvent(ctx, eventspkg.NotificationPresetUpdated, updated); err != nil {
		return Preset{}, err
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, name string) error {
	if err := s.checkStore(); err != nil {
		return err
	}
	normalizedName := normalizePresetName(name)
	if err := s.store.DeletePreset(ctx, normalizedName); err != nil {
		return fmt.Errorf("notifications: delete preset %q: %w", normalizedName, err)
	}
	if err := s.recordPresetLifecycleEvent(
		ctx,
		eventspkg.NotificationPresetDeleted,
		Preset{Name: normalizedName},
	); err != nil {
		return err
	}
	return nil
}

func (s *Service) EnsureBuiltIns(ctx context.Context) error {
	if err := s.checkStore(); err != nil {
		return err
	}
	if err := s.store.EnsureBuiltInPresets(ctx, BuiltInPresets(s.now())); err != nil {
		return fmt.Errorf("notifications: ensure built-in presets: %w", err)
	}
	return nil
}

func (s *Service) Dispatch(ctx context.Context, event Event) (DispatchResult, error) {
	if err := s.checkDispatch(); err != nil {
		return DispatchResult{}, err
	}
	normalizedEvent := event.Normalize(s.now())
	if err := normalizedEvent.Validate(); err != nil {
		return DispatchResult{}, err
	}
	meta, ok := eventspkg.Lookup(normalizedEvent.Type)
	if !ok || !meta.NotificationEligible {
		return DispatchResult{Skipped: 1}, nil
	}
	enabled := true
	presets, err := s.store.ListPresets(ctx, Query{Enabled: &enabled})
	if err != nil {
		return DispatchResult{}, fmt.Errorf(
			"notifications: list enabled presets for dispatch: %w",
			err,
		)
	}

	result := DispatchResult{}
	var joined error
	for _, preset := range presets {
		if !MatchesAny(preset.Events, normalizedEvent.Type) {
			continue
		}
		result.Matched++
		presetResult, dispatchErr := s.dispatchPreset(ctx, preset, normalizedEvent)
		result.Delivered += presetResult.Delivered
		result.Suppressed += presetResult.Suppressed
		result.Skipped += presetResult.Skipped
		result.Failed += presetResult.Failed
		if dispatchErr != nil {
			joined = errors.Join(joined, dispatchErr)
		}
	}
	return result, joined
}

func (s *Service) dispatchPreset(
	ctx context.Context,
	preset Preset,
	event Event,
) (DispatchResult, error) {
	compiled, err := CompileFilter(preset.Filter)
	if err != nil {
		key := cursorKeyForTarget(preset, Target{}, event)
		return DispatchResult{Failed: 1}, s.recordDispatchError(ctx, key, preset, event, err)
	}
	if !compiled.Eval(event) {
		return s.skipPresetTargets(ctx, preset, event, "filter")
	}
	if len(preset.Targets) == 0 {
		key := cursorKeyForTarget(preset, Target{}, event)
		_, advanceErr := s.advance(ctx, key, event, skipDeliveryID(preset, event, "no_targets"))
		return DispatchResult{Skipped: 1}, advanceErr
	}

	result := DispatchResult{}
	var joined error
	for index, target := range preset.Targets {
		cursorKey := cursorKeyForTarget(preset, target, event)
		cursor, cursorErr := s.cursors.Get(ctx, cursorKey)
		if cursorErr != nil && !errors.Is(cursorErr, notifications.ErrCursorNotFound) {
			result.Failed++
			joined = errors.Join(joined, cursorErr)
			continue
		}
		if cursor.LastSequence >= event.Sequence {
			result.Skipped++
			continue
		}
		deliveryID := deliveryIDForTarget(preset, event, index)
		err := s.deliverTarget(ctx, preset, event, target, deliveryID)
		switch {
		case err == nil:
			result.Delivered++
			if _, advanceErr := s.advance(ctx, cursorKey, event, deliveryID); advanceErr != nil {
				result.Failed++
				joined = errors.Join(joined, advanceErr)
			}
		case errors.Is(err, bridgepkg.ErrBridgeNotificationSuppressed):
			result.Suppressed++
			if _, advanceErr := s.advance(
				ctx,
				cursorKey,
				event,
				skipDeliveryID(preset, event, "suppressed"),
			); advanceErr != nil {
				result.Failed++
				joined = errors.Join(joined, advanceErr)
			}
		default:
			result.Failed++
			joined = errors.Join(joined, s.recordDispatchError(ctx, cursorKey, preset, event, err))
		}
	}
	if joined != nil {
		return result, joined
	}
	return result, nil
}

func (s *Service) skipPresetTargets(
	ctx context.Context,
	preset Preset,
	event Event,
	reason string,
) (DispatchResult, error) {
	if len(preset.Targets) == 0 {
		key := cursorKeyForTarget(preset, Target{}, event)
		_, err := s.advance(ctx, key, event, skipDeliveryID(preset, event, reason))
		return DispatchResult{Skipped: 1}, err
	}
	result := DispatchResult{}
	var joined error
	for _, target := range preset.Targets {
		key := cursorKeyForTarget(preset, target, event)
		cursor, cursorErr := s.cursors.Get(ctx, key)
		if cursorErr != nil && !errors.Is(cursorErr, notifications.ErrCursorNotFound) {
			result.Failed++
			joined = errors.Join(joined, cursorErr)
			continue
		}
		if cursor.LastSequence >= event.Sequence {
			result.Skipped++
			continue
		}
		if _, err := s.advance(ctx, key, event, skipDeliveryID(preset, event, reason)); err != nil {
			result.Failed++
			joined = errors.Join(joined, err)
			continue
		}
		result.Skipped++
	}
	return result, joined
}

func (s *Service) deliverTarget(
	ctx context.Context,
	preset Preset,
	event Event,
	target Target,
	deliveryID string,
) error {
	normalizedTarget, instance, err := s.deliverableBridge(ctx, preset, target)
	if err != nil {
		return err
	}
	delivery, err := s.deliveryForTarget(ctx, preset, event, normalizedTarget, instance, deliveryID)
	if err != nil {
		return err
	}
	return s.deliverBridgeEvent(ctx, preset, event, instance, delivery)
}

func (s *Service) deliverableBridge(
	ctx context.Context,
	preset Preset,
	target Target,
) (Target, bridgepkg.BridgeInstance, error) {
	normalizedTarget := target.Normalize()
	if err := normalizedTarget.Validate(); err != nil {
		return Target{}, bridgepkg.BridgeInstance{}, err
	}
	instance, err := s.bridges.GetBridgeInstance(ctx, normalizedTarget.BridgeID)
	if err != nil {
		return Target{}, bridgepkg.BridgeInstance{}, fmt.Errorf(
			"notifications: load bridge %q for preset %q: %w",
			normalizedTarget.BridgeID,
			preset.Name,
			err,
		)
	}
	if instance.NotificationSuppress {
		return Target{}, bridgepkg.BridgeInstance{}, fmt.Errorf(
			"%w: bridge instance %q",
			bridgepkg.ErrBridgeNotificationSuppressed,
			instance.ID,
		)
	}
	if !instance.Enabled || instance.Status.Normalize() != bridgepkg.BridgeStatusReady {
		return Target{}, bridgepkg.BridgeInstance{}, fmt.Errorf(
			"%w: bridge instance %q status %q enabled=%t",
			bridgepkg.ErrBridgeInstanceUnavailable,
			instance.ID,
			instance.Status.Normalize(),
			instance.Enabled,
		)
	}
	return normalizedTarget, instance, nil
}

func (s *Service) deliveryForTarget(
	ctx context.Context,
	preset Preset,
	event Event,
	normalizedTarget Target,
	instance bridgepkg.BridgeInstance,
	deliveryID string,
) (bridgepkg.DeliveryEvent, error) {
	resolved, err := s.resolveTarget(ctx, normalizedTarget)
	if err != nil {
		return bridgepkg.DeliveryEvent{}, err
	}
	deliveryTarget, routingKey, err := deliveryTargetFromBridgeTarget(
		instance,
		*resolved,
		normalizedTarget,
	)
	if err != nil {
		return bridgepkg.DeliveryEvent{}, err
	}
	metadata, err := json.Marshal(struct {
		Preset string `json:"preset"`
		Event  Event  `json:"event"`
		Target Target `json:"target"`
	}{
		Preset: preset.Name,
		Event:  event,
		Target: normalizedTarget,
	})
	if err != nil {
		return bridgepkg.DeliveryEvent{}, fmt.Errorf(
			"notifications: encode preset delivery metadata: %w",
			err,
		)
	}
	delivery := bridgepkg.DeliveryEvent{
		DeliveryID:       deliveryID,
		BridgeInstanceID: instance.ID,
		RoutingKey:       routingKey,
		DeliveryTarget:   deliveryTarget,
		Seq:              event.Sequence,
		EventType:        bridgepkg.DeliveryEventTypeFinal,
		Content:          bridgepkg.MessageContent{Text: notificationText(preset, event)},
		Final:            true,
		Operation:        bridgepkg.DeliveryOperationPost,
		ProviderMetadata: metadata,
	}
	if err := delivery.Validate(); err != nil {
		return bridgepkg.DeliveryEvent{}, err
	}
	return delivery, nil
}

func (s *Service) deliverBridgeEvent(
	ctx context.Context,
	preset Preset,
	event Event,
	instance bridgepkg.BridgeInstance,
	delivery bridgepkg.DeliveryEvent,
) error {
	deliveryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	ack, err := s.bridges.DeliverBridge(
		deliveryCtx,
		instance.ExtensionName,
		bridgepkg.DeliveryRequest{Event: delivery},
	)
	if err != nil {
		return fmt.Errorf(
			"notifications: deliver preset %q event %q to bridge %q: %w",
			preset.Name,
			event.ID,
			instance.ID,
			err,
		)
	}
	if err := ack.ValidateFor(delivery); err != nil {
		return fmt.Errorf("notifications: validate preset %q delivery ack: %w", preset.Name, err)
	}
	return nil
}

func (s *Service) resolveTarget(
	ctx context.Context,
	target Target,
) (*bridgepkg.BridgeTarget, error) {
	query := target.CanonicalRoute
	if query == "" {
		query = target.DisplayName
	}
	resolved, err := s.bridges.ResolveBridgeTarget(ctx, target.BridgeID, query)
	if err != nil {
		return nil, fmt.Errorf(
			"notifications: resolve bridge target %q on %q: %w",
			query,
			target.BridgeID,
			err,
		)
	}
	if resolved.Match == nil {
		return nil, fmt.Errorf(
			"notifications: resolve bridge target %q on %q: %w",
			query,
			target.BridgeID,
			bridgepkg.ErrBridgeTargetUnknown,
		)
	}
	return resolved.Match, nil
}

func (s *Service) advance(
	ctx context.Context,
	key notifications.CursorKey,
	event Event,
	deliveryID string,
) (notifications.Cursor, error) {
	return s.cursors.Advance(ctx, notifications.AdvanceCursor{
		Key:          key,
		LastSequence: event.Sequence,
		DeliveryID:   deliveryID,
		Now:          s.now(),
	})
}

func (s *Service) recordDispatchError(
	ctx context.Context,
	key notifications.CursorKey,
	preset Preset,
	event Event,
	err error,
) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	if len(message) > 2048 {
		message = message[:2048]
	}
	if _, recordErr := s.cursors.RecordError(ctx, notifications.CursorError{
		Key:       key,
		LastError: message,
		Now:       s.now(),
	}); recordErr != nil {
		return errors.Join(err, recordErr)
	}
	if eventErr := s.recordDispatchFailureEvent(ctx, key, preset, event, message); eventErr != nil {
		return errors.Join(err, eventErr)
	}
	return err
}

func (s *Service) recordPresetLifecycleEvent(
	ctx context.Context,
	eventType string,
	preset Preset,
) error {
	if s == nil || s.events == nil {
		return nil
	}
	payload := struct {
		Name                   string   `json:"name"`
		Events                 []string `json:"events,omitempty"`
		Enabled                bool     `json:"enabled"`
		BuiltIn                bool     `json:"built_in"`
		UserModified           bool     `json:"user_modified"`
		DefaultUpdateAvailable bool     `json:"default_update_available"`
	}{
		Name:                   preset.Name,
		Events:                 append([]string(nil), preset.Events...),
		Enabled:                preset.Enabled,
		BuiltIn:                preset.BuiltIn,
		UserModified:           preset.UserModified,
		DefaultUpdateAvailable: preset.DefaultUpdateAvailable,
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("notifications: encode preset lifecycle event: %w", err)
	}
	if err := s.events.WriteEventSummary(detachedContext(ctx), store.EventSummary{
		Type:      eventType,
		Outcome:   string(eventspkg.OutcomeFor(eventType)),
		Content:   content,
		Summary:   notificationPresetLifecycleSummary(eventType, preset.Name),
		Timestamp: s.now().UTC(),
	}); err != nil {
		return fmt.Errorf("notifications: record preset lifecycle event: %w", err)
	}
	return nil
}

func (s *Service) recordDispatchFailureEvent(
	ctx context.Context,
	key notifications.CursorKey,
	preset Preset,
	event Event,
	message string,
) error {
	if s == nil || s.events == nil {
		return nil
	}
	payload := struct {
		Preset    string                  `json:"preset"`
		EventID   string                  `json:"event_id"`
		EventType string                  `json:"event_type"`
		CursorKey notifications.CursorKey `json:"cursor_key"`
		LastError string                  `json:"last_error"`
		Sequence  int64                   `json:"sequence"`
	}{
		Preset:    preset.Name,
		EventID:   event.ID,
		EventType: event.Type,
		CursorKey: key,
		LastError: message,
		Sequence:  event.Sequence,
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("notifications: encode preset dispatch failure event: %w", err)
	}
	if err := s.events.WriteEventSummary(detachedContext(ctx), store.EventSummary{
		Type:      eventspkg.NotificationPresetDispatchFailed,
		Outcome:   string(eventspkg.OutcomeFor(eventspkg.NotificationPresetDispatchFailed)),
		Content:   content,
		Summary:   fmt.Sprintf("notification preset %s dispatch failed", preset.Name),
		Timestamp: s.now().UTC(),
	}); err != nil {
		return fmt.Errorf("notifications: record preset dispatch failure event: %w", err)
	}
	return nil
}

func (s *Service) checkStore() error {
	if s == nil || s.store == nil {
		return fmt.Errorf("%w: store is required", ErrInvalidPreset)
	}
	return nil
}

func (s *Service) checkDispatch() error {
	if err := s.checkStore(); err != nil {
		return err
	}
	if s.cursors == nil {
		return fmt.Errorf("%w: cursor store is required", ErrInvalidPreset)
	}
	if s.bridges == nil {
		return fmt.Errorf("%w: bridge runtime is required", ErrInvalidPreset)
	}
	return nil
}

func normalizeUpdateRequest(req UpdateRequest, now time.Time) UpdateRequest {
	normalized := req
	if req.Events != nil {
		events := normalizePresetEvents(*req.Events)
		normalized.Events = &events
	}
	if req.Targets != nil {
		targets := normalizePresetTargets(*req.Targets)
		normalized.Targets = &targets
	}
	if req.Filter != nil {
		value := strings.TrimSpace(*req.Filter)
		normalized.Filter = &value
	}
	if normalized.Now.IsZero() {
		normalized.Now = now.UTC()
	} else {
		normalized.Now = normalized.Now.UTC()
	}
	return normalized
}

func cursorKeyForTarget(preset Preset, target Target, event Event) notifications.CursorKey {
	workspaceID := strings.TrimSpace(event.WorkspaceID)
	if workspaceID == "" {
		workspaceID = "global"
	}
	return notifications.CursorKey{
		ConsumerID: CursorConsumerPrefix + preset.Name + ":target:" + target.StableHash(),
		StreamName: event.Type,
		SubjectID:  workspaceID + ":" + event.ID,
	}
}

func skipDeliveryID(preset Preset, event Event, reason string) string {
	return "preset:" + preset.Name + ":" + event.ID + ":skip:" + strings.TrimSpace(reason)
}

func deliveryIDForTarget(preset Preset, event Event, index int) string {
	if index < 0 {
		index = 0
	}
	return fmt.Sprintf("preset:%s:%s:%d", preset.Name, event.ID, index+1)
}

func notificationText(preset Preset, event Event) string {
	summary := strings.TrimSpace(event.Summary)
	if summary == "" {
		summary = event.Type
	}
	return fmt.Sprintf("AGH %s: %s", preset.Name, summary)
}

func notificationPresetLifecycleSummary(eventType string, name string) string {
	normalizedName := strings.TrimSpace(name)
	if normalizedName == "" {
		normalizedName = "unknown"
	}
	switch eventType {
	case eventspkg.NotificationPresetCreated:
		return fmt.Sprintf("notification preset %s created", normalizedName)
	case eventspkg.NotificationPresetUpdated:
		return fmt.Sprintf("notification preset %s updated", normalizedName)
	case eventspkg.NotificationPresetDeleted:
		return fmt.Sprintf("notification preset %s deleted", normalizedName)
	default:
		return fmt.Sprintf("notification preset %s changed", normalizedName)
	}
}

func detachedContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithoutCancel(ctx)
}

func deliveryTargetFromBridgeTarget(
	instance bridgepkg.BridgeInstance,
	target bridgepkg.BridgeTarget,
	presetTarget Target,
) (bridgepkg.DeliveryTarget, bridgepkg.RoutingKey, error) {
	req := bridgepkg.ResolveDeliveryTargetRequest{
		BridgeInstanceID: instance.ID,
		Mode:             presetTarget.DeliveryMode,
	}
	switch target.TargetType.Normalize() {
	case bridgepkg.BridgeTargetTypeUser:
		req.PeerID = target.CanonicalRoute
	case bridgepkg.BridgeTargetTypeThread:
		req.ThreadID = target.CanonicalRoute
		if strings.TrimSpace(target.Qualifier) != "" {
			req.GroupID = target.Qualifier
		}
	default:
		req.GroupID = target.CanonicalRoute
	}
	deliveryTarget, err := bridgepkg.BuildDeliveryTarget(instance, req)
	if err != nil {
		return bridgepkg.DeliveryTarget{}, bridgepkg.RoutingKey{}, err
	}
	routingKey := bridgepkg.RoutingKey{
		Scope:            instance.Scope,
		WorkspaceID:      instance.WorkspaceID,
		BridgeInstanceID: instance.ID,
		PeerID:           deliveryTarget.PeerID,
		ThreadID:         deliveryTarget.ThreadID,
		GroupID:          deliveryTarget.GroupID,
	}
	return deliveryTarget, routingKey, nil
}
