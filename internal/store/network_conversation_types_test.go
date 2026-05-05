package store

import (
	"strings"
	"testing"
	"time"
)

func TestNetworkConversationRefValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ref     NetworkConversationRef
		wantErr string
	}{
		{
			name: "Should accept thread references",
			ref: NetworkConversationRef{
				Channel:  "builders",
				Surface:  NetworkSurfaceThread,
				ThreadID: "thread_patch_42",
			},
		},
		{
			name: "Should accept direct references",
			ref: NetworkConversationRef{
				Channel:  "builders",
				Surface:  NetworkSurfaceDirect,
				DirectID: "direct_0123456789abcdef0123456789abcdef",
			},
		},
		{
			name: "Should reject missing thread ids",
			ref: NetworkConversationRef{
				Channel: "builders",
				Surface: NetworkSurfaceThread,
			},
			wantErr: "thread_id",
		},
		{
			name: "Should reject dual containers",
			ref: NetworkConversationRef{
				Channel:  "builders",
				Surface:  NetworkSurfaceThread,
				ThreadID: "thread_patch_42",
				DirectID: "direct_0123456789abcdef0123456789abcdef",
			},
			wantErr: "direct_id",
		},
		{
			name: "Should reject unknown surfaces",
			ref: NetworkConversationRef{
				Channel:  "builders",
				Surface:  "room",
				ThreadID: "thread_patch_42",
			},
			wantErr: "surface",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.ref.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeNetworkDirectRoomPeers(t *testing.T) {
	t.Parallel()

	t.Run("Should return peers in lexicographic order", func(t *testing.T) {
		t.Parallel()

		peerA, peerB, err := NormalizeNetworkDirectRoomPeers("reviewer.sess-xyz", "coder.sess-abc")
		if err != nil {
			t.Fatalf("NormalizeNetworkDirectRoomPeers() error = %v", err)
		}
		if got, want := peerA, "coder.sess-abc"; got != want {
			t.Fatalf("peerA = %q, want %q", got, want)
		}
		if got, want := peerB, "reviewer.sess-xyz"; got != want {
			t.Fatalf("peerB = %q, want %q", got, want)
		}
	})

	t.Run("Should reject same peers", func(t *testing.T) {
		t.Parallel()

		_, _, err := NormalizeNetworkDirectRoomPeers("coder.sess-abc", " coder.sess-abc ")
		if err == nil || !strings.Contains(err.Error(), "must differ") {
			t.Fatalf("NormalizeNetworkDirectRoomPeers() error = %v, want same-peer rejection", err)
		}
	})

	t.Run("Should reject invalid peers", func(t *testing.T) {
		t.Parallel()

		_, _, err := NormalizeNetworkDirectRoomPeers("coder/sess", "reviewer.sess-xyz")
		if err == nil || !strings.Contains(err.Error(), "peer_a") {
			t.Fatalf("NormalizeNetworkDirectRoomPeers() error = %v, want peer_a rejection", err)
		}
	})
}

func TestNetworkConversationSummaryValidation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)

	t.Run("Should validate thread summaries", func(t *testing.T) {
		t.Parallel()

		summary := NetworkThreadSummary{
			Channel:            "builders",
			ThreadID:           "thread_patch_42",
			RootMessageID:      "msg_patch_42",
			OpenedByPeerID:     "coder.sess-abc",
			OpenedAt:           now,
			LastActivityAt:     now,
			MessageCount:       1,
			ParticipantCount:   1,
			OpenWorkCount:      0,
			LastMessagePreview: "hello",
		}
		if err := summary.Validate(); err != nil {
			t.Fatalf("Validate(thread summary) error = %v", err)
		}

		summary.ParticipantCount = -1
		if err := summary.Validate(); err == nil || !strings.Contains(err.Error(), "participant_count") {
			t.Fatalf("Validate(thread summary) error = %v, want participant_count rejection", err)
		}
	})

	t.Run("Should validate direct room summaries", func(t *testing.T) {
		t.Parallel()

		summary := NetworkDirectRoomSummary{
			Channel:        "builders",
			DirectID:       "direct_0123456789abcdef0123456789abcdef",
			PeerA:          "coder.sess-abc",
			PeerB:          "reviewer.sess-xyz",
			OpenedAt:       now,
			LastActivityAt: now,
			MessageCount:   1,
			OpenWorkCount:  0,
		}
		if err := summary.Validate(); err != nil {
			t.Fatalf("Validate(direct summary) error = %v", err)
		}

		summary.PeerA = "reviewer.sess-xyz"
		summary.PeerB = "coder.sess-abc"
		if err := summary.Validate(); err == nil || !strings.Contains(err.Error(), "lexicographic") {
			t.Fatalf("Validate(direct summary) error = %v, want peer ordering rejection", err)
		}
	})
}

