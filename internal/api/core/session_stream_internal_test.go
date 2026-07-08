package core

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/transcript"
	"github.com/gin-gonic/gin"
)

type streamTestFlushWriter struct {
	bytes.Buffer
}

func (streamTestFlushWriter) Flush() {}

func streamTestSessionInfo(id string) *session.Info {
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	return &session.Info{
		ID:          id,
		Name:        "demo",
		AgentName:   "coder",
		WorkspaceID: "ws-workspace",
		Workspace:   "/workspace",
		State:       session.StateActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func TestTranscriptSnapshotReset(t *testing.T) {
	t.Parallel()

	t.Run("Should reset initial subscriptions with subscribe reason", func(t *testing.T) {
		t.Parallel()

		reset, reason := transcriptSnapshotReset(0, 1, 9, session.TranscriptWatermark{})
		if !reset {
			t.Fatal("transcriptSnapshotReset() reset = false, want true")
		}
		if reason != contract.TranscriptSnapshotReasonSubscribe {
			t.Fatalf(
				"transcriptSnapshotReset() reason = %q, want %q",
				reason,
				contract.TranscriptSnapshotReasonSubscribe,
			)
		}
	})

	t.Run("Should report cache rebuild reason for initial subscriptions below watermark", func(t *testing.T) {
		t.Parallel()

		reset, reason := transcriptSnapshotReset(0, 1, 9, session.TranscriptWatermark{
			Sequence: 9,
			Reason:   session.TranscriptWatermarkReasonCacheRebuild,
		})
		if !reset {
			t.Fatal("transcriptSnapshotReset() reset = false, want true")
		}
		if reason != contract.TranscriptSnapshotReasonCacheRebuild {
			t.Fatalf(
				"transcriptSnapshotReset() reason = %q, want %q",
				reason,
				contract.TranscriptSnapshotReasonCacheRebuild,
			)
		}
	})

	t.Run("Should reset stale cursors above the snapshot max as epoch reset", func(t *testing.T) {
		t.Parallel()

		reset, reason := transcriptSnapshotReset(10, 1, 9, session.TranscriptWatermark{
			Sequence: 12,
			Reason:   session.TranscriptWatermarkReasonBelowWindowMutation,
		})
		if !reset {
			t.Fatal("transcriptSnapshotReset() reset = false, want true")
		}
		if reason != contract.TranscriptSnapshotReasonEpochReset {
			t.Fatalf(
				"transcriptSnapshotReset() reason = %q, want %q",
				reason,
				contract.TranscriptSnapshotReasonEpochReset,
			)
		}
	})

	t.Run("Should reset stale cursors above an empty snapshot as epoch reset", func(t *testing.T) {
		t.Parallel()

		reset, reason := transcriptSnapshotReset(10, 0, 0, session.TranscriptWatermark{})
		if !reset {
			t.Fatal("transcriptSnapshotReset() reset = false, want true")
		}
		if reason != contract.TranscriptSnapshotReasonEpochReset {
			t.Fatalf(
				"transcriptSnapshotReset() reason = %q, want %q",
				reason,
				contract.TranscriptSnapshotReasonEpochReset,
			)
		}
	})

	t.Run("Should reset cursors below cache rebuild watermark with cache rebuild reason", func(t *testing.T) {
		t.Parallel()

		reset, reason := transcriptSnapshotReset(8, 4, 12, session.TranscriptWatermark{
			Sequence: 8,
			Reason:   session.TranscriptWatermarkReasonCacheRebuild,
		})
		if !reset {
			t.Fatal("transcriptSnapshotReset() reset = false, want true")
		}
		if reason != contract.TranscriptSnapshotReasonCacheRebuild {
			t.Fatalf(
				"transcriptSnapshotReset() reason = %q, want %q",
				reason,
				contract.TranscriptSnapshotReasonCacheRebuild,
			)
		}
	})

	t.Run("Should reset cursors below mutation watermark with mutation reason", func(t *testing.T) {
		t.Parallel()

		reset, reason := transcriptSnapshotReset(8, 4, 12, session.TranscriptWatermark{
			Sequence: 9,
			Reason:   session.TranscriptWatermarkReasonBelowWindowMutation,
		})
		if !reset {
			t.Fatal("transcriptSnapshotReset() reset = false, want true")
		}
		if reason != contract.TranscriptSnapshotReasonBelowWindowMutation {
			t.Fatalf(
				"transcriptSnapshotReset() reason = %q, want %q",
				reason,
				contract.TranscriptSnapshotReasonBelowWindowMutation,
			)
		}
	})

	t.Run("Should keep cursors above mutation watermark on delta path", func(t *testing.T) {
		t.Parallel()

		reset, reason := transcriptSnapshotReset(10, 4, 12, session.TranscriptWatermark{
			Sequence: 9,
			Reason:   session.TranscriptWatermarkReasonBelowWindowMutation,
		})
		if reset {
			t.Fatal("transcriptSnapshotReset() reset = true, want false")
		}
		if reason != contract.TranscriptSnapshotReasonReconnect {
			t.Fatalf(
				"transcriptSnapshotReset() reason = %q, want %q",
				reason,
				contract.TranscriptSnapshotReasonReconnect,
			)
		}
	})

	t.Run("Should reset cursor gaps below the snapshot window", func(t *testing.T) {
		t.Parallel()

		reset, reason := transcriptSnapshotReset(2, 5, 12, session.TranscriptWatermark{})
		if !reset {
			t.Fatal("transcriptSnapshotReset() reset = false, want true")
		}
		if reason != contract.TranscriptSnapshotReasonReconnect {
			t.Fatalf(
				"transcriptSnapshotReset() reason = %q, want %q",
				reason,
				contract.TranscriptSnapshotReasonReconnect,
			)
		}
	})
}

func TestStreamTranscriptSessionEvents(t *testing.T) {
	t.Parallel()

	t.Run("Should advance cursor to highest transcript delta emitted from burst reads", func(t *testing.T) {
		t.Parallel()

		entries := []transcript.Entry{
			streamTestTranscriptEntry(8, "msg-8"),
			streamTestTranscriptEntry(9, "msg-9"),
			streamTestTranscriptEntry(10, "msg-10"),
		}
		handlers := &BaseHandlers{
			Sessions: sessionManagerStub{
				transcript: func(_ context.Context, _ string, query store.EventQuery) ([]transcript.Entry, error) {
					var out []transcript.Entry
					for _, entry := range entries {
						if entry.Sequence > query.AfterSequence {
							out = append(out, entry)
						}
					}
					return out, nil
				},
			},
		}
		writer := &streamTestFlushWriter{}

		nextSequence, err := handlers.writeTranscriptDeltasForEvents(
			context.Background(),
			writer,
			"sess-a",
			streamTestSessionInfo("sess-a"),
			[]store.SessionEvent{{Sequence: 8}},
			7,
		)
		if err != nil {
			t.Fatalf("writeTranscriptDeltasForEvents(first) error = %v", err)
		}
		if got, want := nextSequence, int64(10); got != want {
			t.Fatalf("writeTranscriptDeltasForEvents(first) sequence = %d, want %d", got, want)
		}

		nextSequence, err = handlers.writeTranscriptDeltasForEvents(
			context.Background(),
			writer,
			"sess-a",
			streamTestSessionInfo("sess-a"),
			[]store.SessionEvent{{Sequence: 9}},
			nextSequence,
		)
		if err != nil {
			t.Fatalf("writeTranscriptDeltasForEvents(second) error = %v", err)
		}
		if got, want := nextSequence, int64(10); got != want {
			t.Fatalf("writeTranscriptDeltasForEvents(second) sequence = %d, want %d", got, want)
		}

		body := writer.String()
		if got, want := strings.Count(body, "event: transcript_delta"), 3; got != want {
			t.Fatalf("transcript_delta frames = %d, want %d\n%s", got, want, body)
		}
		for _, id := range []string{"id: 8", "id: 9", "id: 10"} {
			if got := strings.Count(body, id); got != 1 {
				t.Fatalf("frame %q count = %d, want 1\n%s", id, got, body)
			}
		}
	})

	t.Run("Should rebase poll cursor to snapshot max after epoch reset snapshot", func(t *testing.T) {
		t.Parallel()

		done := make(chan struct{})
		var closeDone sync.Once
		var pollAfter atomic.Int64
		handlers := &BaseHandlers{
			Sessions: sessionManagerStub{
				transcript: func(_ context.Context, _ string, query store.EventQuery) ([]transcript.Entry, error) {
					if query.Limit != defaultSessionReadLimit {
						t.Fatalf("Transcript() query = %#v, want bounded snapshot query", query)
					}
					return []transcript.Entry{{
						Sequence: 5,
						Message: transcript.UIMessage{
							ID:   "msg-5",
							Role: transcript.UIRoleAssistant,
							Parts: []transcript.UIMessagePart{{
								Type:  "text",
								Text:  "post-clear",
								State: "done",
							}},
						},
					}}, nil
				},
				events: func(_ context.Context, _ string, query store.EventQuery) ([]store.SessionEvent, error) {
					pollAfter.Store(query.AfterSequence)
					closeDone.Do(func() { close(done) })
					return nil, nil
				},
				status: func(_ context.Context, id string) (*session.Info, error) {
					return streamTestSessionInfo(id), nil
				},
			},
			PollInterval: time.Millisecond,
		}
		handlers.SetStreamDone(done)

		gin.SetMode(gin.TestMode)
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequestWithContext(context.Background(), "GET", "/stream", http.NoBody)
		writer := &streamTestFlushWriter{}

		handlers.streamTranscriptSessionEvents(
			ctx,
			writer,
			"sess-a",
			streamTestSessionInfo("sess-a"),
			store.EventQuery{AfterSequence: 100, Limit: 0},
			nil,
			sessionStreamOptions{
				frameMode:      contract.SessionStreamFrameTranscript,
				replaySnapshot: true,
			},
			sessionEventStreamSubscription{},
		)
		if got, want := pollAfter.Load(), int64(5); got != want {
			t.Fatalf("poll AfterSequence = %d, want %d after epoch-reset snapshot", got, want)
		}
		if body := writer.String(); !bytes.Contains([]byte(body), []byte(`"reason":"epoch_reset"`)) {
			t.Fatalf("snapshot stream body = %s, want epoch_reset reason", body)
		}
	})
}

func TestSessionStreamFallbackCancelsSubscriptionBeforePolling(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		run  func(*BaseHandlers, *gin.Context, FlushWriter, sessionEventStreamSubscription)
	}{
		{
			name: "Should cancel raw subscription before polling after closed subscription",
			run: func(
				handlers *BaseHandlers,
				ctx *gin.Context,
				writer FlushWriter,
				subscription sessionEventStreamSubscription,
			) {
				handlers.pushAndStreamSessionEvents(
					ctx,
					writer,
					"sess-a",
					streamTestSessionInfo("sess-a"),
					store.EventQuery{},
					1,
					subscription,
				)
			},
		},
		{
			name: "Should cancel raw subscription before polling after sequence gap",
			run: func(
				handlers *BaseHandlers,
				ctx *gin.Context,
				writer FlushWriter,
				subscription sessionEventStreamSubscription,
			) {
				handlers.pushAndStreamSessionEvents(
					ctx,
					writer,
					"sess-a",
					streamTestSessionInfo("sess-a"),
					store.EventQuery{},
					1,
					subscription,
				)
			},
		},
		{
			name: "Should cancel transcript subscription before polling after closed subscription",
			run: func(
				handlers *BaseHandlers,
				ctx *gin.Context,
				writer FlushWriter,
				subscription sessionEventStreamSubscription,
			) {
				handlers.pushAndStreamSessionTranscript(
					ctx,
					writer,
					"sess-a",
					streamTestSessionInfo("sess-a"),
					store.EventQuery{},
					1,
					subscription,
				)
			},
		},
		{
			name: "Should cancel transcript subscription before polling after sequence gap",
			run: func(
				handlers *BaseHandlers,
				ctx *gin.Context,
				writer FlushWriter,
				subscription sessionEventStreamSubscription,
			) {
				handlers.pushAndStreamSessionTranscript(
					ctx,
					writer,
					"sess-a",
					streamTestSessionInfo("sess-a"),
					store.EventQuery{},
					1,
					subscription,
				)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			streamDone := make(chan struct{})
			handlers := &BaseHandlers{PollInterval: time.Hour}
			handlers.SetStreamDone(streamDone)
			gin.SetMode(gin.TestMode)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequestWithContext(context.Background(), "GET", "/stream", http.NoBody)
			writer := &streamTestFlushWriter{}

			events := make(chan store.SessionEvent, 1)
			if strings.Contains(testCase.name, "sequence gap") {
				events <- store.SessionEvent{Sequence: 3}
			}
			close(events)
			canceled := make(chan struct{})
			var cancelOnce sync.Once
			subscription := sessionEventStreamSubscription{
				events: events,
				cancel: func() {
					cancelOnce.Do(func() { close(canceled) })
				},
			}

			done := make(chan struct{})
			go func() {
				defer close(done)
				testCase.run(handlers, ctx, writer, subscription)
			}()

			waitForStreamTestSignal(t, canceled, "subscription cancel")
			close(streamDone)
			waitForStreamTestSignal(t, done, "stream fallback return")
		})
	}
}

func waitForStreamTestSignal(t *testing.T, signal <-chan struct{}, label string) {
	t.Helper()

	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-signal:
	case <-timer.C:
		t.Fatalf("timed out waiting for %s", label)
	}
}

func streamTestTranscriptEntry(sequence int64, id string) transcript.Entry {
	return transcript.Entry{
		Sequence: sequence,
		Message: transcript.UIMessage{
			ID:   id,
			Role: transcript.UIRoleAssistant,
			Parts: []transcript.UIMessagePart{{
				Type:  "text",
				Text:  id,
				State: "done",
			}},
		},
	}
}
