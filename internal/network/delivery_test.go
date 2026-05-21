package network

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
)

func TestInboundQueuePreservesFIFOAndDropsOldestOnOverflow(t *testing.T) {
	t.Parallel()

	queue := newInboundQueue(2)
	first := testDeliveryEnvelope(t, "msg-1", "first")
	second := testDeliveryEnvelope(t, "msg-2", "second")
	third := testDeliveryEnvelope(t, "msg-3", "third")
	now := time.Date(2026, 4, 11, 13, 0, 0, 0, time.UTC)

	if result := queue.enqueue(first, now, false); result.Dropped != nil || result.Depth != 1 {
		t.Fatalf("enqueue(first) = %#v, want depth=1 with no drop", result)
	}
	if result := queue.enqueue(second, now.Add(time.Second), false); result.Dropped != nil || result.Depth != 2 {
		t.Fatalf("enqueue(second) = %#v, want depth=2 with no drop", result)
	}
	if result := queue.enqueue(
		third,
		now.Add(2*time.Second),
		true,
	); result.Dropped == nil || result.Dropped.ID != first.ID ||
		result.Depth != 2 {
		t.Fatalf("enqueue(third) = %#v, want drop=%q depth=2", result, first.ID)
	}

	gotFirst, ok := queue.dequeue()
	if !ok {
		t.Fatal("dequeue(first) = missing, want second envelope")
	}
	gotSecond, ok := queue.dequeue()
	if !ok {
		t.Fatal("dequeue(second) = missing, want third envelope")
	}
	if _, ok := queue.dequeue(); ok {
		t.Fatal("dequeue(third) = present, want queue empty")
	}

	if gotFirst.Envelope.ID != second.ID {
		t.Fatalf("first dequeue id = %q, want %q", gotFirst.Envelope.ID, second.ID)
	}
	if gotSecond.Envelope.ID != third.ID {
		t.Fatalf("second dequeue id = %q, want %q", gotSecond.Envelope.ID, third.ID)
	}
}

func TestFormatNetworkMessageEscapesPreviewAndPreservesCanonicalBody(t *testing.T) {
	t.Parallel()

	envelope := Envelope{
		Protocol:    ProtocolV0,
		WorkspaceID: testWorkspaceID,
		ID:          "msg-direct-01",
		Kind:        KindSay,
		Channel:     "builders",
		Surface:     new(SurfaceDirect),
		DirectID:    new("direct_0123456789abcdef0123456789abcdef"),
		From:        "coder.sess-abc",
		To:          new("reviewer.sess-xyz"),
		WorkID:      new("work_patch-42"),
		ReplyTo:     new("msg-root-00"),
		TraceID:     new("trace-patch-42"),
		CausationID: new("msg-cause-00"),
		TS:          time.Date(2026, 4, 11, 13, 0, 0, 0, time.UTC).Unix(),
		Body: mustRawJSON(t, map[string]any{
			"text":   `look at <auth.go> & run "rm -rf" 'now'`,
			"intent": "review_request",
		}),
	}

	rendered, err := formatNetworkMessage(envelope)
	if err != nil {
		t.Fatalf("formatNetworkMessage() error = %v", err)
	}

	if !strings.Contains(rendered, `trust="untrusted"`) {
		t.Fatalf("rendered message missing trust marker: %s", rendered)
	}
	if !strings.Contains(rendered, `surface="direct"`) {
		t.Fatalf("rendered message missing surface attribute: %s", rendered)
	}
	if !strings.Contains(rendered, `direct-id="direct_0123456789abcdef0123456789abcdef"`) {
		t.Fatalf("rendered message missing direct-id attribute: %s", rendered)
	}
	if !strings.Contains(rendered, `work-id="work_patch-42"`) {
		t.Fatalf("rendered message missing work attribute: %s", rendered)
	}
	if !strings.Contains(rendered, `to="reviewer.sess-xyz"`) {
		t.Fatalf("rendered message missing target attribute: %s", rendered)
	}
	if !strings.Contains(rendered, `reply-to="msg-root-00"`) {
		t.Fatalf("rendered message missing reply-to attribute: %s", rendered)
	}
	if !strings.Contains(rendered, `trace-id="trace-patch-42"`) {
		t.Fatalf("rendered message missing trace-id attribute: %s", rendered)
	}
	if !strings.Contains(rendered, `causation-id="msg-cause-00"`) {
		t.Fatalf("rendered message missing causation-id attribute: %s", rendered)
	}
	escapedPreview := `look at &lt;auth.go&gt; &amp; run &quot;rm -rf&quot; &apos;now&apos;`
	if !strings.Contains(rendered, escapedPreview) {
		t.Fatalf("rendered preview missing escaped text:\n%s", rendered)
	}
	if strings.Contains(rendered, `<network-preview encoding="xml-escaped">look at <auth.go>`) {
		t.Fatalf("rendered preview leaked raw XML-breaking content:\n%s", rendered)
	}
	for _, snippet := range []string{
		"Prefer `agh__network_send` when available; otherwise use `agh network send` to respond.",
		`--causation-id "msg-direct-01"`,
		"--kind say",
		"--surface \"direct\"",
		"--direct \"direct_0123456789abcdef0123456789abcdef\"",
		"--kind receipt",
		`"for_id":"msg-direct-01"`,
		`"status":"accepted"`,
		"--kind trace",
		`"state":"working"`,
		"--kind capability",
		`"capability":{"id":"reply-workflow"`,
		"Direct-room chat uses `--kind say --surface direct`.",
		`--trace-id "trace-patch-42"`,
		"See `agh network --help` for options.",
	} {
		if !strings.Contains(rendered, snippet) {
			t.Fatalf("rendered message missing reply guidance snippet %q:\n%s", snippet, rendered)
		}
	}

	start := strings.Index(rendered, `<network-body encoding="base64-json">`)
	if start < 0 {
		t.Fatalf("rendered message missing network-body: %s", rendered)
	}
	start += len(`<network-body encoding="base64-json">`)
	end := strings.Index(rendered[start:], `</network-body>`)
	if end < 0 {
		t.Fatalf("rendered message missing closing network-body tag: %s", rendered)
	}
	encodedBody := rendered[start : start+end]
	decodedBody, err := base64.StdEncoding.DecodeString(encodedBody)
	if err != nil {
		t.Fatalf("DecodeString(network-body) error = %v", err)
	}

	wantBody, err := json.Marshal(SayBody{
		Text:   `look at <auth.go> & run "rm -rf" 'now'`,
		Intent: "review_request",
	})
	if err != nil {
		t.Fatalf("json.Marshal(wantBody) error = %v", err)
	}
	if !bytes.Equal(decodedBody, wantBody) {
		t.Fatalf("decoded body = %s, want %s", string(decodedBody), string(wantBody))
	}
}

