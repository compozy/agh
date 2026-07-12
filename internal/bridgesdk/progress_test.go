// Suite: bridge progress accumulator
// Invariant: progress preserves bubble ordering, platform limits, throttling, and phase affordances.
// Boundary IN: bridgesdk accumulation and ProgressSink calls.
// Boundary OUT: provider API clients and daemon-side progress projection.
package bridgesdk

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	bridgepkg "github.com/compozy/agh/internal/bridges"
	"github.com/compozy/agh/internal/testutil"
)

func TestProgressAccumulatorAccumulatesProgress(t *testing.T) {
	t.Parallel()

	t.Run("Should batch three progress lines inside one throttle window into one edit", func(t *testing.T) {
		t.Parallel()

		clock := progressTestClock{current: time.Unix(1_000, 0)}
		sink := &progressRecordingSink{}
		accumulator := NewProgressAccumulator(progressTestConfig(), sink, clock.Now)
		ctx := testutil.Context(t)

		for index, label := range []string{"Inspecting", "Reading", "Summarizing"} {
			if index > 0 {
				clock.Advance(300 * time.Millisecond)
			}
			if err := accumulator.OnProgress(ctx, progressTestEvent(index+1, label)); err != nil {
				t.Fatalf("OnProgress(%q) error = %v", label, err)
			}
		}
		if got := sink.EditCalls(); len(got) != 0 {
			t.Fatalf("Edit calls before throttle expiry = %v, want none", got)
		}

		clock.Advance(900 * time.Millisecond)
		if err := accumulator.Flush(ctx); err != nil {
			t.Fatalf("Flush() error = %v", err)
		}

		posts := sink.PostCalls()
		if len(posts) != 1 {
			t.Fatalf("Post calls = %d, want 1", len(posts))
		}
		if posts[0].target != progressTestConfig().Target {
			t.Fatalf("Post target = %#v, want %#v", posts[0].target, progressTestConfig().Target)
		}
		edits := sink.EditCalls()
		if len(edits) != 1 {
			t.Fatalf("Edit calls = %d, want 1", len(edits))
		}
		want := strings.Join([]string{
			"🔧 Inspecting item-1",
			"🔧 Reading item-2",
			"🔧 Summarizing item-3",
		}, "\n")
		if edits[0].text != want {
			t.Fatalf("Edit text = %q, want %q", edits[0].text, want)
		}
	})

	t.Run("Should collapse identical consecutive lines with a repeat counter", func(t *testing.T) {
		t.Parallel()

		clock := progressTestClock{current: time.Unix(2_000, 0)}
		sink := &progressRecordingSink{}
		accumulator := NewProgressAccumulator(progressTestConfig(), sink, clock.Now)
		ctx := testutil.Context(t)
		event := progressTestEvent(1, "Searching")

		for index := 1; index <= 3; index++ {
			event.Index = index
			if err := accumulator.OnProgress(ctx, event); err != nil {
				t.Fatalf("OnProgress(repeat %d) error = %v", index, err)
			}
		}
		clock.Advance(1500 * time.Millisecond)
		if err := accumulator.Flush(ctx); err != nil {
			t.Fatalf("Flush() error = %v", err)
		}

		edits := sink.EditCalls()
		if len(edits) != 1 {
			t.Fatalf("Edit calls = %d, want 1", len(edits))
		}
		if want := "🔧 Searching item-1 (×3)"; edits[0].text != want {
			t.Fatalf("Edit text = %q, want %q", edits[0].text, want)
		}
	})
}

func TestProgressAccumulatorPreservesBubbleOrdering(t *testing.T) {
	t.Parallel()

	t.Run("Should open a fresh bubble after content lands", func(t *testing.T) {
		t.Parallel()

		clock := progressTestClock{current: time.Unix(3_000, 0)}
		sink := &progressRecordingSink{}
		accumulator := NewProgressAccumulator(progressTestConfig(), sink, clock.Now)
		ctx := testutil.Context(t)

		if err := accumulator.OnProgress(ctx, progressTestEvent(1, "Inspecting")); err != nil {
			t.Fatalf("OnProgress(first) error = %v", err)
		}
		if err := accumulator.OnProgress(ctx, progressTestEvent(2, "Reading")); err != nil {
			t.Fatalf("OnProgress(second) error = %v", err)
		}
		if err := accumulator.OnContent(ctx); err != nil {
			t.Fatalf("OnContent() error = %v", err)
		}
		if err := accumulator.OnProgress(ctx, progressTestEvent(3, "Summarizing")); err != nil {
			t.Fatalf("OnProgress(after content) error = %v", err)
		}

		posts := sink.PostCalls()
		if len(posts) != 2 {
			t.Fatalf("Post calls = %d, want 2", len(posts))
		}
		if posts[0].remoteID == posts[1].remoteID {
			t.Fatalf("post remote IDs = %q and %q, want distinct bubbles", posts[0].remoteID, posts[1].remoteID)
		}
		edits := sink.EditCalls()
		if len(edits) != 1 || edits[0].remoteID != posts[0].remoteID {
			t.Fatalf("Edit calls = %v, want one flush of first bubble %q", edits, posts[0].remoteID)
		}
	})

	t.Run("Should freeze an overflowed bubble and keep only the newest continuation editable", func(t *testing.T) {
		t.Parallel()

		clock := progressTestClock{current: time.Unix(4_000, 0)}
		sink := &progressRecordingSink{}
		config := progressTestConfig()
		config.MaxMessageLen = 100
		accumulator := NewProgressAccumulator(config, sink, clock.Now)
		ctx := testutil.Context(t)

		first := progressTestEvent(1, "Inspecting-alpha")
		first.Preview = "one"
		second := progressTestEvent(2, "Reading")
		second.Preview = "two"
		third := progressTestEvent(3, "Done")
		third.Preview = "x"
		for _, event := range []bridgepkg.ToolProgress{first, second, third} {
			if err := accumulator.OnProgress(ctx, event); err != nil {
				t.Fatalf("OnProgress(%q) error = %v", event.Label, err)
			}
		}
		clock.Advance(1500 * time.Millisecond)
		if err := accumulator.Flush(ctx); err != nil {
			t.Fatalf("Flush() error = %v", err)
		}

		posts := sink.PostCalls()
		if len(posts) != 2 {
			t.Fatalf("Post calls = %d, want 2 continuation bubbles", len(posts))
		}
		edits := sink.EditCalls()
		if len(edits) != 1 {
			t.Fatalf("Edit calls = %d, want 1", len(edits))
		}
		if edits[0].remoteID != posts[1].remoteID {
			t.Fatalf("Edit remote ID = %q, want newest continuation %q", edits[0].remoteID, posts[1].remoteID)
		}
		if strings.Contains(edits[0].text, first.Label) {
			t.Fatalf("newest continuation edit = %q, want frozen first bubble omitted", edits[0].text)
		}
	})
}

