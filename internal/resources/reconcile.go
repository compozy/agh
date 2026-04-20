package resources

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultReconcileCoalesceWindow   = 50 * time.Millisecond
	defaultReconcileTimeout          = 5 * time.Second
	defaultReconcileFailureThreshold = 3
	defaultReconcileDegradedBackoff  = 500 * time.Millisecond
)

// ReconcileReason identifies why one kind was scheduled.
type ReconcileReason string

const (
	ReconcileReasonBoot       ReconcileReason = "boot"
	ReconcileReasonWrite      ReconcileReason = "write"
	ReconcileReasonDependency ReconcileReason = "dependency"
)

// Normalize returns the canonical trimmed reason.
func (r ReconcileReason) Normalize() ReconcileReason {
	return ReconcileReason(strings.TrimSpace(string(r)))
}

// Validate reports whether the reconcile reason is supported.
func (r ReconcileReason) Validate(path string) error {
	switch r.Normalize() {
	case ReconcileReasonBoot, ReconcileReasonWrite, ReconcileReasonDependency:
		return nil
	default:
		return fmt.Errorf(
			"%w: %s must be %q, %q, or %q: %q",
			ErrValidation,
			path,
			ReconcileReasonBoot,
			ReconcileReasonWrite,
			ReconcileReasonDependency,
			r,
		)
	}
}

// ReconcileDriver drives boot-time and post-commit resource projection.
type ReconcileDriver interface {
	Trigger(ctx context.Context, kind ResourceKind, reason ReconcileReason) error
	RunBoot(ctx context.Context) error
	Close(ctx context.Context) error
}

// ReconcileEventType identifies one emitted reconcile event.
type ReconcileEventType string

const (
	ReconcileEventRequested ReconcileEventType = "requested"
	ReconcileEventCoalesced ReconcileEventType = "coalesced"
	ReconcileEventFailed    ReconcileEventType = "failed"
	ReconcileEventDegraded  ReconcileEventType = "degraded"
	ReconcileEventApplied   ReconcileEventType = "applied"
)

// ReconcileEvent carries one metric-friendly reconcile observation.
type ReconcileEvent struct {
	Type                ReconcileEventType
	Kind                ResourceKind
	Reason              ReconcileReason
	Duration            time.Duration
	Revision            int64
	Operations          int
	ConsecutiveFailures int
	DegradedUntil       time.Time
	Err                 error
}

// ReconcileEventSink receives reconcile lifecycle events for metrics and observability wiring.
type ReconcileEventSink interface {
	ObserveReconcileEvent(ctx context.Context, event ReconcileEvent)
}

// ReconcileHealthStatus captures the scheduler health state for one kind.
type ReconcileHealthStatus string

const (
	ReconcileHealthStatusHealthy  ReconcileHealthStatus = "healthy"
	ReconcileHealthStatusFailing  ReconcileHealthStatus = "failing"
	ReconcileHealthStatusDegraded ReconcileHealthStatus = "degraded"
)

// ReconcileHealth captures the current health state for one projected kind.
type ReconcileHealth struct {
	Kind                ResourceKind
	Status              ReconcileHealthStatus
	ConsecutiveFailures int
	DegradedUntil       time.Time
	LastError           error
}

// ReconcileHealthSink receives kind-health updates from the driver.
type ReconcileHealthSink interface {
	ReportReconcileHealth(ctx context.Context, health ReconcileHealth)
}

// ReconcileOption configures a reconcile driver instance.
type ReconcileOption func(*reconcileDriver)

// WithReconcileLogger overrides the driver logger.
func WithReconcileLogger(logger *slog.Logger) ReconcileOption {
	return func(d *reconcileDriver) {
		if logger != nil {
			d.logger = logger
		}
	}
}

// WithReconcileNow overrides the clock used by the driver.
func WithReconcileNow(now func() time.Time) ReconcileOption {
	return func(d *reconcileDriver) {
		if now != nil {
			d.now = now
		}
	}
}

// WithReconcileCoalesceWindow overrides the per-kind rerun coalescing window.
func WithReconcileCoalesceWindow(window time.Duration) ReconcileOption {
	return func(d *reconcileDriver) {
		if window > 0 {
			d.coalesceWindow = window
		}
	}
}

