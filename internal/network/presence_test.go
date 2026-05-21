package network

import (
	"testing"
	"time"
)

func TestDerivePresence(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 21, 12, 0, 0, 0, time.UTC)
	greetInterval := 10 * time.Second
	tests := []struct {
		name        string
		peer        PeerInfo
		now         time.Time
		interval    time.Duration
		wantState   PresenceState
		wantAgeSecs *int64
	}{
		{
			name:      "Should classify local peers without last-seen age",
			peer:      PeerInfo{Local: true, LastSeen: presenceTimePtr(now.Add(-30 * time.Second))},
			now:       now,
			interval:  greetInterval,
			wantState: PresenceStateLocal,
		},
		{
			name:        "Should classify remote peers seen now as active",
			peer:        PeerInfo{LastSeen: presenceTimePtr(now)},
			now:         now,
			interval:    greetInterval,
			wantState:   PresenceStateActive,
			wantAgeSecs: presenceInt64Ptr(0),
		},
		{
			name:        "Should classify remote peers within one greet interval as active",
			peer:        PeerInfo{LastSeen: presenceTimePtr(now.Add(-5 * time.Second))},
			now:         now,
			interval:    greetInterval,
			wantState:   PresenceStateActive,
			wantAgeSecs: presenceInt64Ptr(5),
		},
		{
			name:        "Should classify remote peers at the active boundary as active",
			peer:        PeerInfo{LastSeen: presenceTimePtr(now.Add(-10 * time.Second))},
			now:         now,
			interval:    greetInterval,
			wantState:   PresenceStateActive,
			wantAgeSecs: presenceInt64Ptr(10),
		},
		{
			name:        "Should classify remote peers past one interval as inactive",
			peer:        PeerInfo{LastSeen: presenceTimePtr(now.Add(-11 * time.Second))},
			now:         now,
			interval:    greetInterval,
			wantState:   PresenceStateInactive,
			wantAgeSecs: presenceInt64Ptr(11),
		},
		{
			name:        "Should classify remote peers at the expiry boundary as inactive until swept",
			peer:        PeerInfo{LastSeen: presenceTimePtr(now.Add(-20 * time.Second))},
			now:         now,
			interval:    greetInterval,
			wantState:   PresenceStateInactive,
			wantAgeSecs: presenceInt64Ptr(20),
		},
		{
			name:        "Should classify remote peers beyond the expiry window as expired",
			peer:        PeerInfo{LastSeen: presenceTimePtr(now.Add(-21 * time.Second))},
			now:         now,
			interval:    greetInterval,
			wantState:   PresenceStateExpired,
			wantAgeSecs: presenceInt64Ptr(21),
		},
		{
			name:      "Should classify missing remote last-seen as unknown",
			peer:      PeerInfo{},
			now:       now,
			interval:  greetInterval,
			wantState: PresenceStateUnknown,
		},
		{
			name:      "Should classify zero remote last-seen as unknown",
			peer:      PeerInfo{LastSeen: presenceTimePtr(time.Time{})},
			now:       now,
			interval:  greetInterval,
			wantState: PresenceStateUnknown,
		},
		{
			name:      "Should classify remote peers without caller-supplied now as unknown",
			peer:      PeerInfo{LastSeen: presenceTimePtr(now)},
			interval:  greetInterval,
			wantState: PresenceStateUnknown,
		},
		{
			name:      "Should classify remote peers without greet interval as unknown",
			peer:      PeerInfo{LastSeen: presenceTimePtr(now)},
			now:       now,
			wantState: PresenceStateUnknown,
		},
		{
			name:        "Should clamp future last-seen age to zero",
			peer:        PeerInfo{LastSeen: presenceTimePtr(now.Add(5 * time.Second))},
			now:         now,
			interval:    greetInterval,
			wantState:   PresenceStateActive,
			wantAgeSecs: presenceInt64Ptr(0),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			presence := DerivePresence(tc.peer, tc.now, tc.interval)
			if presence.State != tc.wantState {
				t.Fatalf("State = %q, want %q", presence.State, tc.wantState)
			}
			assertOptionalPresenceAge(t, presence.LastSeenAgeSeconds, tc.wantAgeSecs)
		})
	}
}

func assertOptionalPresenceAge(t *testing.T, got *int64, want *int64) {
	t.Helper()

	if got == nil || want == nil {
		if got != want {
			t.Fatalf("LastSeenAgeSeconds = %#v, want %#v", got, want)
		}
		return
	}
	if *got != *want {
		t.Fatalf("LastSeenAgeSeconds = %d, want %d", *got, *want)
	}
}

func presenceTimePtr(value time.Time) *time.Time {
	copyValue := value.UTC()
	return &copyValue
}

func presenceInt64Ptr(value int64) *int64 {
	copyValue := value
	return &copyValue
}
