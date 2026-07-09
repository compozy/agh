package loop

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/compozy/agh/internal/hooks"
	"github.com/compozy/agh/internal/loop/dsl"
	watchpkg "github.com/compozy/agh/internal/loop/watch"
	"github.com/compozy/agh/internal/task"
)

const (
	watchEventsNotReadyReason        = "watch_events_not_ready"
	watchEventsSpecInvalidReason     = "watch_events_spec_invalid"
	watchEventsLedgerUnavailableCode = "watch_events_ledger_unavailable"
	watchEventsFilterReason          = "watch_events_filter"
)

type coordinatorWatchEventsRuntime struct {
	ledger       WatchEventsLedger
	cursorReader WatchEventsCursorReader
	now          func() time.Time
	logger       *slog.Logger
}

func (r *CoordinatorRunner) watchEventsRuntime() coordinatorWatchEventsRuntime {
	now := r.now
	if now == nil {
		now = time.Now
	}
	var cursorReader WatchEventsCursorReader
	if reader, ok := r.watchEventsLedger.(WatchEventsCursorReader); ok {
		cursorReader = reader
	}
	return coordinatorWatchEventsRuntime{
		ledger:       r.watchEventsLedger,
		cursorReader: cursorReader,
		now:          now,
		logger:       r.logger,
	}
}

func isWatchEventsNode(node dsl.Node) bool {
	return node.Class == dsl.NodeClassSource && node.Kind == string(dsl.SourceWatchEvents)
}

func evaluateWatchEventsNode(
	ctx context.Context,
	plan *task.CoordinatorCompletionPlan,
	run Run,
	generation int,
	resolved *ResolvedDefinition,
	topology controlTopology,
	output GenerationOutput,
	node dsl.Node,
	outputs []GenerationOutput,
	outputBlobs *[]GenerationOutputBlob,
	runtime coordinatorWatchEventsRuntime,
) (GenerationOutput, *task.CoordinatorTerminal, error) {
	if runtime.ledger == nil || runtime.cursorReader == nil {
		return output, watchEventsBlockedTerminal(watchEventsLedgerUnavailableCode), nil
	}
	state, terminal, err := recoverWatchEventsState(ctx, run, output, node, runtime)
	if err != nil || terminal != nil {
		return output, terminal, err
	}
	query, err := watchEventsQuery(run, state)
	if err != nil {
		return output, watchEventsBlockedTerminal(watchEventsSpecInvalidReason), nil
	}
	rows, err := runtime.ledger.ReadMatches(ctx, query)
	if err != nil {
		return GenerationOutput{}, nil, fmt.Errorf("loop: read watch-events matches for node %s: %w", node.ID, err)
	}
	matches, nextCursors, readCounts, err := filterWatchEventsRows(
		run,
		generation,
		resolved,
		topology,
		output,
		node,
		outputs,
		state,
		rows,
	)
	if err != nil {
		return GenerationOutput{}, nil, err
	}
	if len(matches) == 0 {
		ref, err := watchpkg.EventsPendingOutputRef(state)
		if err != nil {
			return GenerationOutput{}, nil, err
		}
		output.OutputRef = ref
		if run.Status == StatusWatching {
			plan.Yield = true
			return output, nil, nil
		}
		return output, watchEventsWaitingTerminal(), nil
	}

	ref, err := confirmedWatchEventsOutputRef(matches, nextCursors, outputBlobs, runtime.now().UTC())
	if err != nil {
		return GenerationOutput{}, nil, err
	}
	output.Status = generationOutputSucceeded
	output.OutputRef = ref
	if watchEventsReadMayBeTruncated(readCounts, query.Limit) {
		plan.PostCommitWakes = append(plan.PostCommitWakes, task.CoordinatorWakeSpec{
			LoopRunID:      string(run.ID),
			IdempotencyKey: watchEventsWakeKey(run.ID, node.ID),
		})
	}
	logWatchEventsEvaluation(runtime, run, node, len(rows), len(matches), nextCursors)
	return output, nil, nil
}