// WithReconcileTimeout overrides the default per-kind reconcile timeout.
func WithReconcileTimeout(timeout time.Duration) ReconcileOption {
	return func(d *reconcileDriver) {
		if timeout > 0 {
			d.defaultTimeout = timeout
		}
	}
}

// WithReconcileKindTimeout overrides the timeout for one kind.
func WithReconcileKindTimeout(kind ResourceKind, timeout time.Duration) ReconcileOption {
	return func(d *reconcileDriver) {
		normalizedKind := kind.Normalize()
		if normalizedKind == "" || timeout <= 0 {
			return
		}
		if d.kindTimeouts == nil {
			d.kindTimeouts = make(map[ResourceKind]time.Duration)
		}
		d.kindTimeouts[normalizedKind] = timeout
	}
}

// WithReconcileKindTimeouts overrides timeouts for multiple kinds.
func WithReconcileKindTimeouts(timeouts map[ResourceKind]time.Duration) ReconcileOption {
	return func(d *reconcileDriver) {
		for kind, timeout := range timeouts {
			WithReconcileKindTimeout(kind, timeout)(d)
		}
	}
}

// WithReconcileFailureThreshold overrides the failure count that opens the degraded circuit.
func WithReconcileFailureThreshold(threshold int) ReconcileOption {
	return func(d *reconcileDriver) {
		if threshold > 0 {
			d.failureThreshold = threshold
		}
	}
}

// WithReconcileDegradedBackoff overrides the degraded-circuit backoff.
func WithReconcileDegradedBackoff(backoff time.Duration) ReconcileOption {
	return func(d *reconcileDriver) {
		if backoff > 0 {
			d.degradedBackoff = backoff
		}
	}
}

// WithReconcileEventSink wires a metric-friendly event sink into the driver.
func WithReconcileEventSink(sink ReconcileEventSink) ReconcileOption {
	return func(d *reconcileDriver) {
		d.eventSink = sink
	}
}

// WithReconcileHealthSink wires a health sink into the driver.
func WithReconcileHealthSink(sink ReconcileHealthSink) ReconcileOption {
	return func(d *reconcileDriver) {
		d.healthSink = sink
	}
}

type reconcileDriver struct {
	raw   RawStore
	actor MutationActor

	logger           *slog.Logger
	now              func() time.Time
	coalesceWindow   time.Duration
	defaultTimeout   time.Duration
	kindTimeouts     map[ResourceKind]time.Duration
	failureThreshold int
	degradedBackoff  time.Duration
	eventSink        ReconcileEventSink
	healthSink       ReconcileHealthSink

	mu         sync.Mutex
	closed     bool
	queue      []ResourceKind
	kindStates map[ResourceKind]*reconcileKindState

	projectors   map[ResourceKind]projector
	dependents   map[ResourceKind][]ResourceKind
	bootOrder    []ResourceKind
	topoErr      error
	topoRank     map[ResourceKind]int
	workerCtx    context.Context
	workerCancel context.CancelFunc
	notifyCh     chan struct{}
	doneCh       chan struct{}
}

type reconcileKindState struct {
	pending             bool
	running             bool
	dirty               bool
	pendingReason       ReconcileReason
	dirtyReason         ReconcileReason
	readyAt             time.Time
	degradedUntil       time.Time
	consecutiveFailures int
}

type reconcilePassResult struct {
	reason     ReconcileReason
	revision   int64
	operations int
	duration   time.Duration
	err        error
}

var _ ReconcileDriver = (*reconcileDriver)(nil)