func TestProgressAccumulatorEnforcesProviderMessageLimit(t *testing.T) {
	t.Parallel()

	t.Run("Should keep a first line exactly at the provider limit in one post", func(t *testing.T) {
		t.Parallel()

		config := progressTestConfig()
		config.MaxMessageLen = 100
		limit := config.MaxMessageLen - progressMessageHeadroom
		sink := &progressRecordingSink{}
		accumulator := NewProgressAccumulator(config, sink, nil)
		event := progressTestEvent(1, strings.Repeat("b", limit))
		event.Emoji = ""
		event.Preview = ""

		if err := accumulator.OnProgress(testutil.Context(t), event); err != nil {
			t.Fatalf("OnProgress() error = %v", err)
		}

		assertProgressCallsWithinLimit(t, sink, config)
		assertProgressBubbleText(t, sink, renderProgressLine(event))
		if got, want := len(sink.PostCalls()), 1; got != want {
			t.Fatalf("Post calls = %d, want %d", got, want)
		}
	})

	t.Run("Should split a first line one unit over the provider limit into two posts", func(t *testing.T) {
		t.Parallel()

		config := progressTestConfig()
		config.MaxMessageLen = 100
		limit := config.MaxMessageLen - progressMessageHeadroom
		sink := &progressRecordingSink{}
		accumulator := NewProgressAccumulator(config, sink, nil)
		event := progressTestEvent(1, strings.Repeat("b", limit+1))
		event.Emoji = ""
		event.Preview = ""

		if err := accumulator.OnProgress(testutil.Context(t), event); err != nil {
			t.Fatalf("OnProgress() error = %v", err)
		}

		assertProgressCallsWithinLimit(t, sink, config)
		assertProgressBubbleText(t, sink, renderProgressLine(event))
		if got, want := len(sink.PostCalls()), 2; got != want {
			t.Fatalf("Post calls = %d, want %d continuations", got, want)
		}
	})

	t.Run("Should split an oversized first line without dropping content", func(t *testing.T) {
		t.Parallel()

		config := progressTestConfig()
		config.MaxMessageLen = 100
		sink := &progressRecordingSink{}
		accumulator := NewProgressAccumulator(config, sink, nil)
		event := progressTestEvent(1, strings.Repeat("a", 90))
		event.Emoji = ""
		event.Preview = ""

		if err := accumulator.OnProgress(testutil.Context(t), event); err != nil {
			t.Fatalf("OnProgress() error = %v", err)
		}

		assertProgressCallsWithinLimit(t, sink, config)
		assertProgressBubbleText(t, sink, renderProgressLine(event))
		if got := len(sink.PostCalls()); got < 2 {
			t.Fatalf("Post calls = %d, want continuation bubbles", got)
		}
	})

	t.Run("Should split a verbose preview without capping or dropping it", func(t *testing.T) {
		t.Parallel()

		config := progressTestConfig()
		config.ToolProgress = bridgepkg.ProgressModeVerbose
		config.MaxMessageLen = 100
		sink := &progressRecordingSink{}
		accumulator := NewProgressAccumulator(config, sink, nil)
		event := progressTestEvent(1, "Inspecting")
		event.Preview = strings.Repeat("verbose-detail-", 20)

		if err := accumulator.OnProgress(testutil.Context(t), event); err != nil {
			t.Fatalf("OnProgress() error = %v", err)
		}

		assertProgressCallsWithinLimit(t, sink, config)
		assertProgressBubbleText(t, sink, renderProgressLine(event))
	})

	t.Run("Should split at rune boundaries under a UTF-16 length function", func(t *testing.T) {
		t.Parallel()

		config := progressTestConfig()
		config.MaxMessageLen = 72
		config.Len = func(value string) int {
			return len(utf16.Encode([]rune(value)))
		}
		sink := &progressRecordingSink{}
		accumulator := NewProgressAccumulator(config, sink, nil)
		event := progressTestEvent(1, "分析🚀分析🚀分析🚀")
		event.Emoji = "🛠️"
		event.Preview = "資料🌍資料🌍"

		if err := accumulator.OnProgress(testutil.Context(t), event); err != nil {
			t.Fatalf("OnProgress() error = %v", err)
		}

		assertProgressCallsWithinLimit(t, sink, config)
		assertProgressBubbleText(t, sink, renderProgressLine(event))
		for _, call := range sink.PostCalls() {
			if !utf8.ValidString(call.text) {
				t.Fatalf("Post text = %q, want valid UTF-8", call.text)
			}
		}
	})

	t.Run("Should roll repeat counter growth into a continuation bubble", func(t *testing.T) {
		t.Parallel()

		clock := progressTestClock{current: time.Unix(4_500, 0)}
		config := progressTestConfig()
		config.MaxMessageLen = 100
		sink := &progressRecordingSink{}
		accumulator := NewProgressAccumulator(config, sink, clock.Now)
		event := progressTestEvent(1, strings.Repeat("r", 30))
		event.Emoji = ""
		event.Preview = ""
		ctx := testutil.Context(t)

		for index := 1; index <= 12; index++ {
			event.Index = index
			if err := accumulator.OnProgress(ctx, event); err != nil {
				t.Fatalf("OnProgress(repeat %d) error = %v", index, err)
			}
		}
		clock.Advance(defaultProgressEditInterval)
		if err := accumulator.Flush(ctx); err != nil {
			t.Fatalf("Flush() error = %v", err)
		}

		assertProgressCallsWithinLimit(t, sink, config)
		assertProgressBubbleText(t, sink, fmt.Sprintf("%s (×12)", event.Label))
		posts := sink.PostCalls()
		if len(posts) < 2 {
			t.Fatalf("Post calls = %d, want a continuation for counter growth", len(posts))
		}
		edits := sink.EditCalls()
		if got, want := edits[len(edits)-1].remoteID, posts[len(posts)-1].remoteID; got != want {
			t.Fatalf("latest edit remote ID = %q, want newest continuation %q", got, want)
		}
	})
}

