package tools

import (
	"strings"
	"testing"
	"time"
)

func TestApprovalTokenStore(t *testing.T) {
	t.Parallel()

	t.Run("ShouldMintConsumeAndRejectReplay", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
		clock := func() time.Time {
			return now
		}
		store := NewApprovalTokenStore(
			time.Minute,
			WithApprovalTokenClock(clock),
			WithApprovalTokenRandom(strings.NewReader(strings.Repeat("a", approvalTokenBytes*4))),
		)
		scope := Scope{Operator: true, SessionID: "sess-1", WorkspaceID: "ws-1"}

		grant, err := store.CreateToolApproval(t.Context(), scope, ApprovalRequest{
			ToolID: ToolIDSkillView,
			Input:  []byte(`{"message":"hello"}`),
		})
		if err != nil {
			t.Fatalf("CreateToolApproval() error = %v", err)
		}
		if grant.ApprovalToken == "" || grant.InputDigest == "" || !grant.ExpiresAt.Equal(now.Add(time.Minute)) {
			t.Fatalf("approval grant = %#v, want token, digest, and expiry", grant)
		}

		call := CallRequest{
			ToolID:        ToolIDSkillView,
			SessionID:     "sess-1",
			WorkspaceID:   "ws-1",
			Input:         []byte(`{"message":"hello"}`),
			ApprovalToken: grant.ApprovalToken,
		}
		if err := store.ConsumeToolApproval(t.Context(), scope, call); err != nil {
			t.Fatalf("ConsumeToolApproval() error = %v", err)
		}
		requireToolReason(t, store.ConsumeToolApproval(t.Context(), scope, call), ReasonApprovalTokenReplayed)
	})
}

func TestApprovalTokenStoreRejectsMismatchedAndExpiredTokens(t *testing.T) {
	t.Parallel()

	t.Run("ShouldRejectMismatchedAndExpiredTokens", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
		store := NewApprovalTokenStore(
			time.Second,
			WithApprovalTokenClock(func() time.Time { return now }),
			WithApprovalTokenRandom(strings.NewReader(strings.Repeat("b", approvalTokenBytes*4))),
		)
		scope := Scope{Operator: true, SessionID: "sess-1", WorkspaceID: "ws-1"}
		grant, err := store.CreateToolApproval(t.Context(), scope, ApprovalRequest{
			ToolID: ToolIDSkillView,
			Input:  []byte(`{"message":"hello"}`),
		})
		if err != nil {
			t.Fatalf("CreateToolApproval() error = %v", err)
		}

		mismatched := CallRequest{
			ToolID:        ToolIDSkillView,
			SessionID:     "sess-1",
			WorkspaceID:   "ws-2",
			Input:         []byte(`{"message":"hello"}`),
			ApprovalToken: grant.ApprovalToken,
		}
		requireToolReason(t, store.ConsumeToolApproval(t.Context(), scope, mismatched), ReasonApprovalTokenMismatch)

		mismatchedAgent := CallRequest{
			ToolID:        ToolIDSkillView,
			SessionID:     "sess-1",
			WorkspaceID:   "ws-1",
			AgentName:     "reviewer",
			Input:         []byte(`{"message":"hello"}`),
			ApprovalToken: grant.ApprovalToken,
		}
		requireToolReason(
			t,
			store.ConsumeToolApproval(t.Context(), scope, mismatchedAgent),
			ReasonApprovalTokenMismatch,
		)

		_, err = store.CreateToolApproval(t.Context(), Scope{SessionID: "sess-1"}, ApprovalRequest{
			ToolID:    ToolIDSkillView,
			SessionID: "sess-2",
			Input:     []byte(`{"message":"hello"}`),
		})
		requireToolReason(t, err, ReasonApprovalTokenMismatch)

		now = now.Add(2 * time.Second)
		expired := CallRequest{
			ToolID:        ToolIDSkillView,
			SessionID:     "sess-1",
			WorkspaceID:   "ws-1",
			Input:         []byte(`{"message":"hello"}`),
			ApprovalToken: grant.ApprovalToken,
		}
		requireToolReason(t, store.ConsumeToolApproval(t.Context(), scope, expired), ReasonApprovalTokenExpired)
	})
}

func requireToolReason(t *testing.T, err error, reason ReasonCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want reason %q", reason)
	}
	got, ok := ReasonOf(err)
	if !ok {
		t.Fatalf("ReasonOf(%v) not found, want %q", err, reason)
	}
	if got != reason {
		t.Fatalf("ReasonOf(%v) = %q, want %q", err, got, reason)
	}
}