// NewReconcileDriver constructs the topology-aware reconcile scheduler.
func NewReconcileDriver(
	raw RawStore,
	actor MutationActor,
	registrations []ProjectorRegistration,
	opts ...ReconcileOption,
) (ReconcileDriver, error) {
	projectors, err := buildProjectorSet(registrations)
	if err != nil {
		return nil, err
	}
	if raw == nil && len(projectors) > 0 {
		return nil, errors.New("resources: raw store is required when projectors are registered")
	}

	normalizedActor := actor
	if raw != nil || len(projectors) > 0 {
		normalizedActor, err = normalizeActor(actor)
		if err != nil {
			return nil, err
		}
	}

	driver := &reconcileDriver{
		raw:              raw,
		actor:            normalizedActor,
		logger:           slog.Default(),
		now:              func() time.Time { return time.Now().UTC() },
		coalesceWindow:   defaultReconcileCoalesceWindow,
		defaultTimeout:   defaultReconcileTimeout,
		kindTimeouts:     make(map[ResourceKind]time.Duration),
		failureThreshold: defaultReconcileFailureThreshold,
		degradedBackoff:  defaultReconcileDegradedBackoff,
		kindStates:       make(map[ResourceKind]*reconcileKindState, len(projectors)),
		projectors:       projectors,
		dependents:       make(map[ResourceKind][]ResourceKind),
		topoRank:         make(map[ResourceKind]int, len(projectors)),
		notifyCh:         make(chan struct{}, 1),
		doneCh:           make(chan struct{}),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(driver)
		}
	}
	if driver.logger == nil {
		driver.logger = slog.Default()
	}
	if driver.now == nil {
		return nil, errors.New("resources: reconcile clock is required")
	}
	if driver.coalesceWindow <= 0 {
		return nil, errors.New("resources: reconcile coalesce window must be positive")
	}
	if driver.defaultTimeout <= 0 {
		return nil, errors.New("resources: reconcile timeout must be positive")
	}
	if driver.failureThreshold <= 0 {
		return nil, errors.New("resources: reconcile failure threshold must be positive")
	}
	if driver.degradedBackoff <= 0 {
		return nil, errors.New("resources: reconcile degraded backoff must be positive")
	}

	driver.bootOrder, driver.dependents, driver.topoRank, driver.topoErr = buildReconcileTopology(projectors)
	for kind := range projectors {
		driver.kindStates[kind] = &reconcileKindState{}
	}

	driver.workerCtx, driver.workerCancel = context.WithCancel(context.Background())
	go driver.run()
	return driver, nil
}

func buildProjectorSet(registrations []ProjectorRegistration) (map[ResourceKind]projector, error) {
	projectors := make(map[ResourceKind]projector, len(registrations))
	for _, registration := range registrations {
		projector, err := unwrapProjectorRegistration(registration)
		if err != nil {
			return nil, err
		}
		kind := projector.Kind().Normalize()
		if err := kind.Validate("projector.kind"); err != nil {
			return nil, err
		}
		if _, exists := projectors[kind]; exists {
			return nil, fmt.Errorf("%w: duplicate projector registration for kind %q", ErrConflict, kind)
		}
		projectors[kind] = projector
	}
	return projectors, nil
}

func buildReconcileTopology(
	projectors map[ResourceKind]projector,
) ([]ResourceKind, map[ResourceKind][]ResourceKind, map[ResourceKind]int, error) {
	dependents := make(map[ResourceKind][]ResourceKind, len(projectors))
	indegree := make(map[ResourceKind]int, len(projectors))
	for kind := range projectors {
		indegree[kind] = 0
	}

	for kind, projector := range projectors {
		for _, dependency := range normalizeKinds(projector.DependsOn()) {
			if dependency == "" {
				continue
			}
			if _, ok := projectors[dependency]; !ok {
				continue
			}
			indegree[kind]++
			dependents[dependency] = append(dependents[dependency], kind)
		}
	}

	for dependencyKind := range dependents {
		sort.Slice(dependents[dependencyKind], func(i int, j int) bool {
			return string(dependents[dependencyKind][i]) < string(dependents[dependencyKind][j])
		})
	}

	ready := make([]ResourceKind, 0, len(projectors))
	for kind, count := range indegree {
		if count == 0 {
			ready = append(ready, kind)
		}
	}
	sort.Slice(ready, func(i int, j int) bool {
		return string(ready[i]) < string(ready[j])
	})

	order := make([]ResourceKind, 0, len(projectors))
	for len(ready) > 0 {
		kind := ready[0]
		ready = ready[1:]
		order = append(order, kind)

		for _, dependent := range dependents[kind] {
			indegree[dependent]--
			if indegree[dependent] == 0 {
				ready = append(ready, dependent)
			}
		}
		sort.Slice(ready, func(i int, j int) bool {
			return string(ready[i]) < string(ready[j])
		})
	}

	rank := make(map[ResourceKind]int, len(projectors))
	for idx, kind := range order {
		rank[kind] = idx
	}

	if len(order) != len(projectors) {
		return nil, dependents, rank, fmt.Errorf("%w: reconcile projector dependency cycle detected", ErrValidation)
	}

	for dependencyKind := range dependents {
		sort.Slice(dependents[dependencyKind], func(i int, j int) bool {
			left := dependents[dependencyKind][i]
			right := dependents[dependencyKind][j]
			return rank[left] < rank[right]
		})
	}

	return order, dependents, rank, nil
}