func TestProgressAccumulatorGroupingAndThrottle(t *testing.T) {
	t.Parallel()

	t.Run("Should post one message per line with zero edits under separate grouping", func(t *testing.T) {
		t.Parallel()

		clock := progressTestClock{current: time.Unix(5_000, 0)}
		sink := &progressRecordingSink{}
		config := progressTestConfig()
		config.Grouping = bridgepkg.ProgressGroupingSeparate
		accumulator := NewProgressAccumulator(config, sink, clock.Now)
		ctx := testutil.Context(t)

		for index, label := range []string{"Inspecting", "Reading", "Summarizing"} {
			if err := accumulator.OnProgress(ctx, progressTestEvent(index+1, label)); err != nil {
				t.Fatalf("OnProgress(%q) error = %v", label, err)
			}
		}
		if err := accumulator.Flush(ctx); err != nil {
			t.Fatalf("Flush() error = %v", err)
		}

		if got := len(sink.PostCalls()); got != 3 {
			t.Fatalf("Post calls = %d, want 3", got)
		}
		if got := sink.EditCalls(); len(got) != 0 {
			t.Fatalf("Edit calls = %v, want none", got)
		}
	})

	t.Run("Should defer an edit until the default throttle window expires", func(t *testing.T) {
		t.Parallel()

		clock := progressTestClock{current: time.Unix(6_000, 0)}
		sink := &progressRecordingSink{}
		accumulator := NewProgressAccumulator(progressTestConfig(), sink, clock.Now)
		ctx := testutil.Context(t)

		if err := accumulator.OnProgress(ctx, progressTestEvent(1, "Inspecting")); err != nil {
			t.Fatalf("OnProgress(first) error = %v", err)
		}
		clock.Advance(500 * time.Millisecond)
		if err := accumulator.OnProgress(ctx, progressTestEvent(2, "Reading")); err != nil {
			t.Fatalf("OnProgress(second) error = %v", err)
		}
		if got := sink.EditCalls(); len(got) != 0 {
			t.Fatalf("Edit calls at t=0.5s = %v, want none", got)
		}

		clock.Advance(time.Second)
		if err := accumulator.Flush(ctx); err != nil {
			t.Fatalf("Flush() at t=1.5s error = %v", err)
		}
		if got := len(sink.EditCalls()); got != 1 {
			t.Fatalf("Edit calls at t=1.5s = %d, want 1", got)
		}
	})

	t.Run("Should honor a configured edit interval", func(t *testing.T) {
		t.Parallel()

		clock := progressTestClock{current: time.Unix(6_500, 0)}
		sink := &progressRecordingSink{}
		config := progressTestConfig()
		config.EditInterval = 250 * time.Millisecond
		accumulator := NewProgressAccumulator(config, sink, clock.Now)
		ctx := testutil.Context(t)

		if err := accumulator.OnProgress(ctx, progressTestEvent(1, "Inspecting")); err != nil {
			t.Fatalf("OnProgress(first) error = %v", err)
		}
		clock.Advance(249 * time.Millisecond)
		if err := accumulator.OnProgress(ctx, progressTestEvent(2, "Reading")); err != nil {
			t.Fatalf("OnProgress(second) error = %v", err)
		}
		if got := sink.EditCalls(); len(got) != 0 {
			t.Fatalf("Edit calls at t=0.249s = %v, want none", got)
		}

		clock.Advance(time.Millisecond)
		if err := accumulator.OnProgress(ctx, progressTestEvent(3, "Summarizing")); err != nil {
			t.Fatalf("OnProgress(third) error = %v", err)
		}
		if got := len(sink.EditCalls()); got != 1 {
			t.Fatalf("Edit calls at t=0.25s = %d, want 1", got)
		}
	})
}

