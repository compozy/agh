package automation

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pedronauck/agh/internal/diagnostics"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/vault"
)

var (
	// ErrTriggerAlreadyRegistered reports that the trigger id already exists in the runtime.
	ErrTriggerAlreadyRegistered = errors.New("automation: trigger already registered")
	// ErrTriggerEngineStopped reports that the trigger engine has already been stopped.
	ErrTriggerEngineStopped = errors.New("automation: trigger engine stopped")
	// ErrWebhookEndpointInvalid reports that a webhook endpoint value cannot be normalized.
	ErrWebhookEndpointInvalid = errors.New("automation: invalid webhook endpoint")
	// ErrWebhookTriggerNotRegistered reports that no runtime webhook registration matches the endpoint id.
	ErrWebhookTriggerNotRegistered = errors.New("automation: webhook trigger not registered")
	// ErrWebhookTimestampInvalid reports that a webhook timestamp is outside the accepted freshness window.
	ErrWebhookTimestampInvalid = errors.New("automation: webhook timestamp outside freshness window")
	// ErrWebhookSignatureInvalid reports that a webhook signature does not match the expected HMAC.
	ErrWebhookSignatureInvalid = errors.New("automation: webhook signature invalid")
	// ErrWebhookSecretRequired reports that a webhook registration did not provide auth material.
	ErrWebhookSecretRequired = errors.New("automation: webhook secret is required")
	// ErrWebhookReplayDetected reports that the same authenticated delivery id
	// was already processed within the replay window.
	ErrWebhookReplayDetected = errors.New("automation: webhook delivery already processed")
)

const (
	webhookSignaturePrefix = "sha256="
	triggerEventWebhook    = "webhook"
	sessionEventCreated    = "session.created"
	sessionEventStopped    = "session.stopped"
)

// DefaultWebhookFreshnessWindow is the default accepted clock skew for webhook requests.
const DefaultWebhookFreshnessWindow = 5 * time.Minute

// TriggerDispatcher is the shared execution surface used by matched triggers.
type TriggerDispatcher interface {
	Dispatch(ctx context.Context, req DispatchRequest) (*Run, error)
}

// WebhookDeliveryStore persists authenticated delivery claims across trigger-engine restarts.
type WebhookDeliveryStore interface {
	CreateRun(ctx context.Context, run Run) (Run, error)
	GetRun(ctx context.Context, id string) (Run, error)
	DeleteRun(ctx context.Context, id string) error
}

// HookSessionResolver resolves session metadata for hook-completion ingress.
type HookSessionResolver interface {
	Status(ctx context.Context, id string) (*session.Info, error)
}

// TriggerEngineOption customizes trigger runtime behavior.
type TriggerEngineOption func(*TriggerEngine)

// TriggerResult reports how many triggers matched one activation and which runs were created.
type TriggerResult struct {
	Matched int   `json:"matched"`
	Runs    []Run `json:"runs,omitempty"`
}

// TriggerRegistration stores one runtime trigger definition plus write-only webhook auth material.
type TriggerRegistration struct {
	Trigger        Trigger `json:"trigger"`
	compiledFilter triggerFilter
}

// Validate ensures the runtime registration is internally consistent.
func (r TriggerRegistration) Validate(path string) error {
	if err := r.Trigger.Validate(nestedPath(path, "trigger")); err != nil {
		return err
	}

	if strings.TrimSpace(r.Trigger.Event) != triggerEventWebhook {
		return nil
	}
	if strings.TrimSpace(r.Trigger.WebhookID) == "" {
		return fmt.Errorf(
			"%s is required when trigger.event is %q",
			nestedPath(path, "trigger.webhook_id"),
			triggerEventWebhook,
		)
	}
	return nil
}

// ParsedWebhookEndpoint is the normalized webhook endpoint split into slug and stable webhook id.
type ParsedWebhookEndpoint struct {
	EndpointSlug string `json:"endpoint_slug"`
	WebhookID    string `json:"webhook_id"`
}