func TestPromptNetworkMetaMatchesWrappedConversationFields(t *testing.T) {
	t.Parallel()

	t.Run("Should include direct container work correlation and trust", func(t *testing.T) {
		t.Parallel()

		envelope := Envelope{
			ID:          "msg-direct-meta",
			Kind:        KindTrace,
			Channel:     "builders",
			Surface:     new(SurfaceDirect),
			DirectID:    new("direct_0123456789abcdef0123456789abcdef"),
			From:        "reviewer.sess-xyz",
			To:          new("coder.sess-abc"),
			WorkID:      new("work_review_42"),
			ReplyTo:     new("msg-root-1"),
			TraceID:     new("trace-review-42"),
			CausationID: new("msg-root-1"),
		}

		meta := promptNetworkMeta(envelope)
		if got, want := meta.MessageID, envelope.ID; got != want {
			t.Fatalf("MessageID = %q, want %q", got, want)
		}
		if got, want := meta.Surface, string(SurfaceDirect); got != want {
			t.Fatalf("Surface = %q, want %q", got, want)
		}
		if got, want := meta.DirectID, "direct_0123456789abcdef0123456789abcdef"; got != want {
			t.Fatalf("DirectID = %q, want %q", got, want)
		}
		if got, want := meta.WorkID, "work_review_42"; got != want {
			t.Fatalf("WorkID = %q, want %q", got, want)
		}
		if got, want := meta.ReplyTo, "msg-root-1"; got != want {
			t.Fatalf("ReplyTo = %q, want %q", got, want)
		}
		if got, want := meta.TraceID, "trace-review-42"; got != want {
			t.Fatalf("TraceID = %q, want %q", got, want)
		}
		if got, want := meta.CausationID, "msg-root-1"; got != want {
			t.Fatalf("CausationID = %q, want %q", got, want)
		}
		if got, want := meta.Trust, networkMessageTrustUntrusted; got != want {
			t.Fatalf("Trust = %q, want %q", got, want)
		}
	})

	t.Run("Should omit work id when the message is not lifecycle-bearing", func(t *testing.T) {
		t.Parallel()

		envelope := Envelope{
			ID:      "msg-greet-meta",
			Kind:    KindGreet,
			Channel: "builders",
			From:    "reviewer.sess-xyz",
			WorkID:  new("work_invalid_42"),
			Body: mustRawJSON(t, GreetBody{
				PeerCard: PeerCard{
					PeerID:            "reviewer.sess-xyz",
					ProfilesSupported: []string{ProtocolV0},
				},
				Summary: "available",
			}),
		}

		meta := promptNetworkMeta(envelope)
		if meta.WorkID != "" {
			t.Fatalf("WorkID = %q, want omitted for non-lifecycle kind", meta.WorkID)
		}
		if got, want := meta.Trust, networkMessageTrustUntrusted; got != want {
			t.Fatalf("Trust = %q, want %q", got, want)
		}

		rendered, err := formatNetworkMessage(envelope)
		if err != nil {
			t.Fatalf("formatNetworkMessage() error = %v", err)
		}
		if strings.Contains(rendered, `work-id="work_invalid_42"`) {
			t.Fatalf("rendered message leaked non-lifecycle work-id:\n%s", rendered)
		}
	})
}