func TestProgressAccumulatorDrivesPhaseAffordances(t *testing.T) {
	t.Parallel()

	t.Run("Should drive typing and reaction hooks from progress phases", func(t *testing.T) {
		t.Parallel()

		clock := progressTestClock{current: time.Unix(7_000, 0)}
		sink := &progressRecordingSink{}
		config := progressTestConfig()
		config.Typing = true
		config.Reactions = true
		accumulator := NewProgressAccumulator(config, sink, clock.Now)
		ctx := testutil.Context(t)

		events := []bridgepkg.ToolProgress{
			progressTestEventWithPhase(1, bridgepkg.ToolProgressPhaseStarted),
			progressTestEventWithPhase(2, bridgepkg.ToolProgressPhaseCompleted),
			progressTestEventWithPhase(3, bridgepkg.ToolProgressPhaseFailed),
		}
		for _, event := range events {
			if err := accumulator.OnProgress(ctx, event); err != nil {
				t.Fatalf("OnProgress(%q) error = %v", event.Phase, err)
			}
		}

		if got, want := sink.TypingCalls(), []bool{true, false, false}; fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("Typing calls = %v, want %v", got, want)
		}
		if got, want := sink.ReactionCalls(), []string{"👀", "✅", "❌"}; fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("Reaction calls = %v, want %v", got, want)
		}
	})

	t.Run("Should render long completions and failed details as terminal lifecycle lines", func(t *testing.T) {
		t.Parallel()

		completed := progressTestEventWithPhase(1, bridgepkg.ToolProgressPhaseCompleted)
		completed.DurationMS = 31_000
		failed := progressTestEventWithPhase(2, bridgepkg.ToolProgressPhaseFailed)
		failed.Error = "permission denied"

		if got, want := renderProgressLine(completed), "✅ Inspecting item-1 · 31s"; got != want {
			t.Fatalf("completed progress line = %q, want %q", got, want)
		}
		if got, want := renderProgressLine(failed), "❌ Inspecting item-2 — permission denied"; got != want {
			t.Fatalf("failed progress line = %q, want %q", got, want)
		}
	})

	t.Run("Should stop typing when content closes a new-mode progress bubble", func(t *testing.T) {
		t.Parallel()

		sink := &progressRecordingSink{}
		config := progressTestConfig()
		config.ToolProgress = bridgepkg.ProgressModeNew
		config.Typing = true
		accumulator := NewProgressAccumulator(config, sink, nil)
		ctx := testutil.Context(t)

		if err := accumulator.OnProgress(
			ctx,
			progressTestEventWithPhase(1, bridgepkg.ToolProgressPhaseStarted),
		); err != nil {
			t.Fatalf("OnProgress(started) error = %v", err)
		}
		if err := accumulator.OnContent(ctx); err != nil {
			t.Fatalf("OnContent() error = %v", err)
		}

		if got, want := sink.TypingCalls(), []bool{true, false}; fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("Typing calls = %v, want %v", got, want)
		}
	})

	t.Run("Should treat a nil progress sink as safe no-op affordances", func(t *testing.T) {
		t.Parallel()

		config := progressTestConfig()
		config.Typing = true
		config.Reactions = true
		accumulator := NewProgressAccumulator(config, nil, nil)
		if err := accumulator.OnProgress(testutil.Context(t), progressTestEvent(1, "Inspecting")); err != nil {
			t.Fatalf("OnProgress(first) with nil sink error = %v", err)
		}
		if err := accumulator.OnProgress(testutil.Context(t), progressTestEvent(2, "Reading")); err != nil {
			t.Fatalf("OnProgress(second) with nil sink error = %v", err)
		}
		if err := accumulator.OnContent(testutil.Context(t)); err != nil {
			t.Fatalf("OnContent() with nil sink error = %v", err)
		}
	})

	t.Run("Should ignore progress without platform calls when mode is off", func(t *testing.T) {
		t.Parallel()

		sink := &progressRecordingSink{}
		config := progressTestConfig()
		config.ToolProgress = bridgepkg.ProgressModeOff
		accumulator := NewProgressAccumulator(config, sink, nil)
		if err := accumulator.OnProgress(testutil.Context(t), progressTestEvent(1, "Inspecting")); err != nil {
			t.Fatalf("OnProgress() error = %v", err)
		}
		if calls := sink.TotalCalls(); calls != 0 {
			t.Fatalf("platform calls = %d, want 0", calls)
		}
	})
}

func TestProgressAccumulatorRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	t.Run("Should reject an invalid progress event before calling the platform", func(t *testing.T) {
		t.Parallel()

		sink := &progressRecordingSink{}
		accumulator := NewProgressAccumulator(progressTestConfig(), sink, nil)
		event := progressTestEvent(1, "Inspecting")
		event.Index = 0
		if err := accumulator.OnProgress(testutil.Context(t), event); err == nil {
			t.Fatal("OnProgress(invalid event) error = nil, want non-nil")
		}
		if calls := sink.TotalCalls(); calls != 0 {
			t.Fatalf("platform calls = %d, want 0", calls)
		}
	})

	t.Run("Should reject invalid accumulator settings before calling the platform", func(t *testing.T) {
		t.Parallel()

		sink := &progressRecordingSink{}
		config := progressTestConfig()
		config.EditInterval = -time.Second
		accumulator := NewProgressAccumulator(config, sink, nil)
		if err := accumulator.OnProgress(testutil.Context(t), progressTestEvent(1, "Inspecting")); err == nil {
			t.Fatal("OnProgress(invalid config) error = nil, want non-nil")
		}
		if calls := sink.TotalCalls(); calls != 0 {
			t.Fatalf("platform calls = %d, want 0", calls)
		}
	})

	t.Run("Should use a safe default when the configured message length is too small", func(t *testing.T) {
		t.Parallel()

		sink := &progressRecordingSink{}
		config := progressTestConfig()
		config.MaxMessageLen = 64
		accumulator := NewProgressAccumulator(config, sink, nil)
		if err := accumulator.OnProgress(testutil.Context(t), progressTestEvent(1, "Inspecting")); err != nil {
			t.Fatalf("OnProgress() error = %v", err)
		}
		if got := len(sink.PostCalls()); got != 1 {
			t.Fatalf("Post calls = %d, want 1", got)
		}
	})
}