func recoverWatchEventsState(
	ctx context.Context,
	run Run,
	output GenerationOutput,
	node dsl.Node,
	runtime coordinatorWatchEventsRuntime,
) (watchpkg.EventsPendingState, *task.CoordinatorTerminal, error) {
	if state, ok, err := watchpkg.EventsPendingFromOutputRef(output.OutputRef); err != nil {
		return watchpkg.EventsPendingState{}, nil, err
	} else if ok {
		if err := validateWatchEventsPendingState(state); err != nil {
			return watchpkg.EventsPendingState{}, watchEventsBlockedTerminal(watchEventsSpecInvalidReason), nil
		}
		return state, nil, nil
	}
	subscriptions, err := watchEventsSubscriptionsFromNode(node)
	if err != nil {
		return watchpkg.EventsPendingState{}, watchEventsBlockedTerminal(watchEventsSpecInvalidReason), nil
	}
	streams, kinds, err := watchEventsInitialStreamsAndKinds(subscriptions, run.Inputs)
	if err != nil {
		return watchpkg.EventsPendingState{}, watchEventsBlockedTerminal(watchEventsSpecInvalidReason), nil
	}
	cursors, err := runtime.cursorReader.ReadCursors(ctx, WatchEventsQuery{
		WorkspaceID: string(run.WorkspaceID),
		Streams:     streams,
		Kinds:       kinds,
		Limit:       LoopMaxFanoutWidth,
	})
	if err != nil {
		return watchpkg.EventsPendingState{}, nil, fmt.Errorf(
			"loop: read watch-events cursors for node %s: %w",
			node.ID,
			err,
		)
	}
	return watchpkg.EventsPendingState{
		Subscriptions: subscriptions,
		Cursors:       cursors,
	}, nil, nil
}

func watchEventsSubscriptionsFromNode(node dsl.Node) ([]watchpkg.EventSubscriptionRef, error) {
	if len(node.Events) == 0 {
		return nil, fmt.Errorf("%w: watch-events node %q requires subscriptions", ErrValidation, node.ID)
	}
	supported := SupportedWatchEvents()
	subscriptions := make([]watchpkg.EventSubscriptionRef, 0, len(node.Events))
	for _, subscription := range node.Events {
		kind := hooks.HookEvent(strings.TrimSpace(subscription.Kind))
		if _, ok := supported[kind]; !ok {
			return nil, fmt.Errorf("%w: watch-events kind is unsupported: %q", ErrValidation, subscription.Kind)
		}
		subscriptions = append(subscriptions, watchpkg.EventSubscriptionRef{
			Kind:   string(kind),
			Filter: strings.TrimSpace(subscription.Filter),
		})
	}
	return subscriptions, nil
}

func validateWatchEventsPendingState(state watchpkg.EventsPendingState) error {
	if len(state.Subscriptions) == 0 {
		return fmt.Errorf("%w: watch-events pending subscriptions are required", ErrValidation)
	}
	kinds, err := watchEventsLedgerKinds(state.Subscriptions)
	if err != nil {
		return err
	}
	for stream, cursor := range state.Cursors {
		if strings.TrimSpace(stream) == "" {
			return fmt.Errorf("%w: watch-events cursor stream is required", ErrValidation)
		}
		if cursor < 0 {
			return fmt.Errorf("%w: watch-events cursor for %q must be non-negative", ErrValidation, stream)
		}
	}
	for _, kind := range kinds {
		if !watchEventsPendingStateHasCursorForKind(state, kind) {
			return fmt.Errorf("%w: watch-events cursor for kind %q is required", ErrValidation, kind)
		}
	}
	return nil
}

func watchEventsInitialStreamsAndKinds(
	subscriptions []watchpkg.EventSubscriptionRef,
	inputs map[string]any,
) (map[string]int64, []string, error) {
	streams, err := watchEventsInitialStreams(subscriptions, inputs)
	if err != nil {
		return nil, nil, err
	}
	kinds, err := watchEventsLedgerKinds(subscriptions)
	if err != nil {
		return nil, nil, err
	}
	return streams, kinds, nil
}

func watchEventsInitialStreams(
	subscriptions []watchpkg.EventSubscriptionRef,
	inputs map[string]any,
) (map[string]int64, error) {
	supported := SupportedWatchEvents()
	streams := map[string]int64{}
	for _, subscription := range subscriptions {
		contract, ok := supported[hooks.HookEvent(strings.TrimSpace(subscription.Kind))]
		if !ok {
			return nil, fmt.Errorf("%w: watch-events kind is unsupported: %q", ErrValidation, subscription.Kind)
		}
		streamKeys, err := watchEventsStreamsForSubscription(subscription, contract, inputs)
		if err != nil {
			return nil, err
		}
		for _, stream := range streamKeys {
			streams[stream] = 0
		}
	}
	return streams, nil
}

func watchEventsStreamsForSubscription(
	subscription watchpkg.EventSubscriptionRef,
	contract WatchEventsContract,
	inputs map[string]any,
) ([]string, error) {
	if contract.Stream != watchEventsSessionStream {
		return []string{contract.Stream}, nil
	}
	sessionIDs, ok, err := watchEventsSessionIDsFromFilter(subscription.Filter, inputs, true)
	if err != nil {
		return nil, err
	}
	if !ok || len(sessionIDs) == 0 {
		return nil, fmt.Errorf(
			"%w: watch-events kind %q requires a session_id filter",
			ErrValidation,
			subscription.Kind,
		)
	}
	streams := make([]string, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		stream := WatchEventsSessionStreamForSession(sessionID)
		if stream != "" {
			streams = append(streams, stream)
		}
	}
	if len(streams) == 0 {
		return nil, fmt.Errorf("%w: watch-events session_id filter is empty", ErrValidation)
	}
	slices.Sort(streams)
	return streams, nil
}