// WebhookRequest is the transport-neutral webhook delivery input consumed by the trigger engine.
type WebhookRequest struct {
	Scope       Scope          `json:"scope"`
	WorkspaceID string         `json:"workspace_id,omitempty"`
	Endpoint    string         `json:"endpoint"`
	DeliveryID  string         `json:"delivery_id"`
	Timestamp   time.Time      `json:"timestamp"`
	Signature   string         `json:"signature"`
	Payload     []byte         `json:"payload,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
}

// Validate ensures the request can be normalized before any dispatch occurs.
func (r WebhookRequest) Validate(path string) error {
	if err := ValidateScopeBinding(r.Scope, r.WorkspaceID, path, "workspace_id"); err != nil {
		return err
	}
	if strings.TrimSpace(r.Endpoint) == "" {
		return errors.New(nestedPath(path, "endpoint") + " is required")
	}
	if strings.TrimSpace(r.DeliveryID) == "" {
		return errors.New(nestedPath(path, "delivery_id") + " is required")
	}
	if r.Timestamp.IsZero() {
		return errors.New(nestedPath(path, "timestamp") + " is required")
	}
	if strings.TrimSpace(r.Signature) == "" {
		return errors.New(nestedPath(path, "signature") + " is required")
	}
	return nil
}

// MemoryConsolidatedEvent is the observer-facing completion payload used for normalized memory ingress.
type MemoryConsolidatedEvent struct {
	WorkspaceID string         `json:"workspace_id,omitempty"`
	Timestamp   time.Time      `json:"timestamp"`
	Data        map[string]any `json:"data,omitempty"`
}

// MemoryConsolidationObserver receives dream consolidation completions at the trigger-engine boundary.
type MemoryConsolidationObserver interface {
	OnMemoryConsolidated(ctx context.Context, event MemoryConsolidatedEvent) error
}

// TriggerEngine matches normalized activations against registered triggers and dispatches runs.
type TriggerEngine struct {
	dispatcher TriggerDispatcher
	logger     *slog.Logger
	now        func() time.Time

	webhookFreshnessWindow time.Duration
	hookSessions           HookSessionResolver
	webhookSecrets         WebhookSecretResolver
	webhookDeliveries      WebhookDeliveryStore

	mu            sync.RWMutex
	stopped       bool
	registrations map[string]TriggerRegistration
	webhookIndex  map[string]string
	deliveries    map[string]time.Time
}

// NewTriggerEngine constructs a trigger runtime over the shared dispatcher path.
func NewTriggerEngine(dispatcher TriggerDispatcher, opts ...TriggerEngineOption) (*TriggerEngine, error) {
	if dispatcher == nil {
		return nil, errors.New("automation: trigger dispatcher is required")
	}

	engine := &TriggerEngine{
		dispatcher:             dispatcher,
		logger:                 slog.Default(),
		now:                    func() time.Time { return time.Now().UTC() },
		webhookFreshnessWindow: DefaultWebhookFreshnessWindow,
		registrations:          make(map[string]TriggerRegistration),
		webhookIndex:           make(map[string]string),
		deliveries:             make(map[string]time.Time),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(engine)
		}
	}

	if engine.logger == nil {
		engine.logger = slog.Default()
	}
	if engine.now == nil {
		engine.now = func() time.Time { return time.Now().UTC() }
	}
	if engine.webhookFreshnessWindow <= 0 {
		engine.webhookFreshnessWindow = DefaultWebhookFreshnessWindow
	}
	if engine.registrations == nil {
		engine.registrations = make(map[string]TriggerRegistration)
	}
	if engine.webhookIndex == nil {
		engine.webhookIndex = make(map[string]string)
	}
	if engine.deliveries == nil {
		engine.deliveries = make(map[string]time.Time)
	}

	return engine, nil
}

// WithTriggerEngineLogger overrides the trigger-engine logger.
func WithTriggerEngineLogger(logger *slog.Logger) TriggerEngineOption {
	return func(engine *TriggerEngine) {
		engine.logger = logger
	}
}

// WithTriggerEngineNow overrides the trigger-engine clock.
func WithTriggerEngineNow(now func() time.Time) TriggerEngineOption {
	return func(engine *TriggerEngine) {
		engine.now = now
	}
}

// WithTriggerEngineWebhookFreshnessWindow overrides the accepted webhook clock skew.
func WithTriggerEngineWebhookFreshnessWindow(window time.Duration) TriggerEngineOption {
	return func(engine *TriggerEngine) {
		engine.webhookFreshnessWindow = window
	}
}

// WithTriggerEngineHookSessionResolver injects session lookup support for hook-completion ingress.
func WithTriggerEngineHookSessionResolver(resolver HookSessionResolver) TriggerEngineOption {
	return func(engine *TriggerEngine) {
		engine.hookSessions = resolver
	}
}

// WithTriggerEngineWebhookSecretResolver injects the vault-backed resolver for webhook auth refs.
func WithTriggerEngineWebhookSecretResolver(resolver WebhookSecretResolver) TriggerEngineOption {
	return func(engine *TriggerEngine) {
		engine.webhookSecrets = resolver
	}
}

// WithTriggerEngineWebhookDeliveryStore injects durable replay protection for webhook delivery IDs.
func WithTriggerEngineWebhookDeliveryStore(store WebhookDeliveryStore) TriggerEngineOption {
	return func(engine *TriggerEngine) {
		engine.webhookDeliveries = store
	}
}

// Start validates the runtime start contract. Trigger matching is synchronous so no background work begins here.
func (e *TriggerEngine) Start(ctx context.Context) error {
	if ctx == nil {
		return errors.New("automation: trigger engine start context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.stopped {
		return ErrTriggerEngineStopped
	}
	return nil
}

// Shutdown marks the runtime as stopped and clears registered triggers.
func (e *TriggerEngine) Shutdown(ctx context.Context) error {
	if ctx == nil {
		return errors.New("automation: trigger engine shutdown context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.stopped = true
	e.registrations = make(map[string]TriggerRegistration)
	e.webhookIndex = make(map[string]string)
	e.deliveries = make(map[string]time.Time)
	return nil
}

// Register adds a new trigger definition to the runtime.
func (e *TriggerEngine) Register(registration TriggerRegistration) error {
	normalized, err := normalizeTriggerRegistration(registration)
	if err != nil {
		return err
	}
	if err := e.validateWebhookRegistration(normalized); err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stopped {
		return ErrTriggerEngineStopped
	}
	if _, exists := e.registrations[normalized.Trigger.ID]; exists {
		return fmt.Errorf("%w: %s", ErrTriggerAlreadyRegistered, normalized.Trigger.ID)
	}
	if err := e.ensureUniqueWebhookLocked(normalized, ""); err != nil {
		return err
	}

	e.storeRegistrationLocked(normalized)
	return nil
}

// Update replaces an existing trigger registration.
func (e *TriggerEngine) Update(registration TriggerRegistration) error {
	normalized, err := normalizeTriggerRegistration(registration)
	if err != nil {
		return err
	}
	if err := e.validateWebhookRegistration(normalized); err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stopped {
		return ErrTriggerEngineStopped
	}
	if _, exists := e.registrations[normalized.Trigger.ID]; !exists {
		return fmt.Errorf("%w: %s", ErrTriggerNotFound, normalized.Trigger.ID)
	}
	if err := e.ensureUniqueWebhookLocked(normalized, normalized.Trigger.ID); err != nil {
		return err
	}

	e.deleteWebhookIndexLocked(normalized.Trigger.ID)
	e.storeRegistrationLocked(normalized)
	return nil
}

// Unregister removes one trigger registration by id.
func (e *TriggerEngine) Unregister(id string) error {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return errors.New("automation: trigger id is required")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stopped {
		return ErrTriggerEngineStopped
	}
	if _, exists := e.registrations[trimmedID]; !exists {
		return fmt.Errorf("%w: %s", ErrTriggerNotFound, trimmedID)
	}

	e.deleteWebhookIndexLocked(trimmedID)
	delete(e.registrations, trimmedID)
	return nil
}

// Fire matches one normalized activation envelope against all registered triggers.
func (e *TriggerEngine) Fire(ctx context.Context, envelope ActivationEnvelope) (TriggerResult, error) {
	if ctx == nil {
		return TriggerResult{}, errors.New("automation: trigger fire context is required")
	}
	if err := envelope.Validate("envelope"); err != nil {
		return TriggerResult{}, err
	}

	registrations, err := e.matchingRegistrations(envelope)
	if err != nil {
		return TriggerResult{}, err
	}
	return e.dispatchPreMatched(ctx, envelope, registrations)
}

// FireSessionCreated normalizes a session-created lifecycle event and routes it through the shared matching path.
func (e *TriggerEngine) FireSessionCreated(ctx context.Context, sess *session.Session) (TriggerResult, error) {
	envelope, err := sessionEnvelope(sessionEventCreated, sess)
	if err != nil {
		return TriggerResult{}, err
	}
	return e.Fire(ctx, envelope)
}

// FireSessionStopped normalizes a session-stopped lifecycle event and routes it through the shared matching path.
func (e *TriggerEngine) FireSessionStopped(ctx context.Context, sess *session.Session) (TriggerResult, error) {
	envelope, err := sessionEnvelope(sessionEventStopped, sess)
	if err != nil {
		return TriggerResult{}, err
	}
	return e.Fire(ctx, envelope)
}

// FireMemoryConsolidated normalizes a dream-consolidation completion into the shared matching path.
func (e *TriggerEngine) FireMemoryConsolidated(
	ctx context.Context,
	event MemoryConsolidatedEvent,
) (TriggerResult, error) {
	envelope, err := memoryConsolidatedEnvelope(event)
	if err != nil {
		return TriggerResult{}, err
	}
	return e.Fire(ctx, envelope)
}

// FireHookCompletion normalizes one hook-completion telemetry record into the shared matching path.
func (e *TriggerEngine) FireHookCompletion(
	ctx context.Context,
	sessionID string,
	record hookspkg.HookRunRecord,
) (TriggerResult, error) {
	if ctx == nil {
		return TriggerResult{}, errors.New("automation: trigger fire context is required")
	}
	envelope, err := e.hookCompletionEnvelope(ctx, sessionID, record)
	if err != nil {
		return TriggerResult{}, err
	}
	return e.Fire(ctx, envelope)
}

// HandleWebhook authenticates, normalizes, and dispatches one webhook delivery.
func (e *TriggerEngine) HandleWebhook(ctx context.Context, request WebhookRequest) (TriggerResult, error) {
	if ctx == nil {
		return TriggerResult{}, errors.New("automation: webhook context is required")
	}
	if err := request.Validate("webhook"); err != nil {
		return TriggerResult{}, err
	}

	parsed, err := ParseWebhookEndpoint(request.Endpoint)
	if err != nil {
		return TriggerResult{}, err
	}
	registration, err := e.webhookRegistration(request.Scope, request.WorkspaceID, parsed)
	if err != nil {
		return TriggerResult{}, err
	}
	if err := ValidateWebhookTimestamp(request.Timestamp, e.now(), e.webhookFreshnessWindow); err != nil {
		return TriggerResult{}, err
	}
	webhookSecret, cleanup, err := e.resolveWebhookSecret(ctx, registration.Trigger)
	if err != nil {
		return TriggerResult{}, err
	}
	defer cleanup()
	if err := ValidateWebhookSignature(
		webhookSecret,
		request.Timestamp,
		request.Payload,
		request.Signature,
	); err != nil {
		return TriggerResult{}, err
	}
	claim, err := e.claimWebhookDelivery(ctx, registration.Trigger, request.DeliveryID)
	if err != nil {
		return TriggerResult{}, err
	}

	envelope, err := webhookEnvelope(request, registration.Trigger)
	if err != nil {
		e.releaseWebhookDelivery(ctx, claim)
		return TriggerResult{}, err
	}
	result, err := e.dispatchAfterFilter(ctx, envelope, []TriggerRegistration{registration}, claim.reservedRun)
	if len(result.Runs) == 0 {
		e.releaseWebhookDelivery(ctx, claim)
	}
	return result, err
}

// SessionObserver exposes the existing session notifier shape for internal lifecycle ingress.
func (e *TriggerEngine) SessionObserver() session.Notifier {
	return &triggerSessionObserver{engine: e}
}

// HookTelemetrySink exposes the existing hook telemetry sink shape for hook-completion ingress.
func (e *TriggerEngine) HookTelemetrySink() hookspkg.TelemetrySink {
	return &triggerHookTelemetrySink{engine: e}
}

// MemoryObserver exposes the observer-facing dream-consolidation completion adapter.
func (e *TriggerEngine) MemoryObserver() MemoryConsolidationObserver {
	return &triggerMemoryObserver{engine: e}
}

// ParseWebhookEndpoint resolves the human slug and stable webhook id from an endpoint path segment.
func ParseWebhookEndpoint(endpoint string) (ParsedWebhookEndpoint, error) {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return ParsedWebhookEndpoint{}, ErrWebhookEndpointInvalid
	}

	separator := strings.LastIndex(trimmed, "--")
	if separator <= 0 || separator+2 >= len(trimmed) {
		return ParsedWebhookEndpoint{}, fmt.Errorf("%w: expected <slug>--<webhook_id>", ErrWebhookEndpointInvalid)
	}

	parsed := ParsedWebhookEndpoint{
		EndpointSlug: strings.TrimSpace(trimmed[:separator]),
		WebhookID:    strings.TrimSpace(trimmed[separator+2:]),
	}
	if parsed.EndpointSlug == "" || parsed.WebhookID == "" {
		return ParsedWebhookEndpoint{}, fmt.Errorf(
			"%w: expected non-empty slug and webhook id",
			ErrWebhookEndpointInvalid,
		)
	}
	if !strings.HasPrefix(parsed.WebhookID, "wbh_") {
		return ParsedWebhookEndpoint{}, fmt.Errorf(
			"%w: webhook id %q must start with \"wbh_\"",
			ErrWebhookEndpointInvalid,
			parsed.WebhookID,
		)
	}

	return parsed, nil
}

// FormatWebhookEndpoint returns the stable public endpoint segment for one webhook registration.
func FormatWebhookEndpoint(endpointSlug string, webhookID string) (string, error) {
	trimmedSlug := strings.TrimSpace(endpointSlug)
	trimmedWebhookID := strings.TrimSpace(webhookID)
	if trimmedSlug == "" || trimmedWebhookID == "" {
		return "", ErrWebhookEndpointInvalid
	}
	return trimmedSlug + "--" + trimmedWebhookID, nil
}

// SignWebhookPayload calculates the expected HMAC signature for a webhook request payload.
func SignWebhookPayload(secret string, timestamp time.Time, payload []byte) (string, error) {
	trimmedSecret := strings.TrimSpace(secret)
	if trimmedSecret == "" {
		return "", ErrWebhookSecretRequired
	}
	if timestamp.IsZero() {
		return "", errors.New("automation: webhook timestamp is required")
	}

	mac := hmac.New(sha256.New, []byte(trimmedSecret))
	if _, err := mac.Write(webhookSignaturePayload(timestamp, payload)); err != nil {
		return "", fmt.Errorf("automation: sign webhook payload: %w", err)
	}
	return webhookSignaturePrefix + hex.EncodeToString(mac.Sum(nil)), nil
}

// ValidateWebhookSignature verifies the provided signature before any trigger dispatch occurs.
func ValidateWebhookSignature(secret string, timestamp time.Time, payload []byte, signature string) error {
	expected, err := SignWebhookPayload(secret, timestamp, payload)
	if err != nil {
		return err
	}

	expectedMAC, err := decodeWebhookSignature(expected)
	if err != nil {
		return err
	}
	providedMAC, err := decodeWebhookSignature(signature)
	if err != nil {
		return err
	}
	if !hmac.Equal(providedMAC, expectedMAC) {
		return ErrWebhookSignatureInvalid
	}
	return nil
}

// ValidateWebhookTimestamp rejects stale or far-future webhook timestamps.
func ValidateWebhookTimestamp(timestamp time.Time, now time.Time, window time.Duration) error {
	if timestamp.IsZero() {
		return errors.New("automation: webhook timestamp is required")
	}
	if now.IsZero() {
		return errors.New("automation: current time is required")
	}
	if window <= 0 {
		return errors.New("automation: webhook freshness window must be positive")
	}

	delta := now.UTC().Sub(timestamp.UTC())
	if delta < 0 {
		delta = -delta
	}
	if delta > window {
		return ErrWebhookTimestampInvalid
	}
	return nil
}

func (e *TriggerEngine) matchingRegistrations(envelope ActivationEnvelope) ([]TriggerRegistration, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.stopped {
		return nil, ErrTriggerEngineStopped
	}

	matches := make([]TriggerRegistration, 0)
	for _, registration := range e.registrations {
		if registrationMatchesEnvelope(registration, envelope) {
			matches = append(matches, cloneTriggerRegistration(registration))
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Trigger.ID < matches[j].Trigger.ID
	})
	return matches, nil
}

func (e *TriggerEngine) dispatchPreMatched(
	ctx context.Context,
	envelope ActivationEnvelope,
	registrations []TriggerRegistration,
) (TriggerResult, error) {
	return e.dispatchMatches(ctx, envelope, registrations, false, nil)
}

func (e *TriggerEngine) dispatchAfterFilter(
	ctx context.Context,
	envelope ActivationEnvelope,
	registrations []TriggerRegistration,
	reservedRun *Run,
) (TriggerResult, error) {
	return e.dispatchMatches(ctx, envelope, registrations, true, reservedRun)
}

func (e *TriggerEngine) dispatchMatches(
	ctx context.Context,
	envelope ActivationEnvelope,
	registrations []TriggerRegistration,
	filterRegistrations bool,
	reservedRun *Run,
) (TriggerResult, error) {
	result := TriggerResult{
		Runs: make([]Run, 0, len(registrations)),
	}
	var errs []error
	dispatchKind := DispatchKindTrigger
	if envelope.Source == ActivationSourceExtension {
		dispatchKind = DispatchKindExtension
	}
	for _, registration := range registrations {
		if filterRegistrations && !registrationMatchesEnvelope(registration, envelope) {
			continue
		}

		result.Matched++
		trigger := registration.Trigger
		run, err := e.dispatcher.Dispatch(ctx, DispatchRequest{
			Kind:        dispatchKind,
			Trigger:     &trigger,
			Envelope:    pointerToActivationEnvelope(envelope),
			ReservedRun: cloneRun(reservedRun),
		})
		if run != nil {
			result.Runs = append(result.Runs, *cloneRun(run))
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	if result.Runs == nil {
		result.Runs = []Run{}
	}
	return result, errors.Join(errs...)
}

func (e *TriggerEngine) webhookRegistration(
	scope Scope,
	workspaceID string,
	endpoint ParsedWebhookEndpoint,
) (TriggerRegistration, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.stopped {
		return TriggerRegistration{}, ErrTriggerEngineStopped
	}

	triggerID, ok := e.webhookIndex[strings.TrimSpace(endpoint.WebhookID)]
	if !ok {
		return TriggerRegistration{}, ErrWebhookTriggerNotRegistered
	}
	registration, ok := e.registrations[triggerID]
	if !ok {
		return TriggerRegistration{}, ErrWebhookTriggerNotRegistered
	}
	if registration.Trigger.Scope != scope ||
		strings.TrimSpace(registration.Trigger.WorkspaceID) != strings.TrimSpace(workspaceID) {
		return TriggerRegistration{}, ErrWebhookTriggerNotRegistered
	}
	if strings.TrimSpace(registration.Trigger.EndpointSlug) != strings.TrimSpace(endpoint.EndpointSlug) ||
		strings.TrimSpace(registration.Trigger.WebhookID) != strings.TrimSpace(endpoint.WebhookID) {
		return TriggerRegistration{}, ErrWebhookTriggerNotRegistered
	}
	return cloneTriggerRegistration(registration), nil
}

func (e *TriggerEngine) resolveWebhookSecret(ctx context.Context, trigger Trigger) (string, func(), error) {
	ref := strings.TrimSpace(trigger.WebhookSecretRef)
	if ref == "" {
		return "", func() {}, ErrWebhookSecretRequired
	}
	if err := vault.ValidateRefNamespace(ref, "automation"); err != nil {
		return "", func() {}, fmt.Errorf("%w: %w", ErrWebhookSecretRequired, err)
	}
	if e.webhookSecrets == nil {
		return "", func() {}, ErrWebhookSecretRequired
	}
	value, err := e.webhookSecrets.ResolveRef(ctx, ref)
	if err != nil {
		if errors.Is(err, vault.ErrSecretNotFound) || errors.Is(err, vault.ErrMissingSecret) {
			return "", func() {}, ErrWebhookSecretRequired
		}
		return "", func() {}, err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", func() {}, ErrWebhookSecretRequired
	}
	return value, diagnostics.RegisterDynamicSecret(value), nil
}

func (e *TriggerEngine) validateWebhookRegistration(registration TriggerRegistration) error {
	if !strings.EqualFold(strings.TrimSpace(registration.Trigger.Event), triggerEventWebhook) {
		return nil
	}
	ref := strings.TrimSpace(registration.Trigger.WebhookSecretRef)
	if ref == "" {
		return ErrWebhookSecretRequired
	}
	if err := vault.ValidateRefNamespace(ref, "automation"); err != nil {
		return fmt.Errorf("%w: %w", ErrWebhookSecretRequired, err)
	}
	if e.webhookSecrets == nil {
		return ErrWebhookSecretRequired
	}
	return nil
}

func (e *TriggerEngine) ensureUniqueWebhookLocked(registration TriggerRegistration, allowTriggerID string) error {
	webhookID := strings.TrimSpace(registration.Trigger.WebhookID)
	if webhookID == "" {
		return nil
	}

	existingTriggerID, exists := e.webhookIndex[webhookID]
	if exists && existingTriggerID != strings.TrimSpace(allowTriggerID) {
		return ErrTriggerWebhookIDTaken
	}
	return nil
}

func (e *TriggerEngine) storeRegistrationLocked(registration TriggerRegistration) {
	e.registrations[registration.Trigger.ID] = cloneTriggerRegistration(registration)
	if webhookID := strings.TrimSpace(registration.Trigger.WebhookID); webhookID != "" {
		e.webhookIndex[webhookID] = registration.Trigger.ID
	}
}

func (e *TriggerEngine) deleteWebhookIndexLocked(triggerID string) {
	registration, ok := e.registrations[strings.TrimSpace(triggerID)]
	if !ok {
		return
	}
	if webhookID := strings.TrimSpace(registration.Trigger.WebhookID); webhookID != "" {
		delete(e.webhookIndex, webhookID)
	}
}

type webhookDeliveryClaim struct {
	triggerID   string
	deliveryID  string
	reservedRun *Run
}

func (e *TriggerEngine) claimWebhookDelivery(
	ctx context.Context,
	trigger Trigger,
	deliveryID string,
) (webhookDeliveryClaim, error) {
	claim := webhookDeliveryClaim{
		triggerID:  strings.TrimSpace(trigger.ID),
		deliveryID: strings.TrimSpace(deliveryID),
	}
	if e.webhookDeliveries != nil {
		reserved, err := e.claimPersistentWebhookDelivery(ctx, trigger, deliveryID)
		if err != nil {
			return webhookDeliveryClaim{}, err
		}
		claim.reservedRun = reserved
		return claim, nil
	}
	return claim, e.claimInMemoryWebhookDelivery(trigger.ID, deliveryID)
}

func (e *TriggerEngine) claimInMemoryWebhookDelivery(triggerID string, deliveryID string) error {
	now := e.now()

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stopped {
		return ErrTriggerEngineStopped
	}

	e.purgeDeliveriesLocked(now)

	key := webhookDeliveryKey(triggerID, deliveryID)
	if expiresAt, exists := e.deliveries[key]; exists && expiresAt.After(now) {
		return ErrWebhookReplayDetected
	}

	e.deliveries[key] = now.Add(e.webhookFreshnessWindow)
	return nil
}

func (e *TriggerEngine) claimPersistentWebhookDelivery(
	ctx context.Context,
	trigger Trigger,
	deliveryID string,
) (*Run, error) {
	if ctx == nil {
		return nil, errors.New("automation: webhook delivery claim context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	e.mu.RLock()
	stopped := e.stopped
	e.mu.RUnlock()
	if stopped {
		return nil, ErrTriggerEngineStopped
	}

	now := e.now()
	runID := webhookDeliveryRunID(trigger.ID, deliveryID)
	run := Run{
		ID:        runID,
		TriggerID: strings.TrimSpace(trigger.ID),
		FireID:    webhookDeliveryFireID(trigger.ID, deliveryID),
		Status:    RunScheduled,
		Attempt:   1,
		StartedAt: timePointer(now),
	}
	created, err := e.webhookDeliveries.CreateRun(ctx, run)
	if err == nil {
		return &created, nil
	}
	if !errors.Is(err, ErrRunAlreadyExists) {
		return nil, fmt.Errorf("automation: claim webhook delivery: %w", err)
	}

	existing, getErr := e.webhookDeliveries.GetRun(ctx, runID)
	if getErr != nil && !errors.Is(getErr, ErrRunNotFound) {
		return nil, fmt.Errorf("automation: inspect webhook delivery claim: %w", getErr)
	}
	if getErr == nil && webhookDeliveryClaimActive(existing, now, e.webhookFreshnessWindow) {
		return nil, ErrWebhookReplayDetected
	}
	if getErr == nil {
		if err := e.webhookDeliveries.DeleteRun(ctx, runID); err != nil && !errors.Is(err, ErrRunNotFound) {
			return nil, fmt.Errorf("automation: expire webhook delivery claim: %w", err)
		}
	}

	created, err = e.webhookDeliveries.CreateRun(ctx, run)
	if err != nil {
		if errors.Is(err, ErrRunAlreadyExists) {
			return nil, ErrWebhookReplayDetected
		}
		return nil, fmt.Errorf("automation: reclaim webhook delivery: %w", err)
	}
	return &created, nil
}

func (e *TriggerEngine) releaseWebhookDelivery(ctx context.Context, claim webhookDeliveryClaim) {
	if claim.reservedRun != nil && e.webhookDeliveries != nil {
		if err := e.webhookDeliveries.DeleteRun(
			ctx,
			claim.reservedRun.ID,
		); err != nil &&
			!errors.Is(err, ErrRunNotFound) {
			e.logger.Warn(
				"automation.trigger.webhook_delivery_release_failed",
				"run_id", strings.TrimSpace(claim.reservedRun.ID),
				"error", err,
			)
		}
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.deliveries, webhookDeliveryKey(claim.triggerID, claim.deliveryID))
}

func (e *TriggerEngine) purgeDeliveriesLocked(now time.Time) {
	for key, expiresAt := range e.deliveries {
		if !expiresAt.After(now) {
			delete(e.deliveries, key)
		}
	}
}

func webhookDeliveryKey(triggerID string, deliveryID string) string {
	return strings.TrimSpace(triggerID) + "\x00" + strings.TrimSpace(deliveryID)
}

func (e *TriggerEngine) hookCompletionEnvelope(
	ctx context.Context,
	sessionID string,
	record hookspkg.HookRunRecord,
) (ActivationEnvelope, error) {
	if strings.TrimSpace(sessionID) == "" {
		return ActivationEnvelope{}, errors.New("automation: hook completion session id is required")
	}
	if strings.TrimSpace(record.HookName) == "" {
		return ActivationEnvelope{}, errors.New("automation: hook completion hook name is required")
	}
	if e.hookSessions == nil {
		return ActivationEnvelope{}, errors.New("automation: hook session resolver is required")
	}

	info, err := e.hookSessions.Status(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return ActivationEnvelope{}, fmt.Errorf("automation: resolve hook session %q: %w", sessionID, err)
	}
	if info == nil {
		return ActivationEnvelope{}, fmt.Errorf("automation: resolve hook session %q: empty session", sessionID)
	}

	data := map[string]any{
		"session_id":     strings.TrimSpace(info.ID),
		"session_name":   strings.TrimSpace(info.Name),
		"session_type":   string(info.Type),
		"agent_name":     strings.TrimSpace(info.AgentName),
		"hook_name":      strings.TrimSpace(record.HookName),
		"hook_event":     record.Event.String(),
		"hook_source":    record.Source.String(),
		"hook_mode":      string(record.Mode),
		"hook_outcome":   string(record.Outcome),
		"dispatch_depth": strconv.Itoa(record.DispatchDepth),
		"required":       strconv.FormatBool(record.Required),
	}
	if !record.RecordedAt.IsZero() {
		data["recorded_at"] = record.RecordedAt.UTC().Format(time.RFC3339Nano)
	}
	if trimmedError := strings.TrimSpace(record.Error); trimmedError != "" {
		data["error"] = trimmedError
	}

	workspaceID := strings.TrimSpace(info.WorkspaceID)
	return ActivationEnvelope{
		Kind:        "hook." + strings.TrimSpace(record.HookName) + ".completed",
		Scope:       scopeFromWorkspaceID(workspaceID),
		WorkspaceID: workspaceID,
		Source:      ActivationSourceHook,
		Data:        data,
	}, nil
}

func normalizeTriggerRegistration(registration TriggerRegistration) (TriggerRegistration, error) {
	normalized := TriggerRegistration{
		Trigger: Trigger{
			ID:               strings.TrimSpace(registration.Trigger.ID),
			Scope:            registration.Trigger.Scope,
			Name:             strings.TrimSpace(registration.Trigger.Name),
			AgentName:        strings.TrimSpace(registration.Trigger.AgentName),
			WorkspaceID:      strings.TrimSpace(registration.Trigger.WorkspaceID),
			Prompt:           strings.TrimSpace(registration.Trigger.Prompt),
			Event:            strings.TrimSpace(registration.Trigger.Event),
			Filter:           cloneStringMap(registration.Trigger.Filter),
			Enabled:          registration.Trigger.Enabled,
			Retry:            registration.Trigger.Retry,
			FireLimit:        registration.Trigger.FireLimit,
			Source:           registration.Trigger.Source,
			WebhookID:        strings.TrimSpace(registration.Trigger.WebhookID),
			EndpointSlug:     strings.TrimSpace(registration.Trigger.EndpointSlug),
			WebhookSecretRef: strings.TrimSpace(registration.Trigger.WebhookSecretRef),
			CreatedAt:        registration.Trigger.CreatedAt,
			UpdatedAt:        registration.Trigger.UpdatedAt,
		},
	}
	if err := normalized.Validate("trigger_registration"); err != nil {
		return TriggerRegistration{}, err
	}
	normalized.compiledFilter = compileTriggerFilter(normalized.Trigger.Filter)
	return normalized, nil
}

func registrationMatchesEnvelope(registration TriggerRegistration, envelope ActivationEnvelope) bool {
	trigger := registration.Trigger
	if !trigger.Enabled {
		return false
	}
	if strings.TrimSpace(trigger.Event) != strings.TrimSpace(envelope.Kind) {
		return false
	}
	if trigger.Scope != envelope.Scope {
		return false
	}
	if strings.TrimSpace(trigger.WorkspaceID) != strings.TrimSpace(envelope.WorkspaceID) {
		return false
	}
	if len(trigger.Filter) == 0 {
		return true
	}
	if len(registration.compiledFilter.entries) == len(trigger.Filter) {
		return registration.compiledFilter.matches(envelope)
	}
	return exactFilterMatch(trigger.Filter, envelope)
}

func sessionEnvelope(kind string, sess *session.Session) (ActivationEnvelope, error) {
	if sess == nil {
		return ActivationEnvelope{}, errors.New("automation: session is required")
	}
	info := sess.Info()
	if info == nil {
		return ActivationEnvelope{}, errors.New("automation: session info is required")
	}

	workspaceID := strings.TrimSpace(info.WorkspaceID)
	data := map[string]any{
		"session_id":     strings.TrimSpace(info.ID),
		"session_name":   strings.TrimSpace(info.Name),
		"session_type":   string(info.Type),
		"agent_name":     strings.TrimSpace(info.AgentName),
		"state":          string(info.State),
		"workspace":      strings.TrimSpace(info.Workspace),
		"workspace_id":   workspaceID,
		"acp_session_id": strings.TrimSpace(info.ACPSessionID),
	}
	if !info.CreatedAt.IsZero() {
		data["created_at"] = info.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	if !info.UpdatedAt.IsZero() {
		data["updated_at"] = info.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	if kind == sessionEventStopped {
		if stopReason := strings.TrimSpace(string(info.StopReason)); stopReason != "" {
			data["stop_reason"] = stopReason
		}
		if stopDetail := strings.TrimSpace(info.StopDetail); stopDetail != "" {
			data["stop_detail"] = stopDetail
		}
	}

	return ActivationEnvelope{
		Kind:        kind,
		Scope:       scopeFromWorkspaceID(workspaceID),
		WorkspaceID: workspaceID,
		Source:      ActivationSourceObserver,
		Data:        data,
	}, nil
}

func memoryConsolidatedEnvelope(event MemoryConsolidatedEvent) (ActivationEnvelope, error) {
	workspaceID := strings.TrimSpace(event.WorkspaceID)
	data := cloneAnyMap(event.Data)
	if data == nil {
		data = make(map[string]any)
	}
	if !event.Timestamp.IsZero() {
		data["completed_at"] = event.Timestamp.UTC().Format(time.RFC3339Nano)
	}
	if workspaceID != "" {
		data["workspace_id"] = workspaceID
	}

	envelope := ActivationEnvelope{
		Kind:        "memory.consolidated",
		Scope:       scopeFromWorkspaceID(workspaceID),
		WorkspaceID: workspaceID,
		Source:      ActivationSourceObserver,
		Data:        data,
	}
	return envelope, envelope.Validate("envelope")
}

func webhookEnvelope(request WebhookRequest, trigger Trigger) (ActivationEnvelope, error) {
	endpoint, err := FormatWebhookEndpoint(trigger.EndpointSlug, trigger.WebhookID)
	if err != nil {
		return ActivationEnvelope{}, err
	}
	data := cloneAnyMap(request.Data)
	if data == nil {
		data = make(map[string]any)
	}
	if _, exists := data["payload"]; !exists {
		data["payload"] = string(request.Payload)
	}
	data["endpoint"] = endpoint
	data["endpoint_slug"] = strings.TrimSpace(trigger.EndpointSlug)
	data["webhook_id"] = strings.TrimSpace(trigger.WebhookID)
	data["timestamp"] = request.Timestamp.UTC().Format(time.RFC3339Nano)

	return ActivationEnvelope{
		Kind:        "webhook",
		Scope:       request.Scope,
		WorkspaceID: strings.TrimSpace(request.WorkspaceID),
		Source:      ActivationSourceWebhook,
		Data:        data,
	}, nil
}

func webhookSignaturePayload(timestamp time.Time, payload []byte) []byte {
	message := strconv.FormatInt(timestamp.UTC().Unix(), 10) + "."
	out := make([]byte, 0, len(message)+len(payload))
	out = append(out, []byte(message)...)
	out = append(out, payload...)
	return out
}

func decodeWebhookSignature(signature string) ([]byte, error) {
	trimmed := strings.TrimSpace(signature)
	if !strings.HasPrefix(trimmed, webhookSignaturePrefix) {
		return nil, ErrWebhookSignatureInvalid
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(trimmed, webhookSignaturePrefix))
	if err != nil || len(decoded) != sha256.Size {
		return nil, ErrWebhookSignatureInvalid
	}
	return decoded, nil
}

func cloneTriggerRegistration(src TriggerRegistration) TriggerRegistration {
	return TriggerRegistration{
		Trigger: Trigger{
			ID:               src.Trigger.ID,
			Scope:            src.Trigger.Scope,
			Name:             src.Trigger.Name,
			AgentName:        src.Trigger.AgentName,
			WorkspaceID:      src.Trigger.WorkspaceID,
			Prompt:           src.Trigger.Prompt,
			Event:            src.Trigger.Event,
			Filter:           cloneStringMap(src.Trigger.Filter),
			Enabled:          src.Trigger.Enabled,
			Retry:            src.Trigger.Retry,
			FireLimit:        src.Trigger.FireLimit,
			Source:           src.Trigger.Source,
			WebhookID:        src.Trigger.WebhookID,
			EndpointSlug:     src.Trigger.EndpointSlug,
			WebhookSecretRef: src.Trigger.WebhookSecretRef,
			CreatedAt:        src.Trigger.CreatedAt,
			UpdatedAt:        src.Trigger.UpdatedAt,
		},
		compiledFilter: cloneTriggerFilter(src.compiledFilter),
	}
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(src))
	maps.Copy(cloned, src)
	return cloned
}

func cloneAnyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(src))
	for key, value := range src {
		cloned[key] = cloneAnyValue(value)
	}
	return cloned
}

func cloneAnyValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneAnyMap(typed)
	case map[string]string:
		return cloneStringMap(typed)
	case []any:
		cloned := make([]any, len(typed))
		for idx, item := range typed {
			cloned[idx] = cloneAnyValue(item)
		}
		return cloned
	case []string:
		return append([]string(nil), typed...)
	case []byte:
		return append([]byte(nil), typed...)
	case []map[string]any:
		cloned := make([]map[string]any, len(typed))
		for idx, item := range typed {
			cloned[idx] = cloneAnyMap(item)
		}
		return cloned
	case []map[string]string:
		cloned := make([]map[string]string, len(typed))
		for idx, item := range typed {
			cloned[idx] = cloneStringMap(item)
		}
		return cloned
	default:
		return value
	}
}

func webhookDeliveryClaimActive(run Run, now time.Time, window time.Duration) bool {
	if window <= 0 {
		window = DefaultWebhookFreshnessWindow
	}
	claimedAt := time.Time{}
	if run.StartedAt != nil {
		claimedAt = run.StartedAt.UTC()
	} else if run.ScheduledAt != nil {
		claimedAt = run.ScheduledAt.UTC()
	}
	if claimedAt.IsZero() {
		return true
	}
	return claimedAt.Add(window).After(now.UTC())
}

func webhookDeliveryRunID(triggerID string, deliveryID string) string {
	sum := sha256.Sum256([]byte(webhookDeliveryKey(triggerID, deliveryID)))
	return "run_wbh_" + hex.EncodeToString(sum[:12])
}

func webhookDeliveryFireID(triggerID string, deliveryID string) string {
	sum := sha256.Sum256([]byte(webhookDeliveryKey(triggerID, deliveryID)))
	return "webhook:" + hex.EncodeToString(sum[:])
}

func scopeFromWorkspaceID(workspaceID string) Scope {
	if strings.TrimSpace(workspaceID) == "" {
		return AutomationScopeGlobal
	}
	return AutomationScopeWorkspace
}

func pointerToActivationEnvelope(envelope ActivationEnvelope) *ActivationEnvelope {
	cloned := envelope
	if envelope.Data != nil {
		cloned.Data = cloneAnyMap(envelope.Data)
	}
	return &cloned
}

type triggerSessionObserver struct {
	engine *TriggerEngine
}

func (o *triggerSessionObserver) OnSessionCreated(ctx context.Context, sess *session.Session) {
	if o == nil || o.engine == nil {
		return
	}
	if _, err := o.engine.FireSessionCreated(ctx, sess); err != nil {
		o.engine.logger.Warn("automation.trigger.session_created_failed", "error", err)
	}
}

func (o *triggerSessionObserver) OnSessionStopped(ctx context.Context, sess *session.Session) {
	if o == nil || o.engine == nil {
		return
	}
	if _, err := o.engine.FireSessionStopped(ctx, sess); err != nil {
		o.engine.logger.Warn("automation.trigger.session_stopped_failed", "error", err)
	}
}

func (*triggerSessionObserver) OnAgentEvent(context.Context, string, any) {
}

type triggerHookTelemetrySink struct {
	engine *TriggerEngine
}

func (s *triggerHookTelemetrySink) WriteHookRecord(
	ctx context.Context,
	sessionID string,
	record hookspkg.HookRunRecord,
) error {
	if s == nil || s.engine == nil {
		return nil
	}
	_, err := s.engine.FireHookCompletion(ctx, sessionID, record)
	return err
}

type triggerMemoryObserver struct {
	engine *TriggerEngine
}

func (o *triggerMemoryObserver) OnMemoryConsolidated(ctx context.Context, event MemoryConsolidatedEvent) error {
	if o == nil || o.engine == nil {
		return nil
	}
	_, err := o.engine.FireMemoryConsolidated(ctx, event)
	return err
}