func TestProgressAccumulatorDefaultsModeAndGrouping(t *testing.T) {
	t.Parallel()

	t.Run("Should default empty settings to enabled accumulated progress", func(t *testing.T) {
		t.Parallel()

		clock := progressTestClock{current: time.Unix(7_500, 0)}
		config := progressTestConfig()
		config.ToolProgress = ""
		config.Grouping = ""
		sink := &progressRecordingSink{}
		accumulator := NewProgressAccumulator(config, sink, clock.Now)
		ctx := testutil.Context(t)
		for index, label := range []string{"Inspecting", "Reading"} {
			if err := accumulator.OnProgress(ctx, progressTestEvent(index+1, label)); err != nil {
				t.Fatalf("OnProgress(%q) error = %v", label, err)
			}
		}
		if err := accumulator.Flush(ctx); err != nil {
			t.Fatalf("Flush() error = %v", err)
		}

		if got, want := len(sink.PostCalls()), 1; got != want {
			t.Fatalf("Post calls = %d, want %d accumulated bubble", got, want)
		}
		if got, want := len(sink.EditCalls()), 1; got != want {
			t.Fatalf("Edit calls = %d, want %d accumulated update", got, want)
		}
		assertProgressBubbleText(
			t,
			sink,
			"🔧 Inspecting item-1\n🔧 Reading item-2",
		)
	})
}

func TestProgressAccumulatorRejectsInvalidContexts(t *testing.T) {
	t.Parallel()

	canceledContext, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name      string
		ctx       context.Context
		operation func(*ProgressAccumulator, context.Context) error
		wantError string
	}{
		{
			name: "Should reject a nil context before recording progress",
			operation: func(accumulator *ProgressAccumulator, ctx context.Context) error {
				return accumulator.OnProgress(ctx, progressTestEvent(1, "Inspecting"))
			},
			wantError: "progress context is required",
		},
		{
			name: "Should reject a canceled context before recording progress",
			ctx:  canceledContext,
			operation: func(accumulator *ProgressAccumulator, ctx context.Context) error {
				return accumulator.OnProgress(ctx, progressTestEvent(1, "Inspecting"))
			},
			wantError: "context canceled",
		},
		{
			name: "Should reject a nil context before flushing progress",
			operation: func(accumulator *ProgressAccumulator, ctx context.Context) error {
				return accumulator.Flush(ctx)
			},
			wantError: "progress context is required",
		},
		{
			name: "Should reject a canceled context before flushing progress",
			ctx:  canceledContext,
			operation: func(accumulator *ProgressAccumulator, ctx context.Context) error {
				return accumulator.Flush(ctx)
			},
			wantError: "context canceled",
		},
		{
			name: "Should reject a nil context before content closes progress",
			operation: func(accumulator *ProgressAccumulator, ctx context.Context) error {
				return accumulator.OnContent(ctx)
			},
			wantError: "progress context is required",
		},
		{
			name: "Should reject a canceled context before content closes progress",
			ctx:  canceledContext,
			operation: func(accumulator *ProgressAccumulator, ctx context.Context) error {
				return accumulator.OnContent(ctx)
			},
			wantError: "context canceled",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			sink := &progressRecordingSink{}
			accumulator := NewProgressAccumulator(progressTestConfig(), sink, nil)
			err := test.operation(accumulator, test.ctx)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("progress operation error = %v, want containing %q", err, test.wantError)
			}
			if got := sink.TotalCalls(); got != 0 {
				t.Fatalf("platform calls = %d, want zero after rejected context", got)
			}
		})
	}
}

func TestProgressAccumulatorRejectsNilReceiver(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		operation func(*ProgressAccumulator, context.Context) error
	}{
		{
			name: "Should reject progress on a nil accumulator",
			operation: func(accumulator *ProgressAccumulator, ctx context.Context) error {
				return accumulator.OnProgress(ctx, progressTestEvent(1, "Inspecting"))
			},
		},
		{
			name: "Should reject flush on a nil accumulator",
			operation: func(accumulator *ProgressAccumulator, ctx context.Context) error {
				return accumulator.Flush(ctx)
			},
		},
		{
			name: "Should reject content on a nil accumulator",
			operation: func(accumulator *ProgressAccumulator, ctx context.Context) error {
				return accumulator.OnContent(ctx)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var accumulator *ProgressAccumulator
			err := test.operation(accumulator, testutil.Context(t))
			if err == nil || !strings.Contains(err.Error(), "progress accumulator is required") {
				t.Fatalf("progress operation error = %v, want nil-accumulator rejection", err)
			}
		})
	}
}

func TestProgressAccumulatorRejectsInvalidTargetAcrossLifecycle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		operation func(*ProgressAccumulator, context.Context) error
	}{
		{
			name: "Should reject progress with an invalid target",
			operation: func(accumulator *ProgressAccumulator, ctx context.Context) error {
				return accumulator.OnProgress(ctx, progressTestEvent(1, "Inspecting"))
			},
		},
		{
			name: "Should reject flush with an invalid target",
			operation: func(accumulator *ProgressAccumulator, ctx context.Context) error {
				return accumulator.Flush(ctx)
			},
		},
		{
			name: "Should reject content with an invalid target",
			operation: func(accumulator *ProgressAccumulator, ctx context.Context) error {
				return accumulator.OnContent(ctx)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			config := progressTestConfig()
			config.Target = bridgepkg.DeliveryTarget{
				BridgeInstanceID: "bridge-invalid-target",
				Mode:             bridgepkg.DeliveryModeDirectSend,
			}
			sink := &progressRecordingSink{}
			accumulator := NewProgressAccumulator(config, sink, nil)
			err := test.operation(accumulator, testutil.Context(t))
			if err == nil || !strings.Contains(err.Error(), "invalid progress target") {
				t.Fatalf("progress operation error = %v, want invalid target rejection", err)
			}
			if got := sink.TotalCalls(); got != 0 {
				t.Fatalf("platform calls = %d, want zero after invalid target", got)
			}
		})
	}
}

