package core_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/core"
)

func TestPromptStreamEncoderPermissionDataPartIdentity(t *testing.T) {
	t.Run("ShouldReuseRequestIDForPendingAndFinalPermissionParts", func(t *testing.T) {
		t.Parallel()

		writer := &bufferFlusher{}
		encoder := core.NewPromptStreamEncoder(func() time.Time {
			return time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
		})

		pending := acp.AgentEvent{
			Type:      acp.EventTypePermission,
			SessionID: "sess-1",
			TurnID:    "turn-1",
			RequestID: "req-1",
			Title:     "Bash",
			Action:    "session/request_permission",
			Resource:  "Bash",
		}
		final := pending
		final.Decision = "allow-once"

		if err := encoder.Emit(writer, pending); err != nil {
			t.Fatalf("Emit(pending) error = %v", err)
		}
		if err := encoder.Emit(writer, final); err != nil {
			t.Fatalf("Emit(final) error = %v", err)
		}

		frames := promptPermissionFramesFromSSE(t, writer.String())
		if got, want := len(frames), 2; got != want {
			t.Fatalf("len(permission frames) = %d, want %d; frames=%#v", got, want, frames)
		}
		if got, want := frames[0].ID, "req-1"; got != want {
			t.Fatalf("pending frame ID = %q, want %q", got, want)
		}
		if got, want := frames[1].ID, "req-1"; got != want {
			t.Fatalf("final frame ID = %q, want %q", got, want)
		}
		if frames[0].Data.Decision != "" {
			t.Fatalf("pending decision = %q, want empty", frames[0].Data.Decision)
		}
		if got, want := frames[1].Data.Decision, "allow-once"; got != want {
			t.Fatalf("final decision = %q, want %q", got, want)
		}
	})
}

type promptPermissionFrame struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Data struct {
		RequestID string `json:"request_id"`
		Decision  string `json:"decision"`
	} `json:"data"`
}

func promptPermissionFramesFromSSE(t *testing.T, body string) []promptPermissionFrame {
	t.Helper()

	frames := make([]promptPermissionFrame, 0, 2)
	for record := range strings.SplitSeq(body, "\n\n") {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}
		data := ""
		for line := range strings.SplitSeq(record, "\n") {
			if after, ok := strings.CutPrefix(line, "data: "); ok {
				data += after
			}
		}
		if data == "" || data == "[DONE]" {
			continue
		}

		var frame promptPermissionFrame
		if err := json.Unmarshal([]byte(data), &frame); err != nil {
			t.Fatalf("json.Unmarshal(%q) error = %v", data, err)
		}
		if frame.Type == "data-agh-permission" {
			frames = append(frames, frame)
		}
	}
	return frames
}