func TestFormatNetworkMessageFallsBackToCompactRawJSONWithoutPreview(t *testing.T) {
	t.Parallel()

	envelope := Envelope{
		Protocol:    ProtocolV0,
		WorkspaceID: testWorkspaceID,
		ID:          "msg-direct-raw",
		Kind:        KindSay,
		Channel:     "builders",
		Surface:     new(SurfaceDirect),
		DirectID:    new("direct_0123456789abcdef0123456789abcdef"),
		From:        "coder.sess-abc",
		To:          new("reviewer.sess-xyz"),
		TS:          time.Date(2026, 4, 11, 13, 5, 0, 0, time.UTC).Unix(),
		Body:        json.RawMessage(`["unexpected"]`),
	}

	rendered, err := formatNetworkMessage(envelope)
	if err != nil {
		t.Fatalf("formatNetworkMessage() error = %v", err)
	}
	if strings.Contains(rendered, "<network-preview") {
		t.Fatalf("rendered message unexpectedly included preview:\n%s", rendered)
	}

	start := strings.Index(rendered, `<network-body encoding="base64-json">`)
	if start < 0 {
		t.Fatalf("rendered message missing network-body: %s", rendered)
	}
	start += len(`<network-body encoding="base64-json">`)
	end := strings.Index(rendered[start:], `</network-body>`)
	if end < 0 {
		t.Fatalf("rendered message missing closing network-body tag: %s", rendered)
	}

	decodedBody, err := base64.StdEncoding.DecodeString(rendered[start : start+end])
	if err != nil {
		t.Fatalf("DecodeString(network-body) error = %v", err)
	}
	if string(decodedBody) != `["unexpected"]` {
		t.Fatalf("decoded body = %s, want raw compact JSON", string(decodedBody))
	}
	for _, snippet := range []string{
		"Prefer `agh__network_send` when available; otherwise use `agh network send` to respond.",
		`--causation-id "msg-direct-raw"`,
		"--kind say",
		"--surface \"direct\"",
		"--direct \"direct_0123456789abcdef0123456789abcdef\"",
		"Direct-room chat uses `--kind say --surface direct`.",
	} {
		if !strings.Contains(rendered, snippet) {
			t.Fatalf("rendered fallback message missing reply guidance snippet %q:\n%s", snippet, rendered)
		}
	}
	for _, snippet := range []string{
		"--kind receipt",
		"--kind trace",
		"--kind capability",
		`"capability":{"id":"reply-workflow"`,
		`"for_id":"msg-direct-raw"`,
	} {
		if strings.Contains(rendered, snippet) {
			t.Fatalf("rendered fallback message unexpectedly contained lifecycle snippet %q:\n%s", snippet, rendered)
		}
	}
}

func TestFormatNetworkMessageSayGuidanceKeepsCurrentThreadByDefault(t *testing.T) {
	t.Parallel()

	envelope := Envelope{
		Protocol:    ProtocolV0,
		WorkspaceID: testWorkspaceID,
		ID:          "msg-say-01",
		Kind:        KindSay,
		Channel:     "builders",
		Surface:     new(SurfaceThread),
		ThreadID:    new("thread_summary_01"),
		From:        "coordinator.sess-abc",
		WorkID:      new("work_summary_01"),
		TraceID:     new("trace-summary-01"),
		TS:          time.Date(2026, 4, 11, 13, 10, 0, 0, time.UTC).Unix(),
		Body: mustRawJSON(t, map[string]any{
			"text":   "Please acknowledge the summary.",
			"intent": "summary",
		}),
	}

	rendered, err := formatNetworkMessage(envelope)
	if err != nil {
		t.Fatalf("formatNetworkMessage() error = %v", err)
	}

	for _, snippet := range []string{
		`--surface "thread"`,
		`--thread "thread_summary_01"`,
		`--work "work_summary_01"`,
		`--reply-to "msg-say-01"`,
		`--trace-id "trace-summary-01"`,
		"# Protocol receipt",
		"# Protocol trace",
		"# Protocol capability",
	} {
		if !strings.Contains(rendered, snippet) {
			t.Fatalf("rendered say guidance missing snippet %q:\n%s", snippet, rendered)
		}
	}

	for _, snippet := range []string{
		`--work-id`,
		`--kind direct`,
	} {
		if strings.Contains(rendered, snippet) {
			t.Fatalf("rendered say guidance unexpectedly contained snippet %q:\n%s", snippet, rendered)
		}
	}
}