func watchEventsLedgerKinds(subscriptions []watchpkg.EventSubscriptionRef) ([]string, error) {
	supported := SupportedWatchEvents()
	kindSet := map[string]struct{}{}
	for _, subscription := range subscriptions {
		contract, ok := supported[hooks.HookEvent(strings.TrimSpace(subscription.Kind))]
		if !ok {
			return nil, fmt.Errorf("%w: watch-events kind is unsupported: %q", ErrValidation, subscription.Kind)
		}
		for _, ledgerType := range contract.LedgerTypes {
			trimmed := strings.TrimSpace(ledgerType)
			if trimmed != "" {
				kindSet[trimmed] = struct{}{}
			}
		}
	}
	kinds := make([]string, 0, len(kindSet))
	for kind := range kindSet {
		kinds = append(kinds, kind)
	}
	slices.Sort(kinds)
	return kinds, nil
}

func watchEventsPendingStateHasCursorForKind(state watchpkg.EventsPendingState, ledgerKind string) bool {
	supported := SupportedWatchEvents()
	for _, subscription := range state.Subscriptions {
		contract, ok := supported[hooks.HookEvent(strings.TrimSpace(subscription.Kind))]
		if !ok || !slices.Contains(contract.LedgerTypes, ledgerKind) {
			continue
		}
		for stream := range state.Cursors {
			if watchEventsContractStreamMatches(contract.Stream, stream) {
				return true
			}
		}
	}
	return false
}

func watchEventsQuery(run Run, state watchpkg.EventsPendingState) (WatchEventsQuery, error) {
	kinds, err := watchEventsLedgerKinds(state.Subscriptions)
	if err != nil {
		return WatchEventsQuery{}, err
	}
	return WatchEventsQuery{
		WorkspaceID: string(run.WorkspaceID),
		Streams:     cloneInt64Map(state.Cursors),
		Kinds:       kinds,
		Limit:       LoopMaxFanoutWidth,
	}, nil
}