func (d *reconcileDriver) Trigger(ctx context.Context, kind ResourceKind, reason ReconcileReason) error {
	if ctx == nil {
		return errors.New("resources: reconcile trigger context is required")
	}
	if d.topoErr != nil {
		return d.topoErr
	}

	normalizedKind := kind.Normalize()
	if err := normalizedKind.Validate("kind"); err != nil {
		return err
	}
	normalizedReason := reason.Normalize()
	if err := normalizedReason.Validate("reason"); err != nil {
		return err
	}

	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return errors.New("resources: reconcile driver is closed")
	}
	if _, ok := d.projectors[normalizedKind]; !ok {
		d.mu.Unlock()
		return fmt.Errorf("%w: reconcile kind %q is not registered", ErrValidation, normalizedKind)
	}

	scheduledKinds := d.scheduleCascade(normalizedKind)
	events := make([]ReconcileEvent, 0, len(scheduledKinds)*2)
	for idx, scheduledKind := range scheduledKinds {
		scheduledReason := normalizedReason
		if idx > 0 {
			scheduledReason = ReconcileReasonDependency
		}
		events = append(events, ReconcileEvent{
			Type:   ReconcileEventRequested,
			Kind:   scheduledKind,
			Reason: scheduledReason,
		})
		if coalesced := d.enqueueLocked(scheduledKind, scheduledReason); coalesced != nil {
			events = append(events, *coalesced)
		}
	}
	d.mu.Unlock()

	for _, event := range events {
		d.emitEvent(ctx, event)
	}
	d.notify()
	return nil
}

func (d *reconcileDriver) RunBoot(ctx context.Context) error {
	if ctx == nil {
		return errors.New("resources: reconcile boot context is required")
	}
	if d.topoErr != nil {
		return d.topoErr
	}

	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return errors.New("resources: reconcile driver is closed")
	}
	order := append([]ResourceKind(nil), d.bootOrder...)
	d.mu.Unlock()

	for _, kind := range order {
		d.emitEvent(ctx, ReconcileEvent{
			Type:   ReconcileEventRequested,
			Kind:   kind,
			Reason: ReconcileReasonBoot,
		})
		result := d.runPass(ctx, kind, ReconcileReasonBoot)
		if result.err != nil {
			d.handleBootFailure(ctx, kind, result)
			return result.err
		}
		d.handleBootSuccess(ctx, kind, result)
	}
	return nil
}