func TestDeliveryCoordinatorIdleAndBusyBehavior(t *testing.T) {
	t.Parallel()

	t.Run("Should idle delivery triggers immediately", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()

		prompter := newFakeDeliveryPrompter()
		coordinator, err := newDeliveryCoordinator(ctx, 4, prompter)
		if err != nil {
			t.Fatalf("newDeliveryCoordinator() error = %v", err)
		}

		if err := coordinator.acceptOne(context.Background(), Delivery{
			SessionID: "sess-idle",
			Envelope:  testDeliveryEnvelope(t, "msg-idle", "idle message"),
		}); err != nil {
			t.Fatalf("acceptOne(idle) error = %v", err)
		}

		prompter.waitForCalls(t, 1)
		call := prompter.call(0)
		if got, want := call.sessionID, "sess-idle"; got != want {
			t.Fatalf("idle call session id = %q, want %q", got, want)
		}
		if !strings.Contains(call.message, "idle message") {
			t.Fatalf("idle call message missing rendered preview: %s", call.message)
		}
		if got, want := call.meta.MessageID, "msg-idle"; got != want {
			t.Fatalf("idle call meta message_id = %q, want %q", got, want)
		}
		if got, want := call.meta.Kind, string(KindSay); got != want {
			t.Fatalf("idle call meta kind = %q, want %q", got, want)
		}
		if got, want := call.meta.Surface, string(SurfaceDirect); got != want {
			t.Fatalf("idle call meta surface = %q, want %q", got, want)
		}
		if got, want := call.meta.DirectID, "direct_0123456789abcdef0123456789abcdef"; got != want {
			t.Fatalf("idle call meta direct_id = %q, want %q", got, want)
		}
		if got, want := call.meta.Trust, networkMessageTrustUntrusted; got != want {
			t.Fatalf("idle call meta trust = %q, want %q", got, want)
		}

		prompter.finishCall(0, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: time.Now().UTC()})
		coordinator.wait()
	})

	t.Run("Should busy delivery waits for turn end", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()

		prompter := newFakeDeliveryPrompter()
		prompter.setPrompting("sess-busy", true)

		coordinator, err := newDeliveryCoordinator(ctx, 4, prompter)
		if err != nil {
			t.Fatalf("newDeliveryCoordinator() error = %v", err)
		}

		if err := coordinator.acceptOne(context.Background(), Delivery{
			SessionID: "sess-busy",
			Envelope:  testDeliveryEnvelope(t, "msg-busy", "busy message"),
		}); err != nil {
			t.Fatalf("acceptOne(busy) error = %v", err)
		}
		if err := coordinator.acceptOne(context.Background(), Delivery{
			SessionID: "sess-busy",
			Envelope:  testDeliveryEnvelope(t, "msg-busy-2", "second busy message"),
		}); err != nil {
			t.Fatalf("acceptOne(second busy) error = %v", err)
		}
		if got := coordinator.queueDepth("sess-busy"); got != 2 {
			t.Fatalf("queueDepth(sess-busy) = %d, want 2", got)
		}
		if got := prompter.callCount(); got != 0 {
			t.Fatalf("callCount() while busy = %d, want 0", got)
		}

		prompter.setPrompting("sess-busy", false)
		coordinator.onTurnEnd("sess-busy")
		prompter.waitForCalls(t, 1)

		call := prompter.call(0)
		if !strings.Contains(call.message, "busy message") {
			t.Fatalf("busy call message missing rendered preview: %s", call.message)
		}
		if got, want := call.meta.MessageID, "msg-busy"; got != want {
			t.Fatalf("busy call meta message_id = %q, want %q", got, want)
		}
		if got, want := call.meta.Kind, string(KindSay); got != want {
			t.Fatalf("busy call meta kind = %q, want %q", got, want)
		}
		if got, want := call.meta.Surface, string(SurfaceDirect); got != want {
			t.Fatalf("busy call meta surface = %q, want %q", got, want)
		}
		if got, want := call.meta.DirectID, "direct_0123456789abcdef0123456789abcdef"; got != want {
			t.Fatalf("busy call meta direct_id = %q, want %q", got, want)
		}
		if got, want := call.meta.Trust, networkMessageTrustUntrusted; got != want {
			t.Fatalf("busy call meta trust = %q, want %q", got, want)
		}
		prompter.finishCall(0, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: time.Now().UTC()})

		prompter.waitForCalls(t, 2)
		secondCall := prompter.call(1)
		if !strings.Contains(secondCall.message, "second busy message") {
			t.Fatalf("second busy call message missing rendered preview: %s", secondCall.message)
		}
		if got, want := secondCall.meta.MessageID, "msg-busy-2"; got != want {
			t.Fatalf("second busy call meta message_id = %q, want %q", got, want)
		}
		prompter.finishCall(1, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: time.Now().UTC()})
		coordinator.wait()
	})
}