func TestProgressAccumulatorKeepsDisabledLifecycleSilent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		operation func(*ProgressAccumulator, context.Context) error
	}{
		{
			name: "Should make flush a no-op when progress is disabled",
			operation: func(accumulator *ProgressAccumulator, ctx context.Context) error {
				return accumulator.Flush(ctx)
			},
		},
		{
			name: "Should make content a no-op when progress is disabled",
			operation: func(accumulator *ProgressAccumulator, ctx context.Context) error {
				return accumulator.OnContent(ctx)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			config := progressTestConfig()
			config.ToolProgress = bridgepkg.ProgressModeOff
			sink := &progressRecordingSink{}
			accumulator := NewProgressAccumulator(config, sink, nil)
			if err := test.operation(accumulator, testutil.Context(t)); err != nil {
				t.Fatalf("disabled progress operation error = %v", err)
			}
			if got := sink.TotalCalls(); got != 0 {
				t.Fatalf("platform calls = %d, want zero while disabled", got)
			}
		})
	}
}

func TestProgressAccumulatorPropagatesSinkFailures(t *testing.T) {
	t.Parallel()

	t.Run("Should report a typing affordance failure without dropping the progress line", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("typing unavailable")
		sink := newProgressFailingSink()
		sink.typingError = wantErr
		config := progressTestConfig()
		config.Typing = true
		accumulator := NewProgressAccumulator(config, sink, nil)

		err := accumulator.OnProgress(testutil.Context(t), progressTestEvent(1, "Inspecting"))
		assertProgressOperationError(t, err, wantErr, "set progress typing state")
		if got, want := len(sink.PostCalls()), 1; got != want {
			t.Fatalf("progress posts = %d, want %d after typing failure", got, want)
		}
	})

	t.Run("Should report a reaction affordance failure without dropping the progress line", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("reactions unavailable")
		sink := newProgressFailingSink()
		sink.reactionError = wantErr
		config := progressTestConfig()
		config.Reactions = true
		accumulator := NewProgressAccumulator(config, sink, nil)

		err := accumulator.OnProgress(testutil.Context(t), progressTestEvent(1, "Inspecting"))
		assertProgressOperationError(t, err, wantErr, "apply progress reaction")
		if got, want := len(sink.PostCalls()), 1; got != want {
			t.Fatalf("progress posts = %d, want %d after reaction failure", got, want)
		}
	})

	t.Run("Should propagate an accumulated progress post failure", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("post unavailable")
		sink := newProgressFailingSink()
		sink.postError = wantErr
		accumulator := NewProgressAccumulator(progressTestConfig(), sink, nil)

		err := accumulator.OnProgress(testutil.Context(t), progressTestEvent(1, "Inspecting"))
		assertProgressOperationError(t, err, wantErr, "post progress bubble")
	})

	t.Run("Should propagate a separate progress post failure", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("separate post unavailable")
		sink := newProgressFailingSink()
		sink.postError = wantErr
		config := progressTestConfig()
		config.Grouping = bridgepkg.ProgressGroupingSeparate
		accumulator := NewProgressAccumulator(config, sink, nil)

		err := accumulator.OnProgress(testutil.Context(t), progressTestEvent(1, "Inspecting"))
		assertProgressOperationError(t, err, wantErr, "post separate progress line")
	})

	t.Run("Should reject a progress post without a remote identifier", func(t *testing.T) {
		t.Parallel()

		sink := newProgressFailingSink()
		sink.emptyRemoteID = true
		accumulator := NewProgressAccumulator(progressTestConfig(), sink, nil)

		err := accumulator.OnProgress(testutil.Context(t), progressTestEvent(1, "Inspecting"))
		if err == nil || !strings.Contains(err.Error(), "empty remote id") {
			t.Fatalf("OnProgress() error = %v, want empty remote id rejection", err)
		}
	})

	t.Run("Should propagate an edit failure from an explicit flush", func(t *testing.T) {
		t.Parallel()

		clock := progressTestClock{current: time.Unix(8_000, 0)}
		sink := newProgressFailingSink()
		accumulator := NewProgressAccumulator(progressTestConfig(), sink, clock.Now)
		ctx := testutil.Context(t)
		if err := accumulator.OnProgress(ctx, progressTestEvent(1, "Inspecting")); err != nil {
			t.Fatalf("OnProgress(first) error = %v", err)
		}
		if err := accumulator.OnProgress(ctx, progressTestEvent(2, "Reading")); err != nil {
			t.Fatalf("OnProgress(second) error = %v", err)
		}
		wantErr := errors.New("edit unavailable")
		sink.editError = wantErr

		err := accumulator.Flush(ctx)
		assertProgressOperationError(t, err, wantErr, "edit progress bubble")
	})

	t.Run("Should propagate an edit failure when content closes the bubble", func(t *testing.T) {
		t.Parallel()

		clock := progressTestClock{current: time.Unix(8_500, 0)}
		sink := newProgressFailingSink()
		config := progressTestConfig()
		config.Typing = true
		accumulator := NewProgressAccumulator(config, sink, clock.Now)
		ctx := testutil.Context(t)
		if err := accumulator.OnProgress(ctx, progressTestEvent(1, "Inspecting")); err != nil {
			t.Fatalf("OnProgress(first) error = %v", err)
		}
		if err := accumulator.OnProgress(ctx, progressTestEvent(2, "Reading")); err != nil {
			t.Fatalf("OnProgress(second) error = %v", err)
		}
		wantErr := errors.New("content edit unavailable")
		sink.editError = wantErr

		err := accumulator.OnContent(ctx)
		assertProgressOperationError(t, err, wantErr, "edit progress bubble")
		if got, want := sink.recording.TypingCalls(), []bool{true, true, false}; fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("Typing calls after content flush failure = %v, want cleanup %v", got, want)
		}
	})

	t.Run("Should propagate a typing-stop failure and retry content cleanup", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("typing stop unavailable")
		sink := newProgressFailingSink()
		sink.typingStopError = wantErr
		config := progressTestConfig()
		config.ToolProgress = bridgepkg.ProgressModeNew
		config.Typing = true
		accumulator := NewProgressAccumulator(config, sink, nil)
		ctx := testutil.Context(t)
		if err := accumulator.OnProgress(ctx, progressTestEvent(1, "Inspecting")); err != nil {
			t.Fatalf("OnProgress() error = %v", err)
		}

		err := accumulator.OnContent(ctx)
		assertProgressOperationError(t, err, wantErr, "set progress typing state")
		if got, want := sink.recording.TypingCalls(), []bool{true}; fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("Typing calls after failed stop = %v, want %v", got, want)
		}

		sink.typingStopError = nil
		if err := accumulator.OnContent(ctx); err != nil {
			t.Fatalf("OnContent(retry) error = %v", err)
		}
		if got, want := sink.recording.TypingCalls(), []bool{true, false}; fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("Typing calls after retry = %v, want %v", got, want)
		}
	})

	t.Run("Should preserve the current bubble when an overflow flush edit fails", func(t *testing.T) {
		t.Parallel()

		clock := progressTestClock{current: time.Unix(9_000, 0)}
		sink := newProgressFailingSink()
		config := progressTestConfig()
		config.MaxMessageLen = 100
		accumulator := NewProgressAccumulator(config, sink, clock.Now)
		ctx := testutil.Context(t)
		for index, label := range []string{strings.Repeat("a", 10), strings.Repeat("b", 10)} {
			event := progressTestEvent(index+1, label)
			event.Emoji = ""
			event.Preview = ""
			if err := accumulator.OnProgress(ctx, event); err != nil {
				t.Fatalf("OnProgress(%d) error = %v", index+1, err)
			}
		}
		wantErr := errors.New("overflow edit unavailable")
		sink.editError = wantErr
		overflow := progressTestEvent(3, strings.Repeat("c", 20))
		overflow.Emoji = ""
		overflow.Preview = ""

		err := accumulator.OnProgress(ctx, overflow)
		assertProgressOperationError(t, err, wantErr, "edit progress bubble")
		if got, want := len(sink.PostCalls()), 1; got != want {
			t.Fatalf("Post calls after failed overflow flush = %d, want %d", got, want)
		}
	})
}