func (d *reconcileDriver) Close(ctx context.Context) error {
	if ctx == nil {
		return errors.New("resources: reconcile close context is required")
	}

	d.mu.Lock()
	if !d.closed {
		d.closed = true
		d.queue = nil
		for _, state := range d.kindStates {
			state.pending = false
			state.dirty = false
			state.pendingReason = ""
			state.dirtyReason = ""
			state.readyAt = time.Time{}
		}
		d.workerCancel()
		d.notifyLocked()
	}
	doneCh := d.doneCh
	d.mu.Unlock()

	select {
	case <-doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *reconcileDriver) scheduleCascade(root ResourceKind) []ResourceKind {
	ordered := []ResourceKind{root}
	visited := map[ResourceKind]struct{}{
		root: {},
	}
	var reachable []ResourceKind
	queue := append([]ResourceKind(nil), d.dependents[root]...)
	for len(queue) > 0 {
		kind := queue[0]
		queue = queue[1:]
		if _, seen := visited[kind]; seen {
			continue
		}
		visited[kind] = struct{}{}
		reachable = append(reachable, kind)
		queue = append(queue, d.dependents[kind]...)
	}

	sort.Slice(reachable, func(i int, j int) bool {
		left := reachable[i]
		right := reachable[j]
		leftRank, leftOK := d.topoRank[left]
		rightRank, rightOK := d.topoRank[right]
		switch {
		case leftOK && rightOK:
			return leftRank < rightRank
		case leftOK:
			return true
		case rightOK:
			return false
		default:
			return string(left) < string(right)
		}
	})

	return append(ordered, reachable...)
}

func (d *reconcileDriver) enqueueLocked(kind ResourceKind, reason ReconcileReason) *ReconcileEvent {
	state := d.kindStates[kind]
	if state == nil {
		return nil
	}

	if state.running {
		state.dirty = true
		state.dirtyReason = reason
		return &ReconcileEvent{
			Type:   ReconcileEventCoalesced,
			Kind:   kind,
			Reason: reason,
		}
	}

	if state.pending {
		if reason == ReconcileReasonWrite && state.degradedUntil.After(d.now()) {
			state.degradedUntil = time.Time{}
			state.readyAt = time.Time{}
		}
		state.pendingReason = reason
		return &ReconcileEvent{
			Type:   ReconcileEventCoalesced,
			Kind:   kind,
			Reason: reason,
		}
	}

	state.pending = true
	state.pendingReason = reason
	state.readyAt = time.Time{}
	d.queue = append(d.queue, kind)
	return nil
}

func (d *reconcileDriver) run() {
	defer close(d.doneCh)

	for {
		kind, reason, waitUntil, ok := d.nextPending()
		if !ok {
			if d.shouldExit() {
				return
			}
			if !d.waitForWork(waitUntil) {
				return
			}
			continue
		}

		result := d.runPass(d.workerCtx, kind, reason)
		d.finishAsyncPass(kind, result)
	}
}

func (d *reconcileDriver) shouldExit() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.closed {
		return false
	}
	for _, state := range d.kindStates {
		if state.running {
			return false
		}
	}
	return true
}

func (d *reconcileDriver) nextPending() (ResourceKind, ReconcileReason, time.Time, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := d.now()
	var earliest time.Time
	for idx, kind := range d.queue {
		state := d.kindStates[kind]
		if state == nil || !state.pending {
			continue
		}

		notBefore := state.readyAt
		if state.degradedUntil.After(notBefore) {
			notBefore = state.degradedUntil
		}
		if notBefore.After(now) {
			if earliest.IsZero() || notBefore.Before(earliest) {
				earliest = notBefore
			}
			continue
		}

		d.queue = append(d.queue[:idx], d.queue[idx+1:]...)
		state.pending = false
		state.running = true
		reason := state.pendingReason
		state.pendingReason = ""
		return kind, reason, time.Time{}, true
	}

	return "", "", earliest, false
}

func (d *reconcileDriver) waitForWork(waitUntil time.Time) bool {
	if waitUntil.IsZero() {
		select {
		case <-d.notifyCh:
			return true
		case <-d.workerCtx.Done():
			return false
		}
	}

	delay := time.Until(waitUntil)
	if delay <= 0 {
		return true
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-d.notifyCh:
		return true
	case <-d.workerCtx.Done():
		return false
	}
}

func (d *reconcileDriver) runPass(
	ctx context.Context,
	kind ResourceKind,
	reason ReconcileReason,
) reconcilePassResult {
	startedAt := d.now()

	input, err := d.buildProjectionInput(ctx, kind)
	if err != nil {
		return reconcilePassResult{
			reason:   reason,
			duration: d.now().Sub(startedAt),
			err:      err,
		}
	}

	timeout := d.defaultTimeout
	if override, ok := d.kindTimeouts[kind]; ok && override > 0 {
		timeout = override
	}

	passCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	projector := d.projectors[kind]
	plan, err := projector.Build(passCtx, input)
	if err == nil {
		err = projector.Apply(passCtx, plan)
	}
	if err != nil {
		return reconcilePassResult{
			reason:   reason,
			duration: d.now().Sub(startedAt),
			err:      err,
		}
	}

	return reconcilePassResult{
		reason:     reason,
		revision:   plan.Revision(),
		operations: plan.OperationCount(),
		duration:   d.now().Sub(startedAt),
	}
}