func filterWatchEventsRows(
	run Run,
	generation int,
	resolved *ResolvedDefinition,
	topology controlTopology,
	output GenerationOutput,
	node dsl.Node,
	outputs []GenerationOutput,
	state watchpkg.EventsPendingState,
	rows []WatchEvent,
) ([]WatchEvent, map[string]int64, map[string]int, error) {
	nextCursors := cloneInt64Map(state.Cursors)
	readCounts := map[string]int{}
	matches := make([]WatchEvent, 0, len(rows))
	for _, row := range rows {
		stream := strings.TrimSpace(row.Stream)
		readCounts[stream]++
		if row.Seq > nextCursors[stream] {
			nextCursors[stream] = row.Seq
		}
		matched, event, err := rowMatchesWatchEventsSubscriptions(
			run,
			generation,
			resolved,
			topology,
			output,
			node,
			outputs,
			state.Subscriptions,
			row,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		if matched {
			matches = append(matches, event)
		}
	}
	if len(matches) == 0 {
		return nil, cloneInt64Map(state.Cursors), readCounts, nil
	}
	return matches, nextCursors, readCounts, nil
}

func rowMatchesWatchEventsSubscriptions(
	run Run,
	generation int,
	resolved *ResolvedDefinition,
	topology controlTopology,
	output GenerationOutput,
	node dsl.Node,
	outputs []GenerationOutput,
	subscriptions []watchpkg.EventSubscriptionRef,
	row WatchEvent,
) (bool, WatchEvent, error) {
	for idx, subscription := range subscriptions {
		contract, ok := SupportedWatchEvents()[hooks.HookEvent(strings.TrimSpace(subscription.Kind))]
		if !ok {
			return false, WatchEvent{}, fmt.Errorf(
				"%w: watch-events kind is unsupported: %q",
				ErrValidation,
				subscription.Kind,
			)
		}
		if !watchEventsContractStreamMatches(contract.Stream, row.Stream) ||
			!slices.Contains(contract.LedgerTypes, row.ledgerKind()) {
			continue
		}
		event := cloneWatchEvent(row)
		event.Kind = strings.TrimSpace(subscription.Kind)
		if strings.TrimSpace(subscription.Filter) == "" {
			return true, event, nil
		}
		matched, err := evaluateWatchEventsFilter(
			run,
			generation,
			resolved,
			topology,
			output,
			node,
			outputs,
			idx,
			event,
		)
		if err != nil {
			return false, WatchEvent{}, err
		}
		if matched {
			return true, event, nil
		}
	}
	return false, WatchEvent{}, nil
}

func watchEventsContractStreamMatches(contractStream string, rowStream string) bool {
	return strings.TrimSpace(contractStream) == WatchEventsBaseStream(rowStream)
}

func evaluateWatchEventsFilter(
	run Run,
	generation int,
	resolved *ResolvedDefinition,
	topology controlTopology,
	output GenerationOutput,
	node dsl.Node,
	outputs []GenerationOutput,
	subscriptionIndex int,
	event WatchEvent,
) (bool, error) {
	key := fmt.Sprintf("nodes.%s.events.%d.filter", node.ID, subscriptionIndex)
	condition := resolved.Conditions[key]
	if condition == nil {
		return false, fmt.Errorf("%w: compiled watch-events filter %q is missing", ErrValidation, key)
	}
	namespace, err := runtimeNamespace(
		run,
		generation,
		resolved.Definition.Graph,
		topology,
		outputs,
		node.ID,
		output.ItemIndex,
	)
	if err != nil {
		return false, err
	}
	namespace["event"] = event.eventMap()
	value, _, err := condition.Program.Eval(namespace)
	if err != nil {
		return false, fmt.Errorf("loop: evaluate watch-events filter %s: %w", node.ID, err)
	}
	result, ok := value.Value().(bool)
	if !ok {
		return false, fmt.Errorf(
			"%w: watch-events filter %s returned %T",
			ErrValidation,
			key,
			value.Value(),
		)
	}
	return result, nil
}

func confirmedWatchEventsOutputRef(
	events []WatchEvent,
	cursors map[string]int64,
	outputBlobs *[]GenerationOutputBlob,
	at time.Time,
) (string, error) {
	payload, err := watchpkg.EventsConfirmedOutputPayload(events, cursors)
	if err != nil {
		return "", err
	}
	if !OutputPayloadRequiresRef(payload) {
		return string(payload), nil
	}
	if outputBlobs == nil {
		return "", fmt.Errorf("%w: output blob sink is required", ErrValidation)
	}
	ref := OutputRefForPayload(payload)
	*outputBlobs = append(*outputBlobs, GenerationOutputBlob{
		OutputRef: ref,
		Payload:   payload,
		At:        at,
	})
	return ref, nil
}

func watchEventsReadMayBeTruncated(readCounts map[string]int, limit int) bool {
	if limit <= 0 {
		return false
	}
	for _, count := range readCounts {
		if count >= limit {
			return true
		}
	}
	return false
}

func watchEventsWakeKey(runID RunID, nodeID dsl.NodeID) string {
	return fmt.Sprintf("loop.coordinator.watch_events.%s.%s", runID, nodeID)
}

func watchEventsBlockedTerminal(reasonCode string) *task.CoordinatorTerminal {
	return &task.CoordinatorTerminal{
		Status:     string(StatusBlocked),
		Cause:      string(TransitionCauseContract),
		ReasonCode: reasonCode,
	}
}

func watchEventsWaitingTerminal() *task.CoordinatorTerminal {
	return &task.CoordinatorTerminal{
		Status:     string(StatusWatching),
		Cause:      string(TransitionCauseWatchEvents),
		ReasonCode: watchEventsNotReadyReason,
	}
}

func cloneWatchEvent(src WatchEvent) WatchEvent {
	src.Payload = cloneAnyMap(src.Payload)
	return src
}

func cloneInt64Map(src map[string]int64) map[string]int64 {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]int64, len(src))
	maps.Copy(dst, src)
	return dst
}

func logWatchEventsEvaluation(
	runtime coordinatorWatchEventsRuntime,
	run Run,
	node dsl.Node,
	readCount int,
	matchCount int,
	cursors map[string]int64,
) {
	if runtime.logger == nil {
		return
	}
	runtime.logger.Debug(
		"loop watch-events evaluation",
		"loop_run_id", string(run.ID),
		"workspace_id", string(run.WorkspaceID),
		"node_id", string(node.ID),
		"read_count", readCount,
		"match_count", matchCount,
		"cursors", cursors,
	)
}

var _ WatchEventsLedger = (*watchEventsLedgerFunc)(nil)

type watchEventsLedgerFunc struct {
	read func(context.Context, WatchEventsQuery) ([]WatchEvent, error)
}

func (f *watchEventsLedgerFunc) ReadMatches(ctx context.Context, query WatchEventsQuery) ([]WatchEvent, error) {
	return f.read(ctx, query)
}