func TestProgressAccumulatorRejectsUnrepresentableProviderUnit(t *testing.T) {
	t.Parallel()

	t.Run("Should reject a provider length function where one rune exceeds the limit", func(t *testing.T) {
		t.Parallel()

		config := progressTestConfig()
		config.MaxMessageLen = 100
		config.Len = func(value string) int {
			if value == "" {
				return 0
			}
			return 100
		}
		sink := &progressRecordingSink{}
		accumulator := NewProgressAccumulator(config, sink, nil)

		err := accumulator.OnProgress(testutil.Context(t), progressTestEvent(1, "界"))
		if err == nil || !strings.Contains(err.Error(), "progress rune") {
			t.Fatalf("OnProgress() error = %v, want unrepresentable rune rejection", err)
		}
		if got := sink.TotalCalls(); got != 0 {
			t.Fatalf("platform calls = %d, want zero after provider limit rejection", got)
		}
	})

	t.Run("Should reject an unrepresentable rune under separate grouping", func(t *testing.T) {
		t.Parallel()

		config := progressTestConfig()
		config.Grouping = bridgepkg.ProgressGroupingSeparate
		config.MaxMessageLen = 100
		config.Len = func(value string) int {
			if value == "" {
				return 0
			}
			return 100
		}
		sink := &progressRecordingSink{}
		accumulator := NewProgressAccumulator(config, sink, nil)

		err := accumulator.OnProgress(testutil.Context(t), progressTestEvent(1, "界"))
		if err == nil || !strings.Contains(err.Error(), "progress rune") {
			t.Fatalf("OnProgress() error = %v, want unrepresentable rune rejection", err)
		}
		if got := sink.TotalCalls(); got != 0 {
			t.Fatalf("platform calls = %d, want zero after provider limit rejection", got)
		}
	})
}

func TestProgressAccumulatorFallsBackToToolIdentity(t *testing.T) {
	t.Parallel()

	t.Run("Should render the tool identifier when the presentation label is empty", func(t *testing.T) {
		t.Parallel()

		sink := &progressRecordingSink{}
		accumulator := NewProgressAccumulator(progressTestConfig(), sink, nil)
		event := progressTestEvent(1, "")
		if err := accumulator.OnProgress(testutil.Context(t), event); err != nil {
			t.Fatalf("OnProgress() error = %v", err)
		}

		posts := sink.PostCalls()
		if got, want := len(posts), 1; got != want {
			t.Fatalf("Post calls = %d, want %d", got, want)
		}
		if got, want := posts[0].text, "🔧 agh__test_tool item-1"; got != want {
			t.Fatalf("Post text = %q, want %q", got, want)
		}
	})
}

type progressTestClock struct {
	current time.Time
}

func (c *progressTestClock) Now() time.Time {
	return c.current
}

func (c *progressTestClock) Advance(duration time.Duration) {
	c.current = c.current.Add(duration)
}

type progressPostCall struct {
	target   bridgepkg.DeliveryTarget
	text     string
	remoteID string
}

type progressEditCall struct {
	target   bridgepkg.DeliveryTarget
	remoteID string
	text     string
}

type progressRecordingSink struct {
	mu        sync.Mutex
	posts     []progressPostCall
	edits     []progressEditCall
	typing    []bool
	reactions []string
}

type progressFailingSink struct {
	recording       *progressRecordingSink
	postError       error
	editError       error
	typingError     error
	typingStopError error
	reactionError   error
	emptyRemoteID   bool
}

func newProgressFailingSink() *progressFailingSink {
	return &progressFailingSink{recording: &progressRecordingSink{}}
}