func (d *reconcileDriver) buildProjectionInput(ctx context.Context, kind ResourceKind) (projectionInput, error) {
	if d.raw == nil {
		return projectionInput{}, errors.New("resources: raw store is required for registered projectors")
	}

	records, err := d.raw.ListRaw(ctx, d.actor, ResourceFilter{Kind: kind})
	if err != nil {
		return projectionInput{}, fmt.Errorf("resources: list reconcile records for %q: %w", kind, err)
	}

	projector := d.projectors[kind]
	dependencies := make(map[ResourceKind][]RawRecord, len(projector.DependsOn()))
	for _, dependency := range normalizeKinds(projector.DependsOn()) {
		dependencyRecords, depErr := d.raw.ListRaw(ctx, d.actor, ResourceFilter{Kind: dependency})
		if depErr != nil {
			return projectionInput{}, fmt.Errorf(
				"resources: list reconcile dependency records for %q -> %q: %w",
				kind,
				dependency,
				depErr,
			)
		}
		dependencies[dependency] = dependencyRecords
	}

	return projectionInput{
		kind:         kind,
		revision:     maxRecordVersion(records),
		records:      records,
		dependencies: dependencies,
	}, nil
}

func maxRecordVersion(records []RawRecord) int64 {
	var revision int64
	for _, record := range records {
		if record.Version > revision {
			revision = record.Version
		}
	}
	return revision
}

func (d *reconcileDriver) finishAsyncPass(kind ResourceKind, result reconcilePassResult) {
	health, emitDegraded, ok := d.updateAsyncPassState(kind, result)
	if !ok {
		return
	}

	if result.err != nil {
		d.recordAsyncFailure(kind, result, health, emitDegraded)
		d.notify()
		return
	}

	d.recordAsyncSuccess(kind, result, health)
	d.notify()
}

func (d *reconcileDriver) updateAsyncPassState(
	kind ResourceKind,
	result reconcilePassResult,
) (ReconcileHealth, bool, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	state := d.kindStates[kind]
	if state == nil {
		return ReconcileHealth{}, false, false
	}
	state.running = false

	dirty := state.dirty
	dirtyReason := state.dirtyReason
	state.dirty = false
	state.dirtyReason = ""

	if result.err != nil {
		state.consecutiveFailures++
		if state.consecutiveFailures >= d.failureThreshold {
			state.degradedUntil = d.now().Add(d.degradedBackoff)
		}
	} else {
		state.consecutiveFailures = 0
		state.degradedUntil = time.Time{}
	}

	var health ReconcileHealth
	var emitDegraded bool
	if result.err != nil {
		health = ReconcileHealth{
			Kind:                kind,
			Status:              ReconcileHealthStatusFailing,
			ConsecutiveFailures: state.consecutiveFailures,
			LastError:           result.err,
		}
		if !state.degradedUntil.IsZero() {
			health.Status = ReconcileHealthStatusDegraded
			health.DegradedUntil = state.degradedUntil
			emitDegraded = true
		}
	} else {
		health = ReconcileHealth{
			Kind:   kind,
			Status: ReconcileHealthStatusHealthy,
		}
	}

	if dirty && !d.closed {
		state.pending = true
		state.pendingReason = dirtyReason
		state.readyAt = d.now().Add(d.coalesceWindow)
		d.queue = append([]ResourceKind{kind}, d.queue...)
	}

	return health, emitDegraded, true
}

func (d *reconcileDriver) recordAsyncFailure(
	kind ResourceKind,
	result reconcilePassResult,
	health ReconcileHealth,
	emitDegraded bool,
) {
	d.logger.Error(
		"resources: reconcile pass failed",
		"resource_kind",
		kind,
		"reconcile_reason",
		result.reason,
		"consecutive_failures",
		health.ConsecutiveFailures,
		"degraded_until",
		health.DegradedUntil,
		"error",
		result.err,
	)
	d.emitEvent(context.Background(), ReconcileEvent{
		Type:                ReconcileEventFailed,
		Kind:                kind,
		Reason:              result.reason,
		Duration:            result.duration,
		ConsecutiveFailures: health.ConsecutiveFailures,
		Err:                 result.err,
	})
	if emitDegraded {
		d.emitEvent(context.Background(), ReconcileEvent{
			Type:                ReconcileEventDegraded,
			Kind:                kind,
			Reason:              result.reason,
			ConsecutiveFailures: health.ConsecutiveFailures,
			DegradedUntil:       health.DegradedUntil,
			Err:                 result.err,
		})
	}
	d.reportHealth(context.Background(), health)
}