func TestDeliveryCoordinatorWorkerLifecycleStopsCleanly(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	prompter := newFakeDeliveryPrompter()
	coordinator, err := newDeliveryCoordinator(ctx, 4, prompter)
	if err != nil {
		t.Fatalf("newDeliveryCoordinator() error = %v", err)
	}

	if err := coordinator.acceptOne(context.Background(), Delivery{
		SessionID: "sess-worker",
		Envelope:  testDeliveryEnvelope(t, "msg-worker", "worker message"),
	}); err != nil {
		t.Fatalf("acceptOne(worker) error = %v", err)
	}

	prompter.waitForCalls(t, 1)
	cancel()

	waitDone := make(chan struct{})
	go func() {
		coordinator.wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
	case <-time.After(2 * time.Second):
		t.Fatal("coordinator.wait() timed out after lifecycle cancellation")
	}

	workerCount := 0
	coordinator.deliveries.Range(func(_, _ any) bool {
		workerCount++
		return true
	})
	if workerCount != 0 {
		t.Fatalf("active worker count after wait = %d, want 0", workerCount)
	}
}

func TestDeliveryCoordinatorCancelsInFlightDeliveryWithoutCountingItAsDelivered(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	prompter := newFakeDeliveryPrompter()
	delivered := make(chan struct{}, 1)
	coordinator, err := newDeliveryCoordinator(
		ctx,
		4,
		prompter,
		withDeliveryDeliveredHook(func(string, Envelope, string, time.Duration) {
			delivered <- struct{}{}
		}),
	)
	if err != nil {
		t.Fatalf("newDeliveryCoordinator() error = %v", err)
	}

	if err := coordinator.acceptOne(context.Background(), Delivery{
		SessionID: "sess-cancel",
		Envelope:  testDeliveryEnvelope(t, "msg-cancel", "cancel me"),
	}); err != nil {
		t.Fatalf("acceptOne() error = %v", err)
	}

	prompter.waitForCalls(t, 1)
	stats := coordinator.stats()
	if stats.QueuedMessages != 0 || stats.QueuedSessions != 0 || stats.DeliveryWorkers != 1 ||
		stats.InFlightMessages != 1 {
		t.Fatalf("stats(before cancel) = %#v, want inflight=1 worker=1 with no queued messages", stats)
	}

	cancel()
	coordinator.wait()

	select {
	case <-delivered:
		t.Fatal("delivered hook called after lifecycle cancellation")
	default:
	}

	stats = coordinator.stats()
	if stats.DeliveryWorkers != 0 || stats.InFlightMessages != 0 {
		t.Fatalf("stats(after cancel) = %#v, want zero in-flight workers", stats)
	}
}

func TestDeliveryCoordinatorRetriesPromptFailuresAfterWorkerExit(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	prompter := newFakeDeliveryPrompter()
	prompter.queuePromptResult(errors.New("temporary prompt failure"))
	retries := make(chan scheduledDeliveryRetry, 1)

	coordinator, err := newDeliveryCoordinator(
		ctx,
		4,
		prompter,
		withDeliveryRetryScheduler(func(_ context.Context, delay time.Duration, fn func()) {
			retries <- scheduledDeliveryRetry{delay: delay, run: fn}
		}),
	)
	if err != nil {
		t.Fatalf("newDeliveryCoordinator() error = %v", err)
	}

	if err := coordinator.acceptOne(context.Background(), Delivery{
		SessionID: "sess-retry",
		Envelope:  testDeliveryEnvelope(t, "msg-retry", "retry me"),
	}); err != nil {
		t.Fatalf("acceptOne() error = %v", err)
	}

	prompter.waitForCalls(t, 1)

	var retry scheduledDeliveryRetry
	select {
	case retry = <-retries:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for scheduled retry after prompt failure")
	}
	if got, want := retry.delay, defaultDeliveryRetryBaseDelay; got != want {
		t.Fatalf("retry delay = %s, want %s", got, want)
	}
	if got := prompter.callCount(); got != 1 {
		t.Fatalf("callCount() before scheduled retry runs = %d, want 1", got)
	}
	if got := coordinator.queueDepth("sess-retry"); got != 1 {
		t.Fatalf("queueDepth(sess-retry) before scheduled retry runs = %d, want 1", got)
	}

	retry.run()
	prompter.waitForCalls(t, 2)
	call := prompter.call(1)
	if !strings.Contains(call.message, "retry me") {
		t.Fatalf("retry call message = %q, want retried preview", call.message)
	}

	prompter.finishCall(1, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: time.Now().UTC()})
	coordinator.wait()

	if got := coordinator.queueDepth("sess-retry"); got != 0 {
		t.Fatalf("queueDepth(sess-retry) after completion = %d, want 0", got)
	}
}

