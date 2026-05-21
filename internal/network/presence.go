package network

import "time"

// PresenceState is the daemon-derived activity state for one network peer.
type PresenceState string

const (
	PresenceStateLocal    PresenceState = "local"
	PresenceStateActive   PresenceState = "active"
	PresenceStateInactive PresenceState = "inactive"
	PresenceStateExpired  PresenceState = "expired"
	PresenceStateUnknown  PresenceState = "unknown"
)

// Presence captures derived activity state without introducing a second source
// of truth beyond PeerRegistry timestamps.
type Presence struct {
	State              PresenceState
	LastSeenAgeSeconds *int64
}

// DerivePresence derives a peer's activity state from one captured clock value
// and the network greet interval. It is intentionally pure: callers own
// snapshots and time capture.
func DerivePresence(peer PeerInfo, now time.Time, greetInterval time.Duration) Presence {
	if peer.Local {
		return Presence{State: PresenceStateLocal}
	}
	if peer.LastSeen == nil || peer.LastSeen.IsZero() || greetInterval <= 0 {
		return Presence{State: PresenceStateUnknown}
	}

	if now.IsZero() {
		return Presence{State: PresenceStateUnknown}
	}
	now = now.UTC()
	lastSeen := peer.LastSeen.UTC()
	age := max(now.Sub(lastSeen), 0)
	ageSeconds := int64(age.Seconds())
	presence := Presence{LastSeenAgeSeconds: &ageSeconds}

	switch {
	case age <= greetInterval:
		presence.State = PresenceStateActive
	case age <= 2*greetInterval:
		presence.State = PresenceStateInactive
	default:
		presence.State = PresenceStateExpired
	}
	return presence
}

func applyPresence(peer PeerInfo, now time.Time, greetInterval time.Duration) PeerInfo {
	presence := DerivePresence(peer, now, greetInterval)
	peer.PresenceState = presence.State
	peer.LastSeenAgeSeconds = cloneInt64Ptr(presence.LastSeenAgeSeconds)
	return peer
}
