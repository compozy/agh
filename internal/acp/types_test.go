package acp

import (
	"strings"
	"testing"
	"time"
)

func TestEndPromptClearsActivePromptWhileEmitterIsBackpressured(t *testing.T) {
	t.Parallel()

	proc := &AgentProcess{}
	active, err := proc.beginPrompt("turn-1", 1)
	if err != nil {
		t.Fatalf("beginPrompt() error = %v", err)
	}

	active.events <- AgentEvent{Type: EventTypeAgentMessage, TurnID: "turn-1"}

	emitterStarted := make(chan struct{})
	emitterDone := make(chan struct{})
	go func() {
		close(emitterStarted)
		proc.emitPromptEvent(AgentEvent{Type: EventTypeDone, TurnID: "turn-1"})
		close(emitterDone)
	}()
	<-emitterStarted

	endDone := make(chan struct{})
	go func() {
		proc.endPrompt(active)
		close(endDone)
	}()

	deadline := time.Now().Add(200 * time.Millisecond)
	for proc.currentPrompt() != nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if proc.currentPrompt() != nil {
		t.Fatal("currentPrompt() remained non-nil while endPrompt waited on a backpressured sender")
	}

	select {
	case <-endDone:
		t.Fatal("endPrompt() returned before the blocked emitPromptEvent() send was able to flush")
	default:
	}

	first := <-active.events
	if first.Type != EventTypeAgentMessage {
		t.Fatalf("first queued event = %q, want %q", first.Type, EventTypeAgentMessage)
	}

	select {
	case second, ok := <-active.events:
		if !ok {
			t.Fatal("active.events closed before backpressured event was delivered")
		}
		if second.Type != EventTypeDone {
			t.Fatalf("second queued event = %q, want %q", second.Type, EventTypeDone)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for backpressured emitPromptEvent() send to complete")
	}

	select {
	case <-emitterDone:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("emitPromptEvent() did not return after the queue drained")
	}

	select {
	case <-endDone:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("endPrompt() did not finish after the queue drained")
	}

	if _, ok := <-active.events; ok {
		t.Fatal("active.events remained open after endPrompt()")
	}
}

func TestLockedBufferRetainsOnlyTheLatestBytes(t *testing.T) {
	t.Parallel()

	buffer := &lockedBuffer{}
	payload := strings.Repeat("a", defaultTerminalOutputLimit) + "tail"
	if _, err := buffer.Write([]byte(payload)); err != nil {
		t.Fatalf("lockedBuffer.Write() error = %v", err)
	}

	got := buffer.String()
	if len(got) != defaultTerminalOutputLimit {
		t.Fatalf("len(buffer.String()) = %d, want %d", len(got), defaultTerminalOutputLimit)
	}
	if !strings.HasSuffix(got, "tail") {
		t.Fatalf("buffer.String() suffix = %q, want tail", got[len(got)-4:])
	}
	if strings.HasPrefix(got, "aa") {
		t.Logf("buffer head retained latest window as expected")
	}
}

func TestWaitForPromptQuiescenceHasMaximumDuration(t *testing.T) {
	t.Parallel()

	driver := New(WithPromptDrainWait(20 * time.Millisecond))
	active := &activePromptState{
		activity: make(chan struct{}, 1),
	}

	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				select {
				case active.activity <- struct{}{}:
				default:
				}
			}
		}
	}()
	defer close(stop)

	started := time.Now()
	driver.waitForPromptQuiescence(active)
	elapsed := time.Since(started)

	if elapsed > 120*time.Millisecond {
		t.Fatalf("waitForPromptQuiescence() took %v, want bounded wait", elapsed)
	}
}