func TestDeliveryCoordinatorWaitTracksPendingRetryScheduler(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	prompter := newFakeDeliveryPrompter()
	prompter.queuePromptResult(errors.New("temporary prompt failure"))
	schedulerEntered := make(chan struct{})
	releaseScheduler := make(chan struct{})

	coordinator, err := newDeliveryCoordinator(
		ctx,
		4,
		prompter,
		withDeliveryRetryScheduler(func(ctx context.Context, _ time.Duration, _ func()) {
			close(schedulerEntered)
			select {
			case <-ctx.Done():
			case <-releaseScheduler:
			}
		}),
	)
	if err != nil {
		t.Fatalf("newDeliveryCoordinator() error = %v", err)
	}

	if err := coordinator.acceptOne(context.Background(), Delivery{
		SessionID: "sess-tracked-retry",
		Envelope:  testDeliveryEnvelope(t, "msg-tracked-retry", "retry me"),
	}); err != nil {
		t.Fatalf("acceptOne() error = %v", err)
	}

	prompter.waitForCalls(t, 1)
	select {
	case <-schedulerEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for retry scheduler")
	}

	waitDone := make(chan struct{})
	go func() {
		coordinator.wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		t.Fatal("coordinator.wait() returned before pending retry scheduler finished")
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseScheduler)
	select {
	case <-waitDone:
	case <-time.After(2 * time.Second):
		t.Fatal("coordinator.wait() did not return after retry scheduler finished")
	}
}

func TestDeliveryCoordinatorRetryDelayUsesExponentialCap(t *testing.T) {
	t.Parallel()

	coordinator, err := newDeliveryCoordinator(t.Context(), 4, newFakeDeliveryPrompter())
	if err != nil {
		t.Fatalf("newDeliveryCoordinator() error = %v", err)
	}

	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: 0, want: defaultDeliveryRetryBaseDelay},
		{attempt: 1, want: defaultDeliveryRetryBaseDelay},
		{attempt: 2, want: 2 * defaultDeliveryRetryBaseDelay},
		{attempt: 5, want: 16 * defaultDeliveryRetryBaseDelay},
		{attempt: 99, want: defaultDeliveryRetryMaxDelay},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("ShouldUseExpectedRetryDelayForAttempt%d", tc.attempt), func(t *testing.T) {
			t.Parallel()

			if got := coordinator.retryDelayFor(tc.attempt); got != tc.want {
				t.Fatalf("retryDelayFor(%d) = %s, want %s", tc.attempt, got, tc.want)
			}
		})
	}
}