func TestNetworkDirectRoomEntryValidation(t *testing.T) {
	t.Parallel()

	t.Run("Should validate direct room write rows", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
		entry := NetworkDirectRoomEntry{
			Channel:        "builders",
			DirectID:       "direct_0123456789abcdef0123456789abcdef",
			PeerA:          "coder.sess-abc",
			PeerB:          "reviewer.sess-xyz",
			OpenedAt:       now,
			LastActivityAt: now,
		}
		if err := entry.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}

		entry.LastActivityAt = time.Time{}
		if err := entry.Validate(); err == nil || !strings.Contains(err.Error(), "last_activity_at") {
			t.Fatalf("Validate() error = %v, want last_activity_at rejection", err)
		}
	})
}

func TestNetworkWorkEntryValidation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	valid := NetworkWorkEntry{
		WorkID:         "work_patch_42",
		Channel:        "builders",
		Surface:        NetworkSurfaceThread,
		ThreadID:       "thread_patch_42",
		OpenedByPeerID: "coder.sess-abc",
		State:          NetworkWorkStateSubmitted,
		OpenedAt:       now,
		LastActivityAt: now,
	}

	tests := []struct {
		name    string
		mutate  func(*NetworkWorkEntry)
		wantErr string
	}{
		{
			name: "Should accept valid work rows",
		},
		{
			name: "Should reject dangling thread refs",
			mutate: func(entry *NetworkWorkEntry) {
				entry.ThreadID = ""
			},
			wantErr: "thread_id",
		},
		{
			name: "Should reject dual container refs",
			mutate: func(entry *NetworkWorkEntry) {
				entry.DirectID = "direct_0123456789abcdef0123456789abcdef"
			},
			wantErr: "direct_id",
		},
		{
			name: "Should reject terminal timestamps on active states",
			mutate: func(entry *NetworkWorkEntry) {
				entry.TerminalAt = &now
			},
			wantErr: "terminal_at",
		},
		{
			name: "Should reject unknown states",
			mutate: func(entry *NetworkWorkEntry) {
				entry.State = "paused"
			},
			wantErr: "state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entry := valid
			if tt.mutate != nil {
				tt.mutate(&entry)
			}
			err := entry.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestNetworkConversationQueryValidation(t *testing.T) {
	t.Parallel()

	t.Run("Should reject invalid summary limits", func(t *testing.T) {
		t.Parallel()

		if err := (NetworkThreadQuery{Limit: -1}).Validate(); err == nil || !strings.Contains(err.Error(), "limit") {
			t.Fatalf("Validate(thread query) error = %v, want limit rejection", err)
		}
		if err := (NetworkDirectRoomQuery{Limit: -1}).Validate(); err == nil ||
			!strings.Contains(err.Error(), "limit") {
			t.Fatalf("Validate(direct query) error = %v, want limit rejection", err)
		}
	})

	t.Run("Should reject dual message cursors and invalid work ids", func(t *testing.T) {
		t.Parallel()

		query := NetworkConversationMessageQuery{
			BeforeMessageID: "msg_before",
			AfterMessageID:  "msg_after",
			Limit:           10,
		}
		if err := query.Validate(); err == nil || !strings.Contains(err.Error(), "both before and after") {
			t.Fatalf("Validate(message query) error = %v, want cursor rejection", err)
		}

		query.BeforeMessageID = ""
		query.WorkID = "work/bad"
		if err := query.Validate(); err == nil || !strings.Contains(err.Error(), "work_id") {
			t.Fatalf("Validate(message query) error = %v, want work_id rejection", err)
		}
	})
}

func TestNetworkConversationMessageValidation(t *testing.T) {
	t.Parallel()

	valid := NetworkConversationMessage{
		MessageID:   "msg_patch_42",
		Channel:     "builders",
		Surface:     NetworkSurfaceDirect,
		DirectID:    "direct_0123456789abcdef0123456789abcdef",
		Direction:   "sent",
		PeerFrom:    "coder.sess-abc",
		PeerTo:      "reviewer.sess-xyz",
		Kind:        NetworkKindReceipt,
		WorkID:      "work_patch_42",
		PreviewText: "accepted",
		Body:        []byte(`{"status":"accepted"}`),
	}

	tests := []struct {
		name    string
		mutate  func(*NetworkConversationMessage)
		wantErr string
	}{
		{
			name: "Should accept conversation messages",
		},
		{
			name: "Should reject receipt without work",
			mutate: func(entry *NetworkConversationMessage) {
				entry.WorkID = ""
			},
			wantErr: "work_id",
		},
		{
			name: "Should reject greet with conversation fields",
			mutate: func(entry *NetworkConversationMessage) {
				entry.Kind = NetworkKindGreet
				entry.WorkID = ""
			},
			wantErr: "conversation",
		},
		{
			name: "Should reject messages without matching direct id",
			mutate: func(entry *NetworkConversationMessage) {
				entry.DirectID = ""
			},
			wantErr: "direct_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entry := valid
			if tt.mutate != nil {
				tt.mutate(&entry)
			}
			err := entry.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