func (s *progressFailingSink) Post(
	ctx context.Context,
	target bridgepkg.DeliveryTarget,
	text string,
) (string, error) {
	if s.postError != nil {
		return "", s.postError
	}
	if s.emptyRemoteID {
		return "", nil
	}
	return s.recording.Post(ctx, target, text)
}

func (s *progressFailingSink) Edit(
	ctx context.Context,
	target bridgepkg.DeliveryTarget,
	remoteID string,
	text string,
) error {
	if s.editError != nil {
		return s.editError
	}
	return s.recording.Edit(ctx, target, remoteID, text)
}

func (s *progressFailingSink) Typing(
	ctx context.Context,
	target bridgepkg.DeliveryTarget,
	active bool,
) error {
	if !active && s.typingStopError != nil {
		return s.typingStopError
	}
	if s.typingError != nil {
		return s.typingError
	}
	return s.recording.Typing(ctx, target, active)
}

func (s *progressFailingSink) React(
	ctx context.Context,
	target bridgepkg.DeliveryTarget,
	reaction string,
) error {
	if s.reactionError != nil {
		return s.reactionError
	}
	return s.recording.React(ctx, target, reaction)
}

func (s *progressFailingSink) TotalCalls() int {
	return s.recording.TotalCalls()
}

func (s *progressFailingSink) PostCalls() []progressPostCall {
	return s.recording.PostCalls()
}

func (s *progressRecordingSink) Post(
	_ context.Context,
	target bridgepkg.DeliveryTarget,
	text string,
) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	remoteID := fmt.Sprintf("progress-%d", len(s.posts)+1)
	s.posts = append(s.posts, progressPostCall{target: target, text: text, remoteID: remoteID})
	return remoteID, nil
}

func (s *progressRecordingSink) Edit(
	_ context.Context,
	target bridgepkg.DeliveryTarget,
	remoteID string,
	text string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.edits = append(s.edits, progressEditCall{target: target, remoteID: remoteID, text: text})
	return nil
}

func (s *progressRecordingSink) Typing(
	_ context.Context,
	_ bridgepkg.DeliveryTarget,
	active bool,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.typing = append(s.typing, active)
	return nil
}

func (s *progressRecordingSink) React(
	_ context.Context,
	_ bridgepkg.DeliveryTarget,
	reaction string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reactions = append(s.reactions, reaction)
	return nil
}

func (s *progressRecordingSink) PostCalls() []progressPostCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]progressPostCall(nil), s.posts...)
}

func (s *progressRecordingSink) EditCalls() []progressEditCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]progressEditCall(nil), s.edits...)
}

func (s *progressRecordingSink) TypingCalls() []bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]bool(nil), s.typing...)
}

func (s *progressRecordingSink) ReactionCalls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.reactions...)
}

func (s *progressRecordingSink) TotalCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.posts) + len(s.edits) + len(s.typing) + len(s.reactions)
}

func assertProgressOperationError(t *testing.T, err error, want error, message string) {
	t.Helper()

	if !errors.Is(err, want) {
		t.Fatalf("progress operation error = %v, want wrapping %v", err, want)
	}
	if !strings.Contains(err.Error(), message) {
		t.Fatalf("progress operation error = %q, want containing %q", err.Error(), message)
	}
}

func assertProgressCallsWithinLimit(t *testing.T, sink *progressRecordingSink, config ProgressConfig) {
	t.Helper()

	length := config.Len
	if length == nil {
		length = func(value string) int {
			return len(value)
		}
	}
	limit := config.MaxMessageLen - progressMessageHeadroom
	for _, call := range sink.PostCalls() {
		if got := length(call.text); got > limit {
			t.Fatalf("Post text length = %d, want <= %d: %q", got, limit, call.text)
		}
	}
	for _, call := range sink.EditCalls() {
		if got := length(call.text); got > limit {
			t.Fatalf("Edit text length = %d, want <= %d: %q", got, limit, call.text)
		}
	}
}

func assertProgressBubbleText(t *testing.T, sink *progressRecordingSink, want string) {
	t.Helper()

	posts := sink.PostCalls()
	texts := make([]string, len(posts))
	indices := make(map[string]int, len(posts))
	for index, call := range posts {
		texts[index] = call.text
		indices[call.remoteID] = index
	}
	for _, call := range sink.EditCalls() {
		index, ok := indices[call.remoteID]
		if !ok {
			t.Fatalf("Edit remote ID = %q, want a posted bubble", call.remoteID)
		}
		texts[index] = call.text
	}
	if got := strings.Join(texts, ""); got != want {
		t.Fatalf("provider bubble text = %q, want %q", got, want)
	}
}

func progressTestConfig() ProgressConfig {
	return ProgressConfig{
		ProgressConfig: bridgepkg.ProgressConfig{
			ToolProgress: bridgepkg.ProgressModeAll,
			Grouping:     bridgepkg.ProgressGroupingAccumulate,
		},
		Target: bridgepkg.DeliveryTarget{
			BridgeInstanceID: "bridge-1",
			PeerID:           "peer-1",
			Mode:             bridgepkg.DeliveryModeDirectSend,
		},
		MaxMessageLen: 4_000,
	}
}

func progressTestEvent(index int, label string) bridgepkg.ToolProgress {
	return bridgepkg.ToolProgress{
		ToolCallID: fmt.Sprintf("call-%d", index),
		ToolID:     "agh__test_tool",
		Phase:      bridgepkg.ToolProgressPhaseStarted,
		Label:      label,
		Preview:    fmt.Sprintf("item-%d", index),
		Emoji:      "🔧",
		Index:      index,
	}
}

func progressTestEventWithPhase(index int, phase bridgepkg.ToolProgressPhase) bridgepkg.ToolProgress {
	event := progressTestEvent(index, "Inspecting")
	event.Phase = phase
	return event
}