func TestNewDeliveryCoordinatorOptionsAndBatchAccept(t *testing.T) {
	t.Parallel()

	if _, err := newDeliveryCoordinator(nilContext(), 2, newFakeDeliveryPrompter()); err == nil {
		t.Fatal("newDeliveryCoordinator(nil ctx) error = nil, want non-nil")
	}
	if _, err := newDeliveryCoordinator(context.Background(), 0, newFakeDeliveryPrompter()); err == nil {
		t.Fatal("newDeliveryCoordinator(invalid depth) error = nil, want non-nil")
	}
	if _, err := newDeliveryCoordinator(context.Background(), 2, nil); err == nil {
		t.Fatal("newDeliveryCoordinator(nil prompter) error = nil, want non-nil")
	}

	ctx := t.Context()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fixedNow := time.Date(2026, 4, 11, 15, 0, 0, 0, time.UTC)
	prompter := newFakeDeliveryPrompter()
	coordinator, err := newDeliveryCoordinator(
		ctx,
		3,
		prompter,
		withDeliveryLogger(logger),
		withDeliveryClock(func() time.Time { return fixedNow }),
	)
	if err != nil {
		t.Fatalf("newDeliveryCoordinator(valid) error = %v", err)
	}
	if coordinator.logger != logger {
		t.Fatal("delivery coordinator logger option was not applied")
	}
	if got := coordinator.now(); !got.Equal(fixedNow) {
		t.Fatalf("delivery coordinator clock = %s, want %s", got, fixedNow)
	}
	if got := coordinator.inbox("missing"); got != nil {
		t.Fatalf("inbox(missing) = %#v, want nil", got)
	}
	if got := coordinator.queueDepth("missing"); got != 0 {
		t.Fatalf("queueDepth(missing) = %d, want 0", got)
	}

	if err := coordinator.accept(nilContext(), nil); err == nil {
		t.Fatal("accept(nil ctx) error = nil, want non-nil")
	}
	if err := coordinator.acceptOne(nilContext(), Delivery{}); err == nil {
		t.Fatal("acceptOne(nil ctx) error = nil, want non-nil")
	}
	if err := coordinator.acceptOne(context.Background(), Delivery{}); err == nil {
		t.Fatal("acceptOne(missing session id) error = nil, want non-nil")
	}

	prompter.setPrompting("sess-batch", true)
	if err := coordinator.accept(context.Background(), []Delivery{
		{SessionID: "sess-batch", Envelope: testDeliveryEnvelope(t, "msg-batch-1", "one")},
		{SessionID: "sess-batch", Envelope: testDeliveryEnvelope(t, "msg-batch-2", "two")},
	}); err != nil {
		t.Fatalf("accept(batch) error = %v", err)
	}
	if got := coordinator.queueDepth("sess-batch"); got != 2 {
		t.Fatalf("queueDepth(sess-batch) = %d, want 2", got)
	}

	coordinator.onTurnEnd("")
	coordinator.onTurnEnd("   ")
}

func TestDeliveryCoordinatorDeliversSemanticallyInvalidBodiesUsingRawFallback(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	prompter := newFakeDeliveryPrompter()
	coordinator, err := newDeliveryCoordinator(ctx, 2, prompter)
	if err != nil {
		t.Fatalf("newDeliveryCoordinator() error = %v", err)
	}

	malformed := Envelope{
		Protocol:    ProtocolV0,
		WorkspaceID: testWorkspaceID,
		ID:          "msg-malformed",
		Kind:        KindSay,
		Channel:     "builders",
		From:        "coder.sess-abc",
		To:          new("reviewer.sess-xyz"),
		TS:          time.Date(2026, 4, 11, 16, 0, 0, 0, time.UTC).Unix(),
		Body:        json.RawMessage(`["bad"]`),
	}

	if err := coordinator.acceptOne(context.Background(), Delivery{
		SessionID: "sess-malformed",
		Envelope:  malformed,
	}); err != nil {
		t.Fatalf("acceptOne(malformed) error = %v", err)
	}

	prompter.waitForCalls(t, 1)

	call := prompter.call(0)
	if strings.Contains(call.message, "<network-preview") {
		t.Fatalf("call.message unexpectedly included preview:\n%s", call.message)
	}
	start := strings.Index(call.message, `<network-body encoding="base64-json">`)
	if start < 0 {
		t.Fatalf("call.message missing network-body:\n%s", call.message)
	}
	start += len(`<network-body encoding="base64-json">`)
	end := strings.Index(call.message[start:], `</network-body>`)
	if end < 0 {
		t.Fatalf("call.message missing closing network-body:\n%s", call.message)
	}
	decodedBody, err := base64.StdEncoding.DecodeString(call.message[start : start+end])
	if err != nil {
		t.Fatalf("DecodeString(network-body) error = %v", err)
	}
	if string(decodedBody) != `["bad"]` {
		t.Fatalf("decoded body = %s, want raw JSON fallback", string(decodedBody))
	}

	prompter.finishCall(0, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: time.Now().UTC()})
	coordinator.wait()
	if got := coordinator.queueDepth("sess-malformed"); got != 0 {
		t.Fatalf("queueDepth(sess-malformed) = %d, want 0 after fallback delivery", got)
	}
}