func (d *reconcileDriver) recordAsyncSuccess(
	kind ResourceKind,
	result reconcilePassResult,
	health ReconcileHealth,
) {
	d.logger.Debug(
		"resources: reconcile pass applied",
		"resource_kind",
		kind,
		"reconcile_reason",
		result.reason,
		"revision",
		result.revision,
		"operations",
		result.operations,
		"duration",
		result.duration,
	)
	d.emitEvent(context.Background(), ReconcileEvent{
		Type:       ReconcileEventApplied,
		Kind:       kind,
		Reason:     result.reason,
		Duration:   result.duration,
		Revision:   result.revision,
		Operations: result.operations,
	})
	d.reportHealth(context.Background(), health)
}

func (d *reconcileDriver) handleBootFailure(ctx context.Context, kind ResourceKind, result reconcilePassResult) {
	consecutiveFailures := 1
	degradedUntil := time.Time{}
	if d.failureThreshold <= 1 {
		degradedUntil = d.now().Add(d.degradedBackoff)
	}

	status := ReconcileHealthStatusFailing
	if !degradedUntil.IsZero() {
		status = ReconcileHealthStatusDegraded
	}
	health := ReconcileHealth{
		Kind:                kind,
		Status:              status,
		ConsecutiveFailures: consecutiveFailures,
		DegradedUntil:       degradedUntil,
		LastError:           result.err,
	}

	d.logger.Error(
		"resources: boot reconcile failed",
		"resource_kind",
		kind,
		"reconcile_reason",
		result.reason,
		"degraded_until",
		degradedUntil,
		"error",
		result.err,
	)
	d.emitEvent(ctx, ReconcileEvent{
		Type:                ReconcileEventFailed,
		Kind:                kind,
		Reason:              result.reason,
		Duration:            result.duration,
		ConsecutiveFailures: consecutiveFailures,
		Err:                 result.err,
	})
	if !degradedUntil.IsZero() {
		d.emitEvent(ctx, ReconcileEvent{
			Type:                ReconcileEventDegraded,
			Kind:                kind,
			Reason:              result.reason,
			ConsecutiveFailures: consecutiveFailures,
			DegradedUntil:       degradedUntil,
			Err:                 result.err,
		})
	}
	d.reportHealth(ctx, health)
}

func (d *reconcileDriver) handleBootSuccess(ctx context.Context, kind ResourceKind, result reconcilePassResult) {
	d.logger.Debug(
		"resources: boot reconcile applied",
		"resource_kind",
		kind,
		"reconcile_reason",
		result.reason,
		"revision",
		result.revision,
		"operations",
		result.operations,
		"duration",
		result.duration,
	)
	d.emitEvent(ctx, ReconcileEvent{
		Type:       ReconcileEventApplied,
		Kind:       kind,
		Reason:     result.reason,
		Duration:   result.duration,
		Revision:   result.revision,
		Operations: result.operations,
	})
	d.reportHealth(ctx, ReconcileHealth{
		Kind:   kind,
		Status: ReconcileHealthStatusHealthy,
	})
}

func (d *reconcileDriver) emitEvent(ctx context.Context, event ReconcileEvent) {
	if d.eventSink == nil {
		return
	}
	d.eventSink.ObserveReconcileEvent(ctx, event)
}

func (d *reconcileDriver) reportHealth(ctx context.Context, health ReconcileHealth) {
	if d.healthSink == nil {
		return
	}
	d.healthSink.ReportReconcileHealth(ctx, health)
}

func (d *reconcileDriver) notify() {
	select {
	case d.notifyCh <- struct{}{}:
	default:
	}
}

func (d *reconcileDriver) notifyLocked() {
	select {
	case d.notifyCh <- struct{}{}:
	default:
	}
}