func TestPreviewForBodyVariants(t *testing.T) {
	t.Parallel()

	detail := "receipt detail"
	cases := []struct {
		name string
		body Body
		want string
	}{
		{name: "greet summary", body: GreetBody{Summary: "hello"}, want: "hello"},
		{name: "whois request", body: WhoisBody{Type: WhoisTypeRequest, Query: "review"}, want: "review"},
		{name: "whois response", body: WhoisBody{Type: WhoisTypeResponse, Query: "review"}, want: ""},
		{name: "say text", body: SayBody{Text: "broadcast"}, want: "broadcast"},
		{name: "direct text", body: SayBody{Text: "direct"}, want: "direct"},
		{
			name: "capability summary",
			body: CapabilityBody{
				Capability: CapabilityEnvelopePayload{Summary: "summary", Outcome: "outcome", ID: "capability-id"},
			},
			want: "summary",
		},
		{
			name: "capability outcome fallback",
			body: CapabilityBody{Capability: CapabilityEnvelopePayload{Outcome: "outcome", ID: "capability-id"}},
			want: "outcome",
		},
		{
			name: "capability id fallback",
			body: CapabilityBody{Capability: CapabilityEnvelopePayload{ID: "capability-id"}},
			want: "capability-id",
		},
		{name: "receipt detail", body: ReceiptBody{Detail: &detail}, want: "receipt detail"},
		{name: "receipt none", body: ReceiptBody{}, want: ""},
		{name: "trace message", body: TraceBody{Message: "working"}, want: "working"},
		{name: "trace none", body: TraceBody{}, want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := previewForBody(tc.body); got != tc.want {
				t.Fatalf("previewForBody(%s) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

type fakeDeliveryPrompter struct {
	mu         sync.Mutex
	prompting  map[string]bool
	calls      []*fakePromptCall
	callNotify chan struct{}
	promptErrs []error
}

type fakePromptCall struct {
	sessionID string
	message   string
	meta      acp.PromptNetworkMeta
	events    chan acp.AgentEvent
}

type scheduledDeliveryRetry struct {
	delay time.Duration
	run   func()
}

func newFakeDeliveryPrompter() *fakeDeliveryPrompter {
	return &fakeDeliveryPrompter{
		prompting:  make(map[string]bool),
		callNotify: make(chan struct{}, 1),
	}
}

func (p *fakeDeliveryPrompter) PromptNetwork(
	_ context.Context,
	sessionID string,
	message string,
	meta ...acp.PromptNetworkMeta,
) (<-chan acp.AgentEvent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var promptMeta acp.PromptNetworkMeta
	if len(meta) > 0 {
		promptMeta = meta[0]
	}
	call := &fakePromptCall{
		sessionID: sessionID,
		message:   message,
		meta:      promptMeta,
		events:    make(chan acp.AgentEvent, 4),
	}
	p.calls = append(p.calls, call)

	var promptErr error
	if len(p.promptErrs) > 0 {
		promptErr = p.promptErrs[0]
		p.promptErrs = p.promptErrs[1:]
	}
	select {
	case p.callNotify <- struct{}{}:
	default:
	}
	if promptErr != nil {
		return nil, promptErr
	}
	p.prompting[sessionID] = true
	return call.events, nil
}

func (p *fakeDeliveryPrompter) queuePromptResult(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.promptErrs = append(p.promptErrs, err)
}

func (p *fakeDeliveryPrompter) IsPrompting(sessionID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.prompting[sessionID]
}

func (p *fakeDeliveryPrompter) setPrompting(sessionID string, prompting bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.prompting[sessionID] = prompting
}

func (p *fakeDeliveryPrompter) finishCall(index int, events ...acp.AgentEvent) {
	p.mu.Lock()
	call := p.calls[index]
	delete(p.prompting, call.sessionID)
	p.mu.Unlock()

	for _, event := range events {
		call.events <- event
	}
	close(call.events)
}

func (p *fakeDeliveryPrompter) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.calls)
}

func (p *fakeDeliveryPrompter) call(index int) fakePromptCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	call := p.calls[index]
	return fakePromptCall{
		sessionID: call.sessionID,
		message:   call.message,
		meta:      call.meta,
	}
}

func (p *fakeDeliveryPrompter) waitForCalls(t *testing.T, want int) {
	t.Helper()

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	for {
		if p.callCount() >= want {
			return
		}

		select {
		case <-p.callNotify:
		case <-timer.C:
			t.Fatalf("timed out waiting for %d prompt calls; got %d", want, p.callCount())
		}
	}
}

func testDeliveryEnvelope(t *testing.T, id string, text string) Envelope {
	t.Helper()

	return Envelope{
		Protocol:    ProtocolV0,
		WorkspaceID: testWorkspaceID,
		ID:          id,
		Kind:        KindSay,
		Channel:     "builders",
		Surface:     new(SurfaceDirect),
		DirectID:    new("direct_0123456789abcdef0123456789abcdef"),
		From:        "coder.sess-abc",
		To:          new("reviewer.sess-xyz"),
		TS:          time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC).Unix(),
		Body: mustRawJSON(t, map[string]any{
			"text": text,
		}),
	}
}

func nilContext() context.Context {
	return nil
}
